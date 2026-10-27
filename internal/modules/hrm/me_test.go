package hrm_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
	"github.com/taoworklabs/mmerp/internal/platform"
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

// E sees the posted payroll only, with their own net; a cancelled one is gone.
func TestMyPayslips(t *testing.T) {
	f := payrollAprilFixture(t)
	posted, draft := f.payroll("2026-04"), f.payroll("2026-04")
	f.check(f.send(posted))
	list, err := f.hrm.MyPayslips(f.e)
	f.check(err)
	net := f.payrollDoc(posted).Lines[0].Net
	if len(list) != 1 || list[0] != (hrm.MyPayslipSummary{PayrollID: posted, PeriodStart: "2026-04-01", Net: net}) {
		t.Fatalf("payslips %+v, want %d net %d", list, posted, net)
	}
	if n := f.auditCount("hrm.own_payslips_viewed"); n != 1 {
		t.Fatalf("list audited %d", n)
	}
	f.check(errOf(f.rec.Transition(f.pay, payrollRef(posted), f.payrollDoc(posted).Version, record.Cancelled)))
	if list, err := f.hrm.MyPayslips(f.e); err != nil || len(list) != 0 {
		t.Fatalf("after cancelling: %+v %v", list, err)
	}
	if _, err := f.hrm.MyPayslip(f.e, posted); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("cancelled detail: %v", err)
	}
	if _, err := f.hrm.MyPayslip(f.e, draft); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("draft detail: %v", err)
	}
}

// The payslip on screen is the payslip printed.
func TestMyPayslipMatchesPrint(t *testing.T) {
	f := payrollAprilFixture(t)
	id := f.payroll("2026-04")
	job, err := f.hrm.SavePayrollAdjustments(f.pay, id, hrm.PayrollAdjustments{Version: 2,
		Items: []hrm.PayrollAdjustmentInput{{EmployeeID: f.emp, Amount: -500_000, SourcePeriod: "2026-03", Reason: "Trả thừa"}}})
	f.check(err)
	f.wantJob(job, "completed")
	f.check(f.send(id))
	got, err := f.hrm.MyPayslip(f.e, id)
	f.check(err)
	parts, err := f.hrm.Payslips(f.pay, id)
	f.check(err)
	var want hrm.MyPayslip
	if len(parts) != 1 || parts[0].Key != f.emp {
		t.Fatalf("parts %+v", parts)
	}
	f.check(json.Unmarshal(parts[0].Data, &want))
	want.PayrollID = id
	if !reflect.DeepEqual(got, want) || len(got.Adjustments) != 1 {
		t.Fatalf("payslip %+v, printed %+v", got, want)
	}
}

// M has no line on E's payroll: it does not exist for them.
func TestMyPayslipOtherEmployee(t *testing.T) {
	f := payrollAprilFixture(t)
	id := f.payroll("2026-04")
	f.check(f.send(id))
	if _, err := f.hrm.MyPayslip(f.m, id); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("other's payslip: %v", err)
	}
	if list, err := f.hrm.MyPayslips(f.m); err != nil || len(list) != 0 {
		t.Fatalf("other's list: %+v %v", list, err)
	}
}

// Reading one's payslip is audited on the payroll, without any amount.
func TestMyPayslipAudited(t *testing.T) {
	f := payrollAprilFixture(t)
	id := f.payroll("2026-04")
	f.check(f.send(id))
	p, err := f.hrm.MyPayslip(f.e, id)
	f.check(err)
	var data string
	f.check(f.system.QueryRow(f.admin, `SELECT coalesce(data::text, '') FROM audit.log WHERE action = 'hrm.own_payslip_viewed' AND doc_id = $1`, id).Scan(&data))
	if n := f.auditCount("hrm.own_payslip_viewed"); n != 1 || strings.Contains(data, fmt.Sprint(p.Amounts.Net)) || strings.Contains(data, fmt.Sprint(p.Amounts.Gross)) {
		t.Fatalf("audited %d: %s", n, data)
	}
}

