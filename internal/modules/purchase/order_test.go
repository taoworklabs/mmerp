package purchase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/record/recordtest"
	"github.com/taoworklabs/mmerp/internal/modules/purchase"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/partner"
)

func (f *fixture) deactivateSupplier() {
	f.check(f.purchase.UpdateSupplier(f.staff, f.supplier, purchase.SupplierUpdate{
		SupplierFields: purchase.SupplierFields{PaymentTerms: str("15 ngày"), OrgUnitID: f.a, Active: false}}))
}

func (f *fixture) postedOrders() int {
	l, err := f.purchase.Orders(f.staff, purchase.OrderFilter{Status: "posted", Sort: "-date", Paging: page})
	f.check(err)
	return int(l.Total)
}

func TestOrderRecordContract(t *testing.T) {
	recordtest.Run(t, func(t *testing.T) recordtest.Harness {
		f := newFixture(t)
		deactivate := func(*testing.T, record.Ref) { f.deactivateSupplier() }
		return recordtest.Harness{
			Core: f.core, Actor: f.staff, Admin: f.admin, LegalEntity: f.c, Table: "purchase.orders",
			Get: func(ctx context.Context, r record.Ref) (any, error) { return f.purchase.Order(ctx, r.ID) },
			Gaps: map[string]string{
				"write actions leave allowed_actions when the product is off or the period locked": "a posted order in a locked period still offers create_invoice, dated in an open period",
			},
			Create: func(ctx context.Context, date, requestID string) (record.Ref, error) {
				id, err := f.purchase.CreateOrder(ctx, purchase.NewOrder{RequestID: requestID, OrderFields: f.orderFields(date)})
				return record.Ref{Type: "purchase.order", ID: id}, err
			},
			Edit: func(ctx context.Context, r record.Ref, version int32, date string) error {
				return f.purchase.UpdateOrder(ctx, r.ID, purchase.OrderUpdate{Version: version, OrderFields: f.orderFields(date)})
			},
			RequireApproval: func(_ *testing.T, a record.Action) { f.managerApproves("purchase.order", a) },
			Approve:         func(instance int64) error { return f.appr.Approve(f.manager, instance, 1) },
			Reject:          func(instance int64) error { return f.appr.Reject(f.manager, instance, 1, "không") },
			BreakSubmit:     deactivate,
			BreakPosting:    deactivate,
			Effects:         func(*testing.T, record.Ref) int { return f.postedOrders() },
		}
	})
}

// Saving computes every amount and copies the supplier; sending copies it again.
func TestOrderLifecycle(t *testing.T) {
	f := newFixture(t)
	id := must(t)(f.purchase.CreateOrder(f.staff, purchase.NewOrder{RequestID: newRequest(), OrderFields: f.orderFields("2026-03-10")}))
	o, err := f.purchase.Order(f.staff, id)
	f.check(err)
	// 500,000 + 40,000 VAT; 6,000,000 + 600,000; 500,000 + 50,000.
	if o.Totals != (purchase.Totals{Subtotal: 7_000_000, VatTotal: 690_000, Total: 7_690_000}) || o.Status != "draft" ||
		o.Supplier.Code != "NCC1" || o.Supplier.Name != "Công ty NCC1" || *o.Supplier.TaxCode != "0301234567" {
		t.Fatalf("draft %+v", o)
	}
	if !o.Lines[0].Stocked || !o.Lines[1].Stocked || o.Lines[2].Stocked || o.Lines[0].ItemCode != "SUA" || o.Lines[0].Unit != "thùng" {
		t.Fatalf("lines %+v", o.Lines)
	}
	s, err := f.purchase.Supplier(f.staff, f.supplier)
	f.check(err)
	f.check(f.partners.Update(f.staff, s.PartnerID, partner.PartnerFields{Name: "Công ty NCC Mới", Address: str("9 Lý Thường Kiệt")}))
	f.check(f.move(f.staff, "purchase.order", id, record.Posted))
	o, err = f.purchase.Order(f.staff, id)
	if err != nil || o.Status != "posted" || o.Supplier.Name != "Công ty NCC Mới" {
		t.Fatalf("posted %+v %v", o, err)
	}
	// A partner change after posting never reaches the order.
	f.check(f.partners.Update(f.staff, s.PartnerID, partner.PartnerFields{Name: "Đổi lần nữa"}))
	if o, _ := f.purchase.Order(f.staff, id); o.Supplier.Name != "Công ty NCC Mới" {
		t.Fatalf("after posting %+v", o.Supplier)
	}
}

