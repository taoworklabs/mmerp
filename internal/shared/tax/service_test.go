package tax

import (
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/platform"
)

func TestVATPerLine(t *testing.T) {
	// Each line rounds alone: 10,005 × 8 % = 800.4 → 800 (twice); 72,916 × 10 % = 7,291.6 → 7,292.
	got := VAT([]int64{10_005, 10_005, 72_916}, []string{"8", "8", "10"}, platform.RoundLine)
	if want := []int64{800, 800, 7_292}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestVATTotalAllocates(t *testing.T) {
	got := VAT([]int64{10_005, 10_005}, []string{"8", "8"}, platform.RoundTotal)
	if want := []int64{801, 800}; !slices.Equal(got, want) {
		t.Fatalf("same rate: got %v, want %v", got, want)
	}
	// Rates round apart: 50.5 → 51 and 100.5 → 101; none has no VAT.
	got = VAT([]int64{1_010, 1_005, 0}, []string{"5", "10", "none"}, platform.RoundTotal)
	if want := []int64{51, 101, 0}; !slices.Equal(got, want) {
		t.Fatalf("rates apart: got %v, want %v", got, want)
	}
}

func TestValid(t *testing.T) {
	if !Valid("10") || !Valid("none") || Valid("7") {
		t.Fatal("Valid wrong")
	}
}
