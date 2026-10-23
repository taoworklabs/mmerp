package sales_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/sales"
	"github.com/taoworklabs/mmerp/internal/shared/item"
)

// itemsFixture adds 45 items beyond the fixture's two: codes I001..I045, retired
// every seventh, so the item picker has more than its 20-row limit to choose from.
func itemsFixture(t *testing.T, f *fixture) []int64 {
	t.Helper()
	vats := []string{"none", "0", "5", "8", "10"}
	ids := make([]int64, 45)
	for i := 1; i <= 45; i++ {
		ids[i-1] = must(t)(f.items.CreateItem(f.admin, item.NewItem{
			Code: fmt.Sprintf("I%03d", i),
			ItemFields: item.ItemFields{
				Name: fmt.Sprintf("Hàng %03d", i), Unit: "cái", Price: int64(i * 1000),
				VatRate: vats[i%len(vats)], Active: i%7 != 0,
			},
		}))
	}
	return ids
}

func (f *fixture) itemOptions(q string, ids ...int64) []record.FieldOption {
	f.t.Helper()
	out, err := f.appr.FieldOptions(f.ctxFor(fmt.Sprintf("appr_%d", requests.Add(1)), "approval_admin@*"), "sales.quote", "items", q, ids)
	f.check(err)
	return out
}

// TestItemOptionsUnchanged pins itemOptions, the approval item picker: it still
// returns retired items, at most 20, ordered by code then id.
func TestItemOptionsUnchanged(t *testing.T) {
	f := newFixture(t)
	ids := itemsFixture(t, f)
	options := func(q string) []int64 {
		t.Helper()
		var got []int64
		for _, o := range f.itemOptions(q) {
			got = append(got, o.ID)
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

// An item of the shared catalogue goes on a quotation and shows in the "items"
// approval field, by search and by id.
func TestItemsApprovalFieldAfterMove(t *testing.T) {
	f := newFixture(t)
	id := must(t)(f.items.CreateItem(f.admin, item.NewItem{Code: "MUC", ItemFields: item.ItemFields{Name: "Mực in", Unit: "lọ", Price: 5000, VatRate: "8", Active: true}}))
	fl := f.fields("2026-03-10")
	fl.Lines = append(fl.Lines, fl.Lines[0])
	fl.Lines[len(fl.Lines)-1].ItemID = id
	must(t)(f.sales.CreateQuote(f.staff, sales.NewDoc{RequestID: newRequest(), DocFields: fl}))
	want := []record.FieldOption{{ID: id, Code: "MUC", Name: "Mực in"}}
	if got := f.itemOptions("muc"); !slices.Equal(got, want) {
		t.Fatalf("search: %v", got)
	}
	if got := f.itemOptions("", id); !slices.Equal(got, want) {
		t.Fatalf("labels: %v", got)
	}
}
