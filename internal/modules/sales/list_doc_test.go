package sales_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/sales"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// docSeed: quotations and orders in every status, in teams A and B. One posted
// quotation has a live order, another only a cancelled one.
type docSeed struct {
	*fixture
	quotes, orders []int64
}

func newDocSeed(t *testing.T) *docSeed {
	f := &docSeed{fixture: newFixture(t)}
	staffB := f.ctxFor("staff_b", "staff@B")
	f.ctxFor("manager_b", "manager@B") // B's pending documents need an approver
	xyz := must(t)(f.sales.CreateCustomer(f.staff, customer("XYZ", f.a)))
	kh2 := must(t)(f.sales.CreateCustomer(staffB, customer("KH2", f.b)))
	f.managerApproves("sales.quote", record.Post)
	f.managerApproves("sales.order", record.Post)
	dates := []string{"2026-03-01", "2026-03-02", "2026-03-03"}
	for _, k := range []string{"sales.quote", "sales.order"} {
		create := f.sales.CreateQuote
		if k == "sales.order" {
			create = f.sales.CreateOrder
		}
		for i := range 14 {
			ctx, fl := f.staff, f.fields(dates[i%3])
			fl.Lines[0].Quantity = fmt.Sprint(1 + i%2)
			switch {
			case i%5 == 2:
				ctx, fl.OrgUnitID, fl.CustomerID = staffB, f.b, kh2
			case i%2 == 1:
				fl.CustomerID = xyz
			}
			if k == "sales.order" {
				fl.ValidUntil = nil
			}
			if i%4 == 3 {
				fl.Lines[2].DiscountPercent = "15" // waits for a manager
			}
			id := must(t)(create(ctx, sales.NewDoc{RequestID: newRequest(), DocFields: fl}))
			switch i % 4 {
			case 1, 3:
				f.check(f.move(ctx, k, id, record.Posted))
			case 2:
				f.check(f.move(ctx, k, id, record.Posted))
				f.check(f.move(ctx, k, id, record.Cancelled))
			}
			if k == "sales.quote" {
				f.quotes = append(f.quotes, id)
			} else {
				f.orders = append(f.orders, id)
			}
		}
	}
	// Posted quotations of team A (i = 1 and 5), turned into orders.
	o1 := must(t)(f.order(f.staff, f.quotes[1]))
	o2 := must(t)(f.order(f.staff, f.quotes[5]))
	f.check(f.move(f.staff, "sales.order", o2, record.Posted))
	f.check(f.move(f.staff, "sales.order", o2, record.Cancelled))
	f.orders = append(f.orders, o1, o2)
	return f
}

func (f *docSeed) listed(ctx context.Context, list func(context.Context, sales.DocFilter) (sales.DocList, error)) []sales.DocListItem {
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
	s := &docSeed{fixture: f}
	if q := find(s.listed(f.staff, f.sales.Quotes), quote); !q.Ordered {
		t.Fatalf("with a live order: %+v", q)
	}
	if o := find(s.listed(f.staff, f.sales.Orders), order); !o.FromQuote {
		t.Fatalf("order from a quotation: %+v", o)
	}
	f.check(f.move(f.staff, "sales.order", order, record.Posted))
	f.check(f.move(f.staff, "sales.order", order, record.Cancelled))
	if q := find(s.listed(f.staff, f.sales.Quotes), quote); q.ID != quote || q.Ordered {
		t.Fatalf("order cancelled: %+v", q)
	}
}

func TestDocListKindsSeparate(t *testing.T) {
	f := newDocSeed(t)
	for _, d := range f.listed(f.admin, f.sales.Quotes) {
		if slices.Contains(f.orders, d.ID) {
			t.Fatalf("order %d among quotations", d.ID)
		}
	}
	for _, d := range f.listed(f.admin, f.sales.Orders) {
		if slices.Contains(f.quotes, d.ID) {
			t.Fatalf("quotation %d among orders", d.ID)
		}
	}
	if q, o := len(f.listed(f.admin, f.sales.Quotes)), len(f.listed(f.admin, f.sales.Orders)); q != len(f.quotes) || o != len(f.orders) {
		t.Fatalf("%d quotations, %d orders", q, o)
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
