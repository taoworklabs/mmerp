package sales

import (
	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/tax"
)

var (
	hundred = decimal.NewFromInt(100)
	// maxLine bounds a line's amount, far above any sale, so sums never overflow int64.
	maxLine = decimal.New(1, 15)
)

// amounts are what the server computes for one line.
type amounts struct{ amount, discount, vat int64 }

// compute works out every line's amounts and the document totals; VAT rounding is tax.VAT's.
// It also returns the largest discount, the max_discount approval field: a line priced below
// its catalogue price (list, 0 for none) counts that cut as discount, so lowering the unit
// price never slips past a discount rule.
func compute(lines []LineInput, list []int64, mode platform.Rounding) ([]amounts, Totals, decimal.Decimal, error) {
	out := make([]amounts, len(lines))
	nets := make([]int64, len(lines))
	rates := make([]string, len(lines))
	maxDiscount := decimal.Zero
	for i, l := range lines {
		qty, ok := record.ParseNumber(l.Quantity)
		if !ok || !qty.IsPositive() {
			return nil, Totals{}, maxDiscount, errLine("invalid_quantity", i+1)
		}
		pct, ok := record.ParseNumber(l.DiscountPercent)
		if !ok || pct.IsNegative() || pct.GreaterThan(hundred) {
			return nil, Totals{}, maxDiscount, errLine("invalid_discount", i+1)
		}
		if !tax.Valid(l.VatRate) {
			return nil, Totals{}, maxDiscount, errLine("invalid_vat_rate", i+1)
		}
		maxDiscount = decimal.Max(maxDiscount, pct)
		if list[i] > 0 {
			// 100 − price × (100 − pct) / list, rounded up so 10.001 % is above 10 %.
			cut := hundred.Sub(decimal.NewFromInt(l.UnitPrice).Mul(hundred.Sub(pct)).Div(decimal.NewFromInt(list[i]))).RoundCeil(2)
			maxDiscount = decimal.Max(maxDiscount, cut)
		}
		gross := qty.Mul(decimal.NewFromInt(l.UnitPrice))
		if gross.GreaterThan(maxLine) {
			return nil, Totals{}, maxDiscount, errLine("line_amount_too_large", i+1)
		}
		a := platform.Round(gross)
		d := platform.Round(decimal.NewFromInt(a).Mul(pct).Div(hundred))
		out[i] = amounts{amount: a, discount: d}
		nets[i], rates[i] = a-d, l.VatRate
	}
	for i, v := range tax.VAT(nets, rates, mode) {
		out[i].vat = v
	}
	var t Totals
	for _, a := range out {
		t.Subtotal += a.amount
		t.DiscountTotal += a.discount
		t.VatTotal += a.vat
	}
	t.Total = t.Subtotal - t.DiscountTotal + t.VatTotal
	return out, t, maxDiscount, nil
}
