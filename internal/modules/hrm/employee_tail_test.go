package hrm

import "testing"

func TestDigitTail(t *testing.T) {
	str := func(s string) *string { return &s }
	for _, c := range []struct {
		in   *string
		want string // "" for nil
	}{
		{nil, ""},
		{str("079123456789"), "6789"},
		{str("Vietcombank 0123 456 789"), "6789"},
		{str("0123456789 - VCB"), "6789"},
		{str("C1234567"), ""}, // seven digits: never mostly disclosed
		{str("12345678"), "5678"},
	} {
		got := digitTail(c.in)
		if (got == nil) != (c.want == "") || (got != nil && *got != c.want) {
			t.Errorf("digitTail(%v) = %v, want %q", c.in, got, c.want)
		}
	}
}
