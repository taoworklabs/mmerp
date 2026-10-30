package stock_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/item"
	"github.com/taoworklabs/mmerp/internal/shared/stock"
)

// ledger is a fixture with a default warehouse and someone allowed to create items and documents.
type ledger struct {
	*fixture
	admin context.Context
	items *item.Service
	wh    int64
}

func newLedger(t *testing.T) *ledger {
	t.Helper()
	l := &ledger{fixture: newFixture(t)}
	l.items = item.NewService(item.Deps{IAM: l.core.IAM, Audit: l.core.Audit, Numbering: l.core.Numbering})
	l.admin = l.user(t, "ledger", "admin")
	id, _ := platform.ActorFrom(l.admin)
	if _, err := l.core.IAM.GrantRole(l.core.Admin, id, "item", "admin", nil); err != nil {
		t.Fatal(err)
	}
	err := platform.InTx(l.core.Ctx, func(ctx context.Context) (err error) {
		l.wh, err = l.stock.DefaultWarehouse(ctx, l.unit)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func (l *ledger) item(t *testing.T, tracking string, allowNegative bool) int64 {
	t.Helper()
	id, err := l.items.CreateItem(l.admin, item.NewItem{ItemFields: item.ItemFields{Name: "I", Unit: "cái", VatRate: "8", Active: true,
		Tracking: &tracking, AllowNegative: &allowNegative}})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// doc creates a draft test document of the company dated 2026-11-01.
func (l *ledger) doc(t *testing.T, unit int64) record.Ref {
	t.Helper()
	d, err := l.core.Record.Create(l.admin, testDoc, record.Header{Date: "2026-11-01", OrgUnitID: unit})
	if err != nil {
		t.Fatal(err)
	}
	return d.Ref
}

// move records moves for a new document of the company.
func (l *ledger) move(t *testing.T, moves ...stock.Move) error {
	t.Helper()
	return l.recordFor(l.doc(t, l.unit), moves...)
}

func (l *ledger) recordFor(ref record.Ref, moves ...stock.Move) error {
	return platform.InTx(l.core.Ctx, func(ctx context.Context) error { return l.stock.Record(ctx, ref, moves) })
}

// onHand sums an item's balances in a warehouse.
func (l *ledger) onHand(t *testing.T, wh, item int64) string {
	t.Helper()
	var q decimal.Decimal
	if err := l.core.Pool.QueryRow(l.core.Ctx, `SELECT coalesce(sum(quantity), 0)::text FROM stock.balances WHERE warehouse_id = $1 AND item_id = $2`, wh, item).Scan(&q); err != nil {
		t.Fatal(err)
	}
	return q.String()
}

func qty(n int64) decimal.Decimal { return decimal.NewFromInt(n) }

func lot(code string, expiry *string) *stock.LotRef { return &stock.LotRef{Code: code, Expiry: expiry} }

func wantCode(t *testing.T, err, want error) *platform.Error {
	t.Helper()
	e, ok := errors.AsType[*platform.Error](err)
	if !ok || !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
	return e
}

func TestRecordInAndOut(t *testing.T) {
	l := newLedger(t)
	it := l.item(t, item.TrackLot, false)
	if err := l.move(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Lot: lot("L1", nil), Quantity: qty(10)}); err != nil {
		t.Fatal(err)
	}
	if err := l.move(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Lot: lot("l1", nil), Quantity: qty(-4)}); err != nil {
		t.Fatal(err)
	}
	if got := l.onHand(t, l.wh, it); got != "6" {
		t.Fatalf("on hand %s", got)
	}
	var lots, moves int
	if err := l.core.Pool.QueryRow(l.core.Ctx, `SELECT (SELECT count(*) FROM stock.lots), (SELECT count(*) FROM stock.moves WHERE date = '2026-11-01')`).Scan(&lots, &moves); err != nil || lots != 1 || moves != 2 {
		t.Fatalf("lots %d moves %d %v", lots, moves, err)
	}
}

func TestRecordRefusesShortage(t *testing.T) {
	l := newLedger(t)
	it := l.item(t, item.TrackQuantity, false)
	if err := l.move(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(6)}); err != nil {
		t.Fatal(err)
	}
	e := wantCode(t, l.move(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(-7)}), stock.ErrInsufficient)
	if e.Params["missing"] != "1" || e.Params["item_id"] != it || e.Status != 409 {
		t.Fatalf("params %+v", e)
	}
	if got := l.onHand(t, l.wh, it); got != "6" {
		t.Fatalf("on hand %s", got)
	}
	if v, ok := e.Params["lot_id"]; !ok || v != nil || e.Params["serial_id"] != nil {
		t.Fatalf("absent lot and serial %+v", e.Params)
	}
	// A lot item names its lot.
	lt := l.item(t, item.TrackLot, false)
	if err := l.move(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: lt, Lot: lot("L1", nil), Quantity: qty(2)}); err != nil {
		t.Fatal(err)
	}
	var lotID int64
	if err := l.core.Pool.QueryRow(l.core.Ctx, `SELECT id FROM stock.lots WHERE item_id = $1`, lt).Scan(&lotID); err != nil {
		t.Fatal(err)
	}
	e = wantCode(t, l.move(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: lt, Lot: lot("L1", nil), Quantity: qty(-3)}), stock.ErrInsufficient)
	if e.Params["lot_id"] != lotID || e.Params["missing"] != "1" || e.Params["warehouse_id"] != l.wh {
		t.Fatalf("lot params %+v", e.Params)
	}
	// Moves of one key net out within a document: +5 and -3 from nothing is +2.
	other := l.item(t, item.TrackQuantity, false)
	if err := l.move(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: other, Quantity: qty(-3)},
		stock.Move{LineNo: 2, WarehouseID: l.wh, ItemID: other, Quantity: qty(5)}); err != nil {
		t.Fatal(err)
	}
	if got := l.onHand(t, l.wh, other); got != "2" {
		t.Fatalf("netted %s", got)
	}
}

