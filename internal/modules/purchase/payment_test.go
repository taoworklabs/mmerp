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
)

// payments: a posted order of team A with two posted invoices of its supplier, T1 and T2.
type payments struct {
	*fixture
	order, inv1, inv2 int64
	t1, t2            int64
}

func newPayments(t *testing.T) *payments {
	f := newFixture(t)
	p := &payments{fixture: f, order: f.postedOrder("2026-03-10")}
	p.inv1 = must(t)(f.invoice(f.staff, p.order, invoiceFields("2026-03-12", "1C26TAA", "1", il(1, "4", 50_000, "8"))))
	p.inv2 = must(t)(f.invoice(f.staff, p.order, invoiceFields("2026-03-13", "1C26TAA", "2", il(3, "1", 500_000, "10"))))
	f.check(f.move(f.staff, "purchase.invoice", p.inv1, record.Posted))
	f.check(f.move(f.staff, "purchase.invoice", p.inv2, record.Posted))
	p.t1, p.t2 = 216_000, 550_000
	return p
}

func alloc(inv, amount int64) purchase.AllocationInput {
	return purchase.AllocationInput{InvoiceID: inv, Amount: amount}
}

func (p *payments) fields(date, method string, allocs ...purchase.AllocationInput) purchase.PaymentFields {
	return purchase.PaymentFields{Date: date, OrgUnitID: p.a, SupplierID: p.supplier, Method: method, Allocations: allocs}
}

func (p *payments) pay(ctx context.Context, fl purchase.PaymentFields) (int64, error) {
	return p.purchase.CreatePayment(ctx, purchase.NewPayment{RequestID: newRequest(), PaymentFields: fl})
}

func TestPaymentRecordContract(t *testing.T) {
	recordtest.Run(t, func(t *testing.T) recordtest.Harness {
		p := newPayments(t)
		// The suite dates payments freely, so the invoice is older than any of them.
		if _, err := p.core.System.Exec(p.ctx, `UPDATE record.documents SET date = '2026-01-01' WHERE id = ANY($1)`, []int64{p.order, p.inv2}); err != nil {
			t.Fatal(err)
		}
		fields := func(date string) purchase.PaymentFields { return p.fields(date, "cash", alloc(p.inv2, 1)) }
		return recordtest.Harness{
			Core: p.core, Actor: p.staff, Admin: p.admin, LegalEntity: p.c, Table: "purchase.payments",
			Get: func(ctx context.Context, r record.Ref) (any, error) { return p.purchase.Payment(ctx, r.ID) },
			Create: func(ctx context.Context, date, requestID string) (record.Ref, error) {
				id, err := p.purchase.CreatePayment(ctx, purchase.NewPayment{RequestID: requestID, PaymentFields: fields(date)})
				return record.Ref{Type: "purchase.payment", ID: id}, err
			},
			Edit: func(ctx context.Context, r record.Ref, version int32, date string) error {
				return p.purchase.UpdatePayment(ctx, r.ID, purchase.PaymentUpdate{Version: version, PaymentFields: fields(date)})
			},
			RequireApproval: func(_ *testing.T, a record.Action) { p.managerApproves("purchase.payment", a) },
			Approve:         func(instance int64) error { return p.appr.Approve(p.manager, instance, 1) },
			Reject:          func(instance int64) error { return p.appr.Reject(p.manager, instance, 1, "không") },
			// An invoice cut to nothing leaves no room for the allocation.
			BreakPosting: func(t *testing.T, _ record.Ref) {
				if _, err := p.core.System.Exec(p.ctx, `UPDATE purchase.invoices SET subtotal = 0, discount_total = 0, vat_total = 0, total = 0 WHERE id = $1`, p.inv2); err != nil {
					t.Fatal(err)
				}
			},
			Effects:      func(_ *testing.T, r record.Ref) int { return min(len(p.paymentLines(r.ID)), 1) },
			PostingLines: func(_ *testing.T, r record.Ref) int { return len(p.paymentLines(r.ID)) },
		}
	})
}

func (p *payments) paymentLines(id int64) map[string]int64 {
	return p.postingLines("purchase.payment", id)
}

