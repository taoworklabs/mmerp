package hrm

import "testing"

// The yearly tax is the 2026 monthly schedule with its bounds × 12.
func TestFinaliseUsesYearSchedule(t *testing.T) {
	b, ok := parseBrackets(`[[10000000,0.05],[30000000,0.1],[60000000,0.2],[100000000,0.3],[null,0.35]]`)
	if !ok {
		t.Fatal("brackets")
	}
	// 120M × 5% + 240M × 10% + 140M × 20%.
	p := pitPerson{TNCT: 700_000_000, Deductions: 186_000_000, Insurance: 14_000_000}
	for _, c := range []struct {
		tax         int64
		over, under int64
		waived      bool
	}{{57_960_000, 0, 40_000, true}, {58_100_000, 100_000, 0, false}, {57_900_000, 0, 100_000, false}, {58_000_000, 0, 0, false}} {
		p.Tax = c.tax
		got := finalise(p, b)
		if got.Assessable != 500_000_000 || got.Due != 58_000_000 || got.Over != c.over || got.Under != c.under || got.Waived != c.waived {
			t.Errorf("tax %d: %+v", c.tax, got)
		}
	}
}

func TestClipMonths(t *testing.T) {
	s := func(v string) *string { return &v }
	for _, c := range []struct {
		from, to    *string
		first, last string
		ok          bool
	}{
		{s("2025-06"), nil, "01/2026", "12/2026", true},
		{s("2026-03"), s("2026-08"), "03/2026", "08/2026", true},
		{s("2024-01"), s("2025-12"), "", "", false},
		{s("2027-01"), nil, "", "", false},
		{nil, nil, "01/2026", "12/2026", true},
	} {
		first, last, ok := clipMonths(c.from, c.to, 2026)
		if first != c.first || last != c.last || ok != c.ok {
			t.Errorf("%v-%v: %s %s %v", c.from, c.to, first, last, ok)
		}
	}
}
