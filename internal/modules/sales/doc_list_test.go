package sales_test

import (
	"context"
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/sales"
	"github.com/taoworklabs/mmerp/internal/platform"
)

func (f *fixture) listed(ctx context.Context, list func(context.Context, sales.DocFilter) (sales.DocList, error)) []sales.DocListItem {
	f.t.Helper()
	l, err := list(ctx, sales.DocFilter{Sort: "-date", Paging: platform.Paging{Page: 1, PageSize: 100}})
	f.check(err)
	return l.Items
}

func find(items []sales.DocListItem, id int64) sales.DocListItem {
	i := slices.IndexFunc(items, func(d sales.DocListItem) bool { return d.ID == id })
	if i < 0 {
		return sales.DocListItem{}
	}
	return items[i]
}

// A quotation counts as ordered while one of its orders is not cancelled.
func TestQuoteListOrderedFlag(t *testing.T) {
	f := newFixture(t)
	quote := must(t)(f.sales.CreateQuote(f.staff, sales.NewDoc{RequestID: newRequest(), DocFields: f.fields(todayOf(t, f))}))
	f.check(f.move(f.staff, "sales.quote", quote, record.Posted))
	order := must(t)(f.order(f.staff, quote))
	if q := find(f.listed(f.staff, f.sales.Quotes), quote); !q.Ordered {
		t.Fatalf("with a live order: %+v", q)
	}
	if o := find(f.listed(f.staff, f.sales.Orders), order); !o.FromQuote {
		t.Fatalf("order from a quotation: %+v", o)
	}
	f.check(f.move(f.staff, "sales.order", order, record.Posted))
	f.check(f.move(f.staff, "sales.order", order, record.Cancelled))
	if q := find(f.listed(f.staff, f.sales.Quotes), quote); q.ID != quote || q.Ordered {
		t.Fatalf("order cancelled: %+v", q)
	}
}

func TestDocListKindsSeparate(t *testing.T) {
	f := newFixture(t)
	quote := must(t)(f.sales.CreateQuote(f.staff, sales.NewDoc{RequestID: newRequest(), DocFields: f.fields("2026-03-01")}))
	f.check(f.move(f.staff, "sales.quote", quote, record.Posted))
	fromQuote := must(t)(f.order(f.staff, quote))
	fl := f.fields("2026-03-02")
	fl.ValidUntil = nil
	plain := must(t)(f.sales.CreateOrder(f.staff, sales.NewDoc{RequestID: newRequest(), DocFields: fl}))
	quotes, orders := f.listed(f.admin, f.sales.Quotes), f.listed(f.admin, f.sales.Orders)
	if len(quotes) != 1 || quotes[0].ID != quote {
		t.Fatalf("quotations: %+v", quotes)
	}
	got := []int64{}
	for _, o := range orders {
		got = append(got, o.ID)
	}
	slices.Sort(got)
	if want := []int64{fromQuote, plain}; !slices.Equal(got, want) {
		t.Fatalf("orders %v, want %v", got, want)
	}
}

// A "%" in the search matches a literal percent sign, not everything.
func TestDocListSearchLiteralPercent(t *testing.T) {
	f := newFixture(t)
	f.check(must2(f.sales.CreateQuote(f.staff, sales.NewDoc{RequestID: newRequest(), DocFields: f.fields("2026-03-01")})))
	l, err := f.sales.Quotes(f.admin, sales.DocFilter{Q: "%", Sort: "-date", Paging: platform.Paging{Page: 1, PageSize: 20}})
	if err != nil {
		t.Fatal(err)
	}
	if l.Total != 0 || len(l.Items) != 0 {
		t.Fatalf("q %q matched %d, want 0", "%", l.Total)
	}
}

func TestDocListInvoiceStateFilter(t *testing.T) {
	v := newInvoices(t)
	other := v.newOrder("0")
	v.check(v.move(v.staff, "sales.order", other, record.Posted))
	v.postedInvoice(il(3, "1"))
	f := sales.DocFilter{InvoiceState: "partial", Sort: "-date", Paging: platform.Paging{Page: 1, PageSize: 50}}
	l, err := v.sales.Orders(v.staff, f)
	v.check(err)
	if l.Total != 1 || l.Items[0].ID != v.order || l.Items[0].InvoiceState != "partial" {
		t.Fatalf("partial %+v", l)
	}
	if o := find(v.listed(v.staff, v.sales.Orders), other); o.InvoiceState != "none" {
		t.Fatalf("other %+v", o)
	}
}

func TestDocListPaymentStateFilter(t *testing.T) {
	r := newReceipts(t)
	other := r.newOrder("0")
	r.check(r.move(r.staff, "sales.order", other, record.Posted))
	r.postedReceipt(r.receiptFields("cash", al(r.inv1, total1)))
	f := sales.DocFilter{PaymentState: "partial", Sort: "-date", Paging: platform.Paging{Page: 1, PageSize: 50}}
	l, err := r.sales.Orders(r.staff, f)
	r.check(err)
	if l.Total != 1 || l.Items[0].ID != r.order || l.Items[0].PaymentState != "partial" {
		t.Fatalf("partial %+v", l)
	}
	if o := find(r.listed(r.staff, r.sales.Orders), other); o.PaymentState != "unpaid" {
		t.Fatalf("other %+v", o)
	}
	if q := find(r.listed(r.staff, r.sales.Quotes), 0); q.PaymentState != "" {
		t.Fatalf("quote %+v", q)
	}
}
