package purchase

import (
	"context"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/purchase/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

const orderType = "purchase.order"

func (s *Service) registerOrder() {
	s.d.Record.Register(record.Type{
		Code: orderType, Product: "purchase", Kind: record.Document, NumberPrefix: "DM",
		Fields: []record.Field{
			{Key: "items", Kind: record.Set, Label: "purchase.order.items", Search: s.itemOptions, Labels: s.itemLabels},
		},
		// Reading, printing and seeing files need view; every write, attaching included, needs edit.
		Perms:        record.Perms{View: PermOrderView, Edit: PermOrderEdit, ViewFiles: PermOrderView, Attach: PermOrderEdit},
		OnTransition: s.orderTransition,
		BeforeSubmit: s.orderBeforeSubmit,
		OnDelete:     func(ctx context.Context, id int64) error { return store.DeleteOrder.Run(ctx, id) },
	})
}

// OrderActions lists what the actor may do before picking an order: create.
func (s *Service) OrderActions(ctx context.Context) (Actions, error) {
	return s.typeActions(ctx, orderType)
}

// Orders lists the purchase orders of org units where the actor may view them.
func (s *Service) Orders(ctx context.Context, f OrderFilter) (OrderList, error) {
	out := OrderList{Items: []OrderListItem{}}
	sc, err := s.d.IAM.DataScope(ctx, "purchase", PermOrderView)
	if err != nil {
		return out, err
	}
	page, err := orderList.Run(ctx, orderQuery(sc, f))
	if err != nil {
		return out, err
	}
	out.Total = page.Total
	for _, r := range page.Items {
		out.Items = append(out.Items, OrderListItem{ID: r.ID, Number: r.Number, Status: r.Status, Date: *platform.DatePtr(r.Date),
			SupplierID: r.SupplierID, SupplierCode: r.SupplierCode, SupplierName: r.SupplierName, OrgUnitName: r.OrgUnitName, Total: r.Total})
	}
	return out, nil
}

// Order returns one purchase order; one the actor may not view does not exist.
func (s *Service) Order(ctx context.Context, id int64) (Order, error) {
	d, err := s.d.Record.Read(ctx, record.Ref{Type: orderType, ID: id})
	if err != nil {
		return Order{}, err
	}
	h, err := store.GetOrder.Run(ctx, id)
	if err != nil {
		return Order{}, err
	}
	lines, err := store.GetOrderLines.Run(ctx, id)
	if err != nil {
		return Order{}, err
	}
	out := Order{
		ID: id, Number: h.Number, Status: h.Status, Version: h.Version, Date: *platform.DatePtr(h.Date),
		OrgUnitID: h.OrgUnitID, OrgUnitName: h.OrgUnitName, Supplier: supplierCopy(h),
		ExpectedDate: platform.DatePtr(h.ExpectedDate), PaymentTerms: platform.TextPtr(h.PaymentTerms), Note: platform.TextPtr(h.Note),
		Lines:  make([]Line, len(lines)),
		Totals: Totals{Subtotal: h.Subtotal, DiscountTotal: h.DiscountTotal, VatTotal: h.VatTotal, Total: h.Total},
	}
	for i, l := range lines {
		out.Lines[i] = Line{
			LineInput: LineInput{ItemID: l.ItemID, Description: l.Description, Quantity: platform.PlainDecimal(l.Quantity), UnitPrice: l.UnitPrice,
				DiscountPercent: platform.PlainDecimal(l.DiscountPercent), VatRate: l.VatRate},
			ItemCode: l.ItemCode, Unit: l.Unit, Stocked: l.Stocked, Amount: l.Amount, Discount: l.Discount, Vat: l.Vat,
		}
	}
	out.AllowedActions, err = s.d.Record.Actions(ctx, d)
	return out, err
}

// CreateOrder adds a draft; the same request id returns the first one.
func (s *Service) CreateOrder(ctx context.Context, in NewOrder) (id int64, err error) {
	return id, platform.InTx(ctx, func(ctx context.Context) error {
		if err := s.d.Record.MayCreate(ctx, orderType, record.Header{OrgUnitID: in.OrgUnitID}); err != nil {
			return err
		}
		p, err := s.prepareOrder(ctx, in.OrderFields)
		if err != nil {
			return err
		}
		p.header.RequestID = in.RequestID
		d, err := s.d.Record.Create(ctx, orderType, p.header)
		if err != nil || d.Reused {
			id = d.ID
			return err
		}
		id = d.ID
		c, f := p.supplier, in.OrderFields
		if err := store.InsertOrder.Run(ctx, store.InsertOrderParams{
			ID: id, SupplierID: c.ID, SupplierCode: c.Code, SupplierName: c.Name,
			SupplierTaxCode: platform.NullText(c.TaxCode), SupplierAddress: platform.NullText(c.Address), SupplierPhone: platform.NullText(c.Phone),
			SupplierEmail: platform.NullText(c.Email), ContactName: platform.NullText(c.ContactName),
			ExpectedDate: platform.NullDate(f.ExpectedDate), PaymentTerms: platform.NullText(f.PaymentTerms), Note: platform.NullText(f.Note),
			Subtotal: p.totals.Subtotal, DiscountTotal: p.totals.DiscountTotal, VatTotal: p.totals.VatTotal, Total: p.totals.Total,
		}); err != nil {
			return err
		}
		return s.writeOrderLines(ctx, id, p)
	})
}

func (s *Service) UpdateOrder(ctx context.Context, id int64, in OrderUpdate) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		if _, err := s.d.Record.Read(ctx, record.Ref{Type: orderType, ID: id}); err != nil {
			return err
		}
		p, err := s.prepareOrder(ctx, in.OrderFields)
		if err != nil {
			return err
		}
		if _, err := s.d.Record.Edit(ctx, record.Ref{Type: orderType, ID: id}, in.Version, p.header); err != nil {
			return err
		}
		c, f := p.supplier, in.OrderFields
		if err := store.UpdateOrder.Run(ctx, store.UpdateOrderParams{
			ID: id, SupplierID: c.ID, SupplierCode: c.Code, SupplierName: c.Name,
			SupplierTaxCode: platform.NullText(c.TaxCode), SupplierAddress: platform.NullText(c.Address), SupplierPhone: platform.NullText(c.Phone),
			SupplierEmail: platform.NullText(c.Email), ContactName: platform.NullText(c.ContactName),
			ExpectedDate: platform.NullDate(f.ExpectedDate), PaymentTerms: platform.NullText(f.PaymentTerms), Note: platform.NullText(f.Note),
			Subtotal: p.totals.Subtotal, DiscountTotal: p.totals.DiscountTotal, VatTotal: p.totals.VatTotal, Total: p.totals.Total,
		}); err != nil {
			return err
		}
		if err := store.DeleteOrderLines.Run(ctx, id); err != nil {
			return err
		}
		return s.writeOrderLines(ctx, id, p)
	})
}

