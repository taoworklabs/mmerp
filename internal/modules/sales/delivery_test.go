package sales_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/record/recordtest"
	"github.com/taoworklabs/mmerp/internal/modules/sales"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/item"
	"github.com/taoworklabs/mmerp/internal/shared/stock"
)

// deliveries is the fixture plus the company's default warehouse, goods tracked by quantity
// and by lot, and a posted order of team A: 10 goods, 5 of the lot item, 2 hours of repair.
type deliveries struct {
	*fixture
	wh, goods, lotted, order int64
}

func newDeliveries(t *testing.T) *deliveries {
	t.Helper()
	d := &deliveries{fixture: newFixture(t)}
	d.wh = d.warehouse(d.c)
	d.goods = d.trackedItem("HANG", "quantity")
	d.lotted = d.trackedItem("THUOC", "lot")
	d.order = d.newOrder("0")
	d.check(d.move(d.staff, "sales.order", d.order, record.Posted))
	return d
}

func (d *deliveries) warehouse(legalEntity int64) int64 {
	d.t.Helper()
	var id int64
	d.check(platform.InTx(d.ctx, func(ctx context.Context) (err error) {
		id, err = d.stock.DefaultWarehouse(ctx, legalEntity)
		return err
	}))
	return id
}

func (d *deliveries) trackedItem(code, tracking string) int64 {
	d.t.Helper()
	return must(d.t)(d.items.CreateItem(d.admin, item.NewItem{Code: code, ItemFields: item.ItemFields{Name: code, Unit: "cái", Price: 1_000,
		VatRate: "8", Active: true, Tracking: &tracking}}))
}

// newOrder is a draft order of team A with the fixture's three lines, the first at discount.
func (d *deliveries) newOrder(discount string) int64 {
	d.t.Helper()
	return d.orderOn("2026-03-10", discount)
}

func (d *deliveries) orderOn(date, discount string) int64 {
	d.t.Helper()
	fl := sales.DocFields{Date: date, OrgUnitID: d.a, CustomerID: d.customer, Lines: []sales.LineInput{
		{ItemID: d.goods, Description: "Hàng", Quantity: "10", UnitPrice: 1_000, DiscountPercent: discount, VatRate: "8"},
		{ItemID: d.lotted, Description: "Thuốc", Quantity: "5", UnitPrice: 1_000, DiscountPercent: "0", VatRate: "8"},
		{ItemID: d.fix, Description: "Sửa chữa", Quantity: "2", UnitPrice: 250_000, DiscountPercent: "0", VatRate: "10"},
	}}
	return must(d.t)(d.sales.CreateOrder(d.staff, sales.NewDoc{RequestID: newRequest(), DocFields: fl}))
}

func (d *deliveries) fields(lines ...sales.DeliveryLineInput) sales.DeliveryFields {
	return sales.DeliveryFields{Date: "2026-03-11", WarehouseID: d.wh, Lines: lines}
}

func (d *deliveries) create(ctx context.Context, order int64, f sales.DeliveryFields) (int64, error) {
	return d.sales.CreateDelivery(ctx, sales.NewDelivery{RequestID: newRequest(), OrderID: order, DeliveryFields: f})
}

func dl(position int, qty string) sales.DeliveryLineInput {
	return sales.DeliveryLineInput{OrderPosition: position, Quantity: qty}
}

func lotted(position int, lot, qty string) sales.DeliveryLineInput {
	return sales.DeliveryLineInput{OrderPosition: position, LotCode: &lot, Quantity: qty}
}

func TestDeliveryFromPostedOrderOnly(t *testing.T) {
	d := newDeliveries(t)
	draft := d.newOrder("0")
	if _, err := d.create(d.staff, draft, d.fields(dl(1, "1"))); !errors.Is(err, sales.ErrOrderNotPosted) {
		t.Fatalf("delivery of a draft order: %v", err)
	}
	// A viewer of the order may not deliver it.
	if _, err := d.create(d.ctxFor("viewer", "viewer@A"), d.order, d.fields(dl(1, "1"))); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("viewer delivers: %v", err)
	}
	// Someone who cannot see the order does not learn it exists.
	if _, err := d.create(d.ctxFor("staff_b", "staff@B"), d.order, d.fields(dl(1, "1"))); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("staff of B delivers A's order: %v", err)
	}
}

