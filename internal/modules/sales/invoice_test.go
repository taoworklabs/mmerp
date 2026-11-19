package sales_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/record/recordtest"
	"github.com/taoworklabs/mmerp/internal/core/setting"
	"github.com/taoworklabs/mmerp/internal/modules/sales"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// invoices reuses the delivery fixture: a posted order of team A with 10 goods and 5 of the
// lot item at 1 000 đ and 8 %, and 2 hours of repair at 250 000 đ and 10 %.
type invoices struct{ *deliveries }

func newInvoices(t *testing.T) *invoices { return &invoices{newDeliveries(t)} }

func il(position int, qty string) sales.InvoiceLineInput {
	return sales.InvoiceLineInput{OrderPosition: position, Quantity: qty}
}

func (v *invoices) invoiceFields(lines ...sales.InvoiceLineInput) sales.InvoiceFields {
	return sales.InvoiceFields{Date: "2026-03-12", Lines: lines}
}

func (v *invoices) createInvoice(ctx context.Context, order int64, f sales.InvoiceFields) (int64, error) {
	return v.sales.CreateInvoice(ctx, sales.NewInvoice{RequestID: newRequest(), OrderID: order, InvoiceFields: f})
}

// postedInvoice creates and posts an invoice of the fixture's order.
func (v *invoices) postedInvoice(lines ...sales.InvoiceLineInput) int64 {
	v.t.Helper()
	id := must(v.t)(v.createInvoice(v.staff, v.order, v.invoiceFields(lines...)))
	v.check(v.move(v.staff, "sales.invoice", id, record.Posted))
	return id
}

type postingLine struct {
	kind    string
	unit    int64
	partner *int64
	amount  int64
}

// liveLines are the document's posting lines not voided, by kind.
func (f *fixture) liveLines(docType string, id int64) map[string]postingLine {
	f.t.Helper()
	rows, err := f.system.Query(f.admin, `SELECT kind, org_unit_id, partner_id, amount FROM posting.lines
		WHERE doc_type = $1 AND doc_id = $2 AND voided_at IS NULL`, docType, id)
	f.check(err)
	defer rows.Close()
	out := map[string]postingLine{}
	for rows.Next() {
		var l postingLine
		f.check(rows.Scan(&l.kind, &l.unit, &l.partner, &l.amount))
		out[l.kind] = l
	}
	f.check(rows.Err())
	return out
}

func (f *fixture) customerPartner() int64 {
	f.t.Helper()
	c, err := f.sales.Customer(f.staff, f.customer)
	f.check(err)
	return c.PartnerID
}

func TestInvoiceRecordContract(t *testing.T) {
	recordtest.Run(t, func(t *testing.T) recordtest.Harness {
		v := newInvoices(t)
		order := v.orderOn("2026-01-01", "0")
		v.check(v.move(v.staff, "sales.order", order, record.Posted))
		fields := func(date string) sales.InvoiceFields {
			return sales.InvoiceFields{Date: date, Lines: []sales.InvoiceLineInput{il(1, "1")}}
		}
		count := func(_ *testing.T, r record.Ref) int { return len(v.liveLines(r.Type, r.ID)) }
		return recordtest.Harness{
			Core: v.core, Actor: v.staff, Admin: v.admin, LegalEntity: v.c, Table: "sales.invoices",
			Get: func(ctx context.Context, r record.Ref) (any, error) { return v.sales.Invoice(ctx, r.ID) },
			Create: func(ctx context.Context, date, requestID string) (record.Ref, error) {
				id, err := v.sales.CreateInvoice(ctx, sales.NewInvoice{RequestID: requestID, OrderID: order, InvoiceFields: fields(date)})
				return record.Ref{Type: "sales.invoice", ID: id}, err
			},
			Edit: func(ctx context.Context, r record.Ref, version int32, date string) error {
				return v.sales.UpdateInvoice(ctx, r.ID, sales.InvoiceUpdate{Version: version, InvoiceFields: fields(date)})
			},
			RequireApproval: func(t *testing.T, a record.Action) {
				v.check(v.appr.SaveRule(v.admin, "sales.invoice", a, approval.RuleInput{
					Flow:      approval.Flow{{Step: &approval.StepDef{Approver: approval.Approver{Kind: "role", Product: "sales", Role: "manager"}}}},
					MaxLevels: 3, FallbackProduct: "sales", FallbackRole: "manager",
				}))
			},
			Approve: func(instance int64) error { return v.appr.Approve(v.manager, instance, 1) },
			Reject:  func(instance int64) error { return v.appr.Reject(v.manager, instance, 1, "không") },
			// An order line cut below what the invoice takes leaves it over ordered.
			BreakPosting: func(*testing.T, record.Ref) {
				_, err := v.system.Exec(v.admin, `UPDATE sales.lines SET quantity = 0.001 WHERE doc_id = $1 AND position = 1`, order)
				v.check(err)
			},
			Effects:      func(t *testing.T, r record.Ref) int { return min(count(t, r), 1) },
			PostingLines: count,
			Gaps: map[string]string{
				"write actions leave allowed_actions when the product is off or the period locked": "a posted invoice in a locked period still offers create_receipt, dated in an open period",
			},
		}
	})
}

