package purchase_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/record/recordtest"
	"github.com/taoworklabs/mmerp/internal/modules/purchase"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// il bills qty of an order line at price, discount and VAT rate.
func il(pos int, qty string, price int64, vat string) purchase.InvoiceLineInput {
	return purchase.InvoiceLineInput{OrderPosition: pos, Quantity: qty, UnitPrice: price, DiscountPercent: "0", VatRate: vat}
}

func invoiceFields(date, symbol, no string, lines ...purchase.InvoiceLineInput) purchase.InvoiceFields {
	return purchase.InvoiceFields{Date: date, SupplierInvoiceSymbol: str(symbol), SupplierInvoiceNo: str(no), SupplierInvoiceDate: str(date), Lines: lines}
}

func (f *fixture) invoice(ctx context.Context, order int64, fl purchase.InvoiceFields) (int64, error) {
	return f.purchase.CreateInvoice(ctx, purchase.NewInvoice{RequestID: newRequest(), OrderID: order, InvoiceFields: fl})
}

// billed is the invoice of exit criterion 3: 4 cartons repriced to 52,000 đ, both purifiers, the transport.
func billed(date, no string) purchase.InvoiceFields {
	return invoiceFields(date, "1C26TAA", no, il(1, "4", 52_000, "8"), il(2, "2", 3_000_000, "10"), il(3, "1", 500_000, "10"))
}

// postingLines are ref's posting lines not voided, by kind, with whether they name a partner.
func (f *fixture) postingLines(docType string, id int64) map[string]int64 {
	f.t.Helper()
	rows, err := f.core.Pool.Query(f.ctx, `SELECT kind, amount, partner_id IS NOT NULL, org_unit_id FROM posting.lines
		WHERE doc_type = $1 AND doc_id = $2 AND voided_at IS NULL`, docType, id)
	f.check(err)
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var kind string
		var amount, unit int64
		var partner bool
		f.check(rows.Scan(&kind, &amount, &partner, &unit))
		if partner != (kind == "purchase_payable") || unit != f.a {
			f.t.Fatalf("%s: partner %v, unit %d", kind, partner, unit)
		}
		out[kind] = amount
	}
	f.check(rows.Err())
	return out
}

func TestSupplierInvoiceRecordContract(t *testing.T) {
	recordtest.Run(t, func(t *testing.T) recordtest.Harness {
		f := newFixture(t)
		order := f.postedOrder("2026-01-01")
		n := 0
		fields := func(date string) purchase.InvoiceFields {
			n++
			return invoiceFields(date, "1C26TAA", "C"+string(rune('A'+n%26))+date, il(3, "0.001", 500_000, "10"))
		}
		return recordtest.Harness{
			Core: f.core, Actor: f.staff, Admin: f.admin, LegalEntity: f.c, Table: "purchase.invoices",
			Get: func(ctx context.Context, r record.Ref) (any, error) { return f.purchase.Invoice(ctx, r.ID) },
			Gaps: map[string]string{
				"write actions leave allowed_actions when the product is off or the period locked": "a posted invoice in a locked period still offers create_payment, dated in an open period",
			},
			Create: func(ctx context.Context, date, requestID string) (record.Ref, error) {
				id, err := f.purchase.CreateInvoice(ctx, purchase.NewInvoice{RequestID: requestID, OrderID: order, InvoiceFields: fields(date)})
				return record.Ref{Type: "purchase.invoice", ID: id}, err
			},
			Edit: func(ctx context.Context, r record.Ref, version int32, date string) error {
				return f.purchase.UpdateInvoice(ctx, r.ID, purchase.InvoiceUpdate{Version: version, InvoiceFields: fields(date)})
			},
			RequireApproval: func(_ *testing.T, a record.Action) { f.managerApproves("purchase.invoice", a) },
			Approve:         func(instance int64) error { return f.appr.Approve(f.manager, instance, 1) },
			Reject:          func(instance int64) error { return f.appr.Reject(f.manager, instance, 1, "không") },
			BreakSubmit: func(t *testing.T, r record.Ref) {
				if _, err := f.core.System.Exec(f.ctx, `UPDATE purchase.invoices SET supplier_invoice_no = NULL WHERE id = $1`, r.ID); err != nil {
					t.Fatal(err)
				}
			},
			// Posting checks the supplier reference again.
			BreakPosting: func(t *testing.T, r record.Ref) {
				if _, err := f.core.System.Exec(f.ctx, `UPDATE purchase.invoices SET supplier_invoice_date = NULL WHERE id = $1`, r.ID); err != nil {
					t.Fatal(err)
				}
			},
			Effects:      func(_ *testing.T, r record.Ref) int { return min(len(f.postingLines("purchase.invoice", r.ID)), 1) },
			PostingLines: func(_ *testing.T, r record.Ref) int { return len(f.postingLines("purchase.invoice", r.ID)) },
		}
	})
}