func TestOrderInactiveSupplier(t *testing.T) {
	f := newFixture(t)
	id := must(t)(f.purchase.CreateOrder(f.staff, purchase.NewOrder{RequestID: newRequest(), OrderFields: f.orderFields("2026-03-10")}))
	f.deactivateSupplier()
	wantCode(t, must2(f.purchase.CreateOrder(f.staff, purchase.NewOrder{RequestID: newRequest(), OrderFields: f.orderFields("2026-03-10")})), "supplier_inactive")
	wantCode(t, f.move(f.staff, "purchase.order", id, record.Posted), "supplier_inactive")
}

func TestOrderInactiveItem(t *testing.T) {
	f := newFixture(t)
	fl := f.orderFields("2026-03-10")
	fl.Lines[0].ItemID = 999999
	wantCode(t, must2(f.purchase.CreateOrder(f.staff, purchase.NewOrder{RequestID: newRequest(), OrderFields: fl})), "item_inactive")
}

func TestOrderApprovalAmount(t *testing.T) {
	f := newFixture(t)
	f.managerApproves("purchase.order", record.Post)
	id := must(t)(f.purchase.CreateOrder(f.staff, purchase.NewOrder{RequestID: newRequest(), OrderFields: f.orderFields("2026-03-10")}))
	f.check(f.move(f.staff, "purchase.order", id, record.Posted))
	d, err := f.rec.Get(f.admin, record.Ref{Type: "purchase.order", ID: id})
	if err != nil || d.Status != record.PendingApproval || d.Amount == nil || *d.Amount != 7_690_000 {
		t.Fatalf("held %+v %v", d, err)
	}
}

func TestOrderTeamBoundary(t *testing.T) {
	f := newFixture(t)
	id := f.postedOrder("2026-03-10")
	staffB := f.ctxFor("staff_b", "staff@B")
	if _, err := f.purchase.Order(staffB, id); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("B opens A's order: %v", err)
	}
	if l, err := f.purchase.Orders(staffB, purchase.OrderFilter{Sort: "-date", Paging: page}); err != nil || l.Total != 0 {
		t.Fatalf("B lists %+v %v", l, err)
	}
	fl := f.orderFields("2026-03-10")
	fl.OrgUnitID = f.b
	if _, err := f.purchase.CreateOrder(f.staff, purchase.NewOrder{RequestID: newRequest(), OrderFields: fl}); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("A creates in B: %v", err)
	}
}

func TestOrderList(t *testing.T) {
	f := newFixture(t)
	posted := f.postedOrder("2026-03-10")
	must(t)(f.purchase.CreateOrder(f.staff, purchase.NewOrder{RequestID: newRequest(), OrderFields: f.orderFields("2026-03-11")}))
	l, err := f.purchase.Orders(f.staff, purchase.OrderFilter{Status: "posted", Sort: "-date", Paging: page})
	if err != nil || l.Total != 1 || l.Items[0].ID != posted || l.Items[0].Total != 7_690_000 || l.Items[0].SupplierName != "Công ty NCC1" {
		t.Fatalf("posted %+v %v", l, err)
	}
	if l, _ := f.purchase.Orders(f.staff, purchase.OrderFilter{Q: "ncc1", SupplierID: f.supplier, Sort: "-date", Paging: page}); l.Total != 2 {
		t.Fatalf("by supplier %+v", l)
	}
	items, err := f.purchase.LineItems(f.staff, "sua")
	if err != nil || len(items.Items) != 1 || items.Items[0].Tracking != "lot" {
		t.Fatalf("line items %+v %v", items, err)
	}
	if _, err := f.purchase.LineItems(f.ctxFor("viewer", "viewer@A"), ""); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("viewer line items: %v", err)
	}
}
