package sales_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/record"
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
	fl := sales.DocFields{Date: "2026-03-10", OrgUnitID: d.a, CustomerID: d.customer, Lines: []sales.LineInput{
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
	if _, err := d.create(d.staff, d.order, d.fields(dl(9, "1"))); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("line not on the order: %v", err)
	}
}

func TestDeliveryOverOrdered(t *testing.T) {
	d := newDeliveries(t)
	first := must(t)(d.create(d.staff, d.order, d.fields(dl(1, "6"))))
	// Two lines of one order line add up: 3 + 2 is over the 4 left.
	_, err := d.create(d.staff, d.order, d.fields(dl(2, "1"), dl(1, "3"), dl(1, "2")))
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
}
