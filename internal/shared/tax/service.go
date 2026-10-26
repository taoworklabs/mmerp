// Package tax holds the VAT rates and the VAT computation shared by every product.
package tax

import (
	"slices"

	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// Rates are the VAT rate codes a line may carry; none is not subject to VAT.
var Rates = []string{"none", "0", "5", "8", "10"}

var hundred = decimal.NewFromInt(100)

// Valid reports whether rate is a known rate code.
func Valid(rate string) bool { return slices.Contains(Rates, rate) }

// VAT is the VAT of each line's net amount. RoundLine rounds each line; RoundTotal works out
// each rate once over the document and allocates it back to that rate's lines.
func VAT(nets []int64, rates []string, mode platform.Rounding) []int64 {
	out := make([]int64, len(nets))
	vats := make([]decimal.Decimal, len(nets)) // unrounded
	for i, n := range nets {
		vats[i] = decimal.NewFromInt(n).Mul(fraction(rates[i]))
	}
	if mode != platform.RoundTotal {
		for i, v := range vats {
			out[i] = platform.Round(v)
		}
		return out
	}
	for _, rate := range Rates {
		var idx []int
		var parts []decimal.Decimal
		for i, r := range rates {
			if r == rate {
				idx, parts = append(idx, i), append(parts, vats[i])
			}
		}
		for j, v := range platform.Allocate(parts) {
			out[idx[j]] = v
		}
	}
	return out
}

// fraction is a rate code as a fraction; none is zero.
func fraction(code string) decimal.Decimal {
	if code == "none" {
		return decimal.Zero
	}
	return decimal.RequireFromString(code).Div(hundred)
}
