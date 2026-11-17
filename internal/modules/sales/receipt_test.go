package sales_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/record/recordtest"
	"github.com/taoworklabs/mmerp/internal/modules/sales"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// receipts adds two posted invoices of the fixture's order to the invoice fixture:
// inv1 of 4 goods (total 4 320) and inv2 of one hour of repair (total 275 000).
type receipts struct {
	*invoices
	inv1, inv2 int64
}

const total1, total2 = 4_320, 275_000

func newReceipts(t *testing.T) *receipts {
	t.Helper()
	v := newInvoices(t)
	return &receipts{invoices: v, inv1: v.postedInvoice(il(1, "4")), inv2: v.postedInvoice(il(3, "1"))}
}

func al(invoice, amount int64) sales.AllocationInput {
	return sales.AllocationInput{InvoiceID: invoice, Amount: amount}
}

func (r *receipts) receiptFields(method string, allocs ...sales.AllocationInput) sales.ReceiptFields {
	return sales.ReceiptFields{Date: "2026-03-15", OrgUnitID: r.a, CustomerID: r.customer, Method: method, Allocations: allocs}
}

func (r *receipts) createReceipt(ctx context.Context, f sales.ReceiptFields) (int64, error) {
	return r.sales.CreateReceipt(ctx, sales.NewReceipt{RequestID: newRequest(), ReceiptFields: f})
}

func (r *receipts) postedReceipt(f sales.ReceiptFields) int64 {
	r.t.Helper()
	id := must(r.t)(r.createReceipt(r.staff, f))
	r.check(r.move(r.staff, "sales.receipt", id, record.Posted))
	return id
}

func TestReceiptRecordContract(t *testing.T) {
	recordtest.Run(t, func(t *testing.T) recordtest.Harness {
		v := newInvoices(t)
		order := v.orderOn("2026-01-01", "0")
		v.check(v.move(v.staff, "sales.order", order, record.Posted))
		f := v.invoiceFields(il(3, "2"))
		f.Date = "2026-01-01"
		inv := must(t)(v.createInvoice(v.staff, order, f))
		v.check(v.move(v.staff, "sales.invoice", inv, record.Posted))
		r := &receipts{invoices: v}
		fields := func(date string) sales.ReceiptFields {
			f := r.receiptFields("cash", al(inv, 1))
			f.Date = date
			return f
		}
		count := func(_ *testing.T, ref record.Ref) int { return len(v.liveLines(ref.Type, ref.ID)) }
		return recordtest.Harness{
			Core: v.core, Actor: v.staff, Admin: v.admin, LegalEntity: v.c, Table: "sales.receipts",
			Get: func(ctx context.Context, ref record.Ref) (any, error) { return v.sales.Receipt(ctx, ref.ID) },
			Create: func(ctx context.Context, date, requestID string) (record.Ref, error) {
				id, err := v.sales.CreateReceipt(ctx, sales.NewReceipt{RequestID: requestID, ReceiptFields: fields(date)})
				return record.Ref{Type: "sales.receipt", ID: id}, err
			},
			Edit: func(ctx context.Context, ref record.Ref, version int32, date string) error {
				return v.sales.UpdateReceipt(ctx, ref.ID, sales.ReceiptUpdate{Version: version, ReceiptFields: fields(date)})
			},
			RequireApproval: func(t *testing.T, a record.Action) {
				v.check(v.appr.SaveRule(v.admin, "sales.receipt", a, approval.RuleInput{
					Flow:      approval.Flow{{Step: &approval.StepDef{Approver: approval.Approver{Kind: "role", Product: "sales", Role: "manager"}}}},
					MaxLevels: 3, FallbackProduct: "sales", FallbackRole: "manager",
				}))
			},
			Approve: func(instance int64) error { return v.appr.Approve(v.manager, instance, 1) },
			Reject:  func(instance int64) error { return v.appr.Reject(v.manager, instance, 1, "không") },
			// An invoice cut to nothing leaves no room for the allocation.
			BreakPosting: func(*testing.T, record.Ref) {
				_, err := v.system.Exec(v.admin, `UPDATE sales.invoices SET total = 0 WHERE id = $1`, inv)
				v.check(err)
			},
			// A deactivated customer keeps the receipt a draft.
			BreakSubmit: func(*testing.T, record.Ref) {
				c := customer("KH1", v.a)
				c.Active = false
				v.check(v.sales.UpdateCustomer(v.staff, v.customer, upd(c)))
			},
			Effects:      func(t *testing.T, ref record.Ref) int { return min(count(t, ref), 1) },
			PostingLines: count,
		}
	})
}

