package stock

import (
	"cmp"
	"context"
	"errors"
	"maps"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

	for _, m := range moves {
		if err := check(m, whs, items, h.LegalEntityID); err != nil {
			return err
		}
	}
	// Lots and serials are created in (item, code) order, so two documents creating the
	// same new ones queue on the unique index instead of deadlocking.
	order := make([]int, len(moves))
	for i := range order {
		order[i] = i
	}
	slices.SortFunc(order, func(a, b int) int {
		return cmp.Or(cmp.Compare(moves[a].ItemID, moves[b].ItemID), cmp.Compare(trackCode(moves[a]), trackCode(moves[b])))
	})
	keyOf := make([]key, len(moves))
	for _, i := range order {
		m := moves[i]
		k := key{w: m.WarehouseID, i: m.ItemID}
		if k.l, k.s, err = s.resolve(ctx, items[m.ItemID].Tracking, m); err != nil {
			return err
		}
		keyOf[i] = k
	}

	lines := make([]line, len(moves))
	for i, m := range moves {
		lines[i] = line{k: keyOf[i], lineNo: m.LineNo, qty: m.Quantity}
	}
	return apply(ctx, ref, h.Date, items, lines)
}

// line is one resolved move to write; reverses is the id of the move it undoes, or 0.
type line struct {
	k        key
	lineNo   int
	qty      decimal.Decimal
	reverses int64
}

// apply locks and updates the balances of lines, refuses what would go short, and inserts
// the moves. Record and Void share it so both queue and check the same way.
func apply(ctx context.Context, ref record.Ref, date pgtype.Date, items map[int64]store.MoveItemsRow, lines []line) error {
	ins := store.InsertMovesParams{DocType: ref.Type, DocID: ref.ID, Date: date}
	deltas := map[key]decimal.Decimal{}
	firstLine := map[key]int{}
	var serials []int64
	for _, l := range lines {
		k := l.k
		if k.s != 0 {
			serials = append(serials, k.s)
		}
		if _, seen := firstLine[k]; !seen {
			firstLine[k] = l.lineNo
		}
		deltas[k] = deltas[k].Add(l.qty)
		ins.LineNos = append(ins.LineNos, int32(l.lineNo))
		ins.WarehouseIds = append(ins.WarehouseIds, k.w)
		ins.ItemIds = append(ins.ItemIds, k.i)
		ins.LotIds = append(ins.LotIds, k.l)
		ins.SerialIds = append(ins.SerialIds, k.s)
		ins.Quantities = append(ins.Quantities, l.qty.String())
		ins.ReversesIds = append(ins.ReversesIds, l.reverses)
	}
	// Lock order: serials, then balances by key, so documents touching the same rows queue.
	slices.Sort(serials)
	serials = slices.Compact(serials)
	if err := store.LockSerials.Run(ctx, serials); err != nil {
		return err
	}
	keys := slices.SortedFunc(maps.Keys(deltas), key.compare)
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
	result := map[key]decimal.Decimal{}
	for _, r := range rows {
		if result[key{r.WarehouseID, r.ItemID, r.LotID, r.SerialID}], err = decimal.NewFromString(r.Quantity); err != nil {
			return err
		}
	}
	for _, k := range keys {
		q, it := result[k], items[k.i]
		// Only taking stock out can fail; a balance already below zero may still be refilled.
		if !deltas[k].IsNegative() || !q.IsNegative() || (it.Tracking == item.TrackQuantity && it.AllowNegative) {
			continue
		}
		return &platform.Error{Status: ErrInsufficient.Status, Code: ErrInsufficient.Code, Params: map[string]any{
			"line_no": firstLine[k], "warehouse_id": k.w, "item_id": k.i,
			"lot_id": nullID(k.l), "serial_id": nullID(k.s), "missing": q.Neg().String()}}
	}
	if len(serials) > 0 {
		over, err := store.SerialsOverOne.Run(ctx, serials)
		if err != nil {
			return err
		}
		for _, k := range keys {
			if k.s != 0 && slices.Contains(over, k.s) {
				return &platform.Error{Status: ErrSerialQuantity.Status, Code: ErrSerialQuantity.Code, Params: map[string]any{"line_no": firstLine[k]}}
			}
		}
	}
	return store.InsertMoves.Run(ctx, ins)
}