func TestDeliveryRefusesServiceLine(t *testing.T) {
	d := newDeliveries(t)
	_, err := d.create(d.staff, d.order, d.fields(dl(1, "1"), dl(3, "1")))
	if e, ok := errors.AsType[*platform.Error](err); !ok || !errors.Is(err, sales.ErrDeliveryLineNotStock) || e.Params["line_no"] != 2 {
		t.Fatalf("service line: %v", err)
	}
	wantCode(t, must2(d.create(d.staff, d.order, d.fields(dl(1, "0")))), "invalid_quantity")
	wantCode(t, must2(d.create(d.staff, d.order, d.fields(dl(1, "1.0001")))), "invalid_quantity")
	_, err = d.create(d.staff, d.order, d.fields(dl(1, "1"), dl(9, "1")))
	if e, ok := errors.AsType[*platform.Error](err); !ok || !errors.Is(err, sales.ErrDeliveryPosition) || e.Status != 422 || e.Params["line_no"] != 2 {
		t.Fatalf("line not on the order: %v", err)
	}
}

// Each line's lot, serial and quantity must fit its item's tracking already on saving.
func TestDeliveryLineShapeAtSave(t *testing.T) {
	d := newDeliveries(t)
	serial := d.trackedItem("MAY", "serial")
	fl := sales.DocFields{Date: "2026-03-10", OrgUnitID: d.a, CustomerID: d.customer, Lines: []sales.LineInput{
		{ItemID: d.goods, Description: "Hàng", Quantity: "10", UnitPrice: 1_000, DiscountPercent: "0", VatRate: "8"},
		{ItemID: d.lotted, Description: "Thuốc", Quantity: "5", UnitPrice: 1_000, DiscountPercent: "0", VatRate: "8"},
		{ItemID: serial, Description: "Máy", Quantity: "3", UnitPrice: 1_000, DiscountPercent: "0", VatRate: "8"},
	}}
	order := must(t)(d.sales.CreateOrder(d.staff, sales.NewDoc{RequestID: newRequest(), DocFields: fl}))
	d.check(d.move(d.staff, "sales.order", order, record.Posted))
	sn := func(code, qty string) sales.DeliveryLineInput {
		return sales.DeliveryLineInput{OrderPosition: 3, SerialCode: str(code), Quantity: qty}
	}
	for name, c := range map[string]struct {
		line sales.DeliveryLineInput
		want *platform.Error
	}{
		"lot item without lot":     {dl(2, "1"), stock.ErrLotRequired},
		"serial item without code": {dl(3, "1"), stock.ErrSerialRequired},
		"serial by two":            {sn("S1", "2"), stock.ErrSerialQuantity},
		"lot on a quantity item":   {lotted(1, "L1", "1"), stock.ErrLotNotTracked},
	} {
		_, err := d.create(d.staff, order, d.fields(lotted(2, "L1", "1"), c.line))
		if e, ok := errors.AsType[*platform.Error](err); !ok || !errors.Is(err, c.want) || e.Params["line_no"] != 2 {
			t.Errorf("%s: %v", name, err)
		}
	}
	must(t)(d.create(d.staff, order, d.fields(dl(1, "2"), lotted(2, "L1", "1"), sn("S1", "1"))))
}

func TestDeliveryOverOrdered(t *testing.T) {
	d := newDeliveries(t)
	first := must(t)(d.create(d.staff, d.order, d.fields(dl(1, "6"))))
	// Two lines of one order line add up: 3 + 2 is over the 4 left.
	_, err := d.create(d.staff, d.order, d.fields(lotted(2, "L1", "1"), dl(1, "3"), dl(1, "2")))
	e, ok := errors.AsType[*platform.Error](err)
	if !ok || !errors.Is(err, sales.ErrDeliveryOverOrdered) || e.Params["line_no"] != 3 || e.Params["order_position"] != 1 || e.Params["remaining"] != "4.000" {
		t.Fatalf("over ordered: %v %+v", err, e)
	}
	second := must(t)(d.create(d.staff, d.order, d.fields(dl(1, "4"))))
	// Editing a delivery does not count its own lines against it.
	v, err := d.sales.Delivery(d.staff, first)
	d.check(err)
	d.check(d.sales.UpdateDelivery(d.staff, first, sales.DeliveryUpdate{Version: v.Version, DeliveryFields: d.fields(dl(1, "5.5"))}))
	wantCode(t, d.sales.UpdateDelivery(d.staff, first, sales.DeliveryUpdate{Version: v.Version + 1, DeliveryFields: d.fields(dl(1, "6.5"))}),
		"sales_delivery_over_ordered")
	// A deleted draft takes nothing.
	d.check(d.rec.Delete(d.staff, record.Ref{Type: "sales.delivery", ID: second}, 1))
	must(t)(d.create(d.staff, d.order, d.fields(dl(1, "4.5"))))
}