// Two invoices bill parts of the order; a third beyond what is left is refused.
func TestInvoicePartial(t *testing.T) {
	v := newInvoices(t)
	first := v.postedInvoice(il(1, "4"), il(3, "1"))
	inv, err := v.sales.Invoice(v.staff, first)
	v.check(err)
	if inv.Totals != (sales.Totals{Subtotal: 254_000, VatTotal: 25_320, Total: 279_320}) {
		t.Fatalf("totals %+v", inv.Totals)
	}
	if inv.Customer.Code != "KH1" || inv.OrderID != v.order || len(inv.Lines) != 2 || inv.Lines[0].Vat != 320 || inv.Lines[1].Amount != 250_000 {
		t.Fatalf("invoice %+v", inv)
	}
	err = must2(v.createInvoice(v.staff, v.order, v.invoiceFields(il(1, "7"))))
	wantCode(t, err, "sales_invoice_over_ordered")
	if e, _ := errors.AsType[*platform.Error](err); e.Params["remaining"] != "6.000" || e.Params["line_no"] != 1 {
		t.Fatalf("params %+v", e.Params)
	}
	v.postedInvoice(il(1, "6"), il(3, "1"))
}

// Price, discount and VAT rate are the order line's, whatever the catalogue says now.
func TestInvoicePricesFromOrder(t *testing.T) {
	v := newInvoices(t)
	_, err := v.system.Exec(v.admin, `UPDATE item.items SET price = 9999, vat_rate = '10' WHERE id = $1`, v.goods)
	v.check(err)
	inv, err := v.sales.Invoice(v.staff, must(t)(v.createInvoice(v.staff, v.order, v.invoiceFields(il(1, "1")))))
	v.check(err)
	if l := inv.Lines[0]; l.UnitPrice != 1_000 || l.VatRate != "8" || l.DiscountPercent != "0" || l.ItemCode != "HANG" {
		t.Fatalf("line %+v", l)
	}
}

func TestInvoiceBeforeOrder(t *testing.T) {
	v := newInvoices(t)
	f := v.invoiceFields(il(1, "1"))
	f.Date = "2026-03-09"
	wantCode(t, must2(v.createInvoice(v.staff, v.order, f)), "sales_invoice_before_order")
}

func TestInvoiceOrderNotPosted(t *testing.T) {
	v := newInvoices(t)
	draft := v.newOrder("0")
	wantCode(t, must2(v.createInvoice(v.staff, draft, v.invoiceFields(il(1, "1")))), "sales_order_not_posted")
}

func TestInvoicePositionAndQuantity(t *testing.T) {
	v := newInvoices(t)
	wantCode(t, must2(v.createInvoice(v.staff, v.order, v.invoiceFields(il(9, "1")))), "sales_invoice_position")
	wantCode(t, must2(v.createInvoice(v.staff, v.order, v.invoiceFields(il(1, "0")))), "invalid_quantity")
}

// An order whose cancelling waits for approval takes no new invoice.
func TestInvoiceRefusedWhileOrderCancelHeld(t *testing.T) {
	v := newInvoices(t)
	v.managerApproves("sales.order", record.Cancel)
	order := v.newOrder("15")
	v.check(v.move(v.staff, "sales.order", order, record.Posted))
	v.check(v.move(v.staff, "sales.order", order, record.Cancelled))
	wantCode(t, must2(v.createInvoice(v.staff, order, v.invoiceFields(il(1, "1")))), "awaiting_approval")
}

