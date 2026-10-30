package sales

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/sales/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/stock"
)

const deliveryType = "sales.delivery"

// Deliveries reuse the order's permissions and org unit: who may edit an order may deliver it.
func (s *Service) registerDelivery() {
	s.d.Record.Register(record.Type{
		Code: deliveryType, Product: "sales", Kind: record.Document, NumberPrefix: "GH",
		Perms:        record.Perms{View: PermOrderView, Edit: PermOrderEdit, ViewFiles: PermOrderView, Attach: PermOrderEdit},
		OnTransition: s.deliveryTransition,
		OnDelete:     deleteDelivery,
	})
}

// deliveryTransition takes the goods out of stock on posting, after checking the lines
// against the order again, and puts them back on cancelling.
func (s *Service) deliveryTransition(ctx context.Context, d record.Doc, _ record.Status) error {
	switch d.Status {
	case record.Posted:
		h, err := store.GetDelivery.Run(ctx, d.ID)
		if err != nil {
			return err
		}
		saved, err := store.GetDeliveryLines.Run(ctx, d.ID)
		if err != nil {
			return err
		}
		f := DeliveryFields{WarehouseID: h.WarehouseID, Lines: make([]DeliveryLineInput, len(saved))}
		for i, l := range saved {
			f.Lines[i] = DeliveryLineInput{OrderPosition: int(l.OrderPosition), LotCode: platform.TextPtr(l.LotCode),
				SerialCode: platform.TextPtr(l.SerialCode), Quantity: l.Quantity}
		}
		lines, err := s.checkDelivery(ctx, d.ID, h.OrderID, f)
		if err != nil {
			return err
		}
		moves := make([]stock.Move, len(lines))
		for i, l := range lines {
			q, err := decimal.NewFromString(l.Quantity)
			if err != nil {
				return err
			}
			moves[i] = stock.Move{LineNo: i + 1, WarehouseID: h.WarehouseID, ItemID: l.ItemID, Serial: l.SerialCode, Quantity: q.Neg()}
			if l.LotCode != nil {
				moves[i].Lot = &stock.LotRef{Code: *l.LotCode}
			}
		}
		return s.d.Stock.Record(ctx, d.Ref, moves)
	case record.Cancelled:
		return s.d.Stock.Void(ctx, d.Ref)
	}
	return nil
}

// deleteDelivery deletes a draft's row; its lines go with it.
func deleteDelivery(ctx context.Context, id int64) error {
	return store.DeleteDelivery.Run(ctx, id)
}

// Delivery returns one delivery; one the actor may not view does not exist.
func (s *Service) Delivery(ctx context.Context, id int64) (Delivery, error) {
	d, err := s.d.Record.Read(ctx, record.Ref{Type: deliveryType, ID: id})
	if err != nil {
		return Delivery{}, err
	}
	h, err := store.GetDelivery.Run(ctx, id)
	if err != nil {
		return Delivery{}, err
	}
	lines, err := store.GetDeliveryLines.Run(ctx, id)
	if err != nil {
		return Delivery{}, err
	}
	out := Delivery{ID: id, OrderID: h.OrderID, OrderNumber: h.OrderNumber, Number: h.Number, Status: h.Status, Version: h.Version,
		Date: *platform.DatePtr(h.Date), OrgUnitID: h.OrgUnitID, WarehouseID: h.WarehouseID, WarehouseCode: h.WarehouseCode,
		WarehouseName: h.WarehouseName, Note: platform.TextPtr(h.Note), Lines: make([]DeliveryLine, len(lines))}
	for i, l := range lines {
		out.Lines[i] = DeliveryLine{
			DeliveryLineInput: DeliveryLineInput{OrderPosition: int(l.OrderPosition), LotCode: platform.TextPtr(l.LotCode),
				SerialCode: platform.TextPtr(l.SerialCode), Quantity: platform.PlainDecimal(l.Quantity)},
			ItemID: l.ItemID, ItemCode: l.ItemCode, Description: l.Description, Unit: l.Unit,
		}
	}
	out.AllowedActions, err = s.d.Record.Actions(ctx, d)
	return out, err
}

// CreateDelivery adds a draft delivery of a posted order, at the order's org unit; the same
// request id returns the first one.
func (s *Service) CreateDelivery(ctx context.Context, in NewDelivery) (id int64, err error) {
	return id, platform.InTx(ctx, func(ctx context.Context) error {
		o, err := s.d.Record.Read(ctx, record.Ref{Type: orderType, ID: in.OrderID})
		if err != nil {
			return err
		}
		d, err := s.d.Record.Create(ctx, deliveryType, record.Header{Date: in.Date, OrgUnitID: o.OrgUnitID, RequestID: in.RequestID})
		if id = d.ID; err != nil || d.Reused {
			return err
		}
		lines, err := s.checkDelivery(ctx, id, in.OrderID, in.DeliveryFields)
		if err != nil {
			return err
		}
		if err := store.InsertDelivery.Run(ctx, store.InsertDeliveryParams{ID: id, OrderID: in.OrderID, WarehouseID: in.WarehouseID,
			Note: platform.NullText(in.Note)}); err != nil {
			return err
		}
		return s.writeDeliveryLines(ctx, id, in.WarehouseID, lines)
	})
}

