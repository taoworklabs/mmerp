package purchase

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
// The rules are Sales'; products never share code, so this is a copy without its catalogue discount.
func compute(lines []LineInput, mode platform.Rounding) ([]amounts, Totals, error) {
	out := make([]amounts, len(lines))
	nets := make([]int64, len(lines))
	rates := make([]string, len(lines))
	for i, l := range lines {
		qty, ok := record.ParseNumber(l.Quantity)
		if !ok || !qty.IsPositive() {
			return nil, Totals{}, errLine("invalid_quantity", i+1)
		}
		pct, ok := record.ParseNumber(l.DiscountPercent)
		if !ok || pct.IsNegative() || pct.GreaterThan(hundred) {
			return nil, Totals{}, errLine("invalid_discount", i+1)
		}
		if !tax.Valid(l.VatRate) {
			return nil, Totals{}, errLine("invalid_vat_rate", i+1)
		}
		gross := qty.Mul(decimal.NewFromInt(l.UnitPrice))
		if gross.GreaterThan(maxLine) {
			return nil, Totals{}, errLine("line_amount_too_large", i+1)
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
	return out, t, nil
}
