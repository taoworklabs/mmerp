package stock_test

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/item"
	"github.com/taoworklabs/mmerp/internal/shared/stock"
)

func (l *ledger) count(ref record.Ref, wh int64, counts ...stock.Count) (out []stock.CountResult, err error) {
	return out, platform.InTx(l.core.Ctx, func(ctx context.Context) (err error) {
		out, err = l.stock.RecordCounts(ctx, ref, wh, counts)
		return err
	})
}

func TestRecordCountsMovesToCount(t *testing.T) {
	l := newLedger(t)
	a, b, c := l.item(t, item.TrackQuantity, false), l.item(t, item.TrackQuantity, false), l.item(t, item.TrackQuantity, false)
	if err := l.move(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: a, Quantity: qty(10)},
		stock.Move{LineNo: 2, WarehouseID: l.wh, ItemID: b, Quantity: qty(3)}, stock.Move{LineNo: 3, WarehouseID: l.wh, ItemID: c, Quantity: qty(4)}); err != nil {
		t.Fatal(err)
	}
	res, err := l.count(l.doc(t, l.unit), l.wh, stock.Count{LineNo: 1, ItemID: a, Quantity: qty(7)}, stock.Count{LineNo: 2, ItemID: b, Quantity: qty(5)})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 || res[0].LineNo != 1 || res[0].Book.String() != "10" || res[1].Book.String() != "3" {
		t.Fatalf("results %+v", res)
	}
	for it, want := range map[int64]string{a: "7", b: "5", c: "4"} {
		if got := l.onHand(t, l.wh, it); got != want {
			t.Fatalf("item %d on hand %s, want %s", it, got, want)
		}
	}
}

func TestRecordCountsNewLot(t *testing.T) {
	l := newLedger(t)
	it := l.item(t, item.TrackLot, false)
	res, err := l.count(l.doc(t, l.unit), l.wh, stock.Count{LineNo: 1, ItemID: it, Lot: lot("L-NEW", nil), Quantity: qty(2)})
	if err != nil {
		t.Fatal(err)
	}
	if !res[0].Book.IsZero() || l.onHand(t, l.wh, it) != "2" {
		t.Fatalf("results %+v on hand %s", res, l.onHand(t, l.wh, it))
	}
}

func TestRecordCountsUnknownZero(t *testing.T) {
	l := newLedger(t)
	it, sr := l.item(t, item.TrackLot, false), l.item(t, item.TrackSerial, false)
	s := "S-NONE"
	before := l.moveRows(t)
	res, err := l.count(l.doc(t, l.unit), l.wh, stock.Count{LineNo: 1, ItemID: it, Lot: lot("L-NONE", nil), Quantity: qty(0)},
		stock.Count{LineNo: 2, ItemID: sr, Serial: &s, Quantity: qty(0)})
	if err != nil {
		t.Fatal(err)
	}
	var made int
	if err := l.core.Pool.QueryRow(l.core.Ctx, `SELECT (SELECT count(*) FROM stock.lots) + (SELECT count(*) FROM stock.serials)`).Scan(&made); err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 || !res[0].Book.IsZero() || !res[1].Book.IsZero() || l.moveRows(t) != before || made != 0 {
		t.Fatalf("results %+v moves %d codes %d", res, l.moveRows(t), made)
	}
}

func TestRecordCountsSerialElsewhere(t *testing.T) {
	l := newLedger(t)
	it := l.item(t, item.TrackSerial, false)
	s := "S1"
	if err := l.move(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Serial: &s, Quantity: qty(1)}); err != nil {
		t.Fatal(err)
	}
	w2, err := l.stock.CreateWarehouse(l.admin, stock.NewWarehouse{WarehouseFields: stock.WarehouseFields{Name: "W2", LegalEntityID: l.unit, Active: true}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = l.count(l.doc(t, l.unit), w2, stock.Count{LineNo: 1, ItemID: it, Serial: &s, Quantity: qty(1)})
	_ = wantCode(t, err, stock.ErrSerialQuantity)
	// A serial counts 0 or 1.
	_, err = l.count(l.doc(t, l.unit), l.wh, stock.Count{LineNo: 1, ItemID: it, Serial: &s, Quantity: qty(2)})
	_ = wantCode(t, err, stock.ErrSerialQuantity)
}

func TestRecordCountsVoid(t *testing.T) {
	l := newLedger(t)
	it := l.item(t, item.TrackQuantity, false)
	if err := l.move(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(10)}); err != nil {
		t.Fatal(err)
	}
	ref := l.doc(t, l.unit)
	if _, err := l.count(ref, l.wh, stock.Count{LineNo: 1, ItemID: it, Quantity: qty(6)}); err != nil {
		t.Fatal(err)
	}
	if err := l.void(ref); err != nil {
		t.Fatal(err)
	}
	if got := l.onHand(t, l.wh, it); got != "10" {
		t.Fatalf("on hand %s", got)
	}
}

func TestRecordCountsRefusesTwice(t *testing.T) {
	l := newLedger(t)
	it := l.item(t, item.TrackQuantity, false)
	ref := l.doc(t, l.unit)
	if _, err := l.count(ref, l.wh, stock.Count{LineNo: 1, ItemID: it, Quantity: qty(3)}); err != nil {
		t.Fatal(err)
	}
	_, err := l.count(ref, l.wh, stock.Count{LineNo: 1, ItemID: it, Quantity: qty(3)})
	_ = wantCode(t, err, stock.ErrAlreadyRecorded)
}

func TestRecordCountsSerialFraction(t *testing.T) {
	l := newLedger(t)
	it := l.item(t, item.TrackSerial, false)
	s := "S1"
	_, err := l.count(l.doc(t, l.unit), l.wh, stock.Count{LineNo: 1, ItemID: it, Serial: &s, Quantity: decimal.RequireFromString("0.5")})
	_ = wantCode(t, err, stock.ErrSerialQuantity)
}

// Two counts of one key would each move on hand; the second is refused.
func TestRecordCountsDuplicateKey(t *testing.T) {
	l := newLedger(t)
	q, lt := l.item(t, item.TrackQuantity, false), l.item(t, item.TrackLot, false)
	_, err := l.count(l.doc(t, l.unit), l.wh, stock.Count{LineNo: 1, ItemID: q, Quantity: qty(3)}, stock.Count{LineNo: 2, ItemID: q, Quantity: qty(4)})
	if e := wantCode(t, err, stock.ErrCountDuplicate); e.Params["line_no"] != 2 {
		t.Fatalf("params %+v", e.Params)
	}
	// Unknown lots too, compared as the unique index does.
	_, err = l.count(l.doc(t, l.unit), l.wh, stock.Count{LineNo: 1, ItemID: lt, Lot: lot(" n1", nil), Quantity: qty(0)},
		stock.Count{LineNo: 2, ItemID: lt, Lot: lot("N1", nil), Quantity: qty(0)})
	_ = wantCode(t, err, stock.ErrCountDuplicate)
}