// preparedOrder is a draft's content, checked against the current catalogues and computed.
type preparedOrder struct {
	header   record.Header
	supplier SupplierCopy
	lines    []Line
	totals   Totals
}

// prepareOrder checks the fields, copies the supplier and items as they are now, and computes
// every amount with the rounding mode of the org unit's legal entity.
func (s *Service) prepareOrder(ctx context.Context, f OrderFields) (preparedOrder, error) {
	var p preparedOrder
	if f.ExpectedDate != nil && *f.ExpectedDate < f.Date {
		return p, ErrExpectedDate
	}
	if err := s.d.Record.Visible(ctx, record.Ref{Type: supplierType, ID: f.SupplierID}, record.View); err != nil {
		return p, err
	}
	c, err := store.GetSupplier.Run(ctx, f.SupplierID)
	if err != nil {
		return p, err
	}
	if !c.Active {
		return p, ErrSupplierInactive
	}
	p.supplier = SupplierCopy{ID: c.ID, Code: c.Code, Name: c.Name, TaxCode: platform.TextPtr(c.TaxCode), Address: platform.TextPtr(c.Address),
		Phone: platform.TextPtr(c.Phone), Email: platform.TextPtr(c.Email), ContactName: platform.TextPtr(c.ContactName)}
	ids := make([]int64, len(f.Lines))
	for i, l := range f.Lines {
		ids[i] = l.ItemID
	}
	rows, err := store.ItemsByID.Run(ctx, ids)
	if err != nil {
		return p, err
	}
	items := map[int64]store.ItemItem{}
	for _, r := range rows {
		items[r.ID] = r
	}
	for i, l := range f.Lines {
		if it, ok := items[l.ItemID]; !ok || !it.Active {
			return p, errLine("item_inactive", i+1)
		}
	}
	mode, err := s.d.Setting.RoundingOf(ctx, f.OrgUnitID)
	if err != nil {
		return p, err
	}
	amts, totals, err := compute(f.Lines, mode)
	if err != nil {
		return p, err
	}
	p.lines = make([]Line, len(f.Lines))
	for i, l := range f.Lines {
		it := items[l.ItemID]
		p.lines[i] = Line{LineInput: l, ItemCode: it.Code, Unit: it.Unit, Stocked: it.Tracking != "none",
			Amount: amts[i].amount, Discount: amts[i].discount, Vat: amts[i].vat}
	}
	p.totals = totals
	p.header = record.Header{Date: f.Date, OrgUnitID: f.OrgUnitID, Amount: &totals.Total,
		Fields: map[string]string{"items": record.SetValue(ids)}}
	return p, nil
}

