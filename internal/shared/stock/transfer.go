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

const transferType = "stock.transfer"

func (s *Service) registerTransfer() {
	s.d.Record.Register(record.Type{Code: transferType, Product: "stock", Shared: true, Kind: record.Document, NumberPrefix: "CK",
		Perms:        record.Perms{View: PermView, Edit: PermManage, ViewFiles: PermView, Attach: PermManage},
		OnTransition: s.transferTransition,
		OnDelete:     deleteTransfer,
	})
	s.registerTransferPrint()
}

// TransferActions lists what the actor may do before picking a transfer: create.
func (s *Service) TransferActions(ctx context.Context) (AdjustmentActions, error) {
	acts, err := s.d.Record.TypeActions(ctx, transferType)
	return AdjustmentActions{AllowedActions: acts}, err
}

func (s *Service) Transfers(ctx context.Context, f TransferFilter) (TransferList, error) {
	out := TransferList{Items: []TransferListItem{}}
	sc, err := s.d.IAM.DataScope(ctx, "stock", PermView)
	if err != nil {
		return out, err
	}
	page, err := transferList.Run(ctx, transferQuery(sc, f))
	if err != nil {
		return out, err
	}
	out.Total = page.Total
	for _, r := range page.Items {
		out.Items = append(out.Items, TransferListItem{ID: r.ID, Number: r.Number, Status: r.Status, Date: *platform.DatePtr(r.Date),
			FromWarehouseID: r.FromWarehouseID, FromWarehouseCode: r.FromWarehouseCode, FromWarehouseName: r.FromWarehouseName,
			ToWarehouseID: r.ToWarehouseID, ToWarehouseCode: r.ToWarehouseCode, ToWarehouseName: r.ToWarehouseName, Note: r.Note})
	}
	return out, nil
}

// Transfer returns one transfer; one the actor may not view does not exist.
func (s *Service) Transfer(ctx context.Context, id int64) (Transfer, error) {
	d, err := s.d.Record.Read(ctx, record.Ref{Type: transferType, ID: id})
	if err != nil {
		return Transfer{}, err
	}
	out, err := readTransfer(ctx, id)
	if err != nil {
		return out, err
	}
	out.AllowedActions, err = s.d.Record.Actions(ctx, d)
	return out, err
}

func readTransfer(ctx context.Context, id int64) (Transfer, error) {
	h, err := store.GetTransfer.Run(ctx, id)
	if err != nil {
		return Transfer{}, err
	}
	lines, err := store.GetTransferLines.Run(ctx, id)
	if err != nil {
		return Transfer{}, err
	}
	out := Transfer{ID: id, Number: h.Number, Status: h.Status, Version: h.Version, Date: *platform.DatePtr(h.Date), LegalEntityID: h.LegalEntityID,
		FromWarehouseID: h.FromWarehouseID, FromWarehouseCode: h.FromWarehouseCode, FromWarehouseName: h.FromWarehouseName,
		ToWarehouseID: h.ToWarehouseID, ToWarehouseCode: h.ToWarehouseCode, ToWarehouseName: h.ToWarehouseName,
		Note: h.Note, Lines: make([]TransferLineOut, len(lines))}
	for i, l := range lines {
		out.Lines[i] = TransferLineOut{TransferLine: TransferLine{ItemID: l.ItemID, Lot: platform.TextPtr(l.LotCode), Serial: platform.TextPtr(l.SerialCode),
			Quantity: platform.PlainDecimal(l.Quantity)}, ItemCode: l.ItemCode, ItemName: l.ItemName, Unit: l.Unit}
	}
	return out, nil
}

