package sales

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/sales/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// QuoteActions and OrderActions list what the actor may do before picking a document: create.
func (s *Service) QuoteActions(ctx context.Context) (Actions, error) {
	return s.typeActions(ctx, quoteType)
}
func (s *Service) OrderActions(ctx context.Context) (Actions, error) {
	return s.typeActions(ctx, orderType)
}

func (s *Service) Quotes(ctx context.Context, f DocFilter) (DocList, error) {
	return s.docs(ctx, quoteKind, f)
}
func (s *Service) Orders(ctx context.Context, f DocFilter) (DocList, error) {
	return s.docs(ctx, orderKind, f)
}
func (s *Service) Quote(ctx context.Context, id int64) (Doc, error) { return s.doc(ctx, quoteKind, id) }
func (s *Service) Order(ctx context.Context, id int64) (Doc, error) { return s.doc(ctx, orderKind, id) }

// CreateQuote and CreateOrder add a draft; the same request id returns the first one.
func (s *Service) CreateQuote(ctx context.Context, in NewDoc) (id int64, err error) {
	return id, platform.InTx(ctx, func(ctx context.Context) error {
		id, err = s.create(ctx, quoteKind, in.RequestID, in.DocFields, nil)
		return err
	})
}

func (s *Service) CreateOrder(ctx context.Context, in NewDoc) (id int64, err error) {
	return id, platform.InTx(ctx, func(ctx context.Context) error {
		id, err = s.create(ctx, orderKind, in.RequestID, in.DocFields, nil)
		return err
	})
}

func (s *Service) UpdateQuote(ctx context.Context, id int64, in DocUpdate) error {
	return s.update(ctx, quoteKind, id, in)
}

func (s *Service) UpdateOrder(ctx context.Context, id int64, in DocUpdate) error {
	return s.update(ctx, orderKind, id, in)
}

func (s *Service) docs(ctx context.Context, k kind, f DocFilter) (DocList, error) {
	out := DocList{Items: []DocListItem{}}
	sc, err := s.d.IAM.DataScope(ctx, "sales", k.view)
	if err != nil {
		return out, err
	}
	today, err := s.d.Setting.Today(ctx)
	if err != nil {
		return out, err
	}
	page, err := docList.Run(ctx, docQuery(sc, k, f))
	if err != nil {
		return out, err
	}
	out.Total = page.Total
	for _, r := range page.Items {
		out.Items = append(out.Items, DocListItem{
			ID: r.ID, Number: r.Number, Status: r.Status, Date: *platform.DatePtr(r.Date), CustomerID: r.CustomerID,
			CustomerCode: r.CustomerCode, CustomerName: r.CustomerName, OrgUnitName: r.OrgUnitName, Total: r.Total,
			ValidUntil: platform.DatePtr(r.ValidUntil), Expired: shownExpired(r.Status, r.Ordered, r.ValidUntil, today),
			Ordered: r.Ordered, FromQuote: r.FromQuote,
		})
	}
	return out, nil
}

// shownExpired: only a posted quotation with no order is shown as expired.
func shownExpired(status string, ordered bool, validUntil, today pgtype.Date) bool {
	return status == string(record.Posted) && !ordered && expired(validUntil, today)
}

// expired: a quotation is valid through its validity date, in the tenant time zone.
func expired(validUntil, today pgtype.Date) bool {
	return validUntil.Valid && validUntil.Time.Before(today.Time)
}