// Stock lines clear received not invoiced at the order price; with nothing received, the
// cartons' 2,000 đ over the order price go to price difference. Expense, input VAT and the
// payable with the supplier as before.
func TestSupplierInvoicePostingLines(t *testing.T) {
	f := newFixture(t)
	order := f.postedOrder("2026-03-10")
	id := must(t)(f.invoice(f.staff, order, billed("2026-03-12", "0000123")))
	inv, err := f.purchase.Invoice(f.staff, id)
	// 208,000 + 16,640; 6,000,000 + 600,000; 500,000 + 50,000.
	if err != nil || inv.Totals != (purchase.Totals{Subtotal: 6_708_000, VatTotal: 666_640, Total: 7_374_640}) ||
		inv.Lines[0].UnitPrice != 52_000 || !inv.Lines[0].Stocked || inv.Lines[2].Stocked || *inv.SupplierInvoiceNo != "0000123" ||
		inv.Supplier.Name != "Công ty NCC1" {
		t.Fatalf("invoice %+v %v", inv, err)
	}
	f.check(f.move(f.staff, "purchase.invoice", id, record.Posted))
	got := f.postingLines("purchase.invoice", id)
	want := map[string]int64{"purchase_received_not_invoiced": -6_200_000, "purchase_price_difference": 8_000, "purchase_expense": 500_000,
		"purchase_input_vat": 666_640, "purchase_payable": 7_374_640}
	if len(got) != len(want) {
		t.Fatalf("lines %v", got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %d, want %d", k, got[k], v)
		}
	}
	f.check(f.move(f.staff, "purchase.invoice", id, record.Cancelled))
	if got := f.postingLines("purchase.invoice", id); len(got) != 0 {
		t.Fatalf("after cancel %v", got)
	}
	// No VAT, no VAT line.
	none := must(t)(f.invoice(f.staff, order, invoiceFields("2026-03-13", "1C26TAA", "0000124", il(3, "1", 500_000, "none"))))
	f.check(f.move(f.staff, "purchase.invoice", none, record.Posted))
	if got := f.postingLines("purchase.invoice", none); len(got) != 2 || got["purchase_expense"] != 500_000 {
		t.Fatalf("no VAT %v", got)
	}
}

