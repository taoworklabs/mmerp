package platform_test

import (
	"slices"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/platform"
)

func TestRound(t *testing.T) {
	for in, want := range map[string]int64{"0.5": 1, "-0.5": -1, "1.49": 1, "-2.5": -3, "2.4999": 2, "0": 0} {
		if got := platform.Round(decimal.RequireFromString(in)); got != want {
			t.Errorf("Round(%s) = %d, want %d", in, got, want)
		}
	}
}

func TestAllocate(t *testing.T) {
	for _, c := range []struct {
		parts []string
		want  []int64
	}{
		{[]string{"0.6", "0.6"}, []int64{1, 0}},           // 1.2 rounds to 1
		{[]string{"0.5", "0.5", "0.5"}, []int64{1, 1, 0}}, // equal fractions: earlier first
		{[]string{"-0.6", "-0.6"}, []int64{-1, 0}},        // negative
		{[]string{"0.9", "-0.3"}, []int64{1, 0}},          // mixed signs
		{[]string{"0.4", "-0.9"}, []int64{0, -1}},         // -0.5 rounds away from zero
		{[]string{"0.5", "-0.5"}, []int64{0, 0}},          // total zero
		{[]string{"100.3", "200.3", "300.3"}, []int64{101, 200, 300}},
		{nil, []int64{}},
	} {
		parts := make([]decimal.Decimal, len(c.parts))
		for i, p := range c.parts {
			parts[i] = decimal.RequireFromString(p)
		}
		if got := platform.Allocate(parts); !slices.Equal(got, c.want) {
			t.Errorf("Allocate(%v) = %v, want %v", c.parts, got, c.want)
		}
	}
}
