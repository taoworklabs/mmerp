package purchase_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/record/recordtest"
	"github.com/taoworklabs/mmerp/internal/modules/purchase"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/stock"
)

func (f *fixture) warehouse() int64 {
	f.t.Helper()
	var id int64
	f.check(platform.InTx(f.ctx, func(ctx context.Context) (err error) {
		id, err = f.stock.DefaultWarehouse(ctx, f.c)
		return err
	}))
	return id
}

func (f *fixture) onHand(wh, itemID int64) string {
	f.t.Helper()
	var q string
	f.check(f.core.Pool.QueryRow(f.ctx, `SELECT coalesce(sum(quantity), 0)::numeric(15,3)::text FROM stock.balances
		WHERE warehouse_id = $1 AND item_id = $2`, wh, itemID).Scan(&q))
	return q
}

func (f *fixture) liveMoves(id int64) int {
	f.t.Helper()
	var n int
	f.check(f.core.Pool.QueryRow(f.ctx, `SELECT count(*) FROM stock.moves m WHERE m.doc_type = 'purchase.goods_receipt' AND m.doc_id = $1
		AND m.reverses IS NULL AND NOT EXISTS (SELECT 1 FROM stock.moves r WHERE r.reverses = m.id)`, id).Scan(&n))
	return n
}

func lotLine(pos int, qty, code string, expiry *string) purchase.ReceiptLineInput {
	return purchase.ReceiptLineInput{OrderPosition: pos, Quantity: qty, Lot: &stock.LotRef{Code: code, Expiry: expiry}}
}

func serialLine(pos int, code string) purchase.ReceiptLineInput {
	return purchase.ReceiptLineInput{OrderPosition: pos, Quantity: "1", Serial: &code}
}

func (f *fixture) receive(order int64, date string, lines ...purchase.ReceiptLineInput) (int64, error) {
	return f.purchase.CreateGoodsReceipt(f.staff, purchase.NewGoodsReceipt{RequestID: newRequest(), OrderID: order,
		ReceiptFields: purchase.ReceiptFields{Date: date, WarehouseID: f.warehouse(), Lines: lines}})
}

func TestGoodsReceiptRecordContract(t *testing.T) {
	recordtest.Run(t, func(t *testing.T) recordtest.Harness {
		f := newFixture(t)
		order := f.postedOrder("2026-01-01")
		wh := f.warehouse()
		fields := func(date string) purchase.ReceiptFields {
			return purchase.ReceiptFields{Date: date, WarehouseID: wh, Lines: []purchase.ReceiptLineInput{lotLine(1, "1", "L1", nil)}}
		}
		return recordtest.Harness{
			Core: f.core, Actor: f.staff, Admin: f.admin, LegalEntity: f.c, Table: "purchase.goods_receipts",
			Get: func(ctx context.Context, r record.Ref) (any, error) { return f.purchase.GoodsReceipt(ctx, r.ID) },
			Create: func(ctx context.Context, date, requestID string) (record.Ref, error) {
				id, err := f.purchase.CreateGoodsReceipt(ctx, purchase.NewGoodsReceipt{RequestID: requestID, OrderID: order, ReceiptFields: fields(date)})
				return record.Ref{Type: "purchase.goods_receipt", ID: id}, err
			},
			Edit: func(ctx context.Context, r record.Ref, version int32, date string) error {
				return f.purchase.UpdateGoodsReceipt(ctx, r.ID, purchase.GoodsReceiptUpdate{Version: version, ReceiptFields: fields(date)})
			},
			RequireApproval: func(_ *testing.T, a record.Action) { f.managerApproves("purchase.goods_receipt", a) },
			Approve:         func(instance int64) error { return f.appr.Approve(f.manager, instance, 1) },
			Reject:          func(instance int64) error { return f.appr.Reject(f.manager, instance, 1, "không") },
			// Another receipt takes every carton first, so this one would exceed the order.
			BreakPosting: func(t *testing.T, _ record.Ref) {
				if _, err := f.core.System.Exec(f.ctx, `UPDATE purchase.order_lines SET quantity = 0.001 WHERE order_id = $1 AND position = 1`, order); err != nil {
					t.Fatal(err)
				}
			},
			Effects: func(_ *testing.T, r record.Ref) int { return min(f.liveMoves(r.ID), 1) },
		}
	})
}