func TestDeliveryWarehouseOfOtherEntity(t *testing.T) {
	d := newDeliveries(t)
	other := must(t)(d.core.IAM.CreateOrgUnit(d.admin, iam.OrgUnitInput{Kind: "company", Name: "D"}))
	f := d.fields(dl(1, "1"))
	f.WarehouseID = d.warehouse(other)
	if _, err := d.create(d.staff, d.order, f); !errors.Is(err, sales.ErrDeliveryWarehouse) {
		t.Fatalf("warehouse of another company: %v", err)
	}
	// An inactive warehouse of the company is refused too.
	inactive := must(t)(d.stock.CreateWarehouse(d.admin, stock.NewWarehouse{WarehouseFields: stock.WarehouseFields{Name: "Cũ", LegalEntityID: d.c}}))
	if _, err := d.create(d.staff, d.order, sales.DeliveryFields{Date: "2026-03-11", WarehouseID: inactive, Lines: f.Lines}); !errors.Is(err, sales.ErrDeliveryWarehouse) {
		t.Fatalf("inactive warehouse: %v", err)
	}
	// Changing a draft's warehouse runs the same check.
	id := must(t)(d.create(d.staff, d.order, d.fields(dl(1, "1"))))
	if err := d.sales.UpdateDelivery(d.staff, id, sales.DeliveryUpdate{Version: 1, DeliveryFields: f}); !errors.Is(err, sales.ErrDeliveryWarehouse) {
		t.Fatalf("edit to another company's warehouse: %v", err)
	}
}

func TestDeliveryReadBack(t *testing.T) {
	d := newDeliveries(t)
	in := sales.NewDelivery{RequestID: newRequest(), OrderID: d.order, DeliveryFields: d.fields(dl(1, "4"),
		sales.DeliveryLineInput{OrderPosition: 2, LotCode: str("L1"), Quantity: "2.5"})}
	in.Note = str("Giao buổi sáng")
	id := must(t)(d.sales.CreateDelivery(d.staff, in))
	if again := must(t)(d.sales.CreateDelivery(d.staff, in)); again != id {
		t.Fatalf("same request: %d, then %d", id, again)
	}
	got, err := d.sales.Delivery(d.staff, id)
	d.check(err)
	o, err := d.sales.Order(d.staff, d.order)
	d.check(err)
	l := got.Lines
	if got.OrderID != d.order || got.OrderNumber != o.Number || got.Number != "GH-2026-00001" || got.Status != "draft" || got.Date != "2026-03-11" ||
		got.OrgUnitID != d.a || got.WarehouseID != d.wh || got.WarehouseCode == "" || got.WarehouseName == "" || *got.Note != "Giao buổi sáng" ||
		len(l) != 2 || l[0].ItemID != d.goods || l[0].ItemCode != "HANG" || l[0].Description != "Hàng" || l[0].Unit != "cái" || l[0].Quantity != "4" ||
		l[0].LotCode != nil || l[1].OrderPosition != 2 || *l[1].LotCode != "L1" || l[1].Quantity != "2.5" {
		t.Fatalf("delivery %+v", got)
	}
	if !slices.Equal(got.AllowedActions, []string{"edit", "delete", "submit"}) {
		t.Fatalf("draft actions %v", got.AllowedActions)
	}
	// Who cannot see the order's team cannot open its delivery.
	if _, err := d.sales.Delivery(d.ctxFor("staff_b", "staff@B"), id); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("staff of B opens: %v", err)
	}
}

// An order whose cancelling waits for approval takes no new delivery.
func TestDeliveryRefusedWhileCancelHeld(t *testing.T) {
	d := newDeliveries(t)
	d.managerApproves("sales.order", record.Cancel)
	order := d.newOrder("15")
	d.check(d.move(d.staff, "sales.order", order, record.Posted))
	d.check(d.move(d.staff, "sales.order", order, record.Cancelled))
	if o, _ := d.sales.Order(d.staff, order); o.Status != "posted" {
		t.Fatalf("cancel not held: %s", o.Status)
	}
	wantCode(t, must2(d.create(d.staff, order, d.fields(dl(1, "1")))), "awaiting_approval")
	if d.offersDelivery(d.staff, order) {
		t.Fatal("held order offers create_delivery")
	}
}

func (d *deliveries) offersDelivery(ctx context.Context, order int64) bool {
	d.t.Helper()
	o, err := d.sales.Order(ctx, order)
	d.check(err)
	return slices.Contains(o.AllowedActions, "create_delivery")
}