// Exit criterion 4: one payment settles all of T1 and part of T2.
func TestPaymentSplit(t *testing.T) {
	p := newPayments(t)
	id := must(t)(p.pay(p.staff, p.fields("2026-03-14", "bank", alloc(p.inv1, p.t1), alloc(p.inv2, 300_000))))
	pay, err := p.purchase.Payment(p.staff, id)
	if err != nil || pay.Amount != p.t1+300_000 || pay.SupplierName != "Công ty NCC1" || pay.Allocations[1].Remaining != p.t2 {
		t.Fatalf("payment %+v %v", pay, err)
	}
	p.check(p.move(p.staff, "purchase.payment", id, record.Posted))
	got := p.paymentLines(id)
	if len(got) != 2 || got["purchase_bank_payment"] != 516_000 || got["purchase_payable"] != -516_000 {
		t.Fatalf("lines %v", got)
	}
	i1, _ := p.purchase.Invoice(p.staff, p.inv1)
	i2, _ := p.purchase.Invoice(p.staff, p.inv2)
	if i1.PaymentState != "paid" || i2.PaymentState != "partial" || i2.Paid != 300_000 || i2.Remaining != 250_000 || len(i2.Payments) != 1 {
		t.Fatalf("states %+v / %+v", i1, i2)
	}
	o, _ := p.purchase.Order(p.staff, p.order)
	if o.Paid == nil || *o.Paid != 516_000 || o.PaymentState != "partial" {
		t.Fatalf("order %+v", o)
	}
	l, err := p.purchase.Invoices(p.staff, purchase.InvoiceFilter{PaymentState: "partial", Sort: "-date", Paging: page})
	if err != nil || l.Total != 1 || l.Items[0].ID != p.inv2 {
		t.Fatalf("partial invoices %+v %v", l, err)
	}
	pl, err := p.purchase.Payments(p.staff, purchase.PaymentFilter{Method: "bank", Sort: "-date", Paging: page})
	if err != nil || pl.Total != 1 || pl.Items[0].Amount != 516_000 {
		t.Fatalf("payments %+v %v", pl, err)
	}
	p.check(p.move(p.staff, "purchase.payment", id, record.Cancelled))
	if got := p.paymentLines(id); len(got) != 0 {
		t.Fatalf("after cancel %v", got)
	}
	if i1, _ := p.purchase.Invoice(p.staff, p.inv1); i1.PaymentState != "unpaid" {
		t.Fatalf("after cancel %q", i1.PaymentState)
	}
}

func TestPaymentCashLines(t *testing.T) {
	p := newPayments(t)
	id := must(t)(p.pay(p.staff, p.fields("2026-03-14", "cash", alloc(p.inv2, 1_000))))
	p.check(p.move(p.staff, "purchase.payment", id, record.Posted))
	if got := p.paymentLines(id); got["purchase_cash_payment"] != 1_000 || got["purchase_payable"] != -1_000 {
		t.Fatalf("lines %v", got)
	}
}

func TestPaymentOverInvoice(t *testing.T) {
	p := newPayments(t)
	if e, ok := errors.AsType[*platform.Error](must2(p.pay(p.staff, p.fields("2026-03-14", "cash", alloc(p.inv1, p.t1+1))))); !ok ||
		e.Code != "purchase_payment_over_invoice" || e.Params["remaining"] != p.t1 {
		t.Fatalf("over %v", e)
	}
}

func TestPaymentRules(t *testing.T) {
	p := newPayments(t)
	draft := must(t)(p.invoice(p.staff, p.order, invoiceFields("2026-03-13", "1C26TAA", "3", il(1, "1", 50_000, "8"))))
	wantCode(t, must2(p.pay(p.staff, p.fields("2026-03-14", "cash", alloc(draft, 1)))), "purchase_payment_invoice")
	wantCode(t, must2(p.pay(p.staff, p.fields("2026-03-12", "cash", alloc(p.inv2, 1)))), "purchase_payment_before_invoice")
	wantCode(t, must2(p.pay(p.staff, p.fields("2026-03-14", "cash", alloc(p.inv1, 1), alloc(p.inv1, 1)))), "purchase_payment_duplicate_invoice")
	// Another supplier of team A.
	other := must(t)(p.purchase.CreateSupplier(p.staff, supplier("NCC9", p.a)))
	fl := p.fields("2026-03-14", "cash", alloc(p.inv1, 1))
	fl.SupplierID = other
	wantCode(t, must2(p.pay(p.staff, fl)), "purchase_payment_invoice")
	// The payment at the company, the invoice at team A.
	everywhere := p.ctxFor("staff_all", "staff@*")
	fl = p.fields("2026-03-14", "cash", alloc(p.inv1, 1))
	fl.OrgUnitID = p.c
	wantCode(t, must2(p.pay(everywhere, fl)), "purchase_payment_invoice_org_unit")
}

func TestPaymentConcurrentAllocations(t *testing.T) {
	p := newPayments(t)
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range 2 {
		wg.Go(func() { _, errs[i] = p.pay(p.staff, p.fields("2026-03-14", "cash", alloc(p.inv1, p.t1))) })
	}
	wg.Wait()
	if (errs[0] == nil) == (errs[1] == nil) {
		t.Fatalf("both or neither: %v %v", errs[0], errs[1])
	}
	for _, err := range errs {
		if err != nil {
			wantCode(t, err, "purchase_payment_over_invoice")
		}
	}
}