// Posting writes the receivable with the customer, revenue and VAT at the order's team;
// cancelling voids them. No VAT, no VAT line.
func TestInvoicePostingLines(t *testing.T) {
	v := newInvoices(t)
	id := v.postedInvoice(il(1, "4"), il(3, "1"))
	got := v.liveLines("sales.invoice", id)
	p := v.customerPartner()
	r := got["sales_receivable"]
	if r.amount != 279_320 || r.partner == nil || *r.partner != p || r.unit != v.a {
		t.Fatalf("receivable %+v", r)
	}
	if got["sales_revenue"].amount != 254_000 || got["sales_revenue"].partner != nil || got["sales_output_vat"].amount != 25_320 || len(got) != 3 {
		t.Fatalf("lines %+v", got)
	}
	v.check(v.move(v.staff, "sales.invoice", id, record.Cancelled))
	if got := v.liveLines("sales.invoice", id); len(got) != 0 {
		t.Fatalf("not voided: %+v", got)
	}
	_, err := v.system.Exec(v.admin, `UPDATE sales.lines SET vat_rate = 'none' WHERE doc_id = $1`, v.order)
	v.check(err)
	if got := v.liveLines("sales.invoice", v.postedInvoice(il(1, "1"))); len(got) != 2 || got["sales_revenue"].amount != 1_000 {
		t.Fatalf("no-VAT lines %+v", got)
	}
}

// The fixture's posting has no accounting behind it: invoices post all the same.
func TestInvoicePostsWithoutAccounting(t *testing.T) {
	v := newInvoices(t)
	id := v.postedInvoice(il(3, "2"))
	if inv, err := v.sales.Invoice(v.staff, id); err != nil || inv.Status != "posted" {
		t.Fatalf("%+v %v", inv, err)
	}
}

func TestOrderCancelWithInvoice(t *testing.T) {
	v := newInvoices(t)
	id := must(t)(v.createInvoice(v.staff, v.order, v.invoiceFields(il(3, "1"))))
	wantCode(t, v.move(v.staff, "sales.order", v.order, record.Cancelled), "sales_order_has_invoice")
	v.check(v.rec.Delete(v.staff, record.Ref{Type: "sales.invoice", ID: id}, 1))
	v.check(v.move(v.staff, "sales.order", v.order, record.Cancelled))
}

func TestInvoiceTeamBoundary(t *testing.T) {
	v := newInvoices(t)
	id := v.postedInvoice(il(3, "1"))
	b := v.ctxFor("staff_b", "staff@B")
	if _, err := v.sales.Invoice(b, id); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("staff of B opens: %v", err)
	}
	if _, err := v.sales.InvoiceOptions(b, v.order); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("staff of B reads options: %v", err)
	}
	if _, err := v.createInvoice(b, v.order, v.invoiceFields(il(3, "1"))); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("staff of B invoices: %v", err)
	}
}

// Options give each order line with what non-cancelled invoices leave.
func TestInvoiceOptions(t *testing.T) {
	v := newInvoices(t)
	must(t)(v.createInvoice(v.staff, v.order, v.invoiceFields(il(1, "4"))))
	o, err := v.sales.InvoiceOptions(v.staff, v.order)
	v.check(err)
	if len(o.Lines) != 3 || o.Lines[0].Remaining != "6.000" || o.Lines[0].Ordered != "10.000" || o.Lines[2].Remaining != "2.000" {
		t.Fatalf("options %+v", o.Lines)
	}
}

func (v *invoices) orderState() sales.Doc {
	v.t.Helper()
	o, err := v.sales.Order(v.staff, v.order)
	v.check(err)
	return o
}

