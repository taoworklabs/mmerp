package purchase

import (
	"context"
	"errors"
	"maps"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/purchase/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
	"github.com/taoworklabs/mmerp/internal/shared/stock"
)

const goodsReceiptType = "purchase.goods_receipt"

// Goods receipts reuse the order's permissions and org unit: who may edit an order may receive it.
func (s *Service) registerGoodsReceipt() {
	s.d.Record.Register(record.Type{
		Code: goodsReceiptType, Product: "purchase", Kind: record.Document, NumberPrefix: "NK",
		Perms:        record.Perms{View: PermOrderView, Edit: PermOrderEdit, ViewFiles: PermOrderView, Attach: PermOrderEdit},
		OnTransition: s.receiptTransition,
		OnDelete:     func(ctx context.Context, id int64) error { return store.DeleteGoodsReceipt.Run(ctx, id) },
	})
}

// receiptTransition brings the goods in at the order price on posting, after checking the lines
// against the order again, booking them as received not invoiced; cancelling takes them out, and
// the ledger refuses when they were already issued.
func (s *Service) receiptTransition(ctx context.Context, d record.Doc, _ record.Status) error {
	switch d.Status {
	case record.Posted:
		h, err := store.GetGoodsReceipt.Run(ctx, d.ID)
		if err != nil {
			return err
		}
		saved, err := store.GetGoodsReceiptLines.Run(ctx, d.ID)
		if err != nil {
			return err
		}
		f := ReceiptFields{Date: d.Date, WarehouseID: h.WarehouseID, Lines: make([]ReceiptLineInput, len(saved))}
		for i, r := range saved {
			f.Lines[i] = receiptLineInput(r)
		}
		lines, err := s.checkReceipt(ctx, d.ID, h.OrderID, f)
		if err != nil {
			return err
		}
		prices, err := loadOrderPrices(ctx, h.OrderID)
		if err != nil {
			return err
		}
		moves := make([]stock.Move, len(lines))
		var total int64
		for i, l := range lines {
			q, err := decimal.NewFromString(l.Quantity)
			if err != nil {
				return err
			}
			v := prices.value(l.OrderPosition, q)
			total += v
			moves[i] = stock.Move{LineNo: i + 1, WarehouseID: h.WarehouseID, ItemID: l.ItemID, Lot: l.Lot, Serial: l.Serial, Quantity: q, Value: &v}
		}
		if _, err := s.d.Stock.Record(ctx, d.Ref, moves); err != nil {
			return err
		}
		return s.d.Posting.Record(ctx, d.Ref, nonZero([]posting.Line{
			{Kind: stock.KindInventory, OrgUnitID: d.OrgUnitID, Amount: total},
			{Kind: KindReceivedNotInvoiced, OrgUnitID: d.OrgUnitID, Amount: total},
		}))
	case record.Cancelled:
		if err := s.d.Stock.Void(ctx, d.Ref); err != nil {
			return err
		}
		return s.d.Posting.Void(ctx, d.Ref)
	}
	return nil
}

// orderPrices are an order's lines by position: quantity and net amount.
type orderPrices map[int]store.OrderLineNetsRow

func loadOrderPrices(ctx context.Context, orderID int64) (orderPrices, error) {
	rows, err := store.OrderLineNets.Run(ctx, orderID)
	p := orderPrices{}
	for _, r := range rows {
		p[int(r.Position)] = r
	}
	return p, err
}

// value is q of an order line at its net order price; receipts and invoices value alike, so
// the invoice clears what the receipt booked.
func (p orderPrices) value(pos int, q decimal.Decimal) int64 {
	l := p[pos]
	ordered, err := decimal.NewFromString(l.Quantity)
	if err != nil || ordered.IsZero() {
		return 0 // the column is a positive numeric
	}
	return platform.Round(decimal.NewFromInt(l.Net).Mul(q).Div(ordered))
}

func receiptLineInput(r store.GetGoodsReceiptLinesRow) ReceiptLineInput {
	in := ReceiptLineInput{OrderPosition: int(r.OrderPosition), Quantity: platform.PlainDecimal(r.Quantity), Serial: platform.TextPtr(r.SerialCode)}
	if r.LotCode.Valid {
		in.Lot = &stock.LotRef{Code: r.LotCode.String, Expiry: platform.DatePtr(r.LotExpiry)}
	}
	return in
}