// A confirmed order offers create_delivery to who may edit it while a stock line has
// something left that no delivery, draft or posted, takes.
func TestOrderOffersCreateDelivery(t *testing.T) {
	d := newDeliveries(t)
	if !d.offersDelivery(d.staff, d.order) {
		t.Fatal("posted order does not offer create_delivery")
	}
	if d.offersDelivery(d.staff, d.newOrder("0")) {
		t.Fatal("draft order offers create_delivery")
	}
	if d.offersDelivery(d.ctxFor("viewer", "viewer@A"), d.order) {
		t.Fatal("viewer is offered create_delivery")
	}
	id := must(t)(d.create(d.staff, d.order, d.fields(dl(1, "10"), lotted(2, "L1", "4"))))
	if !d.offersDelivery(d.staff, d.order) {
		t.Fatal("one lot item left, yet no create_delivery")
	}
	d.check(d.sales.UpdateDelivery(d.staff, id, sales.DeliveryUpdate{Version: 1, DeliveryFields: d.fields(dl(1, "10"), lotted(2, "L1", "5"))}))
	if d.offersDelivery(d.staff, d.order) {
		t.Fatal("fully reserved order offers create_delivery")
	}
}

// adjust posts a stock adjustment of the warehouse with lines.
func (d *deliveries) adjust(lines ...stock.AdjustmentLine) {
	d.t.Helper()
	id := must(d.t)(d.stock.CreateAdjustment(d.admin, stock.NewAdjustment{AdjustmentFields: stock.AdjustmentFields{
		Date: "2026-03-01", WarehouseID: d.wh, Reason: "Kiểm kê", Lines: lines}}))
	d.check(d.move(d.admin, "stock.adjustment", id, record.Posted))
}

func (d *deliveries) onHand(itemID int64) string {
	d.t.Helper()
	var q string
	d.check(d.core.Pool.QueryRow(d.ctx, `SELECT coalesce(sum(quantity), 0)::numeric(15,3)::text FROM stock.balances
		WHERE warehouse_id = $1 AND item_id = $2`, d.wh, itemID).Scan(&q))
	return q
}

// liveMoves counts a delivery's moves not reversed.
func (d *deliveries) liveMoves(id int64) int {
	d.t.Helper()
	var n int
	d.check(d.core.Pool.QueryRow(d.ctx, `SELECT count(*) FROM stock.moves m WHERE m.doc_type = 'sales.delivery' AND m.doc_id = $1
		AND m.reverses IS NULL AND NOT EXISTS (SELECT 1 FROM stock.moves r WHERE r.reverses = m.id)`, id).Scan(&n))
	return n
}

func TestDeliveryRecordContract(t *testing.T) {
	recordtest.Run(t, func(t *testing.T) recordtest.Harness {
		d := newDeliveries(t)
		d.adjust(stock.AdjustmentLine{ItemID: d.goods, Quantity: "100"})
		// The suite dates deliveries freely, so the order is older than any of them.
		order := d.orderOn("2026-01-01", "0")
		d.check(d.move(d.staff, "sales.order", order, record.Posted))
		fields := func(date string) sales.DeliveryFields {
			f := d.fields(dl(1, "1"))
			f.Date = date
			return f
		}
		return recordtest.Harness{
			Core: d.core, Actor: d.staff, Admin: d.admin, LegalEntity: d.c, Table: "sales.deliveries",
			Get: func(ctx context.Context, r record.Ref) (any, error) { return d.sales.Delivery(ctx, r.ID) },
			Create: func(ctx context.Context, date, requestID string) (record.Ref, error) {
				id, err := d.sales.CreateDelivery(ctx, sales.NewDelivery{RequestID: requestID, OrderID: order, DeliveryFields: fields(date)})
				return record.Ref{Type: "sales.delivery", ID: id}, err
			},
			Edit: func(ctx context.Context, r record.Ref, version int32, date string) error {
				return d.sales.UpdateDelivery(ctx, r.ID, sales.DeliveryUpdate{Version: version, DeliveryFields: fields(date)})
			},
			RequireApproval: func(t *testing.T, a record.Action) {
				d.check(d.appr.SaveRule(d.admin, "sales.delivery", a, approval.RuleInput{
					Flow:      approval.Flow{{Step: &approval.StepDef{Approver: approval.Approver{Kind: "role", Product: "sales", Role: "manager"}}}},
					MaxLevels: 3, FallbackProduct: "sales", FallbackRole: "manager",
				}))
			},
			Approve: func(instance int64) error { return d.appr.Approve(d.manager, instance, 1) },
			Reject:  func(instance int64) error { return d.appr.Reject(d.manager, instance, 1, "không") },
			// Taking every unit elsewhere leaves the delivery short.
			BreakPosting: func(*testing.T, record.Ref) {
				d.adjust(stock.AdjustmentLine{ItemID: d.goods, Quantity: "-" + d.onHand(d.goods)})
			},
			// An effective posting leaves the delivery's moves live.
			Effects: func(_ *testing.T, r record.Ref) int { return min(d.liveMoves(r.ID), 1) },
		}
	})
}

