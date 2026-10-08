package hrm_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/record/recordtest"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
)

func payrollRef(id int64) record.Ref { return record.Ref{Type: "hrm.payroll", ID: id} }

// heldBy counts the periods a payroll holds.
func (f *fixture) heldBy(id int64) int {
	f.t.Helper()
	var n int
	f.check(f.pool.QueryRow(f.admin, `SELECT count(*) FROM hrm.payroll_periods WHERE posted_payroll_id = $1`, id).Scan(&n))
	return n
}

func TestPayrollRecordContract(t *testing.T) {
	recordtest.Run(t, func(t *testing.T) recordtest.Harness {
		f := newPayrollFixture(t)
		approver := f.ctxFor("approver", "payroll@*")
		create := func(ctx context.Context, month string) (record.Ref, error) {
			p, err := f.hrm.CreatePayroll(ctx, hrm.NewPayroll{LegalEntityID: f.c, Month: month})
			if err == nil {
				f.wantJob(p.JobID, "completed")
			}
			return payrollRef(p.ID), err
		}
		return recordtest.Harness{
			Record: f.rec, Actor: f.pay, Admin: f.admin, LegalEntity: f.c,
			Create: func(ctx context.Context, date string) (record.Ref, error) { return create(ctx, date[:7]) },
			// A payroll never moves to another period: within its month an edit is saving its
			// adjustments, and moving it is left to record, which refuses a locked date first.
			Edit: func(ctx context.Context, r record.Ref, version int32, date string) error {
				d, err := f.rec.Get(f.admin, r)
				if err != nil {
					return err
				}
				if d.Date[:7] != date[:7] {
					_, err = f.rec.Edit(ctx, r, version, record.Header{Date: date, OrgUnitID: f.c})
					return err
				}
				_, err = f.hrm.SavePayrollAdjustments(ctx, r.ID, hrm.PayrollAdjustments{Version: version, Items: []hrm.PayrollAdjustmentInput{}})
				return err
			},
			RequireApproval: func(*testing.T) {
				f.check(f.appr.SaveRule(f.admin, "hrm.payroll", approval.RuleInput{
					Steps: []approval.Step{{Approver: approval.Approver{Kind: "role", Product: "hrm", Role: "payroll"}}}, MaxLevels: 1, FallbackProduct: "hrm", FallbackRole: "payroll",
				}))
			},
			Approve: func(instance int64) error { return f.appr.Approve(approver, instance, 1) },
			// Another payroll of the period is already posted.
			BreakPosting: func(_ *testing.T, r record.Ref) {
				d, err := f.rec.Get(f.admin, r)
				f.check(err)
				other, err := create(f.pay, d.Date[:7])
				f.check(err)
				f.closePeriod(d.Date[:8]+"01", d.Date, other.ID)
			},
			Effects: func(_ *testing.T, r record.Ref) int { return f.heldBy(r.ID) },
		}
	})
}

// payrollAprilFixture: E in department A with a contract and an April timesheet.
func payrollAprilFixture(t *testing.T) *payrollFixture {
	f := newPayrollFixture(t)
	f.contract(f.emp, sample{salary: 22_000_000}, "2026-01-01", "2026-12-31")
	f.timesheet(f.a, "2026-04", map[int64][]string{f.emp: workdays("2026-04-01", "2026-04-30", "2026-04-27", "2026-04-30")})
	return f
}

func (f *payrollFixture) overtime(date string) int64 {
	f.t.Helper()
	id := must(f.t)(f.hrm.CreateOvertime(f.hr, hrm.NewOvertime{EmployeeID: &f.emp, OvertimeFields: ot(date, "weekday", "2", "0")}))
	return id
}

func (f *payrollFixture) send(id int64) error {
	return f.rec.Transition(f.pay, payrollRef(id), f.payrollDoc(id).Version, record.Posted)
}