// Void cancels the moves of a document by inserting a reversal for each one not yet
// reversed, in the caller's transaction. It goes through the same balance checks as Record,
// so goods already issued elsewhere refuse it. A document without moves is a no-op.
func (s *Service) Void(ctx context.Context, ref record.Ref) error {
	h, err := store.DocumentHeader.Run(ctx, store.DocumentHeaderParams{DocType: ref.Type, ID: ref.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return platform.ErrNotFound
	}
	if err != nil {
		return err
	}
	rows, err := store.OpenMoves.Run(ctx, store.OpenMovesParams{DocType: ref.Type, DocID: ref.ID})
	if err != nil || len(rows) == 0 {
		return err
	}
	var ids []int64
	lines := make([]line, len(rows))
	for i, r := range rows {
		q, err := decimal.NewFromString(r.Quantity)
		if err != nil {
			return err
		}
		ids = append(ids, r.ItemID)
		lines[i] = line{k: key{r.WarehouseID, r.ItemID, r.LotID, r.SerialID}, lineNo: int(r.LineNo), qty: q.Neg(), reverses: r.ID}
	}
	irows, err := store.MoveItems.Run(ctx, ids)
	if err != nil {
		return err
	}
	items := map[int64]store.MoveItemsRow{}
	for _, i := range irows {
		items[i.ID] = i
	}
	return apply(ctx, ref, h.Date, items, lines)
}

// nullID is nil for no lot or serial, so the API sends null.
func nullID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
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

// maxQuantity keeps a balance inside numeric(15,3).
var maxQuantity = decimal.New(1, 12)

// check refuses a move before anything is written: its warehouse, quantity and the
// lot or serial its item's tracking asks for.
func check(m Move, whs map[int64]store.MoveWarehousesRow, items map[int64]store.MoveItemsRow, legalEntity int64) error {
	w, ok := whs[m.WarehouseID]
	it, found := items[m.ItemID]
	switch {
	case !ok || !found:
		return platform.ErrNotFound
	case w.LegalEntityID != legalEntity:
		return lineErr(ErrWarehouseEntity, m)
	case !w.Active:
		return lineErr(ErrWarehouseInactive, m)
	case m.Quantity.IsZero() || !m.Quantity.Equal(m.Quantity.Truncate(3)) || m.Quantity.Abs().GreaterThanOrEqual(maxQuantity):
		return lineErr(ErrQuantityInvalid, m)
	}
	hasLot := m.Lot != nil
	hasSerial := m.Serial != nil
	switch it.Tracking {
	case item.TrackNone:
		return lineErr(ErrUntracked, m)
	case item.TrackQuantity:
		if hasLot || hasSerial {
			return lineErr(ErrLotNotTracked, m)
		}
	case item.TrackLot:
		if hasSerial {
			return lineErr(ErrLotNotTracked, m)
		}
		if trackCode(m) == "" {
			return lineErr(ErrLotRequired, m)
		}
	default: // serial
		if hasLot {
			return lineErr(ErrLotNotTracked, m)
		}
		if trackCode(m) == "" {
			return lineErr(ErrSerialRequired, m)
		}
		if !m.Quantity.Abs().Equal(decimal.NewFromInt(1)) {
			return lineErr(ErrSerialQuantity, m)
		}
	}
	return nil
}

// trackCode is the move's lot or serial code as the unique indexes compare it.
func trackCode(m Move) string {
	switch {
	case m.Lot != nil:
		return strings.ToLower(strings.TrimSpace(m.Lot.Code))
	case m.Serial != nil:
		return strings.ToLower(strings.TrimSpace(*m.Serial))
	}
	return ""
}

// resolve returns a checked move's lot and serial ids, creating the lot or serial when
// missing. An outgoing move naming a new one then fails the balance check, rolling the
// creation back.
func (s *Service) resolve(ctx context.Context, tracking string, m Move) (lot, serial int64, err error) {
	switch tracking {
	case item.TrackLot:
		code := strings.TrimSpace(m.Lot.Code)
		if err := store.InsertLot.Run(ctx, store.InsertLotParams{ItemID: m.ItemID, Code: code, Expiry: platform.NullDate(m.Lot.Expiry)}); err != nil {
			return 0, 0, err
		}
		l, err := store.LotByCode.Run(ctx, store.LotByCodeParams{ItemID: m.ItemID, Code: code})
		if err != nil {
			return 0, 0, err
		}
		if m.Lot.Expiry != nil {
			if have := platform.DatePtr(l.Expiry); have == nil || *have != *m.Lot.Expiry {
				return 0, 0, lineErr(ErrLotExpiryMismatch, m)
			}
		}
		return l.ID, 0, nil
	case item.TrackSerial:
		code := strings.TrimSpace(*m.Serial)
		if err := store.InsertSerial.Run(ctx, store.InsertSerialParams{ItemID: m.ItemID, Code: code}); err != nil {
			return 0, 0, err
		}
		id, err := store.SerialByCode.Run(ctx, store.SerialByCodeParams{ItemID: m.ItemID, Code: code})
		return 0, id, err
	}
	return 0, 0, nil
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