// Posting takes the goods out, lots included; cancelling puts them back.
func TestDeliveryPostsAndCancelRestores(t *testing.T) {
	d := newDeliveries(t)
	d.adjust(stock.AdjustmentLine{ItemID: d.goods, Quantity: "10"}, stock.AdjustmentLine{ItemID: d.lotted, Lot: &stock.LotRef{Code: "L1"}, Quantity: "5"})
	id := must(t)(d.create(d.staff, d.order, d.fields(dl(1, "4"), sales.DeliveryLineInput{OrderPosition: 2, LotCode: str("L1"), Quantity: "2"})))
	if d.onHand(d.goods) != "10.000" {
		t.Fatalf("a draft moved stock: %s", d.onHand(d.goods))
	}
	d.check(d.move(d.staff, "sales.delivery", id, record.Posted))
	if g, l := d.onHand(d.goods), d.onHand(d.lotted); g != "6.000" || l != "3.000" {
		t.Fatalf("on hand after posting: %s, %s", g, l)
	}
	d.check(d.move(d.staff, "sales.delivery", id, record.Cancelled))
	if g, l := d.onHand(d.goods), d.onHand(d.lotted); g != "10.000" || l != "5.000" {
		t.Fatalf("on hand after cancelling: %s, %s", g, l)
	}
	// A cancelled delivery takes nothing from the order: all 10 can go again.
	must(t)(d.create(d.staff, d.order, d.fields(dl(1, "10"))))
}

func TestDeliveryPostingRefuses(t *testing.T) {
	d := newDeliveries(t)
	id := must(t)(d.create(d.staff, d.order, d.fields(dl(1, "4"))))
	// Short stock refuses posting and moves nothing.
	d.adjust(stock.AdjustmentLine{ItemID: d.goods, Quantity: "3"})
	wantCode(t, d.move(d.staff, "sales.delivery", id, record.Posted), "stock_insufficient")
	if d.onHand(d.goods) != "3.000" || d.liveMoves(id) != 0 {
		t.Fatalf("refused posting moved stock: %s", d.onHand(d.goods))
	}
}

func TestOrderCancelRefusedWithDelivery(t *testing.T) {
	d := newDeliveries(t)
	id := must(t)(d.create(d.staff, d.order, d.fields(dl(1, "1"))))
	if err := d.move(d.staff, "sales.order", d.order, record.Cancelled); !errors.Is(err, sales.ErrOrderHasDelivery) {
		t.Fatalf("cancel with a draft delivery: %v", err)
	}
	d.check(d.rec.Delete(d.staff, record.Ref{Type: "sales.delivery", ID: id}, 1))
	d.check(d.move(d.staff, "sales.order", d.order, record.Cancelled))
}

// race runs fn in n goroutines released together, each in its own transaction.
func (d *deliveries) race(n int, fn func(ctx context.Context, i int) error) []error {
	errs := make([]error, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs[i] = platform.InTx(d.staff, func(ctx context.Context) error { return fn(ctx, i) })
		}()
	}
	close(start)
	wg.Wait()
	return errs
}

func TestConcurrentDeliverySaves(t *testing.T) {
	d := newDeliveries(t)
	errs := d.race(2, func(ctx context.Context, _ int) error {
		_, err := d.create(ctx, d.order, d.fields(dl(1, "6")))
		return err
	})
	ok, over := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, sales.ErrDeliveryOverOrdered):
			over++
		default:
			t.Fatalf("unexpected: %v", err)
		}
	}
	if ok != 1 || over != 1 {
		t.Fatalf("ok=%d over=%d", ok, over)
	}
}

func TestCancelOrderVsCreateDelivery(t *testing.T) {
	d := newDeliveries(t)
	errs := d.race(2, func(ctx context.Context, i int) error {
		if i == 0 {
			return d.move(ctx, "sales.order", d.order, record.Cancelled)
		}
		_, err := d.create(ctx, d.order, d.fields(dl(1, "1")))
		return err
	})
	cancelErr, createErr := errs[0], errs[1]
	switch {
	case cancelErr == nil && errors.Is(createErr, sales.ErrOrderNotPosted):
	case createErr == nil && errors.Is(cancelErr, sales.ErrOrderHasDelivery):
	default:
		t.Fatalf("cancel: %v, create: %v", cancelErr, createErr)
	}
}

