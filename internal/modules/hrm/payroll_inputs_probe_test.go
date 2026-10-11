package hrm

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

// Probe for how a leave request spanning a weekend or a public holiday reaches the paid
// days, and for what an unpaid type does. 2026-04-04 and 05 are Saturday and Sunday;
// 2026-04-30 is a holiday here.
func probeCalendar() calendar {
	return calendar{Holidays: map[string]string{"2026-04-30": "Giải phóng miền Nam"}}
}

func days(m map[string]decimal.Decimal) string {
	var b []string
	for _, d := range slices.Sorted(maps.Keys(m)) {
		b = append(b, fmt.Sprintf("%s=%s", d, m[d]))
	}
	if len(b) == 0 {
		return "(none)"
	}
	return strings.Join(b, " ")
}

func total(m map[string]decimal.Decimal) decimal.Decimal {
	sum := decimal.Zero
	for _, v := range m {
		sum = sum.Add(v)
	}
	return sum
}

func TestProbeNP04(t *testing.T) {
	cases := []struct {
		name, start, end, entered string
	}{
		{"friday to monday, 4 days entered", "2026-04-03", "2026-04-06", "4"},
		{"friday to monday, 2 days entered", "2026-04-03", "2026-04-06", "2"},
		{"over a holiday and a weekend, 6 days", "2026-04-29", "2026-05-04", "6"},
		{"half day from friday", "2026-04-03", "2026-04-06", "1.5"},
		{"the weekend only, 2 days entered", "2026-04-04", "2026-04-05", "2"},
		{"the holiday only, 1 day entered", "2026-04-30", "2026-04-30", "1"},
	}
	for _, c := range cases {
		b := payrollBasis{Calendar: probeCalendar(), Leaves: []basisLeave{
			{EmployeeID: 1, Start: c.start, End: c.end, Days: dec(c.entered), Paid: true},
		}}
		got := leaveByDay(b, 1)
		t.Logf("NP-04 %-40s entered %-4s → paid %-4s on %s", c.name, c.entered, total(got), days(got))
	}
}

func TestProbeNP10UnpaidType(t *testing.T) {
	b := payrollBasis{Calendar: probeCalendar(), Leaves: []basisLeave{
		{EmployeeID: 1, Start: "2026-04-06", End: "2026-04-07", Days: dec("2"), Paid: false},
	}}
	got := leaveByDay(b, 1)
	t.Logf("NP-10 unpaid leave on two working days → paid %s on %s", total(got), days(got))
}
