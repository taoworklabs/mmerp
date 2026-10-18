package record_test

import (
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/record"
)

func TestParseNumberRefusesExponents(t *testing.T) {
	for _, s := range []string{"1e999999999", "1E5", "", "1.", ".5", "1234567890123456"} {
		if _, ok := record.ParseNumber(s); ok {
			t.Errorf("%q accepted", s)
		}
	}
	if d, ok := record.ParseNumber("-2.5"); !ok || d.String() != "-2.5" {
		t.Errorf("-2.5: %v %v", d, ok)
	}
}