// CreateTransfer adds a draft; the same request id returns the first one.
func (s *Service) CreateTransfer(ctx context.Context, in NewTransfer) (id int64, err error) {
	return id, platform.InTx(ctx, func(ctx context.Context) error {
		h, err := prepareTransfer(ctx, &in.TransferFields)
		if err != nil {
			return err
		}
		h.RequestID = in.RequestID
		d, err := s.d.Record.Create(ctx, transferType, h)
		if id = d.ID; err != nil || d.Reused {
			return err
		}
		if err := store.InsertTransfer.Run(ctx, store.InsertTransferParams{ID: id, FromWarehouseID: in.FromWarehouseID, ToWarehouseID: in.ToWarehouseID, Note: in.Note}); err != nil {
			return err
		}
		return s.writeTransferLines(ctx, id, in.TransferFields)
	})
}

func (s *Service) UpdateTransfer(ctx context.Context, id int64, in TransferUpdate) error {
	ref := record.Ref{Type: transferType, ID: id}
	return platform.InTx(ctx, func(ctx context.Context) error {
		if _, err := s.d.Record.Read(ctx, ref); err != nil {
			return err
		}
		h, err := prepareTransfer(ctx, &in.TransferFields)
		if err != nil {
			return err
		}
		if _, err := s.d.Record.Edit(ctx, ref, in.Version, h); err != nil {
			return err
		}
		if err := store.UpdateTransfer.Run(ctx, store.UpdateTransferParams{ID: id, FromWarehouseID: in.FromWarehouseID, ToWarehouseID: in.ToWarehouseID, Note: in.Note}); err != nil {
			return err
		}
		if err := store.DeleteTransferLines.Run(ctx, id); err != nil {
			return err
		}
		return s.writeTransferLines(ctx, id, in.TransferFields)
	})
}

// prepareTransfer checks what a draft can check without touching stock and returns its
// header, at the source warehouse's legal entity. Tracking and balances are checked at posting.
func prepareTransfer(ctx context.Context, f *TransferFields) (record.Header, error) {
	f.Note = strings.TrimSpace(f.Note)
	if f.FromWarehouseID == f.ToWarehouseID {
		return record.Header{}, ErrTransferSameWarehouse
	}
	from, err := activeWarehouse(ctx, f.FromWarehouseID)
	if err != nil {
		return record.Header{}, err
	}
	to, err := activeWarehouse(ctx, f.ToWarehouseID)
	if err != nil {
		return record.Header{}, err
	}
	if from.LegalEntityID != to.LegalEntityID {
		return record.Header{}, ErrWarehouseEntity
	}
	ids := make([]int64, len(f.Lines))
	for i, l := range f.Lines {
		ids[i] = l.ItemID
	}
	tracking, err := itemTrackings(ctx, ids)
	if err != nil {
		return record.Header{}, err
	}
	for i, l := range f.Lines {
		if l.Lot != nil && strings.TrimSpace(*l.Lot) == "" || l.Serial != nil && strings.TrimSpace(*l.Serial) == "" {
			return record.Header{}, lineErr(ErrCodeBlank, Move{LineNo: i + 1})
		}
		q, err := decimal.NewFromString(l.Quantity)
		if err != nil || !q.IsPositive() || !q.Equal(q.Truncate(3)) || q.GreaterThanOrEqual(maxQuantity) {
			return record.Header{}, lineErr(ErrQuantityInvalid, Move{LineNo: i + 1})
		}
		if err := CheckShape(tracking[l.ItemID], l.Lot, l.Serial, q, i+1); err != nil {
			return record.Header{}, err
		}
	}
	return record.Header{Date: f.Date, OrgUnitID: from.LegalEntityID}, nil
}

// activeWarehouse reads a warehouse a draft names; an inactive one is refused.
func activeWarehouse(ctx context.Context, id int64) (store.AdjustmentWarehouseRow, error) {
	w, err := store.AdjustmentWarehouse.Run(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return w, platform.ErrNotFound
	}
	if err == nil && !w.Active {
		err = ErrWarehouseInactive
	}
	return w, err
}

