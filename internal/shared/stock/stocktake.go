package stock

import (
	"context"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/stock/internal/store"
)

const stocktakeType = "stock.stocktake"

func (s *Service) registerStocktake() {
	s.d.Record.Register(record.Type{Code: stocktakeType, Product: "stock", Shared: true, Kind: record.Document, NumberPrefix: "KK",
		Perms:        record.Perms{View: PermView, Edit: PermManage, ViewFiles: PermView, Attach: PermManage},
		OnTransition: s.stocktakeTransition,
		OnDelete:     deleteStocktake,
	})
	s.registerStocktakePrint()
}

// StocktakeActions lists what the actor may do before picking a stocktake: create.
func (s *Service) StocktakeActions(ctx context.Context) (AdjustmentActions, error) {
	acts, err := s.d.Record.TypeActions(ctx, stocktakeType)
	return AdjustmentActions{AllowedActions: acts}, err
}

func (s *Service) Stocktakes(ctx context.Context, f StocktakeFilter) (StocktakeList, error) {
	out := StocktakeList{Items: []StocktakeListItem{}}
	sc, err := s.d.IAM.DataScope(ctx, "stock", PermView)
	if err != nil {
		return out, err
	}
	page, err := stocktakeList.Run(ctx, stocktakeQuery(sc, f))
	if err != nil {
		return out, err
	}
	out.Total = page.Total
	for _, r := range page.Items {
		out.Items = append(out.Items, StocktakeListItem{ID: r.ID, Number: r.Number, Status: r.Status, Date: *platform.DatePtr(r.Date),
			WarehouseID: r.WarehouseID, WarehouseCode: r.WarehouseCode, WarehouseName: r.WarehouseName, Note: r.Note})
	}
	return out, nil
}

// Stocktake returns one stocktake; one the actor may not view does not exist.
func (s *Service) Stocktake(ctx context.Context, id int64) (Stocktake, error) {
	d, err := s.d.Record.Read(ctx, record.Ref{Type: stocktakeType, ID: id})
	if err != nil {
		return Stocktake{}, err
	}
	out, err := readStocktake(ctx, id)
	if err != nil {
		return out, err
	}
	out.AllowedActions, err = s.d.Record.Actions(ctx, d)
	return out, err
}

func readStocktake(ctx context.Context, id int64) (Stocktake, error) {
	h, err := store.GetStocktake.Run(ctx, id)
	if err != nil {
		return Stocktake{}, err
	}
	lines, err := store.GetStocktakeLines.Run(ctx, id)
	if err != nil {
		return Stocktake{}, err
	}
	out := Stocktake{ID: id, Number: h.Number, Status: h.Status, Version: h.Version, Date: *platform.DatePtr(h.Date),
		LegalEntityID: h.LegalEntityID, WarehouseID: h.WarehouseID, WarehouseCode: h.WarehouseCode, WarehouseName: h.WarehouseName,
		Note: h.Note, Lines: make([]StocktakeLineOut, len(lines))}
	counted := h.Status != string(record.Draft) && h.Status != string(record.PendingApproval)
	for i, l := range lines {
		in := StocktakeLine{ItemID: l.ItemID, Serial: platform.TextPtr(l.SerialCode), Counted: platform.PlainDecimal(l.Counted)}
		if l.LotCode.Valid {
			in.Lot = &LotRef{Code: l.LotCode.String, Expiry: platform.DatePtr(l.LotExpiry)}
		}
		o := StocktakeLineOut{StocktakeLine: in, ItemCode: l.ItemCode, ItemName: l.ItemName, Unit: l.Unit}
		if counted {
			o.Book, o.Difference = plainPtr(l.Book), plainPtr(l.Difference)
		} else {
			onHand := platform.PlainDecimal(l.OnHand)
			o.OnHand = &onHand
		}
		out.Lines[i] = o
	}
	return out, nil
}

// plainPtr is a numeric read as '' when null, as the API writes it.
func plainPtr(v string) *string {
	if v == "" {
		return nil
	}
	p := platform.PlainDecimal(v)
	return &p
}

// CreateStocktake adds a draft; the same request id returns the first one.
func (s *Service) CreateStocktake(ctx context.Context, in NewStocktake) (id int64, err error) {
	return id, platform.InTx(ctx, func(ctx context.Context) error {
		h, err := prepareStocktake(ctx, &in.StocktakeFields)
		if err != nil {
			return err
		}
		h.RequestID = in.RequestID
		d, err := s.d.Record.Create(ctx, stocktakeType, h)
		if id = d.ID; err != nil || d.Reused {
			return err
		}
		if err := store.InsertStocktake.Run(ctx, store.InsertStocktakeParams{ID: id, WarehouseID: in.WarehouseID, Note: in.Note}); err != nil {
			return err
		}
		return s.writeStocktakeLines(ctx, id, in.StocktakeFields)
	})
}

func (s *Service) UpdateStocktake(ctx context.Context, id int64, in StocktakeUpdate) error {
	ref := record.Ref{Type: stocktakeType, ID: id}
	return platform.InTx(ctx, func(ctx context.Context) error {
		if _, err := s.d.Record.Read(ctx, ref); err != nil {
			return err
		}
		h, err := prepareStocktake(ctx, &in.StocktakeFields)
		if err != nil {
			return err
		}
		if _, err := s.d.Record.Edit(ctx, ref, in.Version, h); err != nil {
			return err
		}
		if err := store.UpdateStocktake.Run(ctx, store.UpdateStocktakeParams{ID: id, WarehouseID: in.WarehouseID, Note: in.Note}); err != nil {
			return err
		}
		if err := store.DeleteStocktakeLines.Run(ctx, id); err != nil {
			return err
		}
		return s.writeStocktakeLines(ctx, id, in.StocktakeFields)
	})
}