func TestSupplierInvoicePartial(t *testing.T) {
	f := newFixture(t)
	order := f.postedOrder("2026-03-10")
	first := must(t)(f.invoice(f.staff, order, invoiceFields("2026-03-12", "1C26TAA", "1", il(1, "4", 52_000, "8"))))
	f.check(f.move(f.staff, "purchase.invoice", first, record.Posted))
	if o, _ := f.purchase.Order(f.staff, order); o.InvoiceState != "partial" || *o.Lines[0].Invoiced != "4" || len(o.Invoices) != 1 {
		t.Fatalf("partial %+v", o)
	}
	if e, ok := errors.AsType[*platform.Error](must2(f.invoice(f.staff, order, invoiceFields("2026-03-12", "1C26TAA", "2", il(1, "7", 50_000, "8"))))); !ok ||
		e.Code != "purchase_invoice_over_order" || e.Params["remaining"] != "6.000" {
		t.Fatalf("over order %v", e)
	}
	wantCode(t, must2(f.invoice(f.staff, order, invoiceFields("2026-03-12", "1C26TAA", "2", il(9, "1", 1, "8")))), "purchase_invoice_position")
	wantCode(t, must2(f.invoice(f.staff, order, invoiceFields("2026-03-09", "1C26TAA", "2", il(1, "1", 1, "8")))), "purchase_invoice_before_order")
	rest := must(t)(f.invoice(f.staff, order, invoiceFields("2026-03-12", "1C26TAA", "2", il(1, "6", 50_000, "8"), il(2, "2", 2_900_000, "10"), il(3, "1", 500_000, "10"))))
	f.check(f.move(f.staff, "purchase.invoice", rest, record.Posted))
	o, _ := f.purchase.Order(f.staff, order)
	if o.InvoiceState != "full" || contains(o.AllowedActions, "create_invoice") {
		t.Fatalf("full %q %v", o.InvoiceState, o.AllowedActions)
	}
	opts, err := f.purchase.InvoiceOptions(f.staff, order)
	if err != nil || len(opts.Lines) != 3 || opts.Lines[0].Remaining != "0.000" || opts.Lines[1].UnitPrice != 3_000_000 {
		t.Fatalf("options %+v %v", opts, err)
	}
	l, err := f.purchase.Invoices(f.staff, purchase.InvoiceFilter{OrderID: order, Sort: "-date", Paging: page})
	if err != nil || l.Total != 2 || l.Items[0].SupplierName != "Công ty NCC1" {
		t.Fatalf("list %+v %v", l, err)
	}
	if l, _ := f.purchase.Invoices(f.staff, purchase.InvoiceFilter{Q: "1C26TAA", Sort: "-date", Paging: page}); l.Total != 2 {
		t.Fatalf("search by supplier number %+v", l)
	}
}

func TestSupplierInvoiceDuplicate(t *testing.T) {
	f := newFixture(t)
	order := f.postedOrder("2026-03-10")
	first := must(t)(f.invoice(f.staff, order, invoiceFields("2026-03-12", "1C26TAA", "0000123", il(1, "1", 50_000, "8"))))
	f.check(f.move(f.staff, "purchase.invoice", first, record.Posted))
	dup := must(t)(f.invoice(f.staff, order, invoiceFields("2026-03-12", " 1c26taa ", "0000123", il(1, "1", 50_000, "8"))))
	wantCode(t, f.move(f.staff, "purchase.invoice", dup, record.Posted), "purchase_invoice_duplicate")
	other := must(t)(f.invoice(f.staff, order, invoiceFields("2026-03-12", "1C26TBB", "0000123", il(1, "1", 50_000, "8"))))
	f.check(f.move(f.staff, "purchase.invoice", other, record.Posted))
	// Team B's supplier of the same partner books the same paper: not checked across teams.
	staffB := f.ctxFor("staff_b", "staff@B", "partner.editor@B")
	s, _ := f.purchase.Supplier(f.staff, f.supplier)
	supB := must(t)(f.purchase.CreateSupplier(staffB, purchase.NewSupplier{PartnerID: &s.PartnerID, SupplierFields: purchase.SupplierFields{OrgUnitID: f.b, Active: true}}))
	fl := f.orderFields("2026-03-10")
	fl.OrgUnitID, fl.SupplierID = f.b, supB
	orderB := must(t)(f.purchase.CreateOrder(staffB, purchase.NewOrder{RequestID: newRequest(), OrderFields: fl}))
	f.check(f.move(staffB, "purchase.order", orderB, record.Posted))
	invB := must(t)(f.invoice(staffB, orderB, invoiceFields("2026-03-12", "1C26TAA", "0000123", il(1, "1", 50_000, "8"))))
	f.check(f.move(staffB, "purchase.invoice", invB, record.Posted))
	// Once the first is cancelled, its number is free again.
	f.check(f.move(f.staff, "purchase.invoice", first, record.Cancelled))
	f.check(f.move(f.staff, "purchase.invoice", dup, record.Posted))
}

