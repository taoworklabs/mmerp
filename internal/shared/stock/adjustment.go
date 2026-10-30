package stock

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/stock/internal/store"
)

const adjustmentType = "stock.adjustment"

func (s *Service) registerAdjustment() {
	s.d.Record.Register(record.Type{Code: adjustmentType, Product: "stock", Shared: true, Kind: record.Document, NumberPrefix: "DC",
		Perms:        record.Perms{View: PermView, Edit: PermManage, ViewFiles: PermView, Attach: PermManage},
		OnTransition: s.adjustmentTransition,
		OnDelete:     deleteAdjustment,
	})
	s.registerPrints()
}

// AdjustmentActions lists what the actor may do before picking an adjustment: create.
func (s *Service) AdjustmentActions(ctx context.Context) (AdjustmentActions, error) {
	acts, err := s.d.Record.TypeActions(ctx, adjustmentType)
	return AdjustmentActions{AllowedActions: acts}, err
}

func (s *Service) Adjustments(ctx context.Context, f AdjustmentFilter) (AdjustmentList, error) {
	out := AdjustmentList{Items: []AdjustmentListItem{}}
	sc, err := s.d.IAM.DataScope(ctx, "stock", PermView)
	if err != nil {
		return out, err
	}
	page, err := adjustmentList.Run(ctx, adjustmentQuery(sc, f))
	if err != nil {
		return out, err
	}
	out.Total = page.Total
	for _, r := range page.Items {
		out.Items = append(out.Items, AdjustmentListItem{ID: r.ID, Number: r.Number, Status: r.Status, Date: *platform.DatePtr(r.Date),
			WarehouseID: r.WarehouseID, WarehouseCode: r.WarehouseCode, WarehouseName: r.WarehouseName, Reason: r.Reason})
	}
	return out, nil
}

// Adjustment returns one adjustment; one the actor may not view does not exist.
func (s *Service) Adjustment(ctx context.Context, id int64) (Adjustment, error) {
	d, err := s.d.Record.Read(ctx, record.Ref{Type: adjustmentType, ID: id})
	if err != nil {
		return Adjustment{}, err
	}
	out, err := readAdjustment(ctx, id)
	if err != nil {
		return out, err
	}
	out.AllowedActions, err = s.d.Record.Actions(ctx, d)
	return out, err
}

func readAdjustment(ctx context.Context, id int64) (Adjustment, error) {
	h, err := store.GetAdjustment.Run(ctx, id)
	if err != nil {
		return Adjustment{}, err
	}
	lines, err := store.GetAdjustmentLines.Run(ctx, id)
	if err != nil {
		return Adjustment{}, err
	}
	out := Adjustment{ID: id, Number: h.Number, Status: h.Status, Version: h.Version, Date: *platform.DatePtr(h.Date),
		LegalEntityID: h.LegalEntityID, WarehouseID: h.WarehouseID, WarehouseCode: h.WarehouseCode, WarehouseName: h.WarehouseName,
		Reason: h.Reason, Lines: make([]AdjustmentLineOut, len(lines))}
	for i, l := range lines {
		in := AdjustmentLine{ItemID: l.ItemID, Serial: platform.TextPtr(l.SerialCode), Quantity: platform.PlainDecimal(l.Quantity)}
		if l.LotCode.Valid {
			in.Lot = &LotRef{Code: l.LotCode.String, Expiry: platform.DatePtr(l.LotExpiry)}
		}
		out.Lines[i] = AdjustmentLineOut{AdjustmentLine: in, ItemCode: l.ItemCode, ItemName: l.ItemName, Unit: l.Unit}
	}
	return out, nil
}

// CreateAdjustment adds a draft; the same request id returns the first one.
func (s *Service) CreateAdjustment(ctx context.Context, in NewAdjustment) (id int64, err error) {
	return id, platform.InTx(ctx, func(ctx context.Context) error {
		h, err := prepareAdjustment(ctx, &in.AdjustmentFields)
		if err != nil {
			return err
		}
		h.RequestID = in.RequestID
		d, err := s.d.Record.Create(ctx, adjustmentType, h)
		if id = d.ID; err != nil || d.Reused {
			return err
		}
		if err := store.InsertAdjustment.Run(ctx, store.InsertAdjustmentParams{ID: id, WarehouseID: in.WarehouseID, Reason: in.Reason}); err != nil {
			return err
		}
		return s.writeAdjustmentLines(ctx, id, in.AdjustmentFields)
	})
}

