package hrm_test

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// sample is a row of the payroll samples, as the database needs it; the expected amounts come from the workbook.
type sample struct {
	code       string
	salary     int64
	allowance  int64 // insured, in the overtime rate
	support    int64 // neither, taxable
	deps       int
	hire, quit string
	appendix   int64 // C04: the salary from 16/04
	leave      string
	leaveDays  []string
	overtime   []hrm.OvertimeFields
	adjustment int64
	unused     string // leave left on quitting
}

func ot(date, kind, day, night string) hrm.OvertimeFields {
	return hrm.OvertimeFields{Date: date, DayKind: kind, DayHours: day, NightHours: night}
}

func samples() []sample {
	c17 := []hrm.OvertimeFields{}
	for _, d := range []string{"01", "02", "03", "06", "07", "08", "09", "10", "13", "14", "15"} {
		c17 = append(c17, ot("2026-04-"+d, "weekday", "4", "0"))
	}
	return []sample{
		{code: "S01", salary: 8_000_000}, {code: "S02", salary: 18_000_000}, {code: "S03", salary: 30_000_000},
		{code: "S04", salary: 30_000_000, deps: 2}, {code: "S05", salary: 45_000_000, deps: 1}, {code: "S06", salary: 55_000_000},
		{code: "S07", salary: 90_000_000}, {code: "S08", salary: 120_000_000}, {code: "S09", salary: 150_000_000, deps: 3},
		{code: "S10", salary: 22_000_000, support: 8_000_000}, {code: "S11", salary: 22_000_000, allowance: 3_000_000, deps: 1},
		{code: "C01", salary: 20_000_000, hire: "2026-04-16"}, {code: "C02", salary: 20_000_000, hire: "2026-04-22"},
		{code: "C03", salary: 18_000_000, quit: "2026-04-10", unused: "4"},
		{code: "C04", salary: 20_000_000, appendix: 25_000_000},
		{code: "C05", salary: 25_000_000, leave: "unpaid", leaveDays: []string{"2026-04-06", "2026-04-08"}},
		{code: "C06", salary: 25_000_000, leave: "sick", leaveDays: []string{"2026-04-13", "2026-04-17"}},
		{code: "C07", salary: 25_000_000, leave: "annual", leaveDays: []string{"2026-04-21", "2026-04-22"}},
		{code: "C08", salary: 20_000_000, overtime: []hrm.OvertimeFields{ot("2026-04-06", "weekday", "4", "0"), ot("2026-04-07", "weekday", "6", "0")}},
		{code: "C09", salary: 20_000_000, overtime: []hrm.OvertimeFields{ot("2026-04-05", "weekly_off", "8", "0")}},
		{code: "C10", salary: 20_000_000, overtime: []hrm.OvertimeFields{ot("2026-04-30", "holiday", "8", "0")}},
		{code: "C11", salary: 20_000_000, overtime: []hrm.OvertimeFields{ot("2026-04-07", "weekday", "0", "4")}},
		{code: "C12", salary: 60_000_000, deps: 2, overtime: []hrm.OvertimeFields{ot("2026-04-08", "weekday", "12", "0")}},
		{code: "C13", salary: 30_000_000, adjustment: 2_000_000},
		{code: "C14", salary: 30_000_000, adjustment: -1_500_000},
		{code: "C15", salary: 60_000_000}, // the July payroll
		{code: "C16", salary: 20_000_000, allowance: 3_000_000, overtime: []hrm.OvertimeFields{ot("2026-04-09", "weekday", "10", "0")}},
		{code: "C17", salary: 20_000_000, overtime: c17},
	}
}

// expected reads the workbook's row of each code: column letter → whole đồng.
func expected(t *testing.T) map[string]map[string]int64 {
	t.Helper()
	x, err := excelize.OpenFile("../../../docs/payroll-samples/payroll-input-samples.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = x.Close() }()
	rows, err := x.GetRows("Bộ mẫu", excelize.Options{RawCellValue: true})
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]map[string]int64{}
	for _, r := range rows[1:] {
		m := map[string]int64{}
		for i, v := range r {
			col, _ := excelize.ColumnNumberToName(i + 1)
			if d, err := decimal.NewFromString(v); err == nil {
				m[col] = d.Round(0).IntPart()
			}
		}
		out[r[0]] = m
	}
	return out
}