// E's June: worked days from the draft timesheet, posted leave and overtime, the calendar.
func TestMyDays(t *testing.T) {
	f := newPayrollFixture(t)
	f.check(f.hrm.SaveHoliday(f.pay, f.c, hrm.Holiday{Date: "2026-06-05", Name: "Lễ"}))
	f.grant("12")
	b := must(t)(f.hrm.CreateEmployee(f.hr, employee("B", f.a)))
	ts := must(t)(f.hrm.CreateTimesheet(f.hr, hrm.NewTimesheet{OrgUnitID: f.a, Month: "2026-06"}))
	f.check(f.hrm.SaveTimesheet(f.hr, ts, hrm.TimesheetUpdate{Version: 1, Month: "2026-06", Lines: []hrm.TimesheetLine{
		{EmployeeID: f.emp, Date: "2026-06-01", Days: "1"}, {EmployeeID: f.emp, Date: "2026-06-02", Days: "0.5"},
		{EmployeeID: b, Date: "2026-06-08", Days: "1"}}}))
	leave := must(t)(f.hrm.CreateLeave(f.hr, hrm.NewLeave{EmployeeID: &f.emp, LeaveFields: f.fields("2026-06-03", "2026-06-03", "1")}))
	f.post(record.Ref{Type: "hrm.leave_request", ID: leave}, 1)
	f.post(record.Ref{Type: "hrm.overtime_request", ID: f.overtime("2026-06-04")}, 1)
	// Neither a draft request nor someone else's posted one shows.
	must(t)(f.hrm.CreateOvertime(f.hr, hrm.NewOvertime{EmployeeID: &f.emp, OvertimeFields: ot("2026-06-09", "weekday", "1", "0")}))
	other := must(t)(f.hrm.CreateOvertime(f.hr, hrm.NewOvertime{EmployeeID: &b, OvertimeFields: ot("2026-06-10", "weekday", "1", "0")}))
	f.post(record.Ref{Type: "hrm.overtime_request", ID: other}, 1)

	m, err := f.hrm.MyDays(f.e, "2026-06")
	f.check(err)
	if m.Month != "2026-06" || m.StandardDays != 22 || len(m.Days) != 30 {
		t.Fatalf("month %s, %d standard days, %d days", m.Month, m.StandardDays, len(m.Days))
	}
	want := map[string]hrm.MyDay{
		"2026-06-01": {Date: "2026-06-01", Kind: "work", Worked: str("1")},
		"2026-06-02": {Date: "2026-06-02", Kind: "work", Worked: str("0.5")},
		"2026-06-03": {Date: "2026-06-03", Kind: "work", Leave: &hrm.MyDayLeave{TypeName: "Phép năm", Days: "1", Paid: true}},
		"2026-06-04": {Date: "2026-06-04", Kind: "work", OvertimeHours: str("2")},
		"2026-06-05": {Date: "2026-06-05", Kind: "holiday"},
		"2026-06-06": {Date: "2026-06-06", Kind: "rest"},
		"2026-06-08": {Date: "2026-06-08", Kind: "work"},
		"2026-06-09": {Date: "2026-06-09", Kind: "work"},
		"2026-06-10": {Date: "2026-06-10", Kind: "work"},
	}
	for _, d := range m.Days {
		if w, ok := want[d.Date]; ok && !reflect.DeepEqual(d, w) {
			t.Fatalf("%s: %+v, want %+v", d.Date, d, w)
		}
	}

	f.post(record.Ref{Type: "hrm.timesheet", ID: ts}, 2)
	m, err = f.hrm.MyDays(f.e, "2026-06")
	f.check(err)
	if d := m.Days[0]; !d.WorkedClosed || *d.Worked != "1" {
		t.Fatalf("after posting: %+v", d)
	}
}

// The profile holds the ordinary fields and the contract in force, nothing sensitive.
func TestMyProfile(t *testing.T) {
	f := newPayrollFixture(t)
	f.contract(f.emp, sample{salary: 10_000_000}, "2020-01-01", "2099-12-31")
	p, err := f.hrm.MyProfile(f.e)
	f.check(err)
	want := hrm.MyProfile{EmployeeID: f.emp, Code: "E", FullName: "Nhân viên E", OrgUnitName: "A", ManagerName: str("Nhân viên M"),
		HireDate: "2026-01-05", Contract: &hrm.MyContract{TypeName: "Xác định thời hạn", StartDate: "2020-01-01", EndDate: str("2099-12-31")}}
	if !reflect.DeepEqual(p, want) {
		t.Fatalf("profile %+v, want %+v", p, want)
	}
	b, err := json.Marshal(p)
	f.check(err)
	for _, k := range []string{"id_number", "national_id", "tax_code", "bank_account", "salary", "terms"} {
		if strings.Contains(string(b), k) {
			t.Fatalf("profile has %s: %s", k, b)
		}
	}
}

// An account linked to no employee has no personal space.
func TestMeNotAnEmployee(t *testing.T) {
	f := newFixture(t)
	x := f.ctxFor("x")
	_, err := f.hrm.MyProfile(x)
	wantErr(t, err, "not_an_employee")
	_, err = f.hrm.MyDays(x, "2026-06")
	wantErr(t, err, "not_an_employee")
	_, err = f.hrm.MyPayslips(x)
	wantErr(t, err, "not_an_employee")
	_, err = f.hrm.MyPayslip(x, 1)
	wantErr(t, err, "not_an_employee")
}
