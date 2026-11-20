package stock_test

import (
	"context"
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/item"
	"github.com/taoworklabs/mmerp/internal/shared/stock"
)

func val(v int64) *int64 { return &v }

// valued records moves for a new document and returns the values written.
func (l *ledger) valued(t *testing.T, moves ...stock.Move) (record.Ref, []int64) {
	t.Helper()
	ref := l.doc(t, l.unit)
	var got []int64
	err := platform.InTx(l.core.Ctx, func(ctx context.Context) (err error) {
		got, err = l.stock.Record(ctx, ref, moves)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return ref, got
}

// cost reads the company's running quantity and value of an item.
func (l *ledger) cost(t *testing.T, it int64) (string, int64) {
	t.Helper()
	var q string
	var v int64
	if err := l.core.Pool.QueryRow(l.core.Ctx, `SELECT quantity::text, value FROM stock.costs WHERE legal_entity_id = $1 AND item_id = $2`, l.unit, it).Scan(&q, &v); err != nil {
		t.Fatal(err)
	}
	return q, v
}

func wantValues(t *testing.T, got []int64, want ...int64) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("values %v, want %v", got, want)
	}
}

func wantCost(t *testing.T, l *ledger, it int64, q string, v int64) {
	t.Helper()
	if gq, gv := l.cost(t, it); gq != q || gv != v {
		t.Fatalf("cost %s/%d, want %s/%d", gq, gv, q, v)
	}
}

func TestValueMovingAverage(t *testing.T) {
	l := newLedger(t)
	it := l.item(t, item.TrackQuantity, false)
	_, v := l.valued(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(10), Value: val(1000)})
	wantValues(t, v, 1000)
	l.valued(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(10), Value: val(3000)})
	// Average 200: an issue of 5 takes 1000.
	_, v = l.valued(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(-5)})
	wantValues(t, v, -1000)
	wantCost(t, l, it, "15.000", 3000)
}

func TestValueEmptyingTakesTheRest(t *testing.T) {
	l := newLedger(t)
	it := l.item(t, item.TrackQuantity, false)
	l.valued(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(3), Value: val(100)})
	_, v := l.valued(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(-1)})
	wantValues(t, v, -33)
	_, v = l.valued(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(-2)})
	wantValues(t, v, -67)
	wantCost(t, l, it, "0.000", 0)
}

func TestValueNegativeStock(t *testing.T) {
	l := newLedger(t)
	it := l.item(t, item.TrackQuantity, true)
	// Never had a cost: zero, absorbed by the next receipt.
	_, v := l.valued(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(-2)})
	wantValues(t, v, 0)
	l.valued(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(10), Value: val(1000)})
	wantCost(t, l, it, "8.000", 1000) // unit cost 125
	_, v = l.valued(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(-10)})
	wantValues(t, v, -1250)
	// Below zero: the last unit cost.
	_, v = l.valued(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(-1)})
	wantValues(t, v, -125)
	wantCost(t, l, it, "-3.000", -375)
}

func TestValueVoidAtOriginalValue(t *testing.T) {
	l := newLedger(t)
	it := l.item(t, item.TrackQuantity, false)
	l.valued(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(10), Value: val(1000)})
	out, _ := l.valued(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(-4)})
	l.valued(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(10), Value: val(2000)})
	if err := platform.InTx(l.core.Ctx, func(ctx context.Context) error { return l.stock.Void(ctx, out) }); err != nil {
		t.Fatal(err)
	}
	var back int64
	if err := l.core.Pool.QueryRow(l.core.Ctx, `SELECT value FROM stock.moves WHERE doc_type = $1 AND doc_id = $2 AND reverses IS NOT NULL`, out.Type, out.ID).Scan(&back); err != nil || back != 400 {
		t.Fatalf("reversal %d %v", back, err)
	}
	wantCost(t, l, it, "20.000", 3000)
}

func TestValueOnlyMove(t *testing.T) {
	l := newLedger(t)
	it := l.item(t, item.TrackQuantity, false)
	none := l.item(t, item.TrackQuantity, false)
	l.valued(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(10), Value: val(1000)})
	// Applied while on hand, skipped (value 0, no move) when there is none.
	ref, v := l.valued(t, stock.Move{LineNo: 1, ItemID: it, Value: val(500)}, stock.Move{LineNo: 2, ItemID: none, Value: val(70)})
	wantValues(t, v, 500, 0)
	wantCost(t, l, it, "10.000", 1500)
	var n int
	if err := l.core.Pool.QueryRow(l.core.Ctx, `SELECT count(*) FROM stock.moves WHERE doc_type = $1 AND doc_id = $2`, ref.Type, ref.ID).Scan(&n); err != nil || n != 1 {
		t.Fatalf("moves %d %v", n, err)
	}
	if err := platform.InTx(l.core.Ctx, func(ctx context.Context) error { return l.stock.Void(ctx, ref) }); err != nil {
		t.Fatal(err)
	}
	wantCost(t, l, it, "10.000", 1000)
}

func TestValueTransferCancels(t *testing.T) {
	l := newLedger(t)
	it := l.item(t, item.TrackQuantity, false)
	other := l.warehouse(t, l.unit)
	l.valued(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(4), Value: val(1000)})
	_, v := l.valued(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(-4)}, stock.Move{LineNo: 1, WarehouseID: other, ItemID: it, Quantity: qty(4)})
	wantValues(t, v, -1000, 1000)
	wantCost(t, l, it, "4.000", 1000)
}

func TestValueCounts(t *testing.T) {
	l := newLedger(t)
	it := l.item(t, item.TrackQuantity, false)
	l.valued(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(10), Value: val(1000)})
	ref := l.doc(t, l.unit)
	var res []stock.CountResult
	err := platform.InTx(l.core.Ctx, func(ctx context.Context) (err error) {
		res, err = l.stock.RecordCounts(ctx, ref, l.wh, []stock.Count{{LineNo: 1, ItemID: it, Quantity: qty(7)}})
		return err
	})
	if err != nil || len(res) != 1 || res[0].Value != -300 {
		t.Fatalf("%+v %v", res, err)
	}
}
