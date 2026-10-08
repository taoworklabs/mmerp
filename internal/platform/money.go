package platform

import (
	"slices"

	"github.com/shopspring/decimal"
)

// Rounding is where a document rounds an amount: each line, or once on the total.
type Rounding string

const (
	RoundLine  Rounding = "line"
	RoundTotal Rounding = "total"
)

// Round rounds to a whole minor unit, half away from zero; every rounding of money goes through it.
func Round(d decimal.Decimal) int64 { return d.Round(0).IntPart() }

// Allocate rounds the total of parts once and gives each part a whole amount summing to it:
// each part keeps its integer part (toward zero), and the units left go one by one to the
// parts with the largest fractions (the smallest, when the units are negative); on a tie
// the earlier part goes first.
func Allocate(parts []decimal.Decimal) []int64 {
	out := make([]int64, len(parts))
	if len(parts) == 0 {
		return out
	}
	total, sum := decimal.Zero, int64(0)
	frac := make([]decimal.Decimal, len(parts))
	for i, p := range parts {
		total = total.Add(p)
		out[i] = p.IntPart()
		sum += out[i]
		frac[i] = p.Sub(decimal.NewFromInt(out[i]))
	}
	left := Round(total) - sum
	step := int64(1)
	if left < 0 {
		step, left = -1, -left
	}
	order := make([]int, len(parts))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int { return frac[b].Cmp(frac[a]) * int(step) })
	for i := int64(0); i < left; i++ {
		out[order[i%int64(len(order))]] += step
	}
	return out
}
