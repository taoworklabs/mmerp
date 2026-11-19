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
// legal entity and date, and returns the value written for each move (0 for a value-only move
// skipped). It checks no permission: documents call it from their transitions. It refuses
// while earlier moves of the document are not reversed. Callers hold the document's row lock
// (they run inside record.Transition), so one document never records twice.
func (s *Service) Record(ctx context.Context, ref record.Ref, moves []Move) ([]int64, error) {
	h, err := store.DocumentHeader.Run(ctx, store.DocumentHeaderParams{DocType: ref.Type, ID: ref.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, platform.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if live, err := store.HasLiveMoves.Run(ctx, store.HasLiveMovesParams{DocType: ref.Type, DocID: ref.ID}); err != nil || live {
		return nil, platform.OrErr(err, ErrAlreadyRecorded)
	}
	if len(moves) == 0 {
		return nil, nil
	}
	whs, items, err := s.moveMasters(ctx, moves)
	if err != nil {
		return nil, err
	}

	for _, m := range moves {
		if err := check(m, whs, items, h.LegalEntityID); err != nil {
			return nil, err
		}
	}
	keyOf := make([]key, len(moves))
	for _, i := range creationOrder(moves) {
		m := moves[i]
		k := key{w: m.WarehouseID, i: m.ItemID}
		if !valueOnly(m) {
			if k.l, k.s, err = s.resolve(ctx, items[m.ItemID].Tracking, m); err != nil {
				return nil, err
			}
		}
		keyOf[i] = k
	}

	lines := make([]line, len(moves))
	for i, m := range moves {
		lines[i] = line{k: keyOf[i], lineNo: m.LineNo, qty: m.Quantity, value: m.Value}
	}
	return apply(ctx, ref, h.LegalEntityID, h.Date, items, lines)
}

// valueOnly reports a move changing the value of an item but no quantity, in no warehouse.
func valueOnly(m Move) bool { return m.WarehouseID == 0 }

// creationOrder sorts moves by (item, code), so two documents creating the same new lots or
// serials queue on the unique index instead of deadlocking.
func creationOrder(moves []Move) []int {
	order := make([]int, len(moves))
	for i := range order {
		order[i] = i
	}
	slices.SortFunc(order, func(a, b int) int {
		// Incoming first for the same code, so an outgoing line sees what the document itself creates.
		return cmp.Or(cmp.Compare(moves[a].ItemID, moves[b].ItemID), cmp.Compare(trackCode(moves[a]), trackCode(moves[b])),
			-cmp.Compare(moves[a].Quantity.Sign(), moves[b].Quantity.Sign()), cmp.Compare(moves[a].LineNo, moves[b].LineNo), cmp.Compare(a, b))
	})
	return order
}

// RecordCounts moves on hand of a warehouse to counted quantities, in the caller's transaction:
// each count's difference from on hand, read under the balance lock, becomes a move. A positive
// count of an unknown lot or serial creates it; a zero count of one leaves nothing. Counts name
// distinct keys. Results follow counts. Like Record, it checks no permission and records once.
func (s *Service) RecordCounts(ctx context.Context, ref record.Ref, warehouse int64, counts []Count) ([]CountResult, error) {
	h, err := store.DocumentHeader.Run(ctx, store.DocumentHeaderParams{DocType: ref.Type, ID: ref.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, platform.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if live, err := store.HasLiveMoves.Run(ctx, store.HasLiveMovesParams{DocType: ref.Type, DocID: ref.ID}); err != nil || live {
		return nil, platform.OrErr(err, ErrAlreadyRecorded)
	}
	// Probe moves carry the shape checks and lot creation: +1 for a positive count, -1 (look up only) for zero.
	probes := make([]Move, len(counts))
	for i, c := range counts {
		m := Move{LineNo: c.LineNo, WarehouseID: warehouse, ItemID: c.ItemID, Lot: c.Lot, Serial: c.Serial, Quantity: decimal.NewFromInt(1)}
		if c.Quantity.IsNegative() || !c.Quantity.Equal(c.Quantity.Truncate(3)) || c.Quantity.GreaterThanOrEqual(maxQuantity) {
			return nil, lineErr(ErrQuantityInvalid, m)
		}
		if c.Serial != nil && !c.Quantity.IsZero() && !c.Quantity.Equal(m.Quantity) {
			return nil, lineErr(ErrSerialQuantity, m)
		}
		if c.Quantity.IsZero() {
			m.Quantity = m.Quantity.Neg()
		}
		probes[i] = m
	}
	results := make([]CountResult, len(counts))
	if len(counts) == 0 {
		return results, nil
	}
	whs, items, err := s.moveMasters(ctx, probes)
	if err != nil {
		return nil, err
	}
	// One count per key, codes compared as the unique indexes do: a second would move on hand again.
	type countKey struct {
		item int64
		code string
	}
	seen := map[countKey]bool{}
	for _, m := range probes {
		if err := check(m, whs, items, h.LegalEntityID); err != nil {
			return nil, err
		}
		k := countKey{m.ItemID, trackCode(m)}
		if seen[k] {
			return nil, lineErr(ErrCountDuplicate, m)
		}
		seen[k] = true
	}
	keyOf := make([]key, len(counts))
	known := make([]bool, len(counts))
	for _, i := range creationOrder(probes) {
		m := probes[i]
		k := key{w: m.WarehouseID, i: m.ItemID}
		k.l, k.s, err = s.resolve(ctx, items[m.ItemID].Tracking, m)
		if errors.Is(err, ErrLotNotFound) || errors.Is(err, ErrSerialNotFound) {
			continue // counted 0 of a lot or serial that never existed: on hand is 0 already
		}
		if err != nil {
			return nil, err
		}
		keyOf[i], known[i] = k, true
	}

	// Same lock order as apply, which re-locks these rows as a no-op.
	var serials []int64
	var ws, is, ls, ss []int64
	for i, k := range keyOf {
		if !known[i] {
			continue
		}
		if k.s != 0 {
			serials = append(serials, k.s)
		}
	}
	slices.Sort(serials)
	if err := store.LockSerials.Run(ctx, slices.Compact(serials)); err != nil {
		return nil, err
	}
	sorted := slices.SortedFunc(func(yield func(key) bool) {
		for i, k := range keyOf {
			if known[i] && !yield(k) {
				return
			}
		}
	}, key.compare)
	for _, k := range sorted {
		ws, is, ls, ss = append(ws, k.w), append(is, k.i), append(ls, k.l), append(ss, k.s)
	}
	if err := store.InsertBalances.Run(ctx, store.InsertBalancesParams{WarehouseIds: ws, ItemIds: is, LotIds: ls, SerialIds: ss}); err != nil {
		return nil, err
	}
	if err := store.LockBalances.Run(ctx, store.LockBalancesParams{WarehouseIds: ws, ItemIds: is, LotIds: ls, SerialIds: ss}); err != nil {
		return nil, err
	}
	rows, err := store.BalanceQuantities.Run(ctx, store.BalanceQuantitiesParams{WarehouseIds: ws, ItemIds: is, LotIds: ls, SerialIds: ss})
	if err != nil {
		return nil, err
	}
	book := map[key]decimal.Decimal{}
	for _, r := range rows {
		if book[key{r.WarehouseID, r.ItemID, r.LotID, r.SerialID}], err = decimal.NewFromString(r.Quantity); err != nil {
			return nil, err
		}
	}
	var lines []line
	var of []int // the count each line comes from
	for i, c := range counts {
		results[i] = CountResult{LineNo: c.LineNo, Book: book[keyOf[i]]}
		if !known[i] {
			results[i].Book = decimal.Zero
			continue
		}
		if d := c.Quantity.Sub(results[i].Book); !d.IsZero() {
			lines = append(lines, line{k: keyOf[i], lineNo: c.LineNo, qty: d})
			of = append(of, i)
		}
	}
	if len(lines) == 0 {
		return results, nil
	}
	vals, err := apply(ctx, ref, h.LegalEntityID, h.Date, items, lines)
	for j, v := range vals {
		results[of[j]].Value = v
	}
	return results, err
}

// line is one resolved move to write; reverses is the id of the move it undoes, or 0. A nil
// value is valued at average.
type line struct {
	k        key
	lineNo   int
	qty      decimal.Decimal
	value    *int64
	reverses int64
}

// apply locks and updates the balances of lines, refuses what would go short, values the
// lines and inserts the moves, returning their values. Record and Void share it so both
// queue and check the same way.
func apply(ctx context.Context, ref record.Ref, legalEntity int64, date pgtype.Date, items map[int64]store.MoveItemsRow, lines []line) ([]int64, error) {
	deltas := map[key]decimal.Decimal{}
	firstLine := map[key]int{}
	var serials []int64
	for _, l := range lines {
		k := l.k
		if k.w == 0 {
			continue // value-only: no balance
		}
		if k.s != 0 {
			serials = append(serials, k.s)
		}
		if _, seen := firstLine[k]; !seen {
			firstLine[k] = l.lineNo
		}
		deltas[k] = deltas[k].Add(l.qty)
	}
	// Lock order: items (read with their tracking), serials, then balances by key, so documents touching the same rows queue.
	slices.Sort(serials)
	serials = slices.Compact(serials)
	if err := store.LockSerials.Run(ctx, serials); err != nil {
		return nil, err
	}
	keys := slices.SortedFunc(maps.Keys(deltas), key.compare)
	var ws, is, ls, ss []int64
	var ds []string
	for _, k := range keys {
		ws, is, ls, ss = append(ws, k.w), append(is, k.i), append(ls, k.l), append(ss, k.s)
		ds = append(ds, deltas[k].String())
	}
	if err := store.InsertBalances.Run(ctx, store.InsertBalancesParams{WarehouseIds: ws, ItemIds: is, LotIds: ls, SerialIds: ss}); err != nil {
		return nil, err
	}
	if err := store.LockBalances.Run(ctx, store.LockBalancesParams{WarehouseIds: ws, ItemIds: is, LotIds: ls, SerialIds: ss}); err != nil {
		return nil, err
	}
	rows, err := store.ApplyDeltas.Run(ctx, store.ApplyDeltasParams{WarehouseIds: ws, ItemIds: is, LotIds: ls, SerialIds: ss, Deltas: ds})
	if err != nil {
		return nil, err
	}
	result := map[key]decimal.Decimal{}
	for _, r := range rows {
		if result[key{r.WarehouseID, r.ItemID, r.LotID, r.SerialID}], err = decimal.NewFromString(r.Quantity); err != nil {
			return nil, err
		}
	}
	for _, k := range keys {
		q, it := result[k], items[k.i]
		// Only taking stock out can fail; a balance already below zero may still be refilled.
		if !deltas[k].IsNegative() || !q.IsNegative() || (it.Tracking == item.TrackQuantity && it.AllowNegative) {
			continue
		}
		return nil, &platform.Error{Status: ErrInsufficient.Status, Code: ErrInsufficient.Code, Params: map[string]any{
			"line_no": firstLine[k], "warehouse_id": k.w, "item_id": k.i,
			"lot_id": nullID(k.l), "serial_id": nullID(k.s), "missing": q.Neg().String()}}
	}
	if len(serials) > 0 {
		over, err := store.SerialsOverOne.Run(ctx, serials)
		if err != nil {
			return nil, err
		}
		for _, k := range keys {
			if k.s != 0 && slices.Contains(over, k.s) {
				return nil, &platform.Error{Status: ErrSerialQuantity.Status, Code: ErrSerialQuantity.Code, Params: map[string]any{"line_no": firstLine[k]}}
			}
		}
	}
	vals, skip, err := valuate(ctx, legalEntity, lines)
	if err != nil {
		return nil, err
	}
	ins := store.InsertMovesParams{DocType: ref.Type, DocID: ref.ID, Date: date}
	for i, l := range lines {
		if skip[i] {
			continue
		}
		ins.LineNos = append(ins.LineNos, int32(l.lineNo))
		ins.WarehouseIds = append(ins.WarehouseIds, l.k.w)
		ins.ItemIds = append(ins.ItemIds, l.k.i)
		ins.LotIds = append(ins.LotIds, l.k.l)
		ins.SerialIds = append(ins.SerialIds, l.k.s)
		ins.Quantities = append(ins.Quantities, l.qty.String())
		ins.Values = append(ins.Values, vals[i])
		ins.ReversesIds = append(ins.ReversesIds, l.reverses)
	}
	if len(ins.LineNos) == 0 {
		return vals, nil
	}
	return vals, store.InsertMoves.Run(ctx, ins)
}

// valuate values lines at moving average per item of the legal entity and moves the running
// totals, under the cost rows' locks. A line with a value keeps it; a value-only line is
// skipped (value 0) when the legal entity has none of the item on hand. When an item's
// quantity comes to zero, its last outgoing line valued here takes the rest of the value.
func valuate(ctx context.Context, legalEntity int64, lines []line) ([]int64, []bool, error) {
	var ids []int64
	for _, l := range lines {
		ids = append(ids, l.k.i)
	}
	slices.Sort(ids)
	ids = slices.Compact(ids)
	if err := store.InsertCosts.Run(ctx, store.InsertCostsParams{LegalEntityID: legalEntity, ItemIds: ids}); err != nil {
		return nil, nil, err
	}
	rows, err := store.LockCosts.Run(ctx, store.LockCostsParams{LegalEntityID: legalEntity, ItemIds: ids})
	if err != nil {
		return nil, nil, err
	}
	vals, skip := make([]int64, len(lines)), make([]bool, len(lines))
	up := store.UpdateCostsParams{LegalEntityID: legalEntity}
	for _, r := range rows {
		q, err := decimal.NewFromString(r.Quantity)
		if err != nil {
			return nil, nil, err
		}
		v := r.Value
		avg := decimal.Zero
		switch {
		case q.IsPositive():
			avg = decimal.NewFromInt(v).Div(q)
		case r.UnitCost != "":
			if avg, err = decimal.NewFromString(r.UnitCost); err != nil {
				return nil, nil, err
			}
		}
		had, lastOut := q.IsPositive(), -1
		for i, l := range lines {
			if l.k.i != r.ItemID {
				continue
			}
			switch {
			case l.k.w == 0 && l.reverses == 0 && !had:
				skip[i] = true
				continue
			case l.value != nil:
				vals[i] = *l.value
			default:
				vals[i] = platform.Round(l.qty.Mul(avg))
				if l.qty.IsNegative() {
					lastOut = i
				}
			}
			q, v = q.Add(l.qty), v+vals[i]
		}
		if q.IsZero() && lastOut >= 0 {
			vals[lastOut] -= v
			v = 0
		}
		unit := ""
		if q.IsPositive() {
			unit = decimal.NewFromInt(v).Div(q).String()
		}
		up.ItemIds = append(up.ItemIds, r.ItemID)
		up.Quantities = append(up.Quantities, q.String())
		up.Values = append(up.Values, v)
		up.UnitCosts = append(up.UnitCosts, unit)
	}
	return vals, skip, store.UpdateCosts.Run(ctx, up)
}

// Void cancels the moves of a document by inserting a reversal for each one not yet
// reversed, in the caller's transaction. It goes through the same balance checks as Record,
// so goods already issued elsewhere refuse it. A document without moves is a no-op. Callers
// hold the document's row lock (they run inside record.Transition), so it never voids twice.
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
		back := -r.Value
		lines[i] = line{k: key{r.WarehouseID, r.ItemID, r.LotID, r.SerialID}, lineNo: int(r.LineNo), qty: q.Neg(), value: &back, reverses: r.ID}
	}
	irows, err := store.MoveItems.Run(ctx, ids)
	if err != nil {
		return err
	}
	items := map[int64]store.MoveItemsRow{}
	for _, i := range irows {
		items[i.ID] = i
	}
	_, err = apply(ctx, ref, h.LegalEntityID, h.Date, items, lines)
	return err
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
	if valueOnly(m) {
		if _, found := items[m.ItemID]; !found {
			return platform.ErrNotFound
		}
		if !m.Quantity.IsZero() || m.Value == nil || *m.Value == 0 || m.Lot != nil || m.Serial != nil {
			return lineErr(ErrQuantityInvalid, m)
		}
		return nil
	}
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
	var lot *string
	if m.Lot != nil {
		lot = &m.Lot.Code
	}
	return CheckShape(it.Tracking, lot, m.Serial, m.Quantity, m.LineNo)
}

// CheckShape refuses a line whose lot, serial and quantity do not fit its item's tracking;
// documents call it on saving, before posting checks the lots and serials themselves.
func CheckShape(tracking string, lot, serial *string, qty decimal.Decimal, lineNo int) error {
	m := Move{LineNo: lineNo}
	switch tracking {
	case item.TrackNone:
		return lineErr(ErrUntracked, m)
	case item.TrackQuantity:
		if lot != nil || serial != nil {
			return lineErr(ErrLotNotTracked, m)
		}
	case item.TrackLot:
		if serial != nil {
			return lineErr(ErrLotNotTracked, m)
		}
		if lot == nil || strings.TrimSpace(*lot) == "" {
			return lineErr(ErrLotRequired, m)
		}
	default: // serial
		if lot != nil {
			return lineErr(ErrLotNotTracked, m)
		}
		if serial == nil || strings.TrimSpace(*serial) == "" {
			return lineErr(ErrSerialRequired, m)
		}
		if !qty.Abs().Equal(decimal.NewFromInt(1)) {
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
// missing for an incoming move; an outgoing move naming an unknown one is refused.
func (s *Service) resolve(ctx context.Context, tracking string, m Move) (lot, serial int64, err error) {
	switch tracking {
	case item.TrackLot:
		code := strings.TrimSpace(m.Lot.Code)
		if m.Quantity.IsNegative() {
			l, err := store.LotByCode.Run(ctx, store.LotByCodeParams{ItemID: m.ItemID, Code: code})
			if errors.Is(err, pgx.ErrNoRows) {
				return 0, 0, lineErr(ErrLotNotFound, m)
			}
			if err != nil {
				return 0, 0, err
			}
			return l.ID, 0, expiryMatches(m, l.Expiry)
		}
		if err := store.InsertLot.Run(ctx, store.InsertLotParams{ItemID: m.ItemID, Code: code, Expiry: platform.NullDate(m.Lot.Expiry)}); err != nil {
			return 0, 0, err
		}
		l, err := store.LotByCode.Run(ctx, store.LotByCodeParams{ItemID: m.ItemID, Code: code})
		if err != nil {
			return 0, 0, err
		}
		return l.ID, 0, expiryMatches(m, l.Expiry)
	case item.TrackSerial:
		code := strings.TrimSpace(*m.Serial)
		if m.Quantity.IsNegative() {
			id, err := store.SerialByCode.Run(ctx, store.SerialByCodeParams{ItemID: m.ItemID, Code: code})
			if errors.Is(err, pgx.ErrNoRows) {
				return 0, 0, lineErr(ErrSerialNotFound, m)
			}
			return 0, id, err
		}
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

// SerialsOnHand lists the codes of an item's serials on hand in a warehouse, sorted. It checks no permission.
func (s *Service) SerialsOnHand(ctx context.Context, warehouse, itemID int64) ([]string, error) {
	return store.SerialsOnHand.Run(ctx, store.SerialsOnHandParams{WarehouseID: warehouse, ItemID: itemID})
}

// expiryMatches refuses a move naming an expiry other than the lot's.
func expiryMatches(m Move, have pgtype.Date) error {
	if m.Lot.Expiry == nil {
		return nil
	}
	if h := platform.DatePtr(have); h == nil || *h != *m.Lot.Expiry {
		return lineErr(ErrLotExpiryMismatch, m)
	}
	return nil
}