// GoodsReceipt returns one goods receipt; one the actor may not view does not exist.
func (s *Service) GoodsReceipt(ctx context.Context, id int64) (GoodsReceipt, error) {
	d, err := s.d.Record.Read(ctx, record.Ref{Type: goodsReceiptType, ID: id})
	if err != nil {
		return GoodsReceipt{}, err
	}
	h, err := store.GetGoodsReceipt.Run(ctx, id)
	if err != nil {
		return GoodsReceipt{}, err
	}
	lines, err := store.GetGoodsReceiptLines.Run(ctx, id)
	if err != nil {
		return GoodsReceipt{}, err
	}
	out := GoodsReceipt{ID: id, OrderID: h.OrderID, OrderNumber: h.OrderNumber, Number: h.Number, Status: h.Status, Version: h.Version,
		Date: *platform.DatePtr(h.Date), OrgUnitID: h.OrgUnitID, WarehouseID: h.WarehouseID, WarehouseCode: h.WarehouseCode,
		WarehouseName: h.WarehouseName, Note: platform.TextPtr(h.Note), Lines: make([]ReceiptLine, len(lines))}
	for i, l := range lines {
		out.Lines[i] = ReceiptLine{ReceiptLineInput: receiptLineInput(l), ItemID: l.ItemID, ItemCode: l.ItemCode, Description: l.Description, Unit: l.Unit}
	}
	out.AllowedActions, err = s.d.Record.Actions(ctx, d)
	return out, err
}

// CreateGoodsReceipt adds a draft goods receipt of a posted order, at the order's org unit; the
// same request id returns the first one.
func (s *Service) CreateGoodsReceipt(ctx context.Context, in NewGoodsReceipt) (id int64, err error) {
	return id, platform.InTx(ctx, func(ctx context.Context) error {
		o, err := s.d.Record.Read(ctx, record.Ref{Type: orderType, ID: in.OrderID})
		if err != nil {
			return err
		}
		d, err := s.d.Record.Create(ctx, goodsReceiptType, record.Header{Date: in.Date, OrgUnitID: o.OrgUnitID, RequestID: in.RequestID})
		if id = d.ID; err != nil || d.Reused {
			return err
		}
		lines, err := s.checkReceipt(ctx, id, in.OrderID, in.ReceiptFields)
		if err != nil {
			return err
		}
		if err := store.InsertGoodsReceipt.Run(ctx, store.InsertGoodsReceiptParams{ID: id, OrderID: in.OrderID, WarehouseID: in.WarehouseID,
			Note: platform.NullText(in.Note)}); err != nil {
			return err
		}
		return s.writeReceiptLines(ctx, id, in.WarehouseID, lines)
	})
}

// UpdateGoodsReceipt saves a draft goods receipt again, checked as on creation.
func (s *Service) UpdateGoodsReceipt(ctx context.Context, id int64, in GoodsReceiptUpdate) error {
	ref := record.Ref{Type: goodsReceiptType, ID: id}
	return platform.InTx(ctx, func(ctx context.Context) error {
		d, err := s.d.Record.Read(ctx, ref)
		if err != nil {
			return err
		}
		if _, err := s.d.Record.Edit(ctx, ref, in.Version, record.Header{Date: in.Date, OrgUnitID: d.OrgUnitID}); err != nil {
			return err
		}
		order, err := store.GoodsReceiptOrder.Run(ctx, id)
		if err != nil {
			return err
		}
		lines, err := s.checkReceipt(ctx, id, order, in.ReceiptFields)
		if err != nil {
			return err
		}
		if err := store.UpdateGoodsReceipt.Run(ctx, store.UpdateGoodsReceiptParams{ID: id, WarehouseID: in.WarehouseID, Note: platform.NullText(in.Note)}); err != nil {
			return err
		}
		if err := store.DeleteGoodsReceiptLines.Run(ctx, id); err != nil {
			return err
		}
		return s.writeReceiptLines(ctx, id, in.WarehouseID, lines)
	})
}