// One receipt settles all of one invoice and half of another.
func TestReceiptSplit(t *testing.T) {
	r := newReceipts(t)
	id := r.postedReceipt(r.receiptFields("bank", al(r.inv1, total1), al(r.inv2, total2/2)))
	got, err := r.sales.Receipt(r.staff, id)
	r.check(err)
	if got.Amount != total1+total2/2 || got.Method != "bank" || len(got.Allocations) != 2 || got.CustomerName != "Công ty KH1" {
		t.Fatalf("receipt %+v", got)
	}
	if a := got.Allocations[1]; a.InvoiceID != r.inv2 || a.InvoiceTotal != total2 || a.InvoiceNumber == "" {
		t.Fatalf("allocation %+v", a)
	}
}

func TestReceiptOverInvoice(t *testing.T) {
	r := newReceipts(t)
	err := must2(r.createReceipt(r.staff, r.receiptFields("cash", al(r.inv2, 1), al(r.inv1, total1+1))))
	wantCode(t, err, "sales_receipt_over_invoice")
	if e, _ := errors.AsType[*platform.Error](err); e.Params["remaining"] != int64(total1) || e.Params["line_no"] != 2 {
		t.Fatalf("params %+v", e.Params)
	}
	// Another receipt, even a draft, holds its amount back.
	must(t)(r.createReceipt(r.staff, r.receiptFields("cash", al(r.inv1, 4_000))))
	wantCode(t, must2(r.createReceipt(r.staff, r.receiptFields("cash", al(r.inv1, 321)))), "sales_receipt_over_invoice")
	must(t)(r.createReceipt(r.staff, r.receiptFields("cash", al(r.inv1, 320))))
}

func TestReceiptRules(t *testing.T) {
	r := newReceipts(t)
	other := must(t)(r.sales.CreateCustomer(r.staff, customer("KH2", r.a)))
	f := r.receiptFields("cash", al(r.inv1, 1))
	f.CustomerID = other
	wantCode(t, must2(r.createReceipt(r.staff, f)), "sales_receipt_invoice")

	draft := must(t)(r.createInvoice(r.staff, r.order, r.invoiceFields(il(1, "1"))))
	wantCode(t, must2(r.createReceipt(r.staff, r.receiptFields("cash", al(draft, 1)))), "sales_receipt_invoice")

	f = r.receiptFields("cash", al(r.inv1, 1))
	f.Date = "2026-03-11"
	wantCode(t, must2(r.createReceipt(r.staff, f)), "sales_receipt_before_invoice")

	wantCode(t, must2(r.createReceipt(r.staff, r.receiptFields("cash", al(r.inv1, 1), al(r.inv1, 2)))), "sales_receipt_duplicate_invoice")

	both := r.ctxFor("staff_ab", "staff@A", "staff@B")
	f = r.receiptFields("cash", al(r.inv1, 1))
	f.OrgUnitID = r.b
	wantCode(t, must2(r.createReceipt(both, f)), "sales_receipt_invoice_org_unit")

	if _, err := r.createReceipt(r.ctxFor("staff_b", "staff@B"), r.receiptFields("cash", al(r.inv1, 1))); !errors.Is(err, platform.ErrNotFound) &&
		!errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("staff of B: %v", err)
	}
}

