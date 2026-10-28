package hrm

import "testing"

func TestReasonAllowanceTable(t *testing.T) {
	want := map[string]allowanceKind{
		"contract_expired": allowanceSeverance, "work_completed": allowanceSeverance, "mutual_agreement": allowanceSeverance,
		"imprisonment": allowanceSeverance, "foreigner_expelled": allowanceNone, "death": allowanceSeverance,
		"employer_ceased": allowanceSeverance, "dismissal": allowanceNone, "employee_resignation": allowanceSeverance,
		"employer_unilateral": allowanceSeverance, "absent_without_reason": allowanceNone, "restructuring": allowanceJobLoss,
		"work_permit_expired": allowanceNone, "probation_failed": allowanceNone,
	}
	if len(terminationReasons) != len(want) {
		t.Fatalf("%d reasons, want %d", len(terminationReasons), len(want))
	}
	for code, kind := range want {
		if got, ok := reasonAllowance(code); !ok || got != kind {
			t.Errorf("%s: got %q %v, want %q", code, got, ok, kind)
		}
	}
	if _, ok := reasonAllowance("retirement"); ok {
		t.Error("unknown reason accepted")
	}
}

func TestMonthsWorked(t *testing.T) {
	for _, c := range []struct {
		hire, end string
		want      int
	}{
		{"2007-07-01", "2026-04-15", 225}, {"2024-03-01", "2026-04-30", 26},
		{"2025-04-16", "2026-04-15", 12}, {"2025-04-17", "2026-04-15", 11},
		{"2026-01-31", "2026-02-26", 0}, {"2026-01-31", "2026-02-27", 1},
	} {
		if got := monthsWorked(c.hire, c.end); got != c.want {
			t.Errorf("%s..%s: got %d, want %d", c.hire, c.end, got, c.want)
		}
	}
}

func TestAllowanceYears(t *testing.T) {
	for months, want := range map[int]string{0: "0", 1: "0.5", 6: "0.5", 7: "1", 12: "1", 18: "1.5", 19: "2"} {
		if got := allowanceYears(months); !got.Equal(dec(want)) {
			t.Errorf("%d months: got %s, want %s", months, got, want)
		}
	}
}

func TestAllowanceBasis(t *testing.T) {
	for _, c := range []struct {
		b     allowanceBasis
		kind  allowanceKind
		years string
	}{
		{allowanceBasis{Kind: allowanceSeverance, Worked: 11}, allowanceNone, "1"},
		{allowanceBasis{Kind: allowanceSeverance, Worked: 36, Pension: true}, allowanceNone, "3"},
		{allowanceBasis{Kind: allowanceJobLoss, Worked: 36, Pension: true}, allowanceJobLoss, "3"},
		{allowanceBasis{Kind: allowanceSeverance, Worked: 225, Insured: 207}, allowanceSeverance, "1.5"},
		// Every month insured: no year left to pay severance for, so none, not a severance of 0.
		{allowanceBasis{Kind: allowanceSeverance, Worked: 30, Insured: 20, Paid: 20}, allowanceNone, "0"},
		{allowanceBasis{Kind: allowanceSeverance, Worked: 120, Insured: 120}, allowanceNone, "0"},
		// Job loss keeps its two-month floor at zero years.
		{allowanceBasis{Kind: allowanceJobLoss, Worked: 26, Insured: 26}, allowanceJobLoss, "0"},
	} {
		if got := c.b.kind(); got != c.kind {
			t.Errorf("%+v: kind %q, want %q", c.b, got, c.kind)
		}
		if got := c.b.years(); !got.Equal(dec(c.years)) {
			t.Errorf("%+v: years %s, want %s", c.b, got, c.years)
		}
	}
}

func TestAllowanceAmount(t *testing.T) {
	for _, c := range []struct {
		kind           allowanceKind
		years, average string
		want           string
	}{
		{allowanceSeverance, "1.5", "22000000", "16500000"},
		{allowanceJobLoss, "0", "15000000", "30000000"},
		{allowanceJobLoss, "3", "15000000", "45000000"},
		{allowanceNone, "3", "15000000", "0"},
	} {
		if got := allowanceAmount(c.kind, dec(c.years), dec(c.average)); !got.Equal(dec(c.want)) {
			t.Errorf("%s %s × %s: got %s, want %s", c.kind, c.years, c.average, got, c.want)
		}
	}
}