func TestNegativeAllowedForQuantityItem(t *testing.T) {
	l := newLedger(t)
	it := l.item(t, item.TrackQuantity, true)
	if err := l.move(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(-3)}); err != nil {
		t.Fatal(err)
	}
	if got := l.onHand(t, l.wh, it); got != "-3" {
		t.Fatalf("on hand %s", got)
	}
}

func TestSerialOnHandOnce(t *testing.T) {
	l := newLedger(t)
	it := l.item(t, item.TrackSerial, false)
	s1 := "S1"
	in := stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Serial: &s1, Quantity: qty(1)}
	if err := l.move(t, in); err != nil {
		t.Fatal(err)
	}
	wantCode(t, l.move(t, in), stock.ErrSerialQuantity)
	// Nor in another warehouse.
	w2, err := l.stock.CreateWarehouse(l.admin, stock.NewWarehouse{WarehouseFields: stock.WarehouseFields{Name: "W2", LegalEntityID: l.unit, Active: true}})
	if err != nil {
		t.Fatal(err)
	}
	in.WarehouseID = w2
	wantCode(t, l.move(t, in), stock.ErrSerialQuantity)
	in.Quantity = qty(2)
	wantCode(t, l.move(t, in), stock.ErrSerialQuantity)
	in.WarehouseID, in.Quantity = l.wh, qty(-1)
	if err := l.move(t, in); err != nil {
		t.Fatal(err)
	}
	wantCode(t, l.move(t, in), stock.ErrInsufficient)
}

func TestTrackingChecks(t *testing.T) {
	l := newLedger(t)
	none, q, lt, sr := l.item(t, item.TrackNone, false), l.item(t, item.TrackQuantity, false), l.item(t, item.TrackLot, false), l.item(t, item.TrackSerial, false)
	s := "S"
	for _, c := range []struct {
		m    stock.Move
		want error
	}{
		{stock.Move{ItemID: none, Quantity: qty(1)}, stock.ErrUntracked},
		{stock.Move{ItemID: q, Lot: lot("L", nil), Quantity: qty(1)}, stock.ErrLotNotTracked},
		{stock.Move{ItemID: q, Serial: &s, Quantity: qty(1)}, stock.ErrLotNotTracked},
		{stock.Move{ItemID: lt, Quantity: qty(1)}, stock.ErrLotRequired},
		{stock.Move{ItemID: sr, Quantity: qty(1)}, stock.ErrSerialRequired},
		{stock.Move{ItemID: q, Quantity: qty(0)}, stock.ErrQuantityInvalid},
		{stock.Move{ItemID: q, Quantity: decimal.RequireFromString("1.0005")}, stock.ErrQuantityInvalid},
	} {
		c.m.LineNo, c.m.WarehouseID = 3, l.wh
		if e := wantCode(t, l.move(t, c.m), c.want); e.Params["line_no"] != 3 {
			t.Fatalf("%v params %+v", c.want, e.Params)
		}
	}
	// A lot keeps its expiry: naming another one is refused, naming none reuses it.
	exp, other := "2027-01-01", "2027-02-01"
	if err := l.move(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: lt, Lot: lot("L", &exp), Quantity: qty(1)}); err != nil {
		t.Fatal(err)
	}
	wantCode(t, l.move(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: lt, Lot: lot("L", &other), Quantity: qty(1)}), stock.ErrLotExpiryMismatch)
	if err := l.move(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: lt, Lot: lot("L", nil), Quantity: qty(1)}); err != nil {
		t.Fatal(err)
	}
}