func (s *Service) writeOrderLines(ctx context.Context, id int64, p preparedOrder) error {
	a := store.InsertOrderLinesParams{OrderID: id}
	for i, l := range p.lines {
		a.Positions = append(a.Positions, int32(i+1))
		a.ItemIds = append(a.ItemIds, l.ItemID)
		a.ItemCodes = append(a.ItemCodes, l.ItemCode)
		a.Descriptions = append(a.Descriptions, l.Description)
		a.Units = append(a.Units, l.Unit)
		a.Stocked = append(a.Stocked, l.Stocked)
		a.Quantities = append(a.Quantities, l.Quantity)
		a.UnitPrices = append(a.UnitPrices, l.UnitPrice)
		a.DiscountPercents = append(a.DiscountPercents, l.DiscountPercent)
		a.VatRates = append(a.VatRates, l.VatRate)
		a.Amounts = append(a.Amounts, l.Amount)
		a.Discounts = append(a.Discounts, l.Discount)
		a.Vats = append(a.Vats, l.Vat)
	}
	if err := store.InsertOrderLines.Run(ctx, a); err != nil {
		return err
	}
	return s.d.Audit.RecordFor(ctx, "purchase.order_saved", audit.Ref{Type: orderType, ID: id},
		map[string]any{"supplier_id": p.supplier.ID, "lines": len(p.lines), "total": p.totals.Total})
}

// orderTransition keeps posted orders to active suppliers.
func (s *Service) orderTransition(ctx context.Context, d record.Doc, _ record.Status) error {
	if d.Status != record.Posted {
		return nil
	}
	h, err := store.GetOrder.Run(ctx, d.ID)
	if err != nil {
		return err
	}
	active, err := store.SupplierActive.Run(ctx, h.SupplierID)
	if err != nil {
		return err
	}
	if !active {
		return ErrSupplierInactive
	}
	return nil
}

// orderBeforeSubmit copies the supplier's current details onto a draft being sent, so what goes
// to approval, and what posts, is the supplier as it is at sending; an inactive one keeps it a draft.
func (s *Service) orderBeforeSubmit(ctx context.Context, d record.Doc) error {
	active, err := store.CopySupplier.Run(ctx, d.ID)
	if err != nil {
		return err
	}
	if !active {
		return ErrSupplierInactive
	}
	return nil
}

func supplierCopy(h store.GetOrderRow) SupplierCopy {
	return SupplierCopy{ID: h.SupplierID, Code: h.SupplierCode, Name: h.SupplierName, TaxCode: platform.TextPtr(h.SupplierTaxCode),
		Address: platform.TextPtr(h.SupplierAddress), Phone: platform.TextPtr(h.SupplierPhone), Email: platform.TextPtr(h.SupplierEmail),
		ContactName: platform.TextPtr(h.ContactName)}
}

// canWriteOrders: Purchasing is on and the actor may write orders at the unit, which is also what
// writing their goods receipts, invoices and payments needs.
func (s *Service) canWriteOrders(ctx context.Context, unit int64) (bool, error) {
	if platform.ProductGate(ctx, "purchase", platform.ClassWrite) != nil {
		return false, nil
	}
	return s.d.IAM.Allowed(ctx, "purchase", PermOrderEdit, unit)
}
