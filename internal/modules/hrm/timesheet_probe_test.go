package hrm_test

import (
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
)

// Probes for timesheet and overtime situations no rule settles yet: work entered on a day off,
// work and paid leave on the same day, and an overtime day kind that disagrees with the
// calendar. They log what happens and assert only what the existing rules already say. The
// CC and TC codes label the observations.
//
// April 2026 with the payroll fixture: 22 working days, of which 27/04 and 30/04 are public
// holidays; E's salary is 22,000,000, so one hour of overtime is 125,000 before the multiplier.

func (f *payrollFixture) paidDays(id int64) (string, int64) {
	f.t.Helper()
	l := f.payrollDoc(id).Lines[0]
	return l.PaidDays, l.Overtime
}

// CC-04: work entered on a Sunday and on a public holiday, with one weekday left empty.
func TestProbeCC04WorkOnDaysOff(t *testing.T) {
	f := newPayrollFixture(t)
	f.contract(f.emp, sample{salary: 22_000_000}, "2026-01-01", "2026-12-31")
	days := slices.DeleteFunc(workdays("2026-04-01", "2026-04-30", "2026-04-27", "2026-04-30"), func(d string) bool { return d == "2026-04-06" })
	f.timesheet(f.a, "2026-04", map[int64][]string{f.emp: append(days, "2026-04-05", "2026-04-30")})
	paid, _ := f.paidDays(f.payroll("2026-04"))
	t.Logf("CC-04 19 weekdays + Sunday 05/04 + holiday 30/04 entered: the timesheet saved; paid days %s of 22", paid)
}

// CC-03: a full month of work, plus one day of company-paid annual leave on a day also worked.
func TestProbeCC03WorkAndPaidLeaveSameDay(t *testing.T) {
	f := payrollAprilFixture(t)
	f.grant("12")
	id := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-04-06", "2026-04-06", "1")}))
	f.check(f.leaveFixture.send(f.e, id)) // no rule: posted at once
	paid, _ := f.paidDays(f.payroll("2026-04"))
	t.Logf("CC-03 worked every working day and took 1 paid leave day on 06/04: leave %s, balance %s of 12, paid days %s of 22",
		f.leave(id).Status, f.balance(), paid)
}

// TC-01: overtime on the 30/04 public holiday, entered once as a weekday and once as a holiday.
func TestProbeTC01DayKindAgainstCalendar(t *testing.T) {
	f := payrollAprilFixture(t)
	for _, kind := range []string{"weekday", "holiday"} {
		id, err := f.hrm.CreateOvertime(f.hr, hrm.NewOvertime{EmployeeID: &f.emp, OvertimeFields: ot("2026-04-30", kind, "2", "0")})
		if err != nil {
			t.Logf("TC-01 %s on 30/04 refused: %v", kind, err)
			continue
		}
		f.post(record.Ref{Type: "hrm.overtime_request", ID: id}, 1)
		p := f.payroll("2026-04")
		_, amount := f.paidDays(p)
		t.Logf("TC-01 2 hours on the 30/04 holiday entered as %s: accepted, overtime pay %d", kind, amount)
		f.check(f.rec.Transition(f.pay, record.Ref{Type: "hrm.overtime_request", ID: id}, 2, record.Cancelled))
		f.check(f.hrm.DeletePayroll(f.pay, p, f.payrollDoc(p).Version))
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
	line := func(date string) hrm.TimesheetLine { return hrm.TimesheetLine{EmployeeID: f.emp, Date: date, Days: "1"} }
	f.check(f.hrm.SaveTimesheet(f.hr, id, hrm.TimesheetUpdate{Version: 1, Month: "2026-03", Lines: []hrm.TimesheetLine{line("2026-03-02"), line("2026-03-03")}}))
	f.check(f.hrm.SaveTimesheet(f.hr, id, hrm.TimesheetUpdate{Version: 2, Month: "2026-03", Lines: []hrm.TimesheetLine{line("2026-03-04")}}))
	if ts := f.timesheet(id); !slices.Equal(ts.Lines, []hrm.TimesheetLine{line("2026-03-04")}) {
		t.Fatalf("lines %+v", ts.Lines)
	}
}