// doc returns one document; one the actor may not view does not exist.
func (s *Service) doc(ctx context.Context, k kind, id int64) (Doc, error) {
	d, err := s.d.Record.Read(ctx, record.Ref{Type: k.docType, ID: id})
	if err != nil {
		return Doc{}, err
	}
	h, err := store.GetHeader.Run(ctx, id)
	if err != nil {
		return Doc{}, err
	}
	lines, err := store.GetLines.Run(ctx, id)
	if err != nil {
		return Doc{}, err
	}
	out := Doc{
		ID: id, Kind: k.name, Number: h.Number, Status: h.Status, Version: h.Version, Date: *platform.DatePtr(h.Date),
		OrgUnitID: h.OrgUnitID, OrgUnitName: h.OrgUnitName, Customer: customerCopy(h),
		ValidUntil: platform.DatePtr(h.ValidUntil), DeliveryDate: platform.DatePtr(h.DeliveryDate),
		PaymentTerms: platform.TextPtr(h.PaymentTerms), DeliveryTerms: platform.TextPtr(h.DeliveryTerms), Note: platform.TextPtr(h.Note),
		Lines:       make([]Line, len(lines)),
		Totals:      Totals{Subtotal: h.Subtotal, DiscountTotal: h.DiscountTotal, VatTotal: h.VatTotal, Total: h.Total},
		MaxDiscount: d.Fields["max_discount"],
	}
	for i, l := range lines {
		out.Lines[i] = Line{
			LineInput: LineInput{ItemID: l.ItemID, Description: l.Description, Quantity: platform.PlainDecimal(l.Quantity), UnitPrice: l.UnitPrice,
				DiscountPercent: platform.PlainDecimal(l.DiscountPercent), VatRate: l.VatRate},
			ItemCode: l.ItemCode, Unit: l.Unit, Amount: l.Amount, Discount: l.Discount, Vat: l.Vat,
		}
	}
	// The other document shows only to who may view it.
	if h.QuoteID.Valid {
		if ok, err := s.d.Record.Can(ctx, quoteType, h.QuoteID.Int64, record.View); err != nil {
			return Doc{}, err
		} else if ok {
			qd, err := s.d.Record.Get(ctx, record.Ref{Type: quoteType, ID: h.QuoteID.Int64})
			if err != nil {
				return Doc{}, err
			}
			out.Quote = &DocRef{ID: qd.ID, Number: qd.Number, Status: string(qd.Status)}
		}
	}
	var extra []string
	if k == quoteKind {
		today, err := s.d.Setting.Today(ctx)
		if err != nil {
			return Doc{}, err
		}
		o, err := store.LiveOrder.Run(ctx, pgtype.Int8{Int64: id, Valid: true})
		switch {
		case err == nil:
			out.Ordered = true
			if ok, err := s.d.Record.Can(ctx, orderType, o.ID, record.View); err != nil {
				return Doc{}, err
			} else if ok {
				out.Order = &DocRef{ID: o.ID, Number: o.Number, Status: o.Status}
			}
		case !errors.Is(err, pgx.ErrNoRows):
			return Doc{}, err
		}
		out.Expired = shownExpired(string(d.Status), out.Ordered, h.ValidUntil, today)
		if d.Status == record.Posted && !out.Expired && !out.Ordered {
			if ok, err := s.canMakeOrder(ctx, h.OrgUnitID); err != nil {
				return Doc{}, err
			} else if ok {
				extra = append(extra, "create_order")
			}
		}
	}
	out.AllowedActions, err = s.d.Record.Actions(ctx, d, extra...)
	return out, err
}

func (s *Service) canMakeOrder(ctx context.Context, unit int64) (bool, error) {
	if platform.ProductGate(ctx, "sales", platform.ClassWrite) != nil {
		return false, nil
	}
	return s.d.IAM.Allowed(ctx, "sales", PermOrderEdit, unit)
}