// UpdateDelivery saves a draft delivery again, checked as on creation.
func (s *Service) UpdateDelivery(ctx context.Context, id int64, in DeliveryUpdate) error {
	ref := record.Ref{Type: deliveryType, ID: id}
	return platform.InTx(ctx, func(ctx context.Context) error {
		d, err := s.d.Record.Read(ctx, ref)
		if err != nil {
			return err
		}
		if _, err := s.d.Record.Edit(ctx, ref, in.Version, record.Header{Date: in.Date, OrgUnitID: d.OrgUnitID}); err != nil {
			return err
		}
		order, err := store.DeliveryOrder.Run(ctx, id)
		if err != nil {
			return err
		}
		lines, err := s.checkDelivery(ctx, id, order, in.DeliveryFields)
		if err != nil {
			return err
		}
		if err := store.UpdateDelivery.Run(ctx, store.UpdateDeliveryParams{ID: id, WarehouseID: in.WarehouseID, Note: platform.NullText(in.Note)}); err != nil {
			return err
		}
		if err := store.DeleteDeliveryLines.Run(ctx, id); err != nil {
			return err
		}
		return s.writeDeliveryLines(ctx, id, in.WarehouseID, lines)
	})
}

// checkDelivery locks the order, after the delivery's own row, so saves and posting of its
// deliveries take turns, and refuses while an action on the order awaits approval; it then checks the order, the warehouse and every line, and returns
// the lines with the order line's item and texts.
func (s *Service) checkDelivery(ctx context.Context, id, orderID int64, f DeliveryFields) ([]DeliveryLine, error) {
	o, err := s.d.Record.LockToDepend(ctx, record.Ref{Type: orderType, ID: orderID})
	if err != nil {
		return nil, err
	}
	if o.Status != record.Posted {
		return nil, ErrOrderNotPosted
	}
	w, err := store.DeliveryWarehouse.Run(ctx, f.WarehouseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, platform.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !w.Active || w.LegalEntityID != o.LegalEntityID {
		return nil, ErrDeliveryWarehouse
	}
	rows, err := store.DeliverableLines.Run(ctx, store.DeliverableLinesParams{DeliveryID: id, OrderID: orderID})
	if err != nil {
		return nil, err
	}
	byPos := map[int]store.DeliverableLinesRow{}
	for _, r := range rows {
		byPos[int(r.Position)] = r
	}
	used := map[int]decimal.Decimal{}
	out := make([]DeliveryLine, len(f.Lines))
	for i, l := range f.Lines {
		r, ok := byPos[l.OrderPosition]
		if !ok {
			return nil, deliveryErr(ErrDeliveryPosition, i+1, nil)
		}
		if r.Tracking == "none" {
			return nil, deliveryErr(ErrDeliveryLineNotStock, i+1, nil)
		}
		q, ok := record.ParseNumber(l.Quantity)
		if !ok || !q.IsPositive() || !q.Equal(q.Truncate(3)) {
			return nil, errLine("invalid_quantity", i+1)
		}
		lot, serial := trimmed(l.LotCode), trimmed(l.SerialCode)
		if err := stock.CheckShape(r.Tracking, lot, serial, q, i+1); err != nil {
			return nil, err
		}
		ordered, err := decimal.NewFromString(r.Quantity)
		if err != nil {
			return nil, err
		}
		taken, err := decimal.NewFromString(r.Taken)
		if err != nil {
			return nil, err
		}
		used[l.OrderPosition] = used[l.OrderPosition].Add(q)
		if taken.Add(used[l.OrderPosition]).GreaterThan(ordered) {
			return nil, deliveryErr(ErrDeliveryOverOrdered, i+1, map[string]any{
				"order_position": l.OrderPosition, "remaining": ordered.Sub(taken).StringFixed(3)})
		}
		out[i] = DeliveryLine{
			DeliveryLineInput: DeliveryLineInput{OrderPosition: l.OrderPosition, LotCode: lot, SerialCode: serial,
				Quantity: q.String()},
			ItemID: r.ItemID, ItemCode: r.ItemCode, Description: r.Description, Unit: r.Unit,
		}
	}
	return out, nil
}

// DeliveryOptions lists, for the delivery form of a posted order, the warehouses of its legal
// entity and per stock line what is left to deliver with lots or serials to pick from in the
// warehouse (the legal entity's default when warehouseID is 0). It needs order edit there.
// Without any warehouse it lists the lines with no suggestions, instead of creating the default.
func (s *Service) DeliveryOptions(ctx context.Context, orderID, warehouseID int64) (DeliveryOptions, error) {
	out := DeliveryOptions{Warehouses: []WarehouseOption{}, Lines: []DeliveryOptionLine{}}
	o, err := s.d.Record.Read(ctx, record.Ref{Type: orderType, ID: orderID})
	if err != nil {
		return out, err
	}
	if err := s.require(ctx, PermOrderEdit, o.OrgUnitID); err != nil {
		return out, err
	}
	if o.Status != record.Posted {
		return out, ErrOrderNotPosted
	}
	whs, err := store.DeliveryWarehouses.Run(ctx, o.LegalEntityID)
	if err != nil {
		return out, err
	}
	for _, w := range whs {
		out.Warehouses = append(out.Warehouses, WarehouseOption{ID: w.ID, Code: w.Code, Name: w.Name, IsDefault: w.IsDefault})
	}
	switch {
	case warehouseID == 0 && len(whs) > 0:
		out.WarehouseID = whs[0].ID
	case warehouseID != 0 && slices.ContainsFunc(whs, func(w store.DeliveryWarehousesRow) bool { return w.ID == warehouseID }):
		out.WarehouseID = warehouseID
	case warehouseID != 0:
		return out, ErrDeliveryWarehouse
	}
	today, err := s.d.Setting.Today(ctx)
	if err != nil {
		return out, err
	}
	todayStr := *platform.DatePtr(today)
	rows, err := store.DeliverableLines.Run(ctx, store.DeliverableLinesParams{OrderID: orderID})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		if r.Tracking == "none" {
			continue
		}
		ordered, err := decimal.NewFromString(r.Quantity)
		if err != nil {
			return out, err
		}
		taken, err := decimal.NewFromString(r.Taken)
		if err != nil {
			return out, err
		}
		remaining := decimal.Max(ordered.Sub(taken), decimal.Zero)
		l := DeliveryOptionLine{OrderPosition: int(r.Position), ItemID: r.ItemID, ItemCode: r.ItemCode, Description: r.Description,
			Unit: r.Unit, Tracking: r.Tracking, Ordered: ordered.StringFixed(3), Remaining: remaining.StringFixed(3),
			Lots: []DeliveryLot{}, Serials: []string{}}
		if out.WarehouseID != 0 {
			switch r.Tracking {
			case "lot":
				picks, err := s.d.Stock.SuggestLots(ctx, out.WarehouseID, r.ItemID, remaining, todayStr)
				if err != nil {
					return out, err
				}
				for _, p := range picks {
					l.Lots = append(l.Lots, DeliveryLot{Code: p.Code, Expiry: p.Expiry, Quantity: p.Quantity.StringFixed(3)})
				}
			case "serial":
				if l.Serials, err = s.d.Stock.SerialsOnHand(ctx, out.WarehouseID, r.ItemID); err != nil {
					return out, err
				}
			}
		}
		out.Lines = append(out.Lines, l)
	}
	return out, nil
}