func (s *Service) UpdateAdjustment(ctx context.Context, id int64, in AdjustmentUpdate) error {
	ref := record.Ref{Type: adjustmentType, ID: id}
	return platform.InTx(ctx, func(ctx context.Context) error {
		if _, err := s.d.Record.Read(ctx, ref); err != nil {
			return err
		}
		h, err := prepareAdjustment(ctx, &in.AdjustmentFields)
		if err != nil {
			return err
		}
		if _, err := s.d.Record.Edit(ctx, ref, in.Version, h); err != nil {
			return err
		}
		if err := store.UpdateAdjustment.Run(ctx, store.UpdateAdjustmentParams{ID: id, WarehouseID: in.WarehouseID, Reason: in.Reason}); err != nil {
			return err
		}
		if err := store.DeleteAdjustmentLines.Run(ctx, id); err != nil {
			return err
		}
		return s.writeAdjustmentLines(ctx, id, in.AdjustmentFields)
	})
}

// prepareAdjustment checks what a draft can check without touching stock and returns its
// header, at the warehouse's legal entity. Tracking and balances are checked at posting.
func prepareAdjustment(ctx context.Context, f *AdjustmentFields) (record.Header, error) {
	f.Reason = strings.TrimSpace(f.Reason)
	w, err := store.AdjustmentWarehouse.Run(ctx, f.WarehouseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return record.Header{}, platform.ErrNotFound
	}
	if err != nil {
		return record.Header{}, err
	}
	if !w.Active {
		return record.Header{}, ErrWarehouseInactive
	}
	ids := make([]int64, len(f.Lines))
	for i, l := range f.Lines {
		ids[i] = l.ItemID
		q, err := decimal.NewFromString(l.Quantity)
		if err != nil || q.IsZero() || !q.Equal(q.Truncate(3)) || q.Abs().GreaterThanOrEqual(maxQuantity) {
			return record.Header{}, lineErr(ErrQuantityInvalid, Move{LineNo: i + 1})
		}
	}
	n, err := store.CountItems.Run(ctx, ids)
	if err != nil {
		return record.Header{}, err
	}
	if distinct(ids) != int(n) {
		return record.Header{}, platform.ErrNotFound
	}
	return record.Header{Date: f.Date, OrgUnitID: w.LegalEntityID}, nil
}

func distinct(ids []int64) int {
	seen := map[int64]bool{}
	for _, id := range ids {
		seen[id] = true
	}
	return len(seen)
}

func (s *Service) writeAdjustmentLines(ctx context.Context, id int64, f AdjustmentFields) error {
	a := store.InsertAdjustmentLinesParams{AdjustmentID: id}
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
		a.Quantities = append(a.Quantities, l.Quantity)
	}
	if err := store.InsertAdjustmentLines.Run(ctx, a); err != nil {
		return err
	}
	return s.d.Audit.RecordFor(ctx, "stock.adjustment_saved", audit.Ref{Type: adjustmentType, ID: id},
		map[string]any{"warehouse_id": f.WarehouseID, "lines": len(f.Lines)})
}

// deleteAdjustment deletes a draft's rows, lines first.
func deleteAdjustment(ctx context.Context, id int64) error {
	if err := store.DeleteAdjustmentLines.Run(ctx, id); err != nil {
		return err
	}
	return store.DeleteAdjustment.Run(ctx, id)
}

// adjustmentTransition moves the stock of the lines on posting and reverses it on cancelling.
func (s *Service) adjustmentTransition(ctx context.Context, d record.Doc, _ record.Status) error {
	switch d.Status {
	case record.Posted:
		a, err := readAdjustment(ctx, d.ID)
		if err != nil {
			return err
		}
		moves := make([]Move, len(a.Lines))
		for i, l := range a.Lines {
			q, err := decimal.NewFromString(l.Quantity)
			if err != nil {
				return err
			}
			moves[i] = Move{LineNo: i + 1, WarehouseID: a.WarehouseID, ItemID: l.ItemID, Lot: l.Lot, Serial: l.Serial, Quantity: q}
		}
		return s.Record(ctx, d.Ref, moves)
	case record.Cancelled:
		return s.Void(ctx, d.Ref)
	}
	return nil
}
