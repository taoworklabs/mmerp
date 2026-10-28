package hrm_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// leaving is an employee of unit A hired on hire, with salary from start and, when raise is
// set, raise from 01/01/2026.
func (f *payrollFixture) leaving(code, hire, start string, salary, raise int64) int64 {
	f.t.Helper()
	e := employee(code, f.a)
	e.HireDate = hire
	emp := must(f.t)(f.hrm.CreateEmployee(f.hr, e))
	end := "2026-12-31"
	id := must(f.t)(f.hrm.CreateContract(f.pay, hrm.NewContract{EmployeeID: emp, ContractFields: hrm.ContractFields{
		ContractTypeID: f.fixed, StartDate: start, EndDate: &end, Terms: hrm.ContractTerms{Salary: salary, Lines: []hrm.ContractLine{}}}}))
	f.post(record.Ref{Type: "hrm.contract", ID: id}, 1)
	if raise > 0 {
		app := must(f.t)(f.hrm.CreateContract(f.pay, hrm.NewContract{EmployeeID: emp, ParentID: &id, ContractFields: hrm.ContractFields{
			StartDate: "2026-01-01", Terms: hrm.ContractTerms{Salary: raise, Lines: []hrm.ContractLine{}}}}))
		f.post(record.Ref{Type: "hrm.contract", ID: app}, 1)
	}
	return emp
}

func (f *payrollFixture) terminate(emp int64, date, reason string, insured int) {
	f.t.Helper()
	in := quit(date, reason)
	in.InsuredMonths = insured
	id := must(f.t)(f.hrm.CreateTermination(f.pay, hrm.NewTermination{EmployeeID: emp, TerminationFields: in}))
	f.post(terminationRef(id), 1)
}

func (f *payrollFixture) line(payroll, emp int64) hrm.PayrollLine {
	f.t.Helper()
	for _, l := range f.payrollDoc(payroll).Lines {
		if l.EmployeeID == emp {
			return l
		}
	}
	f.t.Fatalf("no line of %d", emp)
	return hrm.PayrollLine{}
}

// capturedMonthly is the six monthly amounts the payroll captured for the allowance.
func (f *payrollFixture) capturedMonthly(payroll, emp int64) []int64 {
	f.t.Helper()
	var data []byte
	f.check(f.system.QueryRow(f.admin, `SELECT data FROM hrm.payroll_lines WHERE payroll_id = $1 AND employee_id = $2`, payroll, emp).Scan(&data))
	raw, err := platform.Decrypt(f.admin, "hrm.payroll_lines.data", data)
	f.check(err)
	var l struct {
		Input struct {
			Severance *struct{ Monthly []int64 } `json:"severance"`
		} `json:"input"`
	}
	f.check(json.Unmarshal(raw, &l))
	if l.Input.Severance == nil {
		return nil
	}
	return l.Input.Severance.Monthly
}

var april1to15 = workdays("2026-04-01", "2026-04-15")

func TestPayrollSeverance(t *testing.T) {
	f := newPayrollFixture(t)
	emp := f.leaving("Q1", "2007-07-01", "2025-01-01", 20_000_000, 24_000_000)
	f.terminate(emp, "2026-04-15", "mutual_agreement", 207)
	f.timesheet(f.a, "2026-04", map[int64][]string{emp: april1to15})
	id := f.payroll("2026-04")
	l := f.line(id, emp)
	// 225 months worked − 207 insured = 18 months = 1.5 years × ½ × 22,000,000.
	if l.Severance != 16_500_000 || l.Earned != 12_000_000 || l.Gross != 28_500_000 || l.Exempt != 16_500_000 || l.InsuranceBase != 24_000_000 {
		t.Fatalf("line %+v", l.PayrollAmounts)
	}
	if got := f.capturedMonthly(id, emp); !slices.Equal(got, []int64{20_000_000, 20_000_000, 20_000_000, 24_000_000, 24_000_000, 24_000_000}) {
		t.Fatalf("monthly %v", got)
	}
}

func TestPayrollSeveranceStale(t *testing.T) {
	f := newPayrollFixture(t)
	emp := f.leaving("Q1", "2007-07-01", "2025-01-01", 20_000_000, 24_000_000)
	f.timesheet(f.a, "2026-04", map[int64][]string{emp: april1to15})
	id := f.payroll("2026-04")
	if f.line(id, emp).Severance != 0 {
		t.Fatal("severance before any termination")
	}
	f.terminate(emp, "2026-04-15", "mutual_agreement", 207)
	if !f.payrollDoc(id).SourcesChanged {
		t.Fatal("posting a termination left the payroll current")
	}
	job, err := f.hrm.RecomputePayroll(f.pay, id, f.payrollDoc(id).Version)
	f.check(err)
	f.wantJob(job, "completed")
	if l := f.line(id, emp); l.Severance != 16_500_000 || f.payrollDoc(id).SourcesChanged {
		t.Fatalf("after computing %+v", l.PayrollAmounts)
	}
}

func TestPayrollSeveranceKinds(t *testing.T) {
	f := newPayrollFixture(t)
	fired := f.leaving("Q1", "2007-07-01", "2025-01-01", 20_000_000, 0)
	f.terminate(fired, "2026-04-15", "dismissal", 0)
	// Restructuring with every month insured: years 0, still two months' salary.
	cut := f.leaving("Q2", "2024-03-01", "2024-03-01", 15_000_000, 0)
	f.terminate(cut, "2026-04-30", "restructuring", 26)
	f.timesheet(f.a, "2026-04", map[int64][]string{fired: april1to15, cut: workdays("2026-04-01", "2026-04-30", "2026-04-27", "2026-04-30")})
	id := f.payroll("2026-04")
	if l := f.line(id, fired); l.Severance != 0 {
		t.Fatalf("dismissal pays %d", l.Severance)
	}
	if l := f.line(id, cut); l.Severance != 30_000_000 {
		t.Fatalf("restructuring pays %d", l.Severance)
	}
}