func (s *Service) writeDeliveryLines(ctx context.Context, id, warehouse int64, lines []DeliveryLine) error {
	a := store.InsertDeliveryLinesParams{DeliveryID: id}
	for i, l := range lines {
		a.Positions = append(a.Positions, int32(i+1))
		a.OrderPositions = append(a.OrderPositions, int32(l.OrderPosition))
		a.ItemIds = append(a.ItemIds, l.ItemID)
		a.ItemCodes = append(a.ItemCodes, l.ItemCode)
		a.Descriptions = append(a.Descriptions, l.Description)
		a.Units = append(a.Units, l.Unit)
		a.LotCodes = append(a.LotCodes, deref(l.LotCode))
		a.SerialCodes = append(a.SerialCodes, deref(l.SerialCode))
		a.Quantities = append(a.Quantities, l.Quantity)
	}
	if err := store.InsertDeliveryLines.Run(ctx, a); err != nil {
		return err
	}
	return s.d.Audit.RecordFor(ctx, "sales.delivery_saved", audit.Ref{Type: deliveryType, ID: id},
		map[string]any{"warehouse_id": warehouse, "lines": len(lines)})
}

// deliveryErr names the delivery line (1-based) a refusal is about.
func deliveryErr(e *platform.Error, line int, params map[string]any) error {
	p := map[string]any{"line_no": line}
	maps.Copy(p, params)
	return &platform.Error{Status: e.Status, Code: e.Code, Params: p}
}

// trimmed is nil for an absent or blank code.
func trimmed(p *string) *string {
	if p == nil || strings.TrimSpace(*p) == "" {
		return nil
	}
	v := strings.TrimSpace(*p)
	return &v
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