// Two receipts take the stock lines in parts, creating lots and serials; the order shows it.
func TestGoodsReceiptPartial(t *testing.T) {
	f := newFixture(t)
	order := f.postedOrder("2026-03-10")
	wh := f.warehouse()
	first := must(t)(f.receive(order, "2026-03-11", lotLine(1, "6", "L1", str("2027-01-31")), serialLine(2, "S1")))
	if o, _ := f.purchase.Order(f.staff, order); o.ReceiptState != "none" {
		t.Fatalf("draft receipt: state %q", o.ReceiptState)
	}
	f.check(f.move(f.staff, "purchase.goods_receipt", first, record.Posted))
	if q := f.onHand(wh, f.lot); q != "6.000" {
		t.Fatalf("lot on hand %s", q)
	}
	if q := f.onHand(wh, f.serial); q != "1.000" {
		t.Fatalf("serial on hand %s", q)
	}
	o, err := f.purchase.Order(f.staff, order)
	if err != nil || o.ReceiptState != "partial" || *o.Lines[0].Received != "6" || o.Lines[2].Received != nil || len(o.GoodsReceipts) != 1 {
		t.Fatalf("partial %+v %v", o, err)
	}
	if e, ok := errors.AsType[*platform.Error](must2(f.receive(order, "2026-03-12", lotLine(1, "5", "L2", nil)))); !ok ||
		e.Code != "purchase_receipt_over_order" || e.Params["remaining"] != "4.000" || e.Params["line_no"] != 1 {
		t.Fatalf("over order: %v", e)
	}
	wantCode(t, must2(f.receive(order, "2026-03-12", lotLine(3, "1", "X", nil))), "purchase_receipt_line_not_stock")
	second := must(t)(f.receive(order, "2026-03-12", lotLine(1, "4", "L2", nil), serialLine(2, "S2")))
	f.check(f.move(f.staff, "purchase.goods_receipt", second, record.Posted))
	if o, _ := f.purchase.Order(f.staff, order); o.ReceiptState != "full" {
		t.Fatalf("full: %q", o.ReceiptState)
	}
	r, err := f.purchase.GoodsReceipt(f.staff, first)
	if err != nil || r.OrderNumber == "" || len(r.Lines) != 2 || r.Lines[0].ItemCode != "SUA" || *r.Lines[0].Lot.Expiry != "2027-01-31" {
		t.Fatalf("receipt %+v %v", r, err)
	}
	opts, err := f.purchase.ReceiptOptions(f.staff, order)
	if err != nil || len(opts.Lines) != 2 || opts.Lines[0].Remaining != "0.000" || opts.Lines[1].Tracking != "serial" || len(opts.Warehouses) != 1 {
		t.Fatalf("options %+v %v", opts, err)
	}
	l, err := f.purchase.GoodsReceipts(f.staff, purchase.GoodsReceiptFilter{OrderID: order, Sort: "-date", Paging: page})
	if err != nil || l.Total != 2 || l.Items[0].SupplierName != "Công ty NCC1" {
		t.Fatalf("list %+v %v", l, err)
	}
}

func TestGoodsReceiptSerials(t *testing.T) {
	f := newFixture(t)
	order := f.postedOrder("2026-03-10")
	bad := serialLine(2, "S1")
	bad.Quantity = "2"
	wantCode(t, must2(f.receive(order, "2026-03-11", bad)), "stock_serial_quantity")
	wantCode(t, must2(f.receive(order, "2026-03-11", purchase.ReceiptLineInput{OrderPosition: 1, Quantity: "1"})), "stock_lot_required")
	id := must(t)(f.receive(order, "2026-03-11", serialLine(2, "S1")))
	f.check(f.move(f.staff, "purchase.goods_receipt", id, record.Posted))
	again := must(t)(f.receive(order, "2026-03-12", serialLine(2, "S1")))
	if err := f.move(f.staff, "purchase.goods_receipt", again, record.Posted); err == nil {
		t.Fatal("a serial on hand received twice")
	}
}

// Goods already issued cannot be taken back out by cancelling their receipt.
func TestGoodsReceiptCancelAfterIssue(t *testing.T) {
	f := newFixture(t)
	order := f.postedOrder("2026-03-10")
	id := must(t)(f.receive(order, "2026-03-11", lotLine(1, "6", "L1", nil)))
	f.check(f.move(f.staff, "purchase.goods_receipt", id, record.Posted))
	adj := must(t)(f.stock.CreateAdjustment(f.admin, stock.NewAdjustment{AdjustmentFields: stock.AdjustmentFields{
		Date: "2026-03-12", WarehouseID: f.warehouse(), Reason: "Xuất dùng", Lines: []stock.AdjustmentLine{{ItemID: f.lot, Lot: &stock.LotRef{Code: "L1"}, Quantity: "-6"}}}}))
	f.check(f.move(f.admin, "stock.adjustment", adj, record.Posted))
	wantCode(t, f.move(f.staff, "purchase.goods_receipt", id, record.Cancelled), "stock_insufficient")
}

