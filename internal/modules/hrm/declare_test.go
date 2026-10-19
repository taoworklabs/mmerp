package hrm_test

import (
	"context"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/record"
)

type canCase struct {
	ctx    context.Context
	action record.Action
	want   bool
}

func checkCan(t *testing.T, f *fixture, code string, id int64, cases map[string]canCase) {
	t.Helper()
	for name, c := range cases {
		if ok, err := f.rec.Can(c.ctx, code, id, c.action); err != nil || ok != c.want {
			t.Errorf("%s: %v %v, want %v", name, ok, err, c.want)
		}
	}
}

// A contract's signed scan and its print show the salary: both need hrm.salary.view on top
// of the contract's own permission. Attaching is allowed once posted too.
func TestContractFiles(t *testing.T) {
	f := newContractFixture(t)
	id := must(t)(f.original("2026-01-01", "", 10_000_000))
	cases := map[string]canCase{
		"payroll views files": {f.pay, record.ViewFiles, true},
		"payroll attaches":    {f.pay, record.Attach, true},
		"payroll prints":      {f.pay, record.Print, true},
		"hr views files":      {f.hr, record.ViewFiles, false},
		"hr attaches":         {f.hr, record.Attach, false},
		"hr prints":           {f.hr, record.Print, false},
		"hr sends":            {f.hr, record.Post, true},
		"hr edits":            {f.hr, record.Edit, false},
		"manager views":       {f.m, record.View, false},
		"employee views":      {f.e, record.ViewFiles, false},
	}
	checkCan(t, f.fixture, "hrm.contract", id, cases)
	f.check(f.post(id))
	checkCan(t, f.fixture, "hrm.contract", id, map[string]canCase{"payroll attaches posted": {f.pay, record.Attach, true}})
	// An id of another type answers nothing.
	checkCan(t, f.fixture, "hrm.leave_request", id, map[string]canCase{"as a leave request": {f.pay, record.View, false}})
}

// Papers on a profile (an ID card) show what the sensitive fields hold.
func TestEmployeeFiles(t *testing.T) {
	f := newLeaveFixture(t)
	sens := f.ctxFor("sens", "hr@*", "sensitive_viewer@*")
	viewer := f.ctxFor("viewer", "viewer@*", "sensitive_viewer@*")
	checkCan(t, f.fixture, "hrm.employee", f.emp, map[string]canCase{
		"hr views":            {f.hr, record.View, true},
		"hr edits":            {f.hr, record.Edit, true},
		"hr views files":      {f.hr, record.ViewFiles, false},
		"hr attaches":         {f.hr, record.Attach, false},
		"sensitive views":     {sens, record.ViewFiles, true},
		"sensitive attaches":  {sens, record.Attach, true},
		"viewer views files":  {viewer, record.ViewFiles, true},
		"viewer attaches":     {viewer, record.Attach, false},
		"employee views self": {f.e, record.View, false},
	})
	checkCan(t, f.fixture, "hrm.employee", f.emp+1000, map[string]canCase{"missing": {f.hr, record.View, false}})
	if a, err := f.hrm.EmployeeActions(f.hr); err != nil || !equal(a, "create", "import_leave_balances") {
		t.Fatalf("hr before an employee: %v %v", a, err)
	}
}

// A payroll has no attachments; its payslips need hrm.salary.view.
func TestPayrollFiles(t *testing.T) {
	f := payrollAprilFixture(t)
	viewer := f.ctxFor("pviewer", "payroll_viewer@*")
	id := f.payroll("2026-04")
	checkCan(t, f.fixture, "hrm.payroll", id, map[string]canCase{
		"payroll views files": {f.pay, record.ViewFiles, false},
		"payroll attaches":    {f.pay, record.Attach, false},
		"payroll prints":      {f.pay, record.Print, true},
		"payroll exports":     {f.pay, record.Export, true},
		"viewer views":        {viewer, record.View, true},
		"viewer prints":       {viewer, record.Print, false},
	})
}
