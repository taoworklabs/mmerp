package sales_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/modules/sales"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// itemsFixture adds 45 items beyond the fixture's two: codes I001..I045, retired
// every seventh, so the comparison matrix has enough rows to page through.
func itemsFixture(t *testing.T, f *fixture) []int64 {
	t.Helper()
	catalog := f.ctxFor("catalog_items", "catalog_admin@*")
	vats := []string{"none", "0", "5", "8", "10"}
	ids := make([]int64, 45)
	for i := 1; i <= 45; i++ {
		ids[i-1] = must(t)(f.sales.CreateItem(catalog, sales.NewItem{
			Code: fmt.Sprintf("I%03d", i),
			ItemFields: sales.ItemFields{
				Name: fmt.Sprintf("Hàng %03d", i), Unit: "cái", Price: int64(i * 1000),
				VatRate: vats[i%len(vats)], Active: i%7 != 0,
			},
		}))
	}
	return ids
}

// A "%" in the search matches a literal percent sign, not everything: the
// engine escapes LIKE metacharacters.
func TestItemListSearchLiteralPercent(t *testing.T) {
	f := newFixture(t)
	l, err := f.sales.Items(f.ctxFor("viewer_pct", "viewer@A"), sales.ItemFilter{Q: "%", Sort: "code", Paging: platform.Paging{Page: 1, PageSize: 20}})
	if err != nil {
		t.Fatal(err)
	}
	if l.Total != 0 || len(l.Items) != 0 {
		t.Fatalf("q %q matched %d, want 0", "%", l.Total)
	}
}

// TestItemOptionsUnchanged pins itemOptions, the approval item picker: it still
// returns retired items, at most 20, ordered by code then id. Written against
// the fixed-SQL implementation and kept unchanged across the switch.
func TestItemOptionsUnchanged(t *testing.T) {
	f := newFixture(t)
	ids := itemsFixture(t, f)
	admin := f.ctxFor("appr_admin", "approval_admin@*")

	options := func(q string) []int64 {
		t.Helper()
		out, err := f.appr.FieldOptions(admin, "sales.quote", "items", q, nil)
		if err != nil {
			t.Fatal(err)
		}
		got := make([]int64, len(out))
		for i, o := range out {
			got[i] = o.ID
		}
		return got
	}

	// q = "": the 20 lowest codes; "BUT" sorts before every "I0xx" code.
	if want := append([]int64{f.pen}, ids[:19]...); !slices.Equal(options(""), want) {
		t.Fatalf("q=\"\": got %v, want %v", options(""), want)
	}
	// A code fragment: I010..I019, in code order.
	if want := ids[9:19]; !slices.Equal(options("i01"), want) {
		t.Fatalf("q=i01: got %v, want %v", options("i01"), want)
	}
	// A fragment matching only a retired item (i = 7, retired since 7 mod 7 == 0).
	if want := []int64{ids[6]}; !slices.Equal(options("i007"), want) {
		t.Fatalf("q=i007: got %v, want %v", options("i007"), want)
	}
}
