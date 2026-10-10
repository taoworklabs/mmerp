package hrm_test

import (
	"context"
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/record/recordtest"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
	"github.com/taoworklabs/mmerp/internal/platform"
)

func (f *leaveFixture) timesheet(id int64) hrm.Timesheet {
	f.t.Helper()
	t, err := f.hrm.Timesheet(f.hr, id)
	f.check(err)
	return t
}

func (f *leaveFixture) postedTimesheets() int {
	f.t.Helper()
	l, err := f.hrm.Timesheets(f.hr, hrm.TimesheetFilter{Status: "posted", Paging: platform.Paging{Page: 1, PageSize: 20}})
	f.check(err)
	return int(l.Total)
}

func TestTimesheetRecordContract(t *testing.T) {
	recordtest.Run(t, func(t *testing.T) recordtest.Harness {
		f := newLeaveFixture(t)
		approver := f.ctxFor("approver", "hr@*")
		return recordtest.Harness{
			Record: f.rec, Actor: f.hr, Admin: f.admin, LegalEntity: f.c,
			Create: func(ctx context.Context, date string) (record.Ref, error) {
				id, err := f.hrm.CreateTimesheet(ctx, hrm.NewTimesheet{OrgUnitID: f.a, Month: date[:7]})
				return record.Ref{Type: "hrm.timesheet", ID: id}, err
			},
			Edit: func(ctx context.Context, r record.Ref, version int32, date string) error {
				return f.hrm.SaveTimesheet(ctx, r.ID, hrm.TimesheetUpdate{Version: version, Month: date[:7], Lines: []hrm.TimesheetLine{}})
			},
			RequireApproval: func(*testing.T) {
				f.check(f.appr.SaveRule(f.admin, "hrm.timesheet", approval.RuleInput{
					Steps: []approval.Step{{Approver: approval.Approver{Kind: "role", Product: "hrm", Role: "hr"}}}, MaxLevels: 3, FallbackProduct: "hrm", FallbackRole: "hr",
				}))
			},
			Approve:      func(instance int64) error { return f.appr.Approve(approver, instance, 1) },
			BreakPosting: func(_ *testing.T, r record.Ref) { f.closePeriod("2026-03-01", "2026-03-31", r.ID) },
			Effects:      func(*testing.T, record.Ref) int { return f.postedTimesheets() },
		}
	})
}

func TestTimesheet(t *testing.T) {
	f := newLeaveFixture(t)
	// Late joins on 10/03; Gone left before March; Other is in department B.
	late := employee("LATE", f.a)
	late.HireDate = "2026-03-10"
	lateID := must(t)(f.hrm.CreateEmployee(f.hr, late))
	gone := employee("GONE", f.a)
	gone.TerminationDate = str("2026-02-27")
	must(t)(f.hrm.CreateEmployee(f.hr, gone))
	other := must(t)(f.hrm.CreateEmployee(f.hr, employee("OTHER", f.b)))

	id := must(t)(f.hrm.CreateTimesheet(f.hr, hrm.NewTimesheet{OrgUnitID: f.a, Month: "2026-03"}))
	_, err := f.hrm.CreateTimesheet(f.hr, hrm.NewTimesheet{OrgUnitID: f.a, Month: "2026-03"})
	wantErr(t, err, "timesheet_exists")
	ts := f.timesheet(id)
	if ts.Number != "BC-2026-00001" || ts.PeriodStart != "2026-03-01" || ts.PeriodEnd != "2026-03-31" || ts.Status != "draft" {
		t.Fatalf("%+v", ts)
	}
	var codes []string
	for _, e := range ts.Employees {
		codes = append(codes, e.Code)
	}
	if len(codes) != 3 || codes[0] != "E" || codes[1] != "LATE" || codes[2] != "M" {
		t.Fatalf("grid %v", codes)
	}
	if !slices.Equal(ts.AllowedActions, []string{"edit", "delete", "submit", "export", "import"}) {
		t.Fatalf("actions %v", ts.AllowedActions)
	}

	save := func(lines ...hrm.TimesheetLine) error {
		return f.hrm.SaveTimesheet(f.hr, id, hrm.TimesheetUpdate{Version: f.timesheet(id).Version, Month: "2026-03", Lines: lines})
	}
	line := func(emp int64, date, days string) hrm.TimesheetLine {
		return hrm.TimesheetLine{EmployeeID: emp, Date: date, Days: days}
	}
	wantErr(t, save(line(f.emp, "2026-04-01", "1")), "timesheet_date_outside_period")
	wantErr(t, save(line(other, "2026-03-02", "1")), "timesheet_employee_not_in_unit")
	wantErr(t, save(line(lateID, "2026-03-09", "1")), "timesheet_date_outside_employment")
	wantErr(t, save(line(f.emp, "2026-03-02", "2")), "invalid_timesheet_days")
	wantErr(t, save(line(f.emp, "2026-03-02", "1"), line(f.emp, "2026-03-02", "0.5")), "timesheet_line_duplicate")
	f.check(save(line(f.emp, "2026-03-02", "1"), line(lateID, "2026-03-10", "0.5")))
	if ts := f.timesheet(id); len(ts.Lines) != 2 || ts.Lines[1] != line(lateID, "2026-03-10", "0.5") || ts.Version != 2 {
		t.Fatalf("%+v", ts)
	}
	// Someone without the timesheet role does not see it.
	if _, err := f.hrm.Timesheet(f.e, id); errCode(err) != "not_found" {
		t.Fatalf("employee reads it: %v", err)
	}

	// Posted, then a closed March stops the cancel; reopened, the cancel frees the unit's March.
	ref := record.Ref{Type: "hrm.timesheet", ID: id}
	f.check(f.rec.Transition(f.hr, ref, 2, record.Posted))
	f.closePeriod("2026-03-01", "2026-03-31", id)
	wantErr(t, f.rec.Transition(f.hr, ref, 3, record.Cancelled), "payroll_period_closed")
	if _, err := f.pool.Exec(f.admin, `DELETE FROM hrm.payroll_periods`); err != nil {
		t.Fatal(err)
	}
	f.check(f.rec.Transition(f.hr, ref, 3, record.Cancelled))
	must(t)(f.hrm.CreateTimesheet(f.hr, hrm.NewTimesheet{OrgUnitID: f.a, Month: "2026-03"}))

	// Nor can a timesheet be posted into a closed period; import follows edit.
	april := must(t)(f.hrm.CreateTimesheet(f.hr, hrm.NewTimesheet{OrgUnitID: f.a, Month: "2026-04"}))
	if !slices.Contains(f.timesheet(april).AllowedActions, "import") {
		t.Fatalf("actions %v", f.timesheet(april).AllowedActions)
	}
	f.closePeriod("2026-04-01", "2026-04-30", april)
	wantErr(t, f.rec.Transition(f.hr, record.Ref{Type: "hrm.timesheet", ID: april}, 1, record.Posted), "payroll_period_closed")
}