// prepareStocktake checks what a draft can check without touching stock and returns its
// header, at the warehouse's legal entity. Tracking is checked at posting.
func prepareStocktake(ctx context.Context, f *StocktakeFields) (record.Header, error) {
	f.Note = strings.TrimSpace(f.Note)
	w, err := activeWarehouse(ctx, f.WarehouseID)
	if err != nil {
		return record.Header{}, err
	}
	type lineKey struct {
		item int64
		code string
	}
	seen := map[lineKey]bool{}
	ids := make([]int64, len(f.Lines))
	for i, l := range f.Lines {
		ids[i] = l.ItemID
		m := Move{LineNo: i + 1, ItemID: l.ItemID, Lot: l.Lot, Serial: l.Serial}
		if l.Lot != nil && strings.TrimSpace(l.Lot.Code) == "" || l.Serial != nil && strings.TrimSpace(*l.Serial) == "" {
			return record.Header{}, lineErr(ErrCodeBlank, m)
		}
		q, err := decimal.NewFromString(l.Counted)
		if err != nil || q.IsNegative() || !q.Equal(q.Truncate(3)) || q.GreaterThanOrEqual(maxQuantity) {
			return record.Header{}, lineErr(ErrQuantityInvalid, m)
		}
		k := lineKey{l.ItemID, trackCode(m)}
		if seen[k] {
			return record.Header{}, lineErr(ErrCountDuplicate, m)
		}
		seen[k] = true
	}
	if err := itemsExist(ctx, ids); err != nil {
		return record.Header{}, err
	}
	return record.Header{Date: f.Date, OrgUnitID: w.LegalEntityID}, nil
}

func (s *Service) writeStocktakeLines(ctx context.Context, id int64, f StocktakeFields) error {
	a := store.InsertStocktakeLinesParams{StocktakeID: id}
	for i, l := range f.Lines {
		var code, expiry, serial string
		if l.Lot != nil {
			code = strings.TrimSpace(l.Lot.Code)
			if l.Lot.Expiry != nil {
				expiry = *l.Lot.Expiry
			}
		}
		if l.Serial != nil {
			serial = strings.TrimSpace(*l.Serial)
		}
		a.LineNos = append(a.LineNos, int32(i+1))
		a.ItemIds = append(a.ItemIds, l.ItemID)
		a.LotCodes = append(a.LotCodes, code)
		a.LotExpiries = append(a.LotExpiries, expiry)
		a.Serials = append(a.Serials, serial)
		a.Counts = append(a.Counts, l.Counted)
	}
	if err := store.InsertStocktakeLines.Run(ctx, a); err != nil {
		return err
	}
	return s.d.Audit.RecordFor(ctx, "stock.stocktake_saved", audit.Ref{Type: stocktakeType, ID: id},
		map[string]any{"warehouse_id": f.WarehouseID, "lines": len(f.Lines)})
}

// deleteStocktake deletes a draft's rows, lines first.
func deleteStocktake(ctx context.Context, id int64) error {
	if err := store.DeleteStocktakeLines.Run(ctx, id); err != nil {
		return err
	}
	return store.DeleteStocktake.Run(ctx, id)
}

// stocktakeTransition moves on hand to the counts on posting, keeping each line's book
// quantity, and reverses the moves on cancelling.
func (s *Service) stocktakeTransition(ctx context.Context, d record.Doc, _ record.Status) error {
	switch d.Status {
	case record.Posted:
		st, err := readStocktake(ctx, d.ID)
		if err != nil {
			return err
		}
		counts := make([]Count, len(st.Lines))
		for i, l := range st.Lines {
			q, err := decimal.NewFromString(l.Counted)
			if err != nil {
				return err
			}
			counts[i] = Count{LineNo: i + 1, ItemID: l.ItemID, Lot: l.Lot, Serial: l.Serial, Quantity: q}
		}
		res, err := s.RecordCounts(ctx, d.Ref, st.WarehouseID, counts)
		if err != nil {
			return err
		}
		p := store.SetStocktakeResultsParams{StocktakeID: d.ID}
		for _, r := range res {
			p.LineNos, p.Books = append(p.LineNos, int32(r.LineNo)), append(p.Books, r.Book.String())
		}
		return store.SetStocktakeResults.Run(ctx, p)
	case record.Cancelled:
		return s.Void(ctx, d.Ref)
	}
	return nil
}

// OnHandLines lists a warehouse's non-zero balances, optionally of one item, as stocktake lines
// counting what is on hand, to anyone who may view stock somewhere.
func (s *Service) OnHandLines(ctx context.Context, warehouse, itemID int64) (OnHandLines, error) {
	out := OnHandLines{Items: []OnHandLine{}}
	if err := s.requireView(ctx); err != nil {
		return out, err
	}
	rows, err := store.OnHandLines.Run(ctx, store.OnHandLinesParams{WarehouseID: warehouse, ItemID: itemID})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		l := StocktakeLine{ItemID: r.ItemID, Serial: platform.TextPtr(r.SerialCode), Counted: platform.PlainDecimal(r.Quantity)}
		if r.LotCode.Valid {
			l.Lot = &LotRef{Code: r.LotCode.String, Expiry: platform.DatePtr(r.LotExpiry)}
		}
		out.Items = append(out.Items, OnHandLine{StocktakeLine: l, ItemCode: r.ItemCode, ItemName: r.ItemName, Unit: r.Unit})
	}
	return out, nil
}
