package hrm_test

import (
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// How the timesheet, leave and overtime reach the payroll, and the warnings for entries the
// payroll treats differently from what they say.
//
// April 2026 with the payroll fixture: 22 working days, of which 27/04 and 30/04 are public
// holidays; E's salary is 22,000,000, so one hour of overtime is 125,000 before the multiplier.

func (f *payrollFixture) paidDays(id int64) (string, int64) {
	f.t.Helper()
	l := f.payrollDoc(id).Lines[0]
	return l.PaidDays, l.Overtime
}

// Work entered on a day off counts for nothing in the payroll, so the timesheet warns about
// each such cell: a Sunday, and a holiday (paid anyway, the work earns nothing more).
func TestTimesheetWarnsOfWorkOnDaysOff(t *testing.T) {
	f := newPayrollFixture(t)
	f.contract(f.emp, sample{salary: 22_000_000}, "2026-01-01", "2026-12-31")
	days := slices.DeleteFunc(workdays("2026-04-01", "2026-04-30", "2026-04-27", "2026-04-30"), func(d string) bool { return d == "2026-04-06" })
	id := f.timesheet(f.a, "2026-04", map[int64][]string{f.emp: append(days, "2026-04-05", "2026-04-30")})
	if w := f.leaveFixture.timesheet(id).Warnings; !slices.Equal(w, []hrm.TimesheetWarning{
		{EmployeeID: f.emp, Date: "2026-04-05", Code: "work_on_day_off"},
		{EmployeeID: f.emp, Date: "2026-04-30", Code: "work_on_day_off"},
	}) {
		t.Fatalf("warnings %+v", w)
	}
	if paid, _ := f.paidDays(f.payroll("2026-04")); paid != "21" {
		t.Fatalf("paid days %s, want 19 worked + 2 holidays", paid)
	}
}

// Work and approved leave on one day: the balance loses the day and the payroll pays nothing
// more, so both the timesheet cell and the payroll line warn.
func TestWorkOnALeaveDayWarns(t *testing.T) {
	f := payrollAprilFixture(t)
	f.grant("12")
	id := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-04-06", "2026-04-06", "1")}))
	f.check(f.leaveFixture.send(f.e, id)) // no rule: posted at once
	ts, err := f.hrm.Timesheets(f.hr, hrm.TimesheetFilter{Paging: platform.Paging{Page: 1, PageSize: 20}})
	f.check(err)
	if w := f.leaveFixture.timesheet(ts.Items[0].ID).Warnings; !slices.Equal(w, []hrm.TimesheetWarning{{EmployeeID: f.emp, Date: "2026-04-06", Code: "work_on_leave_day"}}) {
		t.Fatalf("timesheet warnings %+v", w)
	}
	l := f.payrollDoc(f.payroll("2026-04")).Lines[0]
	if l.PaidDays != "22" || !slices.Contains(l.Warnings, "work_on_leave_day") || f.balance() != "11" {
		t.Fatalf("payroll line %s %v, balance %s", l.PaidDays, l.Warnings, f.balance())
	}
}

// TC-03: a cancelled overtime request is no longer a source; asserted, since only posted
// documents count as sources under the existing rules.
func TestCancelledOvertimeLeavesThePayroll(t *testing.T) {
	f := payrollAprilFixture(t)
	id := f.overtime("2026-04-06")
	f.post(record.Ref{Type: "hrm.overtime_request", ID: id}, 1)
	if _, amount := f.paidDays(f.payroll("2026-04")); amount != 375_000 {
		t.Fatalf("posted: overtime %d, want 2h × 125,000 × 150%%", amount)
	}
	f.check(f.rec.Transition(f.pay, record.Ref{Type: "hrm.overtime_request", ID: id}, 2, record.Cancelled))
	if _, amount := f.paidDays(f.payroll("2026-04")); amount != 0 {
		t.Fatalf("cancelled: overtime %d, want 0", amount)
	}
}

// CC-06: saving a draft timesheet replaces every line, so a shorter file drops the rest.
func TestTimesheetSaveReplacesEveryLine(t *testing.T) {
	f := newLeaveFixture(t)
	id := must(t)(f.hrm.CreateTimesheet(f.hr, hrm.NewTimesheet{OrgUnitID: f.a, Month: "2026-03"}))
	line := func(date string) hrm.TimesheetLine {
		return hrm.TimesheetLine{EmployeeID: f.emp, Date: date, Days: "1"}
	}
	f.check(f.hrm.SaveTimesheet(f.hr, id, hrm.TimesheetUpdate{Version: 1, Month: "2026-03", Lines: []hrm.TimesheetLine{line("2026-03-02"), line("2026-03-03")}}))
	f.check(f.hrm.SaveTimesheet(f.hr, id, hrm.TimesheetUpdate{Version: 2, Month: "2026-03", Lines: []hrm.TimesheetLine{line("2026-03-04")}}))
	if ts := f.timesheet(id); !slices.Equal(ts.Lines, []hrm.TimesheetLine{line("2026-03-04")}) {
		t.Fatalf("lines %+v", ts.Lines)
	}
}
