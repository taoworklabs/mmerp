package purchase

import (
	"testing"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// The rules are Sales': these are its cases without the catalogue discount.
func TestCalcMatchesSales(t *testing.T) {
	line := func(qty string, price int64, pct, vat string) LineInput {
		return LineInput{ItemID: 1, Description: "x", Quantity: qty, UnitPrice: price, DiscountPercent: pct, VatRate: vat}
	}
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
		{"fractional quantity and discount", []LineInput{line("2.5", 33_333, "12.5", "10")}, platform.RoundLine,
			[]int64{7_292}, Totals{83_333, 10_417, 7_292, 80_208}},
		{"rates apart", []LineInput{line("1", 1_010, "0", "5"), line("1", 1_005, "0", "10"), line("3", 500, "100", "none")}, platform.RoundTotal,
			[]int64{51, 101, 0}, Totals{3_515, 1_500, 152, 2_167}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, tot, err := compute(c.lines, c.mode)
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
	for _, bad := range []struct {
		l    LineInput
		code string
	}{
		{line("0", 1, "0", "8"), "invalid_quantity"},
		{line("1", 1, "100.01", "8"), "invalid_discount"},
		{line("1", 1, "0", "7"), "invalid_vat_rate"},
		{line("999999999999", 999_999_999, "0", "8"), "line_amount_too_large"},
	} {
		_, _, err := compute([]LineInput{line("1", 1, "0", "0"), bad.l}, platform.RoundLine)
		if e, ok := err.(*platform.Error); !ok || e.Code != bad.code || e.Params["line"] != 2 {
			t.Errorf("%+v: %v, want %s on line 2", bad.l, err, bad.code)
		}
	}
}