// goodsOrder is a draft order of team A for qty of the goods only.
func (d *deliveries) goodsOrder(qty string) int64 {
	d.t.Helper()
	fl := sales.DocFields{Date: "2026-03-10", OrgUnitID: d.a, CustomerID: d.customer, Lines: []sales.LineInput{
		{ItemID: d.goods, Description: "Hàng", Quantity: qty, UnitPrice: 1_000, DiscountPercent: "0", VatRate: "8"}}}
	return must(d.t)(d.sales.CreateOrder(d.staff, sales.NewDoc{RequestID: newRequest(), DocFields: fl}))
}

func (d *deliveries) posted(order int64, lines ...sales.DeliveryLineInput) int64 {
	d.t.Helper()
	id := must(d.t)(d.create(d.staff, order, d.fields(lines...)))
	d.check(d.move(d.staff, "sales.delivery", id, record.Posted))
	return id
}

func (d *deliveries) orderState(want string, delivered ...string) sales.Doc {
	d.t.Helper()
	o, err := d.sales.Order(d.staff, d.order)
	d.check(err)
	got := []string{}
	for _, l := range o.Lines {
		if l.Delivered == nil {
			got = append(got, "-")
		} else {
			got = append(got, *l.Delivered)
		}
	}
	if o.DeliveryState != want || !slices.Equal(got, delivered) {
		d.t.Fatalf("state %s delivered %v, want %s %v", o.DeliveryState, got, want, delivered)
	}
	l, err := d.sales.Orders(d.staff, sales.DocFilter{DeliveryState: want, Sort: "-date", Paging: platform.Paging{Page: 1, PageSize: 100}})
	d.check(err)
	if i := slices.IndexFunc(l.Items, func(x sales.DocListItem) bool { return x.ID == d.order }); i < 0 || l.Items[i].DeliveryState != want {
		d.t.Fatalf("list filtered by %s: %+v", want, l.Items)
	}
	return o
}

func TestDeliveryStateDerived(t *testing.T) {
	d := newDeliveries(t)
	d.adjust(stock.AdjustmentLine{ItemID: d.goods, Quantity: "10"}, stock.AdjustmentLine{ItemID: d.lotted, Lot: &stock.LotRef{Code: "L1"}, Quantity: "5"})
	d.orderState("none", "0", "0", "-")
	// A draft delivery delivers nothing yet.
	draft := must(t)(d.create(d.staff, d.order, d.fields(dl(1, "1"))))
	d.orderState("none", "0", "0", "-")
	d.check(d.rec.Delete(d.staff, record.Ref{Type: "sales.delivery", ID: draft}, 1))
	first := d.posted(d.order, dl(1, "4"))
	d.orderState("partial", "4", "0", "-")
	second := d.posted(d.order, dl(1, "6"), sales.DeliveryLineInput{OrderPosition: 2, LotCode: str("L1"), Quantity: "5"})
	o := d.orderState("full", "10", "5", "-")
	if len(o.Deliveries) != 2 || o.Deliveries[0].ID != first || o.Deliveries[1].ID != second || o.Deliveries[0].Status != "posted" {
		t.Fatalf("deliveries %+v", o.Deliveries)
	}
	d.check(d.move(d.staff, "sales.delivery", first, record.Cancelled))
	o = d.orderState("partial", "6", "5", "-")
	if len(o.Deliveries) != 1 || o.Deliveries[0].ID != second {
		t.Fatalf("cancelled one still listed: %+v", o.Deliveries)
	}
	// An order of services only has nothing to deliver; quotations show no state.
	services := must(t)(d.sales.CreateOrder(d.staff, sales.NewDoc{RequestID: newRequest(), DocFields: sales.DocFields{Date: "2026-03-10",
		OrgUnitID: d.a, CustomerID: d.customer, Lines: []sales.LineInput{
			{ItemID: d.fix, Description: "Sửa chữa", Quantity: "2", UnitPrice: 250_000, DiscountPercent: "0", VatRate: "10"}}}}))
	if s, err := d.sales.Order(d.staff, services); err != nil || s.DeliveryState != "not_applicable" || s.Lines[0].Delivered != nil {
		t.Fatalf("services only: %+v %v", s, err)
	}
	q := must(t)(d.sales.CreateQuote(d.staff, sales.NewDoc{RequestID: newRequest(), DocFields: d.fixture.fields("2026-03-10")}))
	if s, err := d.sales.Quote(d.staff, q); err != nil || s.DeliveryState != "" || s.Lines[0].Delivered != nil || s.Lines[0].Available != nil {
		t.Fatalf("quotation: %+v %v", s, err)
	}
}

