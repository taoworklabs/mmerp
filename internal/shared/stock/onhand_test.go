package stock_test

import (
	"errors"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/item"
	"github.com/taoworklabs/mmerp/internal/shared/stock"
)

// docOn creates a draft test document dated date.
func (l *ledger) docOn(t *testing.T, date string) record.Ref {
	t.Helper()
	d, err := l.core.Record.Create(l.admin, testDoc, record.Header{Date: date, OrgUnitID: l.unit})
	if err != nil {
		t.Fatal(err)
	}
	return d.Ref
}

func TestOnHandHidesZero(t *testing.T) {
	l := newLedger(t)
	kept, gone := l.item(t, item.TrackLot, false), l.item(t, item.TrackQuantity, false)
	exp := "2026-12-31"
	if err := l.move(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: kept, Lot: lot("L1", &exp), Quantity: qty(10)},
		stock.Move{LineNo: 2, WarehouseID: l.wh, ItemID: gone, Quantity: qty(3)}); err != nil {
		t.Fatal(err)
	}
	if err := l.move(t, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: gone, Quantity: qty(-3)}); err != nil {
		t.Fatal(err)
	}
	page := platform.Paging{Page: 1, PageSize: 50}
	got, err := l.stock.OnHand(l.admin, stock.OnHandFilter{Paging: page})
	if err != nil || got.Total != 1 || got.Items[0].ItemID != kept || got.Items[0].Quantity != "10.000" ||
		got.Items[0].LotCode == nil || *got.Items[0].LotCode != "L1" || *got.Items[0].Expiry != exp {
		t.Fatalf("%+v %v", got, err)
	}
	before := "2026-12-01"
	if got, err = l.stock.OnHand(l.admin, stock.OnHandFilter{ExpiringBefore: before, Paging: page}); err != nil || got.Total != 0 {
		t.Fatalf("%+v %v", got, err)
	}
	if got, err = l.stock.OnHand(l.admin, stock.OnHandFilter{Q: "l1", Paging: page}); err != nil || got.Total != 1 {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestStockCardRunningBalance(t *testing.T) {
	l := newLedger(t)
	it := l.item(t, item.TrackQuantity, false)
	if err := l.recordFor(l.docOn(t, "2026-11-01"), stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(10)}); err != nil {
		t.Fatal(err)
	}
	out := l.docOn(t, "2026-11-05")
	if err := l.recordFor(out, stock.Move{LineNo: 1, WarehouseID: l.wh, ItemID: it, Quantity: qty(-4)}); err != nil {
		t.Fatal(err)
	}
	if err := l.void(out); err != nil {
		t.Fatal(err)
	}
	c, err := l.stock.StockCard(l.admin, l.wh, it, "2026-11-02", "2026-11-30")
	if err != nil || len(c.Rows) != 2 || c.Opening != "10.000" || c.Closing != "10.000" ||
		c.Rows[0].Quantity != "-4.000" || c.Rows[0].Balance != "6.000" || c.Rows[1].Quantity != "4.000" || c.Rows[1].Balance != "10.000" ||
		c.Rows[0].Number == "" || c.Rows[0].DocID != out.ID {
		t.Fatalf("%+v %v", c, err)
	}
}

func TestStockCardBounds(t *testing.T) {
	l := newLedger(t)
	it := l.item(t, item.TrackQuantity, false)
	c, err := l.stock.StockCard(l.admin, l.wh, it, "2026-01-01", "2026-12-31")
	if err != nil || c.Opening != "0.000" || c.Closing != "0.000" || len(c.Rows) != 0 {
		t.Fatalf("%+v %v", c, err)
	}
	if _, err := l.stock.StockCard(l.admin, l.wh, it, "2026-11-02", "2026-11-01"); !errors.Is(err, stock.ErrStockCardRange) {
		t.Fatalf("from after to: %v", err)
	}
	// 366 days counting both ends is the longest card.
	if _, err := l.stock.StockCard(l.admin, l.wh, it, "2027-01-01", "2028-01-01"); err != nil {
		t.Fatalf("366 days: %v", err)
	}
	if _, err := l.stock.StockCard(l.admin, l.wh, it, "2026-01-01", "2027-01-02"); !errors.Is(err, stock.ErrStockCardRangeLen) {
		t.Fatalf("too long: %v", err)
	}
	if _, err := l.stock.StockCard(l.user(t, "nobody"), l.wh, it, "2026-11-01", "2026-11-30"); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("no role: %v", err)
	}
}