// create checks the actor may create at the org unit and see the customer, then adds the draft.
func (s *Service) create(ctx context.Context, k kind, requestID string, f DocFields, quoteID *int64) (int64, error) {
	if err := s.d.Record.MayCreate(ctx, k.docType, record.Header{OrgUnitID: f.OrgUnitID}); err != nil {
		return 0, err
	}
	p, err := s.prepare(ctx, k, f)
	if err != nil {
		return 0, err
	}
	p.header.RequestID = requestID
	d, err := s.d.Record.Create(ctx, k.docType, p.header)
	if err != nil || d.Reused {
		return d.ID, err
	}
	c := p.customer
	if err := store.InsertHeader.Run(ctx, store.InsertHeaderParams{
		ID: d.ID, Kind: k.name, CustomerID: c.ID, CustomerCode: c.Code, CustomerName: c.Name,
		CustomerTaxCode: platform.NullText(c.TaxCode), CustomerAddress: platform.NullText(c.Address), CustomerPhone: platform.NullText(c.Phone),
		CustomerEmail: platform.NullText(c.Email), ContactName: platform.NullText(c.ContactName),
		ValidUntil: platform.NullDate(f.ValidUntil), DeliveryDate: platform.NullDate(f.DeliveryDate), QuoteID: platform.NullInt8(quoteID),
		PaymentTerms: platform.NullText(f.PaymentTerms), DeliveryTerms: platform.NullText(f.DeliveryTerms), Note: platform.NullText(f.Note),
		Subtotal: p.totals.Subtotal, DiscountTotal: p.totals.DiscountTotal, VatTotal: p.totals.VatTotal, Total: p.totals.Total,
	}); err != nil {
		return 0, err
	}
	return d.ID, s.writeLines(ctx, k, d.ID, p)
}

func (s *Service) update(ctx context.Context, k kind, id int64, in DocUpdate) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		if _, err := s.d.Record.Read(ctx, record.Ref{Type: k.docType, ID: id}); err != nil {
			return err
		}
		p, err := s.prepare(ctx, k, in.DocFields)
		if err != nil {
			return err
		}
		if _, err := s.d.Record.Edit(ctx, record.Ref{Type: k.docType, ID: id}, in.Version, p.header); err != nil {
			return err
		}
		c, f := p.customer, in.DocFields
		if err := store.UpdateHeader.Run(ctx, store.UpdateHeaderParams{
			ID: id, CustomerID: c.ID, CustomerCode: c.Code, CustomerName: c.Name,
			CustomerTaxCode: platform.NullText(c.TaxCode), CustomerAddress: platform.NullText(c.Address), CustomerPhone: platform.NullText(c.Phone),
			CustomerEmail: platform.NullText(c.Email), ContactName: platform.NullText(c.ContactName),
			ValidUntil: platform.NullDate(f.ValidUntil), DeliveryDate: platform.NullDate(f.DeliveryDate),
			PaymentTerms: platform.NullText(f.PaymentTerms), DeliveryTerms: platform.NullText(f.DeliveryTerms), Note: platform.NullText(f.Note),
			Subtotal: p.totals.Subtotal, DiscountTotal: p.totals.DiscountTotal, VatTotal: p.totals.VatTotal, Total: p.totals.Total,
		}); err != nil {
			return err
		}
		if err := store.DeleteLines.Run(ctx, id); err != nil {
			return err
		}
		return s.writeLines(ctx, k, id, p)
	})
}

// deleteDoc deletes a draft's header; its lines go with it.
func deleteDoc(ctx context.Context, id int64) error {
	return store.DeleteHeader.Run(ctx, id)
}

// prepared is a draft's content, checked against the current catalogues and computed.
type prepared struct {
	header   record.Header
	customer CustomerCopy
	lines    []Line
	totals   Totals
}

