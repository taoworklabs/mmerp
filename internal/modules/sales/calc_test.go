package sales

import (
	"testing"

	"github.com/taoworklabs/mmerp/internal/platform"
)

func TestCompute(t *testing.T) {
	line := func(qty string, price int64, pct, vat string) LineInput {
		return LineInput{ItemID: 1, Description: "x", Quantity: qty, UnitPrice: price, DiscountPercent: pct, VatRate: vat}
	}
	// Two lines of net 10,005 at 8 %: VAT 800.4 each.
	same := []LineInput{line("1", 10_005, "0", "8"), line("1", 10_005, "0", "8")}
	cases := []struct {
		name  string
		lines []LineInput
		mode  platform.Rounding
		vats  []int64
		want  Totals
	}{
		{"line rounds each line", same, platform.RoundLine, []int64{800, 800}, Totals{20_010, 0, 1_600, 21_610}},
		{"total rounds the rate once, then allocates", same, platform.RoundTotal, []int64{801, 800}, Totals{20_010, 0, 1_601, 21_611}},
		{
			// 2.5 × 33,333 = 83,332.5 → 83,333; 12.5 % of it = 10,416.625 → 10,417; net 72,916 × 10 % = 7,291.6 → 7,292.
			"fractional quantity and discount", []LineInput{line("2.5", 33_333, "12.5", "10")}, platform.RoundLine,
			[]int64{7_292}, Totals{83_333, 10_417, 7_292, 80_208},
		},
		{
			// Rates round apart: 50.5 → 51 and 100.5 → 101, where together they would make 151; none has no VAT.
			"rates apart", []LineInput{line("1", 1_010, "0", "5"), line("1", 1_005, "0", "10"), line("3", 500, "100", "none")}, platform.RoundTotal,
			[]int64{51, 101, 0}, Totals{3_515, 1_500, 152, 2_167},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, tot, _, err := compute(c.lines, make([]int64, len(c.lines)), c.mode)
			if err != nil {
				t.Fatal(err)
			}
			for i, v := range c.vats {
				if a[i].vat != v {
					t.Errorf("line %d vat %d, want %d", i+1, a[i].vat, v)
				}
			}
			if tot != c.want {
				t.Errorf("totals %+v, want %+v", tot, c.want)
			}
		})
	}
	_, _, maxDiscount, _ := compute([]LineInput{line("1", 10, "5", "0"), line("1", 10, "15.5", "0")}, []int64{0, 0}, platform.RoundLine)
	if maxDiscount.String() != "15.5" {
		t.Fatalf("max discount %s", maxDiscount)
	}
	// A unit price cut below the catalogue price counts as discount: 85 of 100 is 15 %, and
	// 85 less 10 % is 76.5 of 100, 23.5 %; a price above the list counts only its discount; a one-đồng
	// cut of 100,000 rounds up to 0.01 %, so it is never lost.
	for _, c := range []struct {
		price     int64
		pct, want string
		list      int64
	}{{85, "0", "15", 100}, {85, "10", "23.5", 100}, {120, "5", "5", 100}, {99_999, "0", "0.01", 100_000}} {
		_, _, got, _ := compute([]LineInput{line("1", c.price, c.pct, "0")}, []int64{c.list}, platform.RoundLine)
		if got.String() != c.want {
			t.Errorf("price %d less %s%% of %d: max discount %s, want %s", c.price, c.pct, c.list, got, c.want)
		}
	}
	for _, bad := range []struct {
		l    LineInput
		code string
	}{
		{line("0", 1, "0", "8"), "invalid_quantity"},
		{line("1", 1, "100.01", "8"), "invalid_discount"},
		{line("1", 1, "0", "7"), "invalid_vat_rate"},
		{line("999999999999", 999_999_999, "0", "8"), "line_amount_too_large"},
	} {
		_, _, _, err := compute([]LineInput{line("1", 1, "0", "0"), bad.l}, []int64{0, 0}, platform.RoundLine)
		if e, ok := err.(*platform.Error); !ok || e.Code != bad.code || e.Params["line"] != 2 {
			t.Errorf("%+v: %v, want %s on line 2", bad.l, err, bad.code)
		}
	}
}
