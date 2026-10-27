package hrm_test

import (
	"testing"
	"time"
)

// Last month: A and B each have an employee, only A has a posted timesheet; the entity's
// payroll closes once it is posted. Counts are nil without the permission.
func TestMonthClose(t *testing.T) {
	f := newPayrollFixture(t)
	mc, err := f.hrm.MonthClose(f.admin)
	f.check(err)
	start, err := time.Parse(time.DateOnly, mc.PeriodStart)
	f.check(err)
	month, last := start.Format("2006-01"), start.AddDate(0, 1, -1).Format(time.DateOnly)

	f.contract(f.emp, sample{salary: 22_000_000}, "2026-01-01", "2099-12-31")
	must(t)(f.hrm.CreateEmployee(f.hr, employee("B1", f.b)))
	f.timesheet(f.a, month, map[int64][]string{f.emp: workdays(mc.PeriodStart, last)})

	check := func(ctx_ string, got, want *int) {
		t.Helper()
		if (got == nil) != (want == nil) || got != nil && *got != *want {
			t.Fatalf("%s: got %v, want %v", ctx_, got, want)
		}
	}
	one, zero := 1, 0
	mc, err = f.hrm.MonthClose(f.pay)
	f.check(err)
	check("timesheets", mc.OpenTimesheetUnits, &one)
	check("payrolls", mc.OpenPayrollEntities, &one)

	f.check(f.send(f.payroll(month)))
	mc, err = f.hrm.MonthClose(f.pay)
	f.check(err)
	check("payrolls after posting", mc.OpenPayrollEntities, &zero)

	mc, err = f.hrm.MonthClose(f.e)
	f.check(err)
	check("no permission timesheets", mc.OpenTimesheetUnits, nil)
	check("no permission payrolls", mc.OpenPayrollEntities, nil)
}