// workdays lists the Monday–Friday dates of [from, to] that are not holidays.
func workdays(from, to string, holidays ...string) []string {
	var out []string
	start, _ := time.Parse(time.DateOnly, from)
	end, _ := time.Parse(time.DateOnly, to)
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		day := d.Format(time.DateOnly)
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday && !slices.Contains(holidays, day) {
			out = append(out, day)
		}
	}
	return out
}

// payrollFixture adds payroll staff, kinds of leave and contract, and the April holidays.
type payrollFixture struct {
	*leaveFixture
	pay                 context.Context
	fixed, sick, unpaid int64
	employees           map[string]int64
}

func newPayrollFixture(t *testing.T) *payrollFixture {
	f := &payrollFixture{leaveFixture: newLeaveFixture(t), employees: map[string]int64{}}
	f.pay = f.ctxFor("pay", "payroll@*", "hr@*", "sensitive_viewer@*", "leave_admin@*")
	f.fixed = must(t)(f.hrm.SaveContractType(f.hr, 0, hrm.ContractTypeInput{Name: "Xác định thời hạn", FixedTerm: true, Active: true}))
	f.check(f.pool.QueryRow(f.admin, `UPDATE hrm.leave_types SET paid = true WHERE id = $1 RETURNING id`, f.annual).Scan(new(int64)))
	f.sick = must(t)(f.hrm.SaveLeaveType(f.hr, 0, hrm.LeaveTypeInput{Name: "Nghỉ ốm", Active: true}))
	f.unpaid = must(t)(f.hrm.SaveLeaveType(f.hr, 0, hrm.LeaveTypeInput{Name: "Nghỉ không lương", Active: true}))
	f.check(f.hrm.SaveHoliday(f.pay, f.c, hrm.Holiday{Date: "2026-04-27", Name: "Nghỉ bù Giỗ Tổ"}))
	f.check(f.hrm.SaveHoliday(f.pay, f.c, hrm.Holiday{Date: "2026-04-30", Name: "Giải phóng miền Nam"}))
	return f
}

func (f *payrollFixture) post(ref record.Ref, version int32) {
	f.t.Helper()
	f.check(f.rec.Transition(f.pay, ref, version, record.Posted))
}

// contract posts a fixed-term contract from start to end, with an appendix from 16/04 when appendix is set.
func (f *payrollFixture) contract(emp int64, s sample, start, end string) {
	f.t.Helper()
	terms := hrm.ContractTerms{Salary: s.salary, Lines: []hrm.ContractLine{}}
	if s.allowance > 0 {
		terms.Lines = append(terms.Lines, hrm.ContractLine{Kind: "allowance", Name: "Trách nhiệm", Amount: s.allowance, Taxable: true})
	}
	if s.support > 0 {
		terms.Lines = append(terms.Lines, hrm.ContractLine{Kind: "support", Name: "Xăng xe", Amount: s.support, Taxable: true})
	}
	id := must(f.t)(f.hrm.CreateContract(f.pay, hrm.NewContract{EmployeeID: emp, ContractFields: hrm.ContractFields{
		ContractTypeID: f.fixed, StartDate: start, EndDate: &end, Terms: terms}}))
	f.post(record.Ref{Type: "hrm.contract", ID: id}, 1)
	if s.appendix > 0 {
		terms.Salary = s.appendix
		app := must(f.t)(f.hrm.CreateContract(f.pay, hrm.NewContract{EmployeeID: emp, ParentID: &id, ContractFields: hrm.ContractFields{StartDate: "2026-04-16", Terms: terms}}))
		f.post(record.Ref{Type: "hrm.contract", ID: app}, 1)
	}
}

