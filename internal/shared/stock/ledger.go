package stock

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/item"
	"github.com/taoworklabs/mmerp/internal/shared/stock/internal/store"
)

// key is a balance row; no lot or serial is 0.
type key struct{ w, i, l, s int64 }

func (k key) compare(o key) int {
	return cmp.Or(cmp.Compare(k.w, o.w), cmp.Compare(k.i, o.i), cmp.Compare(k.l, o.l), cmp.Compare(k.s, o.s))
}

func lineErr(e *platform.Error, m Move) error {
	return &platform.Error{Status: e.Status, Code: e.Code, Params: map[string]any{"line_no": m.LineNo}}
}

// Record writes the moves of a document, in the caller's transaction, under the document's
// legal entity and date. It checks no permission: documents call it from their transitions.
// It refuses while earlier moves of the document are not reversed.
func (s *Service) Record(ctx context.Context, ref record.Ref, moves []Move) error {
	h, err := store.DocumentHeader.Run(ctx, store.DocumentHeaderParams{DocType: ref.Type, ID: ref.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return platform.ErrNotFound
	}
	if err != nil {
		return err
	}
	if live, err := store.HasLiveMoves.Run(ctx, store.HasLiveMovesParams{DocType: ref.Type, DocID: ref.ID}); err != nil || live {
		return platform.OrErr(err, ErrAlreadyRecorded)
	}
	if len(moves) == 0 {
		return nil
	}
	whs, items, err := s.moveMasters(ctx, moves)
	if err != nil {
		return err
	}

	ins := store.InsertMovesParams{DocType: ref.Type, DocID: ref.ID, Date: h.Date}
	deltas := map[key]decimal.Decimal{}
	firstLine := map[key]Move{}
	var serials []int64
	for _, m := range moves {
		w, ok := whs[m.WarehouseID]
		it, found := items[m.ItemID]
		switch {
		case !ok || !found:
			return platform.ErrNotFound
		case w.LegalEntityID != h.LegalEntityID:
			return lineErr(ErrWarehouseEntity, m)
		case !w.Active:
			return lineErr(ErrWarehouseInactive, m)
		}
		k := key{w: m.WarehouseID, i: m.ItemID}
		if k.l, k.s, err = s.resolve(ctx, it.Tracking, m); err != nil {
			return err
		}
		if k.s != 0 {
			serials = append(serials, k.s)
		}
		if _, seen := firstLine[k]; !seen {
			firstLine[k] = m
		}
		deltas[k] = deltas[k].Add(m.Quantity)
		ins.LineNos = append(ins.LineNos, int32(m.LineNo))
		ins.WarehouseIds = append(ins.WarehouseIds, k.w)
		ins.ItemIds = append(ins.ItemIds, k.i)
		ins.LotIds = append(ins.LotIds, k.l)
		ins.SerialIds = append(ins.SerialIds, k.s)
		ins.Quantities = append(ins.Quantities, m.Quantity.String())
	}

	// Lock order: serials, then balances by key, so documents touching the same rows queue.
	slices.Sort(serials)
	serials = slices.Compact(serials)
	if err := store.LockSerials.Run(ctx, serials); err != nil {
		return err
	}
	keys := slices.SortedFunc(func(yield func(key) bool) {
		for k := range deltas {
			if !yield(k) {
				return
			}
		}
	}, key.compare)
	var ws, is, ls, ss []int64
	var ds []string
	for _, k := range keys {
		ws, is, ls, ss = append(ws, k.w), append(is, k.i), append(ls, k.l), append(ss, k.s)
		ds = append(ds, deltas[k].String())
	}
	if err := store.InsertBalances.Run(ctx, store.InsertBalancesParams{WarehouseIds: ws, ItemIds: is, LotIds: ls, SerialIds: ss}); err != nil {
		return err
	}
	if err := store.LockBalances.Run(ctx, store.LockBalancesParams{WarehouseIds: ws, ItemIds: is, LotIds: ls, SerialIds: ss}); err != nil {
		return err
	}
	rows, err := store.ApplyDeltas.Run(ctx, store.ApplyDeltasParams{WarehouseIds: ws, ItemIds: is, LotIds: ls, SerialIds: ss, Deltas: ds})
	if err != nil {
		return err
	}
	for _, r := range rows {
		k := key{r.WarehouseID, r.ItemID, r.LotID, r.SerialID}
		q, err := decimal.NewFromString(r.Quantity)
		if err != nil {
			return err
		}
		it := items[k.i]
		// Only taking stock out can fail; a balance already below zero may still be refilled.
		if !deltas[k].IsNegative() || !q.IsNegative() || (it.Tracking == item.TrackQuantity && it.AllowNegative) {
			continue
		}
		return &platform.Error{Status: ErrInsufficient.Status, Code: ErrInsufficient.Code, Params: map[string]any{
			"line_no": firstLine[k].LineNo, "warehouse_id": k.w, "item_id": k.i, "missing": q.Neg().String()}}
	}
	if len(serials) > 0 {
		over, err := store.SerialsOverOne.Run(ctx, serials)
		if err != nil {
			return err
		}
		if len(over) > 0 {
			for k, m := range firstLine {
				if k.s == over[0] {
					return lineErr(ErrSerialQuantity, m)
				}
			}
		}
	}
	return store.InsertMoves.Run(ctx, ins)
}

func (s *Service) moveMasters(ctx context.Context, moves []Move) (map[int64]store.MoveWarehousesRow, map[int64]store.MoveItemsRow, error) {
	var wids, iids []int64
	for _, m := range moves {
		wids, iids = append(wids, m.WarehouseID), append(iids, m.ItemID)
	}
	wrows, err := store.MoveWarehouses.Run(ctx, wids)
	if err != nil {
		return nil, nil, err
	}
	irows, err := store.MoveItems.Run(ctx, iids)
	if err != nil {
		return nil, nil, err
	}
	whs, items := map[int64]store.MoveWarehousesRow{}, map[int64]store.MoveItemsRow{}
	for _, w := range wrows {
		whs[w.ID] = w
	}
	for _, i := range irows {
		items[i.ID] = i
	}
	return whs, items, nil
}

// resolve checks a move against its item's tracking and returns its lot and serial ids,
// creating the lot or serial when missing. An outgoing move naming a new one then fails
// the balance check, rolling the creation back.
func (s *Service) resolve(ctx context.Context, tracking string, m Move) (lot, serial int64, err error) {
	var lotCode, serialCode string
	if m.Lot != nil {
		lotCode = strings.TrimSpace(m.Lot.Code)
	}
	if m.Serial != nil {
		serialCode = strings.TrimSpace(*m.Serial)
	}
	switch tracking {
	case item.TrackNone:
		return 0, 0, lineErr(ErrUntracked, m)
	case item.TrackQuantity:
		if m.Lot != nil || m.Serial != nil {
			return 0, 0, lineErr(ErrLotNotTracked, m)
		}
		return 0, 0, nil
	case item.TrackLot:
		if m.Serial != nil {
			return 0, 0, lineErr(ErrLotNotTracked, m)
		}
		if lotCode == "" {
			return 0, 0, lineErr(ErrLotRequired, m)
		}
		if err := store.InsertLot.Run(ctx, store.InsertLotParams{ItemID: m.ItemID, Code: lotCode, Expiry: platform.NullDate(m.Lot.Expiry)}); err != nil {
			return 0, 0, err
		}
		l, err := store.LotByCode.Run(ctx, store.LotByCodeParams{ItemID: m.ItemID, Code: lotCode})
		if err != nil {
			return 0, 0, err
		}
		if m.Lot.Expiry != nil {
			if have := platform.DatePtr(l.Expiry); have == nil || *have != *m.Lot.Expiry {
				return 0, 0, lineErr(ErrLotExpiryMismatch, m)
			}
		}
		return l.ID, 0, nil
	default: // serial
		if m.Lot != nil {
			return 0, 0, lineErr(ErrLotNotTracked, m)
		}
		if serialCode == "" {
			return 0, 0, lineErr(ErrSerialRequired, m)
		}
		if !m.Quantity.Abs().Equal(decimal.NewFromInt(1)) {
			return 0, 0, lineErr(ErrSerialQuantity, m)
		}
		if err := store.InsertSerial.Run(ctx, store.InsertSerialParams{ItemID: m.ItemID, Code: serialCode}); err != nil {
			return 0, 0, err
		}
		id, err := store.SerialByCode.Run(ctx, store.SerialByCodeParams{ItemID: m.ItemID, Code: serialCode})
		return 0, id, err
	}
}

// SuggestLots picks on-hand lots of an item for qty, first expiring first, skipping those
// expired before today (YYYY-MM-DD). Short stock returns what there is. It checks no permission.
func (s *Service) SuggestLots(ctx context.Context, warehouse, itemID int64, qty decimal.Decimal, today string) ([]LotPick, error) {
	rows, err := store.LotsOnHand.Run(ctx, store.LotsOnHandParams{WarehouseID: warehouse, ItemID: itemID, Today: platform.NullDate(&today)})
	if err != nil {
		return nil, err
	}
	var out []LotPick
	for _, r := range rows {
		if !qty.IsPositive() {
			break
		}
		have, err := decimal.NewFromString(r.Quantity)
		if err != nil {
			return nil, err
		}
		take := decimal.Min(have, qty)
		out = append(out, LotPick{LotID: r.ID, Code: r.Code, Expiry: platform.DatePtr(r.Expiry), Quantity: take})
		qty = qty.Sub(take)
	}
	return out, nil
}