// An inactive supplier is still paid what it is owed.
func TestPaymentInactiveSupplier(t *testing.T) {
	p := newPayments(t)
	p.deactivateSupplier()
	id := must(t)(p.pay(p.staff, p.fields("2026-03-14", "cash", alloc(p.inv1, p.t1))))
	p.check(p.move(p.staff, "purchase.payment", id, record.Posted))
}

func TestCancelChain(t *testing.T) {
	p := newPayments(t)
	id := must(t)(p.pay(p.staff, p.fields("2026-03-14", "cash", alloc(p.inv1, 1_000))))
	p.check(p.move(p.staff, "purchase.payment", id, record.Posted))
	wantCode(t, p.move(p.staff, "purchase.invoice", p.inv1, record.Cancelled), "purchase_invoice_has_payment")
	p.check(p.move(p.staff, "purchase.payment", id, record.Cancelled))
	p.check(p.move(p.staff, "purchase.invoice", p.inv1, record.Cancelled))
	p.check(p.move(p.staff, "purchase.invoice", p.inv2, record.Cancelled))
	p.check(p.move(p.staff, "purchase.order", p.order, record.Cancelled))
}

// A draft payment holds its amount back from create_payment and open invoices; the invoice lists it.
func TestPaymentDraftHoldsAmount(t *testing.T) {
	p := newPayments(t)
	i1, _ := p.purchase.Invoice(p.staff, p.inv1)
	if !contains(i1.AllowedActions, "create_payment") {
		t.Fatalf("actions %v", i1.AllowedActions)
	}
	id := must(t)(p.pay(p.staff, p.fields("2026-03-14", "cash", alloc(p.inv1, p.t1))))
	i1, _ = p.purchase.Invoice(p.staff, p.inv1)
	if contains(i1.AllowedActions, "create_payment") || i1.PaymentState != "unpaid" || len(i1.Payments) != 1 || i1.Payments[0].ID != id {
		t.Fatalf("after a draft %+v", i1)
	}
	open, err := p.purchase.OpenInvoices(p.staff, p.supplier, p.a, 0)
	if err != nil || len(open) != 1 || open[0].InvoiceID != p.inv2 || open[0].Remaining != p.t2 {
		t.Fatalf("open %+v %v", open, err)
	}
	if open, _ := p.purchase.OpenInvoices(p.staff, p.supplier, p.a, id); len(open) != 2 {
		t.Fatalf("open for the draft itself %+v", open)
	}
}

func TestOrderPaymentState(t *testing.T) {
	p := newPayments(t)
	id := must(t)(p.pay(p.staff, p.fields("2026-03-14", "cash", alloc(p.inv1, p.t1), alloc(p.inv2, p.t2))))
	p.check(p.move(p.staff, "purchase.payment", id, record.Posted))
	// Every invoice paid, but the order is not fully invoiced yet.
	if o, _ := p.purchase.Order(p.staff, p.order); o.PaymentState != "partial" || *o.Paid != p.t1+p.t2 {
		t.Fatalf("partly invoiced %+v", o)
	}
	rest := must(t)(p.invoice(p.staff, p.order, invoiceFields("2026-03-15", "1C26TAA", "4", il(1, "6", 50_000, "8"), il(2, "2", 3_000_000, "10"))))
	p.check(p.move(p.staff, "purchase.invoice", rest, record.Posted))
	r, _ := p.purchase.Invoice(p.staff, rest)
	id2 := must(t)(p.pay(p.staff, p.fields("2026-03-16", "bank", alloc(rest, r.Totals.Total))))
	p.check(p.move(p.staff, "purchase.payment", id2, record.Posted))
	if o, _ := p.purchase.Order(p.staff, p.order); o.PaymentState != "paid" || o.InvoiceState != "full" {
		t.Fatalf("all paid %+v", o)
	}
	l, _ := p.purchase.Orders(p.staff, purchase.OrderFilter{PaymentState: "paid", Sort: "-date", Paging: page})
	if l.Total != 1 || l.Items[0].PaymentState != "paid" {
		t.Fatalf("list %+v", l)
	}
}

func TestPaymentTeamBoundary(t *testing.T) {
	p := newPayments(t)
	id := must(t)(p.pay(p.staff, p.fields("2026-03-14", "cash", alloc(p.inv1, 1))))
	staffB := p.ctxFor("staff_b", "staff@B")
	if _, err := p.purchase.Payment(staffB, id); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("B opens A's payment: %v", err)
	}
	if l, _ := p.purchase.Payments(staffB, purchase.PaymentFilter{Sort: "-date", Paging: page}); l.Total != 0 {
		t.Fatalf("B lists %+v", l)
	}
}
