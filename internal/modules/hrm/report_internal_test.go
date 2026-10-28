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

// A dependant is listed for the months the payroll deducted them: from a set first month, and
// only months the employee was paid in.
func TestDeductedMonths(t *testing.T) {
	s := func(v string) *string { return &v }
	year := []string{"2026-01", "2026-02", "2026-03", "2026-04", "2026-05", "2026-06", "2026-07", "2026-08", "2026-09", "2026-10", "2026-11", "2026-12"}
	for _, c := range []struct {
		from, to    *string
		paid        []string
		first, last string
		ok          bool
	}{
		{s("2025-06"), nil, year, "01/2026", "12/2026", true},
		{s("2026-03"), s("2026-08"), year, "03/2026", "08/2026", true},
		{s("2024-01"), s("2025-12"), year, "", "", false},
		{s("2027-01"), nil, year, "", "", false},
		{nil, nil, year, "", "", false},
		{s("2025-06"), nil, year[:3], "01/2026", "03/2026", true},
		{s("2026-05"), nil, year[:3], "", "", false},
	} {
		first, last, ok := deductedMonths(DependentInput{DeductionFrom: c.from, DeductionTo: c.to}, c.paid)
		if first != c.first || last != c.last || ok != c.ok {
			t.Errorf("%v-%v over %d months: %s %s %v", c.from, c.to, len(c.paid), first, last, ok)
		}
	}
}

func TestQuarterParam(t *testing.T) {
	for in, want := range map[string]bool{"2026-Q3": true, "2026-Q4": true, "2026-Q31": false, "2026-Q0": false, "2026-Q5": false, "26-Q3": false, "2026-Q3 ": false} {
		if _, _, err := parseQuarter(in); (err == nil) != want {
			t.Errorf("%q: %v", in, err)
		}
	}
}