// timesheet posts the month's timesheet of unit: a full day on every working day each employee
// worked, leaving out holidays and leave.
func (f *payrollFixture) timesheet(unit int64, month string, worked map[int64][]string) int64 {
	f.t.Helper()
	id := must(f.t)(f.hrm.CreateTimesheet(f.hr, hrm.NewTimesheet{OrgUnitID: unit, Month: month}))
	lines := []hrm.TimesheetLine{}
	for emp, days := range worked {
		for _, d := range days {
			lines = append(lines, hrm.TimesheetLine{EmployeeID: emp, Date: d, Days: "1"})
		}
	}
	f.check(f.hrm.SaveTimesheet(f.hr, id, hrm.TimesheetUpdate{Version: 1, Month: month, Lines: lines}))
	f.post(record.Ref{Type: "hrm.timesheet", ID: id}, 2)
	return id
}

// payroll creates the payroll of month and works its jobs until computed.
func (f *payrollFixture) payroll(month string) int64 {
	f.t.Helper()
	p, err := f.hrm.CreatePayroll(f.pay, hrm.NewPayroll{LegalEntityID: f.c, Month: month})
	f.check(err)
	f.wantJob(p.JobID, "completed")
	return p.ID
}

// wantJob works queued jobs until job ends, and checks how: completed, or the code it failed with.
func (f *fixture) wantJob(job int64, want string) {
	f.t.Helper()
	f.work()
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		j, err := f.jobs.JobGet(f.admin, job)
		f.check(err)
		if j.FinalizedAt == nil {
			continue
		}
		got := string(j.State)
		var out platform.JobOutput
		if json.Unmarshal(j.Output(), &out) == nil && out.Error != nil {
			got = out.Error.Code
		} else if got != "completed" && len(j.Errors) > 0 {
			got = j.Errors[len(j.Errors)-1].Error
		}
		if got != want {
			f.t.Fatalf("job %d: %s, want %s", job, got, want)
		}
		return
	}
	f.t.Fatalf("job %d did not end", job)
}

func (f *fixture) work() {
	f.t.Helper()
	if f.started {
		return
	}
	f.started = true
	f.check(f.jobs.Start(f.t.Context()))
	f.t.Cleanup(func() { _ = f.jobs.Stop(context.Background()) })
}

func (f *fixture) check(err error) {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
}