func TestSupplierInvoiceDuplicateConcurrent(t *testing.T) {
	f := newFixture(t)
	// Two orders of the same supplier: only the supplier's row lock makes the sends take turns.
	ids := []int64{
		must(t)(f.invoice(f.staff, f.postedOrder("2026-03-10"), invoiceFields("2026-03-12", "1C26TAA", "77", il(1, "1", 50_000, "8")))),
		must(t)(f.invoice(f.staff, f.postedOrder("2026-03-10"), invoiceFields("2026-03-12", "1C26TAA", "77", il(1, "1", 50_000, "8")))),
	}
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range 2 {
		wg.Go(func() { errs[i] = f.move(f.staff, "purchase.invoice", ids[i], record.Posted) })
	}
	wg.Wait()
	if (errs[0] == nil) == (errs[1] == nil) {
		t.Fatalf("both or neither: %v %v", errs[0], errs[1])
	}
}

func TestSupplierInvoiceSupplierRef(t *testing.T) {
	f := newFixture(t)
	order := f.postedOrder("2026-03-10")
	fl := invoiceFields("2026-03-12", "1C26TAA", "1", il(1, "1", 50_000, "8"))
	fl.SupplierInvoiceSymbol = nil
	id := must(t)(f.invoice(f.staff, order, fl))
	wantCode(t, f.move(f.staff, "purchase.invoice", id, record.Posted), "purchase_invoice_supplier_ref_required")
}

func TestSupplierInvoicePostsWithoutAccounting(t *testing.T) {
	f := newFixture(t)
	order := f.postedOrder("2026-03-10")
	id := must(t)(f.invoice(f.staff, order, billed("2026-03-12", "9")))
	f.check(f.move(f.staff, "purchase.invoice", id, record.Posted))
}

func TestOrderCancelWithInvoice(t *testing.T) {
	f := newFixture(t)
	order := f.postedOrder("2026-03-10")
	id := must(t)(f.invoice(f.staff, order, billed("2026-03-12", "9")))
	f.check(f.move(f.staff, "purchase.invoice", id, record.Posted))
	wantCode(t, f.move(f.staff, "purchase.order", order, record.Cancelled), "purchase_order_has_invoice")
	f.check(f.move(f.staff, "purchase.invoice", id, record.Cancelled))
	f.check(f.move(f.staff, "purchase.order", order, record.Cancelled))
}

func TestSupplierInvoiceTeamBoundary(t *testing.T) {
	f := newFixture(t)
	order := f.postedOrder("2026-03-10")
	id := must(t)(f.invoice(f.staff, order, billed("2026-03-12", "9")))
	staffB := f.ctxFor("staff_b", "staff@B")
	if _, err := f.purchase.Invoice(staffB, id); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("B opens A's invoice: %v", err)
	}
	if _, err := f.purchase.InvoiceOptions(staffB, order); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("B reads options: %v", err)
	}
	if _, err := f.invoice(staffB, order, billed("2026-03-12", "10")); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("B invoices A's order: %v", err)
	}
}