func TestOrderCancelWithReceipt(t *testing.T) {
	f := newFixture(t)
	order := f.postedOrder("2026-03-10")
	id := must(t)(f.receive(order, "2026-03-11", lotLine(1, "6", "L1", nil)))
	wantCode(t, f.move(f.staff, "purchase.order", order, record.Cancelled), "purchase_order_has_receipt")
	f.check(f.move(f.staff, "purchase.goods_receipt", id, record.Posted))
	wantCode(t, f.move(f.staff, "purchase.order", order, record.Cancelled), "purchase_order_has_receipt")
	f.check(f.move(f.staff, "purchase.goods_receipt", id, record.Cancelled))
	f.check(f.move(f.staff, "purchase.order", order, record.Cancelled))
	wantCode(t, must2(f.receive(order, "2026-03-11", lotLine(1, "1", "L1", nil))), "purchase_order_not_posted")
}

func TestGoodsReceiptRefusedWhileOrderCancelHeld(t *testing.T) {
	f := newFixture(t)
	order := f.postedOrder("2026-03-10")
	f.managerApproves("purchase.order", record.Cancel)
	f.check(f.move(f.staff, "purchase.order", order, record.Cancelled))
	if _, err := f.receive(order, "2026-03-11", lotLine(1, "1", "L1", nil)); err == nil {
		t.Fatal("receipt made while the order's cancel awaits approval")
	}
}

func TestGoodsReceiptBeforeOrder(t *testing.T) {
	f := newFixture(t)
	order := f.postedOrder("2026-03-10")
	wantCode(t, must2(f.receive(order, "2026-03-09", lotLine(1, "1", "L1", nil))), "purchase_receipt_before_order")
}

// A draft receipt holds its quantity back from the next one and from create_goods_receipt.
func TestGoodsReceiptDraftHoldsQuantity(t *testing.T) {
	f := newFixture(t)
	order := f.postedOrder("2026-03-10")
	o, _ := f.purchase.Order(f.staff, order)
	if !contains(o.AllowedActions, "create_goods_receipt") {
		t.Fatalf("actions %v", o.AllowedActions)
	}
	must(t)(f.receive(order, "2026-03-11", lotLine(1, "10", "L1", nil), serialLine(2, "S1"), serialLine(2, "S2")))
	o, _ = f.purchase.Order(f.staff, order)
	if contains(o.AllowedActions, "create_goods_receipt") || o.ReceiptState != "none" {
		t.Fatalf("after a draft taking all: %v %q", o.AllowedActions, o.ReceiptState)
	}
	wantCode(t, must2(f.receive(order, "2026-03-11", lotLine(1, "1", "L9", nil))), "purchase_receipt_over_order")
}

func TestGoodsReceiptTeamBoundary(t *testing.T) {
	f := newFixture(t)
	order := f.postedOrder("2026-03-10")
	id := must(t)(f.receive(order, "2026-03-11", lotLine(1, "1", "L1", nil)))
	staffB := f.ctxFor("staff_b", "staff@B")
	if _, err := f.purchase.GoodsReceipt(staffB, id); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("B opens A's receipt: %v", err)
	}
	if _, err := f.purchase.ReceiptOptions(staffB, order); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("B reads options of A's order: %v", err)
	}
	if l, _ := f.purchase.GoodsReceipts(staffB, purchase.GoodsReceiptFilter{Sort: "-date", Paging: page}); l.Total != 0 {
		t.Fatalf("B lists %+v", l)
	}
}

// Two receipts for the whole remaining quantity at once: exactly one gets it.
func TestGoodsReceiptConcurrent(t *testing.T) {
	f := newFixture(t)
	order := f.postedOrder("2026-03-10")
	wh := f.warehouse()
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range 2 {
		wg.Go(func() {
			_, errs[i] = f.purchase.CreateGoodsReceipt(f.staff, purchase.NewGoodsReceipt{RequestID: newRequest(), OrderID: order,
				ReceiptFields: purchase.ReceiptFields{Date: "2026-03-11", WarehouseID: wh, Lines: []purchase.ReceiptLineInput{lotLine(1, "10", "L1", nil)}}})
		})
	}
	wg.Wait()
	if (errs[0] == nil) == (errs[1] == nil) {
		t.Fatalf("both or neither: %v %v", errs[0], errs[1])
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