// Invoice state counts posted invoices over all lines; create_invoice disappears once
// non-cancelled invoices, drafts included, take everything.
func TestOrderInvoiceState(t *testing.T) {
	v := newInvoices(t)
	if o := v.orderState(); o.InvoiceState != "none" || !slices.Contains(o.AllowedActions, "create_invoice") {
		t.Fatalf("fresh order %s %v", o.InvoiceState, o.AllowedActions)
	}
	v.postedInvoice(il(1, "4"))
	o := v.orderState()
	if o.InvoiceState != "partial" || o.Lines[0].Invoiced == nil || *o.Lines[0].Invoiced != "4" || len(o.Invoices) != 1 {
		t.Fatalf("after one invoice %+v %v", o.InvoiceState, o.Lines[0].Invoiced)
	}
	rest := must(t)(v.createInvoice(v.staff, v.order, v.invoiceFields(il(1, "6"), il(2, "5"), il(3, "2"))))
	if o := v.orderState(); o.InvoiceState != "partial" || slices.Contains(o.AllowedActions, "create_invoice") {
		t.Fatalf("draft takes the rest: %s %v", o.InvoiceState, o.AllowedActions)
	}
	v.check(v.move(v.staff, "sales.invoice", rest, record.Posted))
	if o := v.orderState(); o.InvoiceState != "full" || len(o.Invoices) != 2 {
		t.Fatalf("all invoiced: %s", o.InvoiceState)
	}
	if o, _ := v.sales.Order(v.ctxFor("viewer", "viewer@A"), v.order); slices.Contains(o.AllowedActions, "create_invoice") {
		t.Fatal("viewer offered create_invoice")
	}
}

func TestInvoiceList(t *testing.T) {
	v := newInvoices(t)
	posted := v.postedInvoice(il(1, "4"))
	draft := must(t)(v.createInvoice(v.staff, v.order, v.invoiceFields(il(3, "1"))))
	all := sales.InvoiceFilter{Sort: "-date", Paging: platform.Paging{Page: 1, PageSize: 50}}
	l, err := v.sales.Invoices(v.staff, all)
	v.check(err)
	if l.Total != 2 || l.Items[0].CustomerName != "Công ty KH1" || l.Items[0].OrderID != v.order {
		t.Fatalf("list %+v", l)
	}
	f := all
	f.Status = "posted"
	if l, _ := v.sales.Invoices(v.staff, f); l.Total != 1 || l.Items[0].ID != posted || l.Items[0].Total != 4_320 {
		t.Fatalf("posted %+v", l)
	}
	f = all
	d, err := v.sales.Invoice(v.staff, draft)
	v.check(err)
	f.Q = d.Number
	if l, _ := v.sales.Invoices(v.staff, f); l.Total != 1 || l.Items[0].ID != draft {
		t.Fatalf("by number %+v", l)
	}
	if l, _ := v.sales.Invoices(v.ctxFor("staff_b", "staff@B"), all); l.Total != 0 {
		t.Fatalf("staff of B: %+v", l)
	}
}

// Sending the same create again, after the first billed all that was left, returns the first invoice.
func TestInvoiceRetryReturnsFirst(t *testing.T) {
	v := newInvoices(t)
	in := sales.NewInvoice{RequestID: newRequest(), OrderID: v.order, InvoiceFields: v.invoiceFields(il(1, "10"), il(2, "5"), il(3, "2"))}
	first := must(t)(v.sales.CreateInvoice(v.staff, in))
	if again := must(t)(v.sales.CreateInvoice(v.staff, in)); again != first {
		t.Fatalf("retry made %d, first %d", again, first)
	}
}

// Posting writes what the invoice stored, even when the rounding setting changed since the save.
func TestInvoicePostsStoredTotals(t *testing.T) {
	v := newInvoices(t)
	// Two lines of 1 006 đ at 8 %: line rounding gives 80 + 80, total rounding 161.
	id := must(t)(v.createInvoice(v.staff, v.order, v.invoiceFields(il(1, "1.006"), il(2, "1.006"))))
	v.check(v.set.SetFor(v.admin, v.c, setting.Rounding, "total"))
	v.check(v.move(v.staff, "sales.invoice", id, record.Posted))
	inv, err := v.sales.Invoice(v.staff, id)
	v.check(err)
	got := v.liveLines("sales.invoice", id)
	if inv.Totals.Total != 2_172 || got["sales_receivable"].amount != inv.Totals.Total || got["sales_output_vat"].amount != inv.Totals.VatTotal {
		t.Fatalf("stored %+v, lines %+v", inv.Totals, got)
	}
}
