package sales

import (
	"context"
	"errors"
	"maps"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/sales/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
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

// deliveryTransition is where posting and cancelling move the stock.
func (s *Service) deliveryTransition(context.Context, record.Doc, record.Status) error {
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
			return nil, platform.ErrNotFound
		}
		if r.Tracking == "none" {
			return nil, deliveryErr(ErrDeliveryLineNotStock, i+1, nil)
		}
		q, ok := record.ParseNumber(l.Quantity)
		if !ok || !q.IsPositive() || !q.Equal(q.Truncate(3)) {
			return nil, errLine("invalid_quantity", i+1)
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
			DeliveryLineInput: DeliveryLineInput{OrderPosition: l.OrderPosition, LotCode: trimmed(l.LotCode), SerialCode: trimmed(l.SerialCode),
				Quantity: q.String()},
			ItemID: r.ItemID, ItemCode: r.ItemCode, Description: r.Description, Unit: r.Unit,
		}
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