func TestWarehouseOfOtherEntity(t *testing.T) {
	l := newLedger(t)
	it := l.item(t, item.TrackQuantity, false)
	other, err := l.core.IAM.CreateOrgUnit(l.core.Admin, iam.OrgUnitInput{Kind: "company", Name: "D"})
	if err != nil {
		t.Fatal(err)
	}
	wantCode(t, l.recordFor(l.doc(t, other), stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(1)}), stock.ErrWarehouseEntity)
	if err := l.stock.UpdateWarehouse(l.admin, l.wh, stock.WarehouseFields{Name: "Kho", Active: false}); err != nil {
		t.Fatal(err)
	}
	wantCode(t, l.move(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(1)}), stock.ErrWarehouseInactive)
}

func TestRecordOncePerDocument(t *testing.T) {
	l := newLedger(t)
	it := l.item(t, item.TrackQuantity, false)
	ref := l.doc(t, l.unit)
	m := stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(1)}
	if err := l.recordFor(ref, m); err != nil {
		t.Fatal(err)
	}
	wantCode(t, l.recordFor(ref, m), stock.ErrAlreadyRecorded)
}

func TestSuggestLotsFEFO(t *testing.T) {
	l := newLedger(t)
	it := l.item(t, item.TrackLot, false)
	a, b, d := "2026-12-01", "2026-11-15", "2026-10-31"
	if err := l.move(t,
		stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Lot: lot("A", &a), Quantity: qty(2)},
		stock.Move{LineNo: 2, WarehouseID: l.wh, ItemID: it, Lot: lot("B", &b), Quantity: qty(2)},
		stock.Move{LineNo: 3, WarehouseID: l.wh, ItemID: it, Lot: lot("C", nil), Quantity: qty(5)},
		stock.Move{LineNo: 4, WarehouseID: l.wh, ItemID: it, Lot: lot("D", &d), Quantity: qty(9)},
	); err != nil {
		t.Fatal(err)
	}
	picks, err := l.stock.SuggestLots(l.core.Ctx, l.wh, it, qty(5), "2026-11-01")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, p := range picks {
		got = append(got, p.Code+":"+p.Quantity.String())
	}
	if len(got) != 3 || got[0] != "B:2" || got[1] != "A:2" || got[2] != "C:1" || *picks[0].Expiry != b || picks[2].Expiry != nil {
		t.Fatalf("picks %v", got)
	}
	// Short: everything usable, 9 of 20.
	picks, err = l.stock.SuggestLots(l.core.Ctx, l.wh, it, qty(20), "2026-11-01")
	if err != nil || len(picks) != 3 || picks[2].Quantity.String() != "5" {
		t.Fatalf("short %+v %v", picks, err)
	}
}

// race runs fn for each document at once and returns the errors in order.
func race(docs []record.Ref, fn func(i int, ref record.Ref) error) []error {
	errs := make([]error, len(docs))
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i, ref := range docs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs[i] = fn(i, ref)
		}()
	}
	close(start)
	wg.Wait()
	return errs
}

func TestConcurrentIssueOfLastUnit(t *testing.T) {
	l := newLedger(t)
	it := l.item(t, item.TrackLot, false)
	if err := l.move(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Lot: lot("L1", nil), Quantity: qty(1)}); err != nil {
		t.Fatal(err)
	}
	docs := []record.Ref{l.doc(t, l.unit), l.doc(t, l.unit)}
	errs := race(docs, func(_ int, ref record.Ref) error {
		return l.recordFor(ref, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Lot: lot("L1", nil), Quantity: qty(-1)})
	})
	ok, short := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, stock.ErrInsufficient):
			short++
		default:
			t.Fatal(err)
		}
	}
	if ok != 1 || short != 1 {
		t.Fatalf("ok %d short %d", ok, short)
	}
	if got := l.onHand(t, l.wh, it); got != "0" {
		t.Fatalf("on hand %s", got)
	}
}

func TestOppositeOrderNoDeadlock(t *testing.T) {
	l := newLedger(t)
	x, y := l.item(t, item.TrackQuantity, false), l.item(t, item.TrackQuantity, false)
	if err := l.move(t,
		stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: x, Quantity: qty(5)},
		stock.Move{LineNo: 2, WarehouseID: l.wh, ItemID: y, Quantity: qty(5)}); err != nil {
		t.Fatal(err)
	}
	docs := []record.Ref{l.doc(t, l.unit), l.doc(t, l.unit)}
	done := make(chan []error, 1)
	go func() {
		done <- race(docs, func(i int, ref record.Ref) error {
			a := stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: x, Quantity: qty(-1)}
			b := stock.Move{LineNo: 2, WarehouseID: l.wh, ItemID: y, Quantity: qty(-1)}
			if i == 1 {
				return l.recordFor(ref, b, a)
			}
			return l.recordFor(ref, a, b)
		})
	}()
	select {
	case errs := <-done:
		for _, err := range errs {
			if err != nil {
				t.Fatal(err)
			}
		}
	case <-time.After(5 * time.Second):
		t.Fatal("deadlock")
	}
	if a, b := l.onHand(t, l.wh, x), l.onHand(t, l.wh, y); a != "3" || b != "3" {
		t.Fatalf("on hand %s %s", a, b)
	}
}
