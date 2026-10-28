package hrm

import "testing"

// The worked figures of Decree 145/2020, Article 66 on a Saturday-and-Sunday-off calendar.
func TestEntitlement(t *testing.T) {
	cal := calendar{Holidays: map[string]string{}}
	str := func(s string) *string { return &s }
	cases := []struct {
		name        string
		hire        string
		termination *string
		month       int
		want        string
	}{
		{"full year", "2023-01-01", nil, 12, "12"},
		{"first month", "2023-01-01", nil, 1, "1"},
		{"before any month", "2023-01-01", nil, 0, "0"},
		// Five years complete on 2026-07-01: June is still the fourth year.
		{"seniority step mid-year", "2021-07-01", nil, 12, "12.5"},
		{"before the step", "2021-07-01", nil, 6, "6"},
		{"three steps", "2010-01-01", nil, 12, "15"},
		{"one step all year", "2020-01-01", nil, 12, "13"},
		{"1.25 rounds down", "2010-01-01", nil, 1, "1"},
		{"2.5 stays", "2010-01-01", nil, 2, "2.5"},
		// March 2026 has 22 working days; from the 10th it keeps 16, from the 20th only 8.
		{"joined before mid-March", "2026-03-10", nil, 12, "10"},
		{"joined after mid-March", "2026-03-20", nil, 12, "9"},
		// April 1–5 holds 3 working days of 22; April 1–20 holds 14.
		{"left early April", "2023-01-01", str("2026-04-05"), 12, "3"},
		{"left late April", "2023-01-01", str("2026-04-20"), 12, "4"},
		{"one working day of November", "2026-11-30", nil, 12, "1"},
		{"hired after the year", "2027-02-01", nil, 12, "0"},
	}
	for _, c := range cases {
		if got := entitlementThrough(cal, c.hire, c.termination, 2026, c.month); got.String() != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
	for m := 1; m <= 12; m++ {
		if got := monthCredit(cal, "2023-01-01", nil, 2026, m); got.String() != "1" {
			t.Errorf("month %d credits %s, want 1", m, got)
		}
	}
}