// On hand 10 and a posted order committing 6 leave 4 for a draft order; a posted
// order shows no availability.
func TestAvailableOnDraftOrder(t *testing.T) {
	d := newDeliveries(t)
	d.check(d.move(d.staff, "sales.order", d.order, record.Cancelled))
	d.adjust(stock.AdjustmentLine{ItemID: d.goods, Quantity: "10"})
	a := d.goodsOrder("6")
	d.check(d.move(d.staff, "sales.order", a, record.Posted))
	b := d.goodsOrder("5")
	if o, err := d.sales.Order(d.staff, b); err != nil || o.Lines[0].Available == nil || *o.Lines[0].Available != "4.000" {
		t.Fatalf("draft order: %+v %v", o.Lines, err)
	}
	c := d.goodsOrder("1")
	d.check(d.move(d.staff, "sales.order", c, record.Posted))
	if o, err := d.sales.Order(d.staff, c); err != nil || o.Lines[0].Available != nil {
		t.Fatalf("posted order: %+v %v", o.Lines, err)
	}
	// The draft's services line shows none either.
	o, err := d.sales.Order(d.staff, d.newOrder("0"))
	d.check(err)
	if o.Lines[0].Available == nil || *o.Lines[0].Available != "3.000" || o.Lines[2].Available != nil {
		t.Fatalf("mixed draft: %+v", o.Lines)
	}
}

func TestAvailabilityEndpoint(t *testing.T) {
	d := newDeliveries(t)
	d.adjust(stock.AdjustmentLine{ItemID: d.goods, Quantity: "15"}, stock.AdjustmentLine{ItemID: d.lotted, Lot: &stock.LotRef{Code: "L1"}, Quantity: "3"})
	// Delivering 4 takes them from both on hand and committed.
	d.posted(d.order, dl(1, "4"))
	got, err := d.sales.Availability(d.staff, d.a, []int64{d.goods, d.lotted})
	d.check(err)
	want := []sales.ItemAvailability{
		{ItemID: d.goods, OnHand: "11.000", Committed: "6.000", Available: "5.000"},
		{ItemID: d.lotted, OnHand: "3.000", Committed: "5.000", Available: "-2.000"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("availability %+v, want %+v", got, want)
	}
	if _, err := d.sales.Availability(d.ctxFor("nobody"), d.a, []int64{d.goods}); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("without order view: %v", err)
	}
}

func TestDeliveriesList(t *testing.T) {
	d := newDeliveries(t)
	other := d.newOrder("0")
	d.check(d.move(d.staff, "sales.order", other, record.Posted))
	id := must(t)(d.create(d.staff, d.order, d.fields(dl(1, "1"))))
	must(t)(d.create(d.staff, other, d.fields(dl(1, "1"))))
	f := sales.DeliveryFilter{OrderID: d.order, Sort: "-date", Paging: platform.Paging{Page: 1, PageSize: 20}}
	l, err := d.sales.Deliveries(d.staff, f)
	d.check(err)
	o, err := d.sales.Order(d.staff, d.order)
	d.check(err)
	if l.Total != 1 || len(l.Items) != 1 || l.Items[0].ID != id || l.Items[0].OrderNumber != o.Number || l.Items[0].Status != "draft" ||
		l.Items[0].WarehouseID != d.wh || l.Items[0].CustomerName == "" {
		t.Fatalf("deliveries of the order: %+v", l)
	}
	f.OrderID = 0
	if l, err := d.sales.Deliveries(d.staff, f); err != nil || l.Total != 2 {
		t.Fatalf("all: %+v %v", l, err)
	}
	if l, err := d.sales.Deliveries(d.ctxFor("staff_b", "staff@B"), f); err != nil || l.Total != 0 {
		t.Fatalf("staff of B: %+v %v", l, err)
	}
}