// itemTrackings reads the tracking of the items a draft names, without locking them;
// an id naming no item is refused.
func itemTrackings(ctx context.Context, ids []int64) (map[int64]string, error) {
	rows, err := store.ItemTrackings.Run(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]string, len(rows))
	for _, r := range rows {
		out[r.ID] = r.Tracking
	}
	if distinct(ids) != len(out) {
		return nil, platform.ErrNotFound
	}
	return out, nil
}

func (s *Service) writeTransferLines(ctx context.Context, id int64, f TransferFields) error {
	a := store.InsertTransferLinesParams{TransferID: id}
	for i, l := range f.Lines {
		var code, serial string
		if l.Lot != nil {
			code = strings.TrimSpace(*l.Lot)
		}
		if l.Serial != nil {
			serial = strings.TrimSpace(*l.Serial)
		}
		a.LineNos = append(a.LineNos, int32(i+1))
		a.ItemIds = append(a.ItemIds, l.ItemID)
		a.LotCodes = append(a.LotCodes, code)
		a.Serials = append(a.Serials, serial)
		a.Quantities = append(a.Quantities, l.Quantity)
	}
	if err := store.InsertTransferLines.Run(ctx, a); err != nil {
		return err
	}
	return s.d.Audit.RecordFor(ctx, "stock.transfer_saved", audit.Ref{Type: transferType, ID: id},
		map[string]any{"from_warehouse_id": f.FromWarehouseID, "to_warehouse_id": f.ToWarehouseID, "lines": len(f.Lines)})
}

// deleteTransfer deletes a draft's rows, lines first.
func deleteTransfer(ctx context.Context, id int64) error {
	if err := store.DeleteTransferLines.Run(ctx, id); err != nil {
		return err
	}
	return store.DeleteTransfer.Run(ctx, id)
}

// transferTransition takes each line out of the source and into the destination on posting,
// and reverses both on cancelling.
func (s *Service) transferTransition(ctx context.Context, d record.Doc, _ record.Status) error {
	switch d.Status {
	case record.Posted:
		t, err := readTransfer(ctx, d.ID)
		if err != nil {
			return err
		}
		moves := make([]Move, 0, 2*len(t.Lines))
		for i, l := range t.Lines {
			q, err := decimal.NewFromString(l.Quantity)
			if err != nil {
				return err
			}
			// The lot exists in the source, so its expiry is not restated.
			var lot *LotRef
			if l.Lot != nil {
				lot = &LotRef{Code: *l.Lot}
			}
			if err := knownCode(ctx, l, i+1); err != nil {
				return err
			}
			moves = append(moves,
				Move{LineNo: i + 1, WarehouseID: t.FromWarehouseID, ItemID: l.ItemID, Lot: lot, Serial: l.Serial, Quantity: q.Neg()},
				Move{LineNo: i + 1, WarehouseID: t.ToWarehouseID, ItemID: l.ItemID, Lot: lot, Serial: l.Serial, Quantity: q})
		}
		return s.Record(ctx, d.Ref, moves)
	case record.Cancelled:
		return s.Void(ctx, d.Ref)
	}
	return nil
}

// knownCode refuses a line naming a lot or serial its item never had: the incoming side would
// create it and the outgoing side then report a shortage instead.
func knownCode(ctx context.Context, l TransferLineOut, lineNo int) error {
	m := Move{LineNo: lineNo}
	switch {
	case l.Lot != nil:
		_, err := store.LotByCode.Run(ctx, store.LotByCodeParams{ItemID: l.ItemID, Code: *l.Lot})
		if errors.Is(err, pgx.ErrNoRows) {
			return lineErr(ErrLotNotFound, m)
		}
		return err
	case l.Serial != nil:
		_, err := store.SerialByCode.Run(ctx, store.SerialByCodeParams{ItemID: l.ItemID, Code: *l.Serial})
		if errors.Is(err, pgx.ErrNoRows) {
			return lineErr(ErrSerialNotFound, m)
		}
		return err
	}
	return nil
}
