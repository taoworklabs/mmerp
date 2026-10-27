package hrm_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
)

// printPayslip prints the payslip of employee on payroll as ctx's actor, the job ending
// as want ("completed" or an error code), and returns the PDF of a completed one.
func (f *payrollFixture) printPayslip(ctx context.Context, payroll, employee int64, want string) []byte {
	f.t.Helper()
	job, err := f.core.DataIO.StartExport(ctx, "printing.hrm.payslip", fmt.Appendf(nil, `{"id":%d,"parts":[%d]}`, payroll, employee))
	f.check(err)
	f.wantJob(job, want)
	if want != "completed" {
		return nil
	}
	j, err := f.core.DataIO.Job(ctx, job)
	f.check(err)
	_, file, err := f.core.DataIO.File(ctx, *j.FileID)
	f.check(err)
	defer func() { f.check(file.Close()) }()
	b, err := io.ReadAll(file)
	f.check(err)
	return b
}

// An employee with no role prints their own payslip of a posted payroll, and no one else's.
func TestOwnPayslipPrint(t *testing.T) {
	f := payrollAprilFixture(t)
	id := f.payroll("2026-04")
	f.check(f.send(id))

	if b := f.printPayslip(f.e, id, f.emp, "completed"); !bytes.HasPrefix(b, []byte("%PDF")) {
		t.Fatal("own payslip is no PDF")
	}
	f.printPayslip(f.m, id, f.emp, "not_found")
}