// prepare checks the fields, copies the customer and items as they are now, and computes
// every amount with the rounding mode of the org unit's legal entity.
func (s *Service) prepare(ctx context.Context, k kind, f DocFields) (prepared, error) {
	var p prepared
	switch {
	case k == quoteKind && (f.ValidUntil == nil || *f.ValidUntil < f.Date):
		return p, ErrValidUntil
	case k == orderKind && f.DeliveryDate != nil && *f.DeliveryDate < f.Date:
		return p, ErrDeliveryDate
	}
	if k == quoteKind {
		f.DeliveryDate = nil
	} else {
		f.ValidUntil = nil
	}
	if err := s.d.Record.Visible(ctx, record.Ref{Type: customerType, ID: f.CustomerID}, record.View); err != nil {
		return p, err
	}
	c, err := store.GetCustomer.Run(ctx, f.CustomerID)
	if err != nil {
		return p, err
	}
	if !c.Active {
		return p, ErrCustomerInactive
	}
	p.customer = CustomerCopy{ID: c.ID, Code: c.Code, Name: c.Name, TaxCode: platform.TextPtr(c.TaxCode), Address: platform.TextPtr(c.Address),
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
	mode, err := s.d.Setting.RoundingOf(ctx, f.OrgUnitID)
	if err != nil {
		return p, err
	}
	list := make([]int64, len(f.Lines))
	for i, l := range f.Lines {
		it, ok := items[l.ItemID]
		if !ok || !it.Active {
			return p, errLine("item_inactive", i+1)
		}
		list[i] = it.Price
	}
	amts, totals, maxDiscount, err := compute(f.Lines, list, mode)
	if err != nil {
		return p, err
	}
	p.lines = make([]Line, len(f.Lines))
	for i, l := range f.Lines {
		it := items[l.ItemID]
		p.lines[i] = Line{LineInput: l, ItemCode: it.Code, Unit: it.Unit, Amount: amts[i].amount, Discount: amts[i].discount, Vat: amts[i].vat}
	}
	p.totals = totals
	p.header = record.Header{Date: f.Date, OrgUnitID: f.OrgUnitID, Amount: &totals.Total,
		Fields: map[string]string{"max_discount": maxDiscount.String(), "items": record.SetValue(ids)}}
	return p, nil
}

func (s *Service) writeLines(ctx context.Context, k kind, id int64, p prepared) error {
	a := store.InsertLinesParams{DocID: id}
	for i, l := range p.lines {
		a.Positions = append(a.Positions, int32(i+1))
		a.ItemIds = append(a.ItemIds, l.ItemID)
		a.ItemCodes = append(a.ItemCodes, l.ItemCode)
		a.Descriptions = append(a.Descriptions, l.Description)
		a.Units = append(a.Units, l.Unit)
		a.Quantities = append(a.Quantities, l.Quantity)
		a.UnitPrices = append(a.UnitPrices, l.UnitPrice)
		a.DiscountPercents = append(a.DiscountPercents, l.DiscountPercent)
		a.VatRates = append(a.VatRates, l.VatRate)
		a.Amounts = append(a.Amounts, l.Amount)
		a.Discounts = append(a.Discounts, l.Discount)
		a.Vats = append(a.Vats, l.Vat)
	}
	if err := store.InsertLines.Run(ctx, a); err != nil {
		return err
	}
	return s.d.Audit.RecordFor(ctx, "sales.document_saved", audit.Ref{Type: k.docType, ID: id},
		map[string]any{"customer_id": p.customer.ID, "lines": len(p.lines), "total": p.totals.Total})
}

// CreateOrderFromQuote makes a draft order of a posted, unexpired quotation: its customer,
// terms and lines as quoted, dated today. While an order from it is not cancelled, it
// returns that order instead, so a retried click never makes a second one. When an
// approval rule holds the quotation for it, it returns held and makes nothing yet.
func (s *Service) CreateOrderFromQuote(ctx context.Context, quoteID int64) (id int64, held bool, err error) {
	ref := record.Ref{Type: quoteType, ID: quoteID}
	err = platform.InTx(ctx, func(ctx context.Context) error {
		if err := s.mayOrder(ctx, ref); err != nil {
			return err
		}
		d, err := s.d.Record.LockToDepend(ctx, ref)
		if err != nil {
			return err
		}
		if id, err = s.liveOrder(ctx, quoteID); err != nil || id != 0 {
			return err
		}
		f, err := s.orderFields(ctx, d)
		if err != nil {
			return err
		}
		if held, err = s.d.Record.Request(ctx, ref, actionCreateOrder, nil); err != nil || held {
			return err
		}
		id, err = s.create(ctx, orderKind, "", f, &quoteID)
		return err
	})
	return id, held, err
}

// runCreateOrder makes the order of an approved create_order request, as its submitter,
// in the approver's transaction; the quotation is already locked.
func (s *Service) runCreateOrder(ctx context.Context, d record.Doc, _ json.RawMessage) error {
	if err := s.mayOrder(ctx, d.Ref); err != nil {
		return err
	}
	if id, err := s.liveOrder(ctx, d.ID); err != nil || id != 0 {
		return platform.OrErr(err, ErrQuoteHasOrder)
	}
	f, err := s.orderFields(ctx, d)
	if err != nil {
		return err
	}
	_, err = s.create(ctx, orderKind, "", f, &d.ID)
	return err
}

// mayOrder checks the actor may see the quotation and make orders where it is; checked
// before an existing order is returned, so only who may make one learns of it.
func (s *Service) mayOrder(ctx context.Context, ref record.Ref) error {
	qd, err := s.d.Record.Read(ctx, ref)
	if err != nil {
		return err
	}
	if err := s.d.Record.WriteGate(ctx, ref); err != nil {
		return err
	}
	return s.require(ctx, PermOrderEdit, qd.OrgUnitID)
}

// liveOrder is the order from the quotation not cancelled, 0 for none; one the actor may
// not view is refused.
func (s *Service) liveOrder(ctx context.Context, quoteID int64) (int64, error) {
	live, err := store.LiveOrder.Run(ctx, pgtype.Int8{Int64: quoteID, Valid: true})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if ok, err := s.d.Record.Can(ctx, orderType, live.ID, record.View); err != nil || !ok {
		return 0, platform.OrErr(err, ErrQuoteHasOrder)
	}
	return live.ID, nil
}

// orderFields is the order a posted, unexpired quotation makes.
func (s *Service) orderFields(ctx context.Context, d record.Doc) (DocFields, error) {
	if d.Status != record.Posted {
		return DocFields{}, ErrQuoteNotPosted
	}
	h, err := store.GetHeader.Run(ctx, d.ID)
	if err != nil {
		return DocFields{}, err
	}
	today, err := s.d.Setting.Today(ctx)
	if err != nil {
		return DocFields{}, err
	}
	if expired(h.ValidUntil, today) {
		return DocFields{}, ErrQuoteExpired
	}
	lines, err := store.GetLines.Run(ctx, d.ID)
	if err != nil {
		return DocFields{}, err
	}
	f := DocFields{Date: *platform.DatePtr(today), OrgUnitID: h.OrgUnitID, CustomerID: h.CustomerID,
		PaymentTerms: platform.TextPtr(h.PaymentTerms), DeliveryTerms: platform.TextPtr(h.DeliveryTerms), Note: platform.TextPtr(h.Note),
		Lines: make([]LineInput, len(lines))}
	for i, l := range lines {
		f.Lines[i] = LineInput{ItemID: l.ItemID, Description: l.Description, Quantity: l.Quantity, UnitPrice: l.UnitPrice,
			DiscountPercent: l.DiscountPercent, VatRate: l.VatRate}
	}
	return f, nil
}

func customerCopy(h store.GetHeaderRow) CustomerCopy {
	return CustomerCopy{ID: h.CustomerID, Code: h.CustomerCode, Name: h.CustomerName, TaxCode: platform.TextPtr(h.CustomerTaxCode),
		Address: platform.TextPtr(h.CustomerAddress), Phone: platform.TextPtr(h.CustomerPhone), Email: platform.TextPtr(h.CustomerEmail),
		ContactName: platform.TextPtr(h.ContactName)}
}