func TestPayrollLifecycle(t *testing.T) {
	f := payrollAprilFixture(t)
	viewer := f.ctxFor("accounting", "payroll_viewer@*")

	// May has no timesheet yet.
	_, err := f.hrm.CreatePayroll(f.pay, hrm.NewPayroll{LegalEntityID: f.c, Month: "2026-05"})
	wantErr(t, err, "payroll_timesheets_missing")
	if units := errParams(err)["org_units"].([]map[string]any); len(units) != 1 || units[0]["id"] != f.a {
		t.Fatalf("units %v", units)
	}
	_, err = f.hrm.CreatePayroll(f.hr, hrm.NewPayroll{LegalEntityID: f.c, Month: "2026-04"})
	wantErr(t, err, "forbidden")

	// Creating computes, so it needs the salary permission too.
	if a, _ := f.hrm.PayrollActions(viewer); len(a) != 0 {
		t.Fatalf("viewer may %v", a)
	}
	if a, _ := f.hrm.PayrollActions(f.pay); !slices.Equal(a, []string{"create"}) {
		t.Fatalf("pay may %v", a)
	}
	first, second := f.payroll("2026-04"), f.payroll("2026-04")
	p := f.payrollDoc(first)
	if p.Number != "BL-2026-00001" || p.ComputedAt == nil || p.SourcesChanged || len(p.Lines) != 1 || p.Lines[0].Earned != 22_000_000 ||
		!slices.Equal(p.AllowedActions, []string{"edit", "delete", "submit", "compute", "adjust", "export", "print"}) {
		t.Fatalf("payroll %+v", p)
	}
	if d, err := f.rec.Get(f.admin, payrollRef(first)); err != nil || d.Amount == nil || *d.Amount != p.Totals[0].Cost {
		t.Fatalf("document amount %v, want the total cost %d: %v", d.Amount, p.Totals[0].Cost, err)
	}
	// Totals by department only, without anyone's amounts, and no audited read.
	before := f.auditCount("hrm.payroll_lines_viewed")
	v, err := f.hrm.Payroll(viewer, first)
	f.check(err)
	if v.Lines != nil || v.Adjustments != nil || len(v.Totals) != 1 || v.Totals[0].Gross != 22_000_000 || v.Totals[0].OrgUnitName != "A" ||
		f.auditCount("hrm.payroll_lines_viewed") != before || slices.Contains(v.AllowedActions, "export") {
		t.Fatalf("viewer sees %+v", v)
	}
	if _, err := f.hrm.Payroll(f.hr, first); errCode(err) != "not_found" {
		t.Fatalf("hr reads it: %v", err)
	}
	// A one-person department's totals are that person's pay: nothing stored in clear.
	var row string
	f.check(f.pool.QueryRow(f.admin, `SELECT row_to_json(p)::text FROM hrm.payrolls p WHERE id = $1`, first).Scan(&row))
	if strings.Contains(row, "22000000") {
		t.Fatalf("payroll row %s", row)
	}

	// A new overtime request changes the sources: no sending until computed again.
	ot1 := f.overtime("2026-04-06")
	f.post(record.Ref{Type: "hrm.overtime_request", ID: ot1}, 1)
	if !f.payrollDoc(first).SourcesChanged {
		t.Fatal("sources unchanged")
	}
	wantErr(t, f.send(first), "payroll_sources_changed")
	job, err := f.hrm.RecomputePayroll(f.pay, first, f.payrollDoc(first).Version)
	f.check(err)
	f.wantJob(job, "completed")
	if p := f.payrollDoc(first); p.SourcesChanged || p.Lines[0].Overtime != 375_000 {
		t.Fatalf("after computing: %+v", p.Lines)
	}
	f.check(f.send(first))
	if f.heldBy(first) != 1 {
		t.Fatal("period not held")
	}

	// Posting lines: by department and legal entity, balanced, no line per person.
	type line struct {
		kind string
		unit int64
		sum  int64
	}
	var lines []line
	rows, err := f.pool.Query(f.admin, `SELECT kind, org_unit_id, amount FROM posting.lines WHERE doc_type = 'hrm.payroll' AND doc_id = $1 AND voided_at IS NULL ORDER BY id`, first)
	f.check(err)
	for rows.Next() {
		var l line
		f.check(rows.Scan(&l.kind, &l.unit, &l.sum))
		lines = append(lines, l)
	}
	f.check(rows.Err())
	p = f.payrollDoc(first)
	want := []line{{"salary_expense", f.a, p.Totals[0].Gross}, {"employer_insurance_expense", f.a, 22_000_000 * 235 / 1000},
		{"salary_payable", f.a, p.Totals[0].Net}, {"insurance_payable", f.c, 22_000_000 * 340 / 1000}, {"pit_payable", f.c, p.Totals[0].IncomeTax}}
	if !slices.Equal(lines, want) {
		t.Fatalf("posting lines %+v, want %+v", lines, want)
	}

	// One posted payroll per period; the other may not post, nor cancel what it does not hold.
	f.check(f.payrollJob(second))
	wantErr(t, f.send(second), "payroll_already_posted")
	// 11': the period is closed to its sources.
	ot2 := f.overtime("2026-04-07")
	wantErr(t, f.rec.Transition(f.pay, record.Ref{Type: "hrm.overtime_request", ID: ot2}, 1, record.Posted), "payroll_period_closed")
	// 13: nor can the timesheet be cancelled.
	ts, err := f.hrm.Timesheets(f.hr, hrm.TimesheetFilter{Month: "2026-04", Page: 1, PageSize: 20})
	f.check(err)
	wantErr(t, f.rec.Transition(f.hr, record.Ref{Type: "hrm.timesheet", ID: ts.Items[0].ID}, 3, record.Cancelled), "payroll_period_closed")

	// Cancelling reopens the period and voids the lines; then the second one posts.
	f.check(f.rec.Transition(f.pay, payrollRef(first), f.payrollDoc(first).Version, record.Cancelled))
	var voided int
	f.check(f.pool.QueryRow(f.admin, `SELECT count(*) FROM posting.lines WHERE doc_id = $1 AND voided_at IS NOT NULL`, first).Scan(&voided))
	if voided != 5 || f.heldBy(first) != 0 {
		t.Fatalf("voided %d, held %d", voided, f.heldBy(first))
	}
	f.check(f.send(second))
	// A posted payroll not holding its period (only through bad data) cannot be cancelled.
	third := f.payroll("2026-04")
	f.check(f.pool.QueryRow(f.admin, `UPDATE hrm.payroll_periods SET posted_payroll_id = $1 RETURNING id`, third).Scan(new(int64)))
	wantErr(t, f.rec.Transition(f.pay, payrollRef(second), f.payrollDoc(second).Version, record.Cancelled), "payroll_not_holding_period")
}