// checkReceipt locks the order, after the receipt's own row, so saves and posting of its
// receipts take turns, and refuses while an action on the order awaits approval; it then checks
// the order, the warehouse and every line, and returns the lines with the order line's item and texts.
func (s *Service) checkReceipt(ctx context.Context, id, orderID int64, f ReceiptFields) ([]ReceiptLine, error) {
	o, err := s.d.Record.LockToDepend(ctx, record.Ref{Type: orderType, ID: orderID})
	if err != nil {
		return nil, err
	}
	if o.Status != record.Posted {
		return nil, ErrOrderNotPosted
	}
	if f.Date < o.Date {
		return nil, ErrReceiptBeforeOrder
	}
	w, err := store.ReceiptWarehouse.Run(ctx, f.WarehouseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, platform.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !w.Active || w.LegalEntityID != o.LegalEntityID {
		return nil, ErrReceiptWarehouse
	}
	rows, err := store.ReceivableLines.Run(ctx, store.ReceivableLinesParams{ReceiptID: id, OrderID: orderID})
	if err != nil {
		return nil, err
	}
	byPos := map[int]store.ReceivableLinesRow{}
	for _, r := range rows {
		byPos[int(r.Position)] = r
	}
	used := map[int]decimal.Decimal{}
	out := make([]ReceiptLine, len(f.Lines))
	for i, l := range f.Lines {
		r, ok := byPos[l.OrderPosition]
		if !ok {
			return nil, lineErr(ErrReceiptPosition, i+1, nil)
		}
		if !r.Stocked {
			return nil, lineErr(ErrReceiptLineNotStock, i+1, nil)
		}
		q, ok := record.ParseNumber(l.Quantity)
		if !ok || !q.IsPositive() || !q.Equal(q.Truncate(3)) {
			return nil, errLine("invalid_quantity", i+1)
		}
		var lotCode *string
		lot := l.Lot
		if lot != nil {
			lotCode = trimmed(&lot.Code)
			if lotCode != nil {
				lot = &stock.LotRef{Code: *lotCode, Expiry: lot.Expiry}
			} else {
				lot = nil
			}
		}
		serial := trimmed(l.Serial)
		if err := stock.CheckShape(r.Tracking, lotCode, serial, q, i+1); err != nil {
			return nil, err
		}
		ordered, taken, err := decimals(r.Quantity, r.Taken)
		if err != nil {
			return nil, err
		}
		used[l.OrderPosition] = used[l.OrderPosition].Add(q)
		if taken.Add(used[l.OrderPosition]).GreaterThan(ordered) {
			return nil, lineErr(ErrReceiptOverOrder, i+1, map[string]any{
				"order_position": l.OrderPosition, "remaining": ordered.Sub(taken).StringFixed(3)})
		}
		out[i] = ReceiptLine{
			ReceiptLineInput: ReceiptLineInput{OrderPosition: l.OrderPosition, Quantity: q.String(), Lot: lot, Serial: serial},
			ItemID:           r.ItemID, ItemCode: r.ItemCode, Description: r.Description, Unit: r.Unit,
		}
	}
	return out, nil
}

// ReceiptOptions lists, for the receipt form of a posted order, the warehouses of its legal entity
// and per stock line what is left to receive. It needs order edit there.
func (s *Service) ReceiptOptions(ctx context.Context, orderID int64) (ReceiptOptions, error) {
	out := ReceiptOptions{Warehouses: []WarehouseOption{}, Lines: []ReceiptOptionLine{}}
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
	whs, err := store.ReceiptWarehouses.Run(ctx, o.LegalEntityID)
	if err != nil {
		return out, err
	}
	for _, w := range whs {
		out.Warehouses = append(out.Warehouses, WarehouseOption{ID: w.ID, Code: w.Code, Name: w.Name, IsDefault: w.IsDefault})
	}
	rows, err := store.ReceivableLines.Run(ctx, store.ReceivableLinesParams{OrderID: orderID})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		if !r.Stocked {
			continue
		}
		ordered, taken, err := decimals(r.Quantity, r.Taken)
		if err != nil {
			return out, err
		}
		out.Lines = append(out.Lines, ReceiptOptionLine{OrderPosition: int(r.Position), ItemID: r.ItemID, ItemCode: r.ItemCode,
			Description: r.Description, Unit: r.Unit, Tracking: r.Tracking, Ordered: ordered.StringFixed(3),
			Remaining: decimal.Max(ordered.Sub(taken), decimal.Zero).StringFixed(3)})
	}
	return out, nil
}

func (s *Service) writeReceiptLines(ctx context.Context, id, warehouse int64, lines []ReceiptLine) error {
	a := store.InsertGoodsReceiptLinesParams{ReceiptID: id}
	for i, l := range lines {
		a.Positions = append(a.Positions, int32(i+1))
		a.OrderPositions = append(a.OrderPositions, int32(l.OrderPosition))
		a.ItemIds = append(a.ItemIds, l.ItemID)
		a.ItemCodes = append(a.ItemCodes, l.ItemCode)
		a.Descriptions = append(a.Descriptions, l.Description)
		a.Units = append(a.Units, l.Unit)
		lot, expiry := "", ""
		if l.Lot != nil {
			lot, expiry = l.Lot.Code, deref(l.Lot.Expiry)
		}
		a.LotCodes = append(a.LotCodes, lot)
		a.LotExpiries = append(a.LotExpiries, expiry)
		a.SerialCodes = append(a.SerialCodes, deref(l.Serial))
		a.Quantities = append(a.Quantities, l.Quantity)
	}
	if err := store.InsertGoodsReceiptLines.Run(ctx, a); err != nil {
		return err
	}
	return s.d.Audit.RecordFor(ctx, "purchase.goods_receipt_saved", audit.Ref{Type: goodsReceiptType, ID: id},
		map[string]any{"warehouse_id": warehouse, "lines": len(lines)})
}

// decimals parses an ordered and a taken quantity as the queries return them.
func decimals(ordered, taken string) (decimal.Decimal, decimal.Decimal, error) {
	o, err := decimal.NewFromString(ordered)
	if err != nil {
		return o, o, err
	}
	t, err := decimal.NewFromString(taken)
	return o, t, err
}

// lineErr names the line (1-based) of a receipt, invoice or payment a refusal is about, as line_no.
func lineErr(e *platform.Error, line int, params map[string]any) error {
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