func TestDeliveryOptions(t *testing.T) {
	d := newDeliveries(t)
	exp1, exp2 := "2026-12-01", "2026-11-15"
	d.adjust(stock.AdjustmentLine{ItemID: d.goods, Quantity: "15"},
		stock.AdjustmentLine{ItemID: d.lotted, Lot: &stock.LotRef{Code: "L1", Expiry: &exp1}, Quantity: "3"},
		stock.AdjustmentLine{ItemID: d.lotted, Lot: &stock.LotRef{Code: "L2", Expiry: &exp2}, Quantity: "2"})
	// A draft delivery of 2 lot items leaves 3 to suggest.
	must(t)(d.create(d.staff, d.order, d.fields(dl(1, "4"), lotted(2, "L1", "2"))))
	o, err := d.sales.DeliveryOptions(d.staff, d.order, 0)
	d.check(err)
	if len(o.Warehouses) != 1 || o.Warehouses[0].ID != d.wh || !o.Warehouses[0].IsDefault {
		t.Fatalf("warehouses %+v", o.Warehouses)
	}
	if len(o.Lines) != 2 {
		t.Fatalf("service line must be absent: %+v", o.Lines)
	}
	g, l := o.Lines[0], o.Lines[1]
	if g.OrderPosition != 1 || g.Ordered != "10.000" || g.Remaining != "6.000" || len(g.Lots) != 0 {
		t.Fatalf("goods %+v", g)
	}
	if l.Remaining != "3.000" || len(l.Lots) != 2 || l.Lots[0].Code != "L2" || l.Lots[0].Quantity != "2.000" || l.Lots[1].Code != "L1" || l.Lots[1].Quantity != "1.000" {
		t.Fatalf("lots %+v", l)
	}
	if _, err := d.sales.DeliveryOptions(d.staff, d.order, d.wh); err != nil {
		t.Fatal(err)
	}
	if _, err := d.sales.DeliveryOptions(d.ctxFor("viewer", "viewer@A"), d.order, d.wh); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("viewer: %v", err)
	}
	if _, err := d.sales.DeliveryOptions(d.staff, d.newOrder("0"), 0); !errors.Is(err, sales.ErrOrderNotPosted) {
		t.Fatalf("draft order: %v", err)
	}
}

func TestDeliveryOptionsSerials(t *testing.T) {
	d := newDeliveries(t)
	ser := d.trackedItem("MAY", "serial")
	fl := sales.DocFields{Date: "2026-03-10", OrgUnitID: d.a, CustomerID: d.customer, Lines: []sales.LineInput{
		{ItemID: ser, Description: "Máy", Quantity: "2", UnitPrice: 1_000, DiscountPercent: "0", VatRate: "8"}}}
	o := must(t)(d.sales.CreateOrder(d.staff, sales.NewDoc{RequestID: newRequest(), DocFields: fl}))
	d.check(d.move(d.staff, "sales.order", o, record.Posted))
	d.adjust(stock.AdjustmentLine{ItemID: ser, Serial: new("S2"), Quantity: "1"}, stock.AdjustmentLine{ItemID: ser, Serial: new("S1"), Quantity: "1"})
	got, err := d.sales.DeliveryOptions(d.staff, o, d.wh)
	d.check(err)
	if len(got.Lines) != 1 || !slices.Equal(got.Lines[0].Serials, []string{"S1", "S2"}) {
		t.Fatalf("serials %+v", got.Lines)
	}
}

func TestDeliveryRefusesExpiredLot(t *testing.T) {
	d := newDeliveries(t)
	exp := "2026-11-01"
	d.adjust(stock.AdjustmentLine{ItemID: d.lotted, Lot: &stock.LotRef{Code: "L1", Expiry: &exp}, Quantity: "3"})
	f := d.fields(lotted(2, "l1", "1"))
	f.Date = "2026-11-02"
	_, err := d.create(d.staff, d.order, f)
	if e, ok := errors.AsType[*platform.Error](err); !ok || !errors.Is(err, sales.ErrLotExpired) || e.Params["line_no"] != 1 ||
		e.Params["lot_code"] != "l1" || e.Params["expiry"] != exp {
		t.Fatalf("expired lot: %v", err)
	}
	f.Date = exp
	id := must(t)(d.create(d.staff, d.order, f))
	// Moving the date past expiry at save is refused too.
	f.Date = "2026-11-02"
	_, err = d.sales.Delivery(d.staff, id)
	d.check(err)
	err = d.sales.UpdateDelivery(d.staff, id, sales.DeliveryUpdate{Version: 1, DeliveryFields: f})
	if !errors.Is(err, sales.ErrLotExpired) {
		t.Fatalf("update past expiry: %v", err)
	}
}

func TestDeliveryDateBeforeOrder(t *testing.T) {
	d := newDeliveries(t)
	f := d.fields(dl(1, "1"))
	f.Date = "2026-03-09"
	if _, err := d.create(d.staff, d.order, f); !errors.Is(err, sales.ErrDeliveryBeforeOrder) {
		t.Fatalf("early delivery: %v", err)
	}
}