// payrollJob computes a draft again and waits.
func (f *payrollFixture) payrollJob(id int64) error {
	job, err := f.hrm.RecomputePayroll(f.pay, id, f.payrollDoc(id).Version)
	if err == nil {
		f.wantJob(job, "completed")
	}
	return err
}

// A source changing while the payroll awaits approval fails the last approval; the instance stays open.
func TestPayrollSourcesChangeWhilePending(t *testing.T) {
	f := payrollAprilFixture(t)
	approver := f.ctxFor("approver", "payroll@*")
	f.check(f.appr.SaveRule(f.admin, "hrm.payroll", approval.RuleInput{
		Steps: []approval.Step{{Approver: approval.Approver{Kind: "role", Product: "hrm", Role: "payroll"}}}, MaxLevels: 1, FallbackProduct: "hrm", FallbackRole: "payroll",
	}))
	id := f.payroll("2026-04")
	f.check(f.send(id))
	d, err := f.rec.Get(f.admin, payrollRef(id))
	f.check(err)
	f.post(record.Ref{Type: "hrm.overtime_request", ID: f.overtime("2026-04-06")}, 1)
	wantErr(t, f.appr.Approve(approver, *d.ApprovalTicket, 1), "payroll_sources_changed")
	if d, _ := f.rec.Get(f.admin, payrollRef(id)); d.Status != record.PendingApproval || f.heldBy(id) != 0 {
		t.Fatalf("after the failed approval: %+v", d)
	}
	// Withdraw, compute, send again.
	f.check(f.rec.Transition(f.pay, payrollRef(id), d.Version, record.Draft))
	f.check(f.payrollJob(id))
	f.check(f.send(id))
	d, _ = f.rec.Get(f.admin, payrollRef(id))
	f.check(f.appr.Approve(approver, *d.ApprovalTicket, 1))
	if f.heldBy(id) != 1 {
		t.Fatal("not posted")
	}
}