// The payroll samples are the acceptance test of payroll: every money column of every case,
// to the đồng, computed by the job from data entered as a user would.
func TestPayrollSamples(t *testing.T) {
	f := newPayrollFixture(t)
	want := expected(t)
	holidays := []string{"2026-04-27", "2026-04-30"}
	april, july := map[int64][]string{}, map[int64][]string{}
	var adjustments []hrm.PayrollAdjustmentInput
	for _, s := range samples() {
		unit := f.a
		if s.code == "C15" {
			unit = f.b
		}
		e := employee(s.code, unit)
		e.HireDate = "2026-01-01"
		if s.hire != "" {
			e.HireDate = s.hire
		}
		if s.quit != "" {
			e.TerminationDate = &s.quit
		}
		emp := must(t)(f.hrm.CreateEmployee(f.hr, e))
		f.employees[s.code] = emp
		if s.code == "C15" {
			f.contract(emp, s, "2026-07-01", "2026-12-31")
			july[emp] = workdays("2026-07-01", "2026-07-31")
			continue
		}
		f.contract(emp, s, "2026-01-01", "2026-06-30")
		for range s.deps {
			must(t)(f.hrm.SaveDependent(f.pay, emp, 0, hrm.DependentInput{FullName: "Con", Relationship: "child", DeductionFrom: new("2026-01")}))
		}
		days := workdays(max(e.HireDate, "2026-04-01"), min(*cmpOr(e.TerminationDate, "2026-04-30"), "2026-04-30"), holidays...)
		if s.leave != "" {
			types := map[string]int64{"annual": f.annual, "sick": f.sick, "unpaid": f.unpaid}
			if s.leave == "annual" {
				f.check(f.hrm.AdjustLeaveBalance(f.pay, emp, 2026, hrm.BalanceAdjustment{Delta: "12", Reason: "đầu năm"}))
			}
			off := workdays(s.leaveDays[0], s.leaveDays[1])
			id := must(t)(f.hrm.CreateLeave(f.hr, hrm.NewLeave{EmployeeID: &emp, LeaveFields: hrm.LeaveFields{
				LeaveTypeID: types[s.leave], StartDate: s.leaveDays[0], EndDate: s.leaveDays[1], Days: fmt.Sprint(len(off))}}))
			f.post(record.Ref{Type: "hrm.leave_request", ID: id}, 1)
			days = slices.DeleteFunc(days, func(d string) bool { return slices.Contains(off, d) })
		}
		if s.unused != "" {
			f.check(f.hrm.AdjustLeaveBalance(f.pay, emp, 2026, hrm.BalanceAdjustment{Delta: s.unused, Reason: "đầu năm"}))
		}
		for _, o := range s.overtime {
			id := must(t)(f.hrm.CreateOvertime(f.hr, hrm.NewOvertime{EmployeeID: &emp, OvertimeFields: o}))
			f.post(record.Ref{Type: "hrm.overtime_request", ID: id}, 1)
		}
		if s.adjustment != 0 {
			adjustments = append(adjustments, hrm.PayrollAdjustmentInput{EmployeeID: emp, Amount: s.adjustment, SourcePeriod: "2026-03", Reason: "Kỳ 03/2026"})
		}
		april[emp] = days
	}
	f.timesheet(f.a, "2026-04", april)
	f.timesheet(f.b, "2026-07", july)

	id := f.payroll("2026-04")
	job, err := f.hrm.SavePayrollAdjustments(f.pay, id, hrm.PayrollAdjustments{Version: f.payrollDoc(id).Version, Items: adjustments})
	f.check(err)
	f.wantJob(job, "completed")
	// Computing opened the dependents, a read of sensitive data audited like any other.
	if f.auditCount("hrm.dependents_viewed") == 0 {
		t.Fatal("computing read the dependents unaudited")
	}
	lines := map[int64]hrm.PayrollLine{}
	for _, l := range f.payrollDoc(id).Lines {
		lines[l.EmployeeID] = l
	}
	for _, l := range f.payrollDoc(f.payroll("2026-07")).Lines {
		lines[l.EmployeeID] = l
	}

	for _, s := range samples() {
		l, ok := lines[f.employees[s.code]]
		w := want[s.code]
		if !ok || w == nil {
			t.Errorf("%s: no line or no expected row", s.code)
			continue
		}
		paid, _ := decimal.NewFromString(l.PaidDays)
		for col, got := range map[string]int64{
			"I": int64(l.StandardDays), "J": paid.IntPart(), "K": l.Earned, "N": l.Overtime, "O": l.UnusedLeave, "P": l.Adjustment,
			"Q": l.Gross, "R": l.Exempt, "S": l.InsuranceBase, "T": l.SocialInsurance, "U": l.HealthInsurance, "V": l.UnemploymentIns,
			"W": l.PersonalDeduction, "X": l.DependentDeduction, "Y": l.Taxable, "Z": l.IncomeTax, "AA": l.Net,
			"AB": l.EmployerSocial, "AC": l.EmployerHealth, "AD": l.EmployerUnemploy, "AE": l.EmployerUnion, "AF": l.Cost,
		} {
			if got != w[col] {
				t.Errorf("%s column %s: got %d, want %d", s.code, col, got, w[col])
			}
		}
		if s.code == "C17" && !slices.Contains(l.Warnings, "overtime_month_limit") {
			t.Errorf("C17: no overtime warning: %v", l.Warnings)
		}
	}
}

func cmpOr(p *string, def string) *string {
	if p == nil {
		return &def
	}
	return p
}

func (f *payrollFixture) payrollDoc(id int64) hrm.Payroll {
	f.t.Helper()
	p, err := f.hrm.Payroll(f.pay, id)
	f.check(err)
	return p
}
