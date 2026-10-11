package hrm

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

// 2026-04-04 and 05 are a Saturday and a Sunday; 2026-04-30 is a holiday here.
func allocCalendar() calendar {
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

func TestWorkingDays(t *testing.T) {
	c := allocCalendar()
	cases := []struct {
		start, end string
		want       int
	}{
		{"2026-04-03", "2026-04-06", 2}, // Fri, Sat, Sun, Mon
		{"2026-04-03", "2026-04-03", 1},
		{"2026-04-04", "2026-04-05", 0}, // the weekend only
		{"2026-04-30", "2026-04-30", 0}, // the holiday only
		{"2026-04-29", "2026-05-04", 3}, // 04-29, 05-01 and 05-04
		{"2026-04-06", "2026-04-10", 5}, // a whole working week
	}
	for _, c2 := range cases {
		got := c.workingDays(dayOf(c2.start), dayOf(c2.end))
		if got != c2.want {
			t.Errorf("workingDays(%s, %s) = %d, want %d", c2.start, c2.end, got, c2.want)
		}
	}
}

func TestLeaveAllocation(t *testing.T) {
	c := allocCalendar()
	cases := []struct {
		name, start, end, days string
		want                   map[string]string
	}{
		{"a whole range of working days", "2026-04-06", "2026-04-07", "2",
			map[string]string{"2026-04-06": "1", "2026-04-07": "1"}},
		{"skips the weekend in the middle", "2026-04-03", "2026-04-06", "2",
			map[string]string{"2026-04-03": "1", "2026-04-06": "1"}},
		{"a half day lands as a half day", "2026-04-03", "2026-04-06", "1.5",
			map[string]string{"2026-04-03": "1", "2026-04-06": "0.5"}},
		{"skips the holiday", "2026-04-29", "2026-05-04", "3",
			map[string]string{"2026-04-29": "1", "2026-05-01": "1", "2026-05-04": "1"}},
		{"nothing to place on a weekend", "2026-04-04", "2026-04-05", "2", map[string]string{}},
		{"fewer days than the range offers go on the earliest dates", "2026-04-06", "2026-04-10", "2",
			map[string]string{"2026-04-06": "1", "2026-04-07": "1"}},
	}
	for _, tc := range cases {
		got := leaveAllocation(c, tc.start, tc.end, dec(tc.days))
		if len(got) != len(tc.want) {
			t.Errorf("%s: got %s, want %v", tc.name, days(got), tc.want)
			continue
		}
		for d, want := range tc.want {
			if !got[d].Equal(dec(want)) {
				t.Errorf("%s: on %s got %s, want %s", tc.name, d, got[d], want)
			}
		}
	}
}

// The cap on entry is what keeps the allocation whole; without it the remainder is dropped,
// which is what used to cost an employee balance nobody paid them for.
func TestLeaveAllocationDropsWhatDoesNotFit(t *testing.T) {
	got := leaveAllocation(allocCalendar(), "2026-04-03", "2026-04-06", dec("4"))
	if total(got).String() != "2" {
		t.Fatalf("4 days over a Friday-to-Monday range allocate %s, want 2", total(got))
	}
}

func TestLeaveAllocationAddsUpAcrossRequests(t *testing.T) {
	b := payrollBasis{Calendar: allocCalendar(), Leaves: []basisLeave{
		{EmployeeID: 1, Start: "2026-04-06", End: "2026-04-07", Days: dec("2"), Paid: true},
		{EmployeeID: 1, Start: "2026-04-07", End: "2026-04-08", Days: dec("2"), Paid: true},
	}}
	got := leaveByDay(b, 1)
	if !got["2026-04-07"].Equal(dec("2")) {
		t.Fatalf("the shared date holds %s, want 2 before the payroll caps it", got["2026-04-07"])
	}
}

func TestLeaveAllocationSkipsUnpaidInPayroll(t *testing.T) {
	b := payrollBasis{Calendar: allocCalendar(), Leaves: []basisLeave{
		{EmployeeID: 1, Start: "2026-04-06", End: "2026-04-07", Days: dec("2"), Paid: false},
	}}
	if got := leaveByDay(b, 1); len(got) != 0 {
		t.Fatalf("unpaid leave reached the paid days: %s", days(got))
	}
}

func TestWorkingDaysUsesTheWeekVersionInForce(t *testing.T) {
	// Sunday only is off from 2026-03-16, so the Saturday after it is a working day.
	c := calendar{Weeks: []WorkWeek{{EffectiveFrom: "2026-03-16", OffDays: []int{int(time.Sunday)}}}}
	if got := c.workingDays(dayOf("2026-03-21"), dayOf("2026-03-21")); got != 1 {
		t.Errorf("Saturday 2026-03-21 under the Sunday-only week: %d working days, want 1", got)
	}
	if got := c.workingDays(dayOf("2026-03-14"), dayOf("2026-03-14")); got != 0 {
		t.Errorf("Saturday 2026-03-14 before that version: %d working days, want 0", got)
	}
}