// Adjustments are sensitive, recompute the payroll, and leave it uncomputed until then.
func TestPayrollAdjustments(t *testing.T) {
	f := payrollAprilFixture(t)
	id := f.payroll("2026-04")
	items := []hrm.PayrollAdjustmentInput{{EmployeeID: f.emp, Amount: -500_000, SourcePeriod: "2026-03", Reason: "Trả thừa"}}
	_, err := f.hrm.SavePayrollAdjustments(f.pay, id, hrm.PayrollAdjustments{Version: 2, Items: []hrm.PayrollAdjustmentInput{{EmployeeID: f.emp, Amount: 1, SourcePeriod: "2026-04", Reason: "x"}}})
	wantErr(t, err, "invalid_payroll_adjustment")
	// Nor for someone of another legal entity, whose pay would be read into this payroll.
	other := must(t)(f.iam.CreateOrgUnit(f.admin, iam.OrgUnitInput{Kind: "company", Name: "C2"}))
	stranger := must(t)(f.hrm.CreateEmployee(f.pay, employee("X", other)))
	_, err = f.hrm.SavePayrollAdjustments(f.pay, id, hrm.PayrollAdjustments{Version: 2, Items: []hrm.PayrollAdjustmentInput{{EmployeeID: stranger, Amount: 1, SourcePeriod: "2026-03", Reason: "x"}}})
	wantErr(t, err, "invalid_payroll_adjustment")
	job, err := f.hrm.SavePayrollAdjustments(f.pay, id, hrm.PayrollAdjustments{Version: 2, Items: items})
	f.check(err)
	if p := f.payrollDoc(id); p.ComputedAt != nil || p.Version != 3 {
		t.Fatalf("before the job: %+v", p)
	}
	wantErr(t, f.send(id), "payroll_not_computed")
	f.wantJob(job, "completed")
	p := f.payrollDoc(id)
	if p.Lines[0].Adjustment != -500_000 || p.Lines[0].Gross != 21_500_000 || len(p.Adjustments) != 1 || p.Adjustments[0].Reason != "Trả thừa" {
		t.Fatalf("after: %+v %+v", p.Lines, p.Adjustments)
	}
	if n := f.auditCount("hrm.payroll_adjusted"); n != 1 {
		t.Fatalf("audited %d", n)
	}
}

// Six working days a week make 27 in July 2026; the standard is capped at 26, and so is the pay.
func TestPayrollSixDayWeek(t *testing.T) {
	f := newPayrollFixture(t)
	f.check(f.hrm.SaveWorkWeek(f.pay, f.c, hrm.WorkWeek{EffectiveFrom: "2026-01-01", OffDays: []int{0}}))
	f.contract(f.emp, sample{salary: 26_000_000}, "2026-01-01", "2026-12-31")
	var days []string
	for d := 1; d <= 31; d++ {
		date := fmt.Sprintf("2026-07-%02d", d)
		if day, _ := time.Parse(time.DateOnly, date); day.Weekday() != time.Sunday {
			days = append(days, date)
		}
	}
	f.timesheet(f.a, "2026-07", map[int64][]string{f.emp: days})
	l := f.payrollDoc(f.payroll("2026-07")).Lines[0]
	if l.StandardDays != 26 || l.PaidDays != "26" || l.Earned != 26_000_000 || l.SocialInsurance == 0 {
		t.Fatalf("%+v", l)
	}
}