// Cash and bank go to their own kinds; the receivable falls by the amount, with the customer.
func TestReceiptPostingLines(t *testing.T) {
	r := newReceipts(t)
	cash := r.postedReceipt(r.receiptFields("cash", al(r.inv1, total1)))
	got := r.liveLines("sales.receipt", cash)
	p := r.customerPartner()
	rv := got["sales_receivable"]
	if len(got) != 2 || got["sales_cash_receipt"].amount != total1 || got["sales_cash_receipt"].unit != r.a ||
		rv.amount != -total1 || rv.partner == nil || *rv.partner != p || rv.unit != r.a {
		t.Fatalf("cash lines %+v", got)
	}
	bank := r.postedReceipt(r.receiptFields("bank", al(r.inv2, 1_000)))
	if got := r.liveLines("sales.receipt", bank); got["sales_bank_receipt"].amount != 1_000 || got["sales_receivable"].amount != -1_000 {
		t.Fatalf("bank lines %+v", got)
	}
	r.check(r.move(r.staff, "sales.receipt", cash, record.Cancelled))
	if got := r.liveLines("sales.receipt", cash); len(got) != 0 {
		t.Fatalf("not voided %+v", got)
	}
}

// Two receipts for all of one invoice at once: exactly one gets it.
func TestReceiptConcurrentAllocations(t *testing.T) {
	r := newReceipts(t)
	errs := r.race(2, func(ctx context.Context, _ int) error {
		_, err := r.createReceipt(ctx, r.receiptFields("cash", al(r.inv1, total1)))
		return err
	})
	ok, over := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, sales.ErrReceiptOverInvoice):
			over++
		default:
			t.Fatal(err)
		}
	}
	if ok != 1 || over != 1 {
		t.Fatalf("ok %d, over %d", ok, over)
	}
}

// Cancelling runs backwards: receipt, then invoice, then order.
func TestCancelChain(t *testing.T) {
	r := newReceipts(t)
	id := r.postedReceipt(r.receiptFields("cash", al(r.inv1, 100)))
	wantCode(t, r.move(r.staff, "sales.invoice", r.inv1, record.Cancelled), "sales_invoice_has_receipt")
	wantCode(t, r.move(r.staff, "sales.order", r.order, record.Cancelled), "sales_order_has_invoice")
	r.check(r.move(r.staff, "sales.receipt", id, record.Cancelled))
	r.check(r.move(r.staff, "sales.invoice", r.inv1, record.Cancelled))
	r.check(r.move(r.staff, "sales.invoice", r.inv2, record.Cancelled))
	r.check(r.move(r.staff, "sales.order", r.order, record.Cancelled))
}

func TestReceiptTeamBoundary(t *testing.T) {
	r := newReceipts(t)
	id := r.postedReceipt(r.receiptFields("cash", al(r.inv1, 100)))
	if _, err := r.sales.Receipt(r.ctxFor("staff_b", "staff@B"), id); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("staff of B opens: %v", err)
	}
}

func (r *receipts) invoice(id int64) sales.Invoice {
	r.t.Helper()
	v, err := r.sales.Invoice(r.staff, id)
	r.check(err)
	return v
}

func TestInvoicePaymentState(t *testing.T) {
	r := newReceipts(t)
	if v := r.invoice(r.inv1); v.PaymentState != "unpaid" || v.Paid != 0 || v.Remaining != total1 || !slices.Contains(v.AllowedActions, "create_receipt") {
		t.Fatalf("fresh %+v", v)
	}
	first := r.postedReceipt(r.receiptFields("cash", al(r.inv1, 1_000)))
	if v := r.invoice(r.inv1); v.PaymentState != "partial" || v.Paid != 1_000 || v.Remaining != total1-1_000 {
		t.Fatalf("partial %+v", v)
	}
	r.postedReceipt(r.receiptFields("cash", al(r.inv1, total1-1_000)))
	if v := r.invoice(r.inv1); v.PaymentState != "paid" || v.Remaining != 0 || slices.Contains(v.AllowedActions, "create_receipt") {
		t.Fatalf("paid %+v", v)
	}
	r.check(r.move(r.staff, "sales.receipt", first, record.Cancelled))
	if v := r.invoice(r.inv1); v.PaymentState != "partial" || v.Paid != total1-1_000 {
		t.Fatalf("after cancelling %+v", v)
	}
	f := sales.InvoiceFilter{PaymentState: "partial", Sort: "-date", Paging: platform.Paging{Page: 1, PageSize: 50}}
	if l, err := r.sales.Invoices(r.staff, f); err != nil || l.Total != 1 || l.Items[0].ID != r.inv1 || l.Items[0].PaymentState != "partial" {
		t.Fatalf("list %+v %v", l, err)
	}
}