// A small invoice posting directly while a manager approves a large one of the same order: both
// take the order before the supplier, so neither waits on the other forever.
func TestSupplierInvoiceDirectPostRacesApproval(t *testing.T) {
	f := newFixture(t)
	f.check(f.appr.SaveRule(f.admin, "purchase.invoice", record.Post, approval.RuleInput{
		Flow: approval.Flow{{Split: &approval.Split{Mode: approval.ChooseOne, Branches: []approval.Branch{{
			Condition: &approval.Expr{Field: "amount", Op: "gt", Value: "1000000"},
			Body:      approval.Flow{{Step: &approval.StepDef{Approver: approval.Approver{Kind: "role", Product: "purchase", Role: "manager"}}}},
		}}}}},
		MaxLevels: 3, FallbackProduct: "purchase", FallbackRole: "manager",
	}))
	for n := range 15 {
		order := f.postedOrder("2026-03-10")
		big := must(t)(f.invoice(f.staff, order, invoiceFields("2026-03-12", "1C26TAA", fmt.Sprint("B", n), il(2, "1", 3_000_000, "10"))))
		f.check(f.move(f.staff, "purchase.invoice", big, record.Posted)) // held for approval
		d, err := f.rec.Get(f.admin, record.Ref{Type: "purchase.invoice", ID: big})
		f.check(err)
		instance := f.pendingInstance(big)
		small := must(t)(f.invoice(f.staff, order, invoiceFields("2026-03-12", "1C26TAA", fmt.Sprint("S", n), il(1, "1", 50_000, "8"))))
		var wg sync.WaitGroup
		errs := make([]error, 2)
		wg.Go(func() { errs[0] = f.move(f.staff, "purchase.invoice", small, record.Posted) })
		wg.Go(func() { errs[1] = f.appr.Approve(f.manager, instance, 1) })
		wg.Wait()
		for _, err := range errs {
			if err != nil {
				t.Fatalf("round %d (doc v%d): %v", n, d.Version, err)
			}
		}
	}
}

// pendingInstance is the approval instance waiting on a document.
func (f *fixture) pendingInstance(doc int64) int64 {
	f.t.Helper()
	var id int64
	f.check(f.core.Pool.QueryRow(f.ctx, `SELECT id FROM approval.instances WHERE doc_id = $1 ORDER BY id DESC LIMIT 1`, doc).Scan(&id))
	return id
}

// stockCost is the company's running quantity and value of an item.
func (f *fixture) stockCost(itemID int64) (string, int64) {
	f.t.Helper()
	var q string
	var v int64
	if err := f.core.Pool.QueryRow(f.ctx, `SELECT quantity::text, value FROM stock.costs WHERE legal_entity_id = $1 AND item_id = $2`, f.c, itemID).Scan(&q, &v); err != nil {
		f.t.Fatal(err)
	}
	return q, v
}

// A receipt books the goods at the order price; the invoice's price difference goes to the
// goods still on hand, and cancelling takes both back.
func TestReceiptAndInvoiceValueStock(t *testing.T) {
	f := newFixture(t)
	order := f.postedOrder("2026-03-10")
	rc := must(t)(f.receive(order, "2026-03-11", lotLine(1, "6", "L1", nil)))
	f.check(f.move(f.staff, "purchase.goods_receipt", rc, record.Posted))
	if got := f.postingLines("purchase.goods_receipt", rc); len(got) != 2 || got["stock_inventory"] != 300_000 || got["purchase_received_not_invoiced"] != 300_000 {
		t.Fatalf("receipt %v", got)
	}
	id := must(t)(f.invoice(f.staff, order, billed("2026-03-12", "0000123")))
	f.check(f.move(f.staff, "purchase.invoice", id, record.Posted))
	got := f.postingLines("purchase.invoice", id)
	if got["stock_inventory"] != 8_000 || got["purchase_price_difference"] != 0 || got["purchase_received_not_invoiced"] != -6_200_000 {
		t.Fatalf("invoice %v", got)
	}
	if q, v := f.stockCost(f.lot); q != "6.000" || v != 308_000 {
		t.Fatalf("cost %s %d", q, v)
	}
	f.check(f.move(f.staff, "purchase.invoice", id, record.Cancelled))
	f.check(f.move(f.staff, "purchase.goods_receipt", rc, record.Cancelled))
	if q, v := f.stockCost(f.lot); q != "0.000" || v != 0 {
		t.Fatalf("after cancel %s %d", q, v)
	}
	if got := f.postingLines("purchase.goods_receipt", rc); len(got) != 0 {
		t.Fatalf("receipt after cancel %v", got)
	}
}