// A draft receipt holds its amount back: nothing is paid yet, but nothing is left to receive.
func TestReceiptDraftHoldsAmount(t *testing.T) {
	r := newReceipts(t)
	draft := must(t)(r.createReceipt(r.staff, r.receiptFields("cash", al(r.inv1, total1))))
	v := r.invoice(r.inv1)
	if v.PaymentState != "unpaid" || slices.Contains(v.AllowedActions, "create_receipt") || len(v.Receipts) != 1 ||
		v.Receipts[0].ID != draft || v.Receipts[0].Status != "draft" || v.Receipts[0].Amount != total1 {
		t.Fatalf("invoice %+v", v)
	}
	open, err := r.sales.OpenInvoices(r.staff, r.customer, r.a, 0)
	r.check(err)
	if len(open) != 1 || open[0].InvoiceID != r.inv2 || open[0].Amount != total2 {
		t.Fatalf("open %+v", open)
	}
	// The draft being edited sees its own amount as still open.
	if open, _ := r.sales.OpenInvoices(r.staff, r.customer, r.a, draft); len(open) != 2 {
		t.Fatalf("open for the draft %+v", open)
	}
}

func TestOrderPaymentState(t *testing.T) {
	r := newReceipts(t)
	if o := r.orderState(); o.PaymentState != "unpaid" || o.Paid == nil || *o.Paid != 0 {
		t.Fatalf("fresh %+v", o.PaymentState)
	}
	r.postedReceipt(r.receiptFields("bank", al(r.inv1, total1), al(r.inv2, total2)))
	if o := r.orderState(); o.PaymentState != "partial" || *o.Paid != total1+total2 {
		t.Fatalf("invoices paid, order not all invoiced: %s %d", o.PaymentState, *o.Paid)
	}
	last := r.postedInvoice(il(1, "6"), il(2, "5"), il(3, "1"))
	r.postedReceipt(r.receiptFields("cash", al(last, r.invoice(last).Totals.Total)))
	if o := r.orderState(); o.PaymentState != "paid" {
		t.Fatalf("all paid: %s", o.PaymentState)
	}
}

func TestReceiptList(t *testing.T) {
	r := newReceipts(t)
	cash := r.postedReceipt(r.receiptFields("cash", al(r.inv1, 100)))
	must(t)(r.createReceipt(r.staff, r.receiptFields("bank", al(r.inv2, 200))))
	all := sales.ReceiptFilter{Sort: "-date", Paging: platform.Paging{Page: 1, PageSize: 50}}
	if l, err := r.sales.Receipts(r.staff, all); err != nil || l.Total != 2 || l.Items[0].CustomerName != "Công ty KH1" {
		t.Fatalf("all %+v %v", l, err)
	}
	f := all
	f.Method = "cash"
	if l, _ := r.sales.Receipts(r.staff, f); l.Total != 1 || l.Items[0].ID != cash || l.Items[0].Amount != 100 {
		t.Fatalf("cash %+v", l)
	}
	f = all
	f.Status = "draft"
	f.CustomerID = r.customer
	if l, _ := r.sales.Receipts(r.staff, f); l.Total != 1 || l.Items[0].Method != "bank" {
		t.Fatalf("draft %+v", l)
	}
	if l, _ := r.sales.Receipts(r.ctxFor("staff_b", "staff@B"), all); l.Total != 0 {
		t.Fatalf("staff of B %+v", l)
	}
}
