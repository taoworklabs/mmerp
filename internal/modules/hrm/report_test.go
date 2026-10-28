package hrm_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// reportFixture is a payroll fixture whose E has a social insurance number and a tax code.
type reportFixture struct {
	*payrollFixture
	rep context.Context // payroll and sensitive_viewer, tenant-wide
}

func newReportFixture(t *testing.T) *reportFixture {
	f := &reportFixture{payrollFixture: newPayrollFixture(t)}
	f.rep = f.ctxFor("rep", "payroll@*", "sensitive_viewer@*")
	f.setIdentifiers(f.emp, "0123456789", "8000000001", "001099000001")
	return f
}

func (f *reportFixture) setIdentifiers(emp int64, si, tax, id string) {
	f.t.Helper()
	e, err := f.hrm.Employee(f.pay, emp)
	f.check(err)
	sv := &hrm.SensitiveValues{SocialInsuranceNo: &si, TaxCode: &tax, NationalID: &id}
	f.check(f.hrm.UpdateEmployee(f.pay, emp, hrm.EmployeeInput{EmployeeFields: e.EmployeeFields, Sensitive: sv}))
}

// edit changes an employee's plain fields.
func (f *reportFixture) edit(emp int64, change func(*hrm.EmployeeFields)) {
	f.t.Helper()
	e, err := f.hrm.Employee(f.pay, emp)
	f.check(err)
	change(&e.EmployeeFields)
	f.check(f.hrm.UpdateEmployee(f.pay, emp, hrm.EmployeeInput{EmployeeFields: e.EmployeeFields}))
}

// hire creates an employee in unit, hired on date, with a contract to the end of the year.
func (f *reportFixture) hire(code string, unit int64, date string, salary int64) int64 {
	f.t.Helper()
	e := employee(code, unit)
	e.HireDate = date
	id := must(f.t)(f.hrm.CreateEmployee(f.hr, e))
	f.contract(id, sample{salary: salary}, date, "2026-12-31")
	return id
}

// appendix posts an addendum to emp's contract with a new salary from date.
func (f *reportFixture) appendix(emp int64, date string, salary int64) {
	f.t.Helper()
	var parent int64
	f.check(f.system.QueryRow(f.admin, `SELECT id FROM hrm.contracts WHERE employee_id = $1 AND parent_id IS NULL ORDER BY start_date DESC LIMIT 1`, emp).Scan(&parent))
	id := must(f.t)(f.hrm.CreateContract(f.pay, hrm.NewContract{EmployeeID: emp, ParentID: &parent, ContractFields: hrm.ContractFields{
		StartDate: date, Terms: hrm.ContractTerms{Salary: salary, Lines: []hrm.ContractLine{}}}}))
	f.post(record.Ref{Type: "hrm.contract", ID: id}, 1)
}

// close posts each unit's timesheet of month, then the legal entity's payroll of month.
func (f *reportFixture) close(entity int64, month string, units map[int64]map[int64][]string) int64 {
	f.t.Helper()
	for unit, worked := range units {
		f.timesheet(unit, month, worked)
	}
	p, err := f.hrm.CreatePayroll(f.pay, hrm.NewPayroll{LegalEntityID: entity, Month: month})
	f.check(err)
	f.wantJob(p.JobID, "completed")
	f.check(f.send(p.ID))
	return p.ID
}

// full is every working day of month.
func full(month string) []string { return workdays(month+"-01", lastDay(month)) }

func lastDay(month string) string {
	switch month[5:] {
	case "02":
		return month + "-28"
	case "04", "06", "09", "11":
		return month + "-30"
	}
	return month + "-31"
}

// exportReport runs a report as ctx's actor and returns its rows and file name.
func (f *fixture) exportReport(ctx context.Context, kind, params string) ([][]string, string) {
	f.t.Helper()
	job, err := f.core.DataIO.StartExport(ctx, kind, []byte(params))
	f.check(err)
	f.wantJob(job, "completed")
	j, err := f.core.DataIO.Job(ctx, job)
	f.check(err)
	name, file, err := f.core.DataIO.File(ctx, *j.FileID)
	f.check(err)
	defer func() { f.check(file.Close()) }()
	b, err := io.ReadAll(file)
	f.check(err)
	x, err := excelize.OpenReader(bytes.NewReader(b))
	f.check(err)
	rows, err := x.GetRows(x.GetSheetName(0), excelize.Options{RawCellValue: true})
	f.check(err)
	return rows, name
}

// reportJob starts a report and works it until it ends as want; it returns the job.
func (f *fixture) reportJob(ctx context.Context, kind, params, want string) int64 {
	f.t.Helper()
	job, err := f.core.DataIO.StartExport(ctx, kind, []byte(params))
	f.check(err)
	f.wantJob(job, want)
	return job
}

// rowOf finds the row whose column col holds v.
func rowOf(t *testing.T, rows [][]string, col int, v string) []string {
	t.Helper()
	for _, r := range rows {
		if len(r) > col && r[col] == v {
			return pad(r, 25)
		}
	}
	t.Fatalf("no row with %q in column %d: %v", v, col, rows)
	return nil
}

func pad(r []string, n int) []string {
	for len(r) < n {
		r = append(r, "")
	}
	return r
}

// D02-LT columns, 0-based: B name, (n) at n+1.
const (
	dName = 1
	dSI   = 2
	dSal  = 5
	dFix  = 12
	dEff  = 14
	dFrom = 15
	dTo   = 16
	dKind = 19
	dNote = 20
)

func wantCells(t *testing.T, row []string, want map[int]string) {
	t.Helper()
	for col, v := range want {
		if row[col] != v {
			t.Errorf("column %d = %q, want %q (row %v)", col, row[col], v, row)
		}
	}
}

// May to June in C: E's salary rises, T left at the end of May, N joins on 1 June, U works
// five days of June, too few to contribute.
func TestD02LTMovements(t *testing.T) {
	f := newReportFixture(t)
	f.contract(f.emp, sample{salary: 22_000_000}, "2026-01-01", "2026-12-31")
	quit := f.hire("T", f.a, "2026-01-05", 15_000_000)
	f.edit(quit, func(e *hrm.EmployeeFields) { e.TerminationDate = str("2026-05-31") })
	off := f.hire("U", f.a, "2026-01-05", 12_000_000)
	f.close(f.c, "2026-05", map[int64]map[int64][]string{f.a: {f.emp: full("2026-05"), quit: full("2026-05"), off: full("2026-05")}})
	f.appendix(f.emp, "2026-06-01", 25_000_000)
	joined := f.hire("N", f.a, "2026-06-01", 10_000_000)
	f.close(f.c, "2026-06", map[int64]map[int64][]string{f.a: {f.emp: full("2026-06"), joined: full("2026-06"), off: workdays("2026-06-01", "2026-06-05")}})

	rows, name := f.exportReport(f.rep, "hrm.report.d02lt", `{"legal_entity_id":`+itoa(f.c)+`,"month":"2026-06"}`)
	if !strings.Contains(name, "D02-LT") || !strings.Contains(name, "06-2026") {
		t.Errorf("file name %q", name)
	}
	if !strings.HasPrefix(rows[0][dKind], "(18)") || len(rows) != 5 {
		t.Fatalf("rows %v", rows)
	}
	t.Run("increase", func(t *testing.T) {
		wantCells(t, rowOf(t, rows, dName, "Nhân viên N"), map[int]string{dKind: "Tăng lao động", dEff: "01/06/2026", dFrom: "06/2026", dFix: "x", dSal: "10000000"})
	})
	t.Run("decrease", func(t *testing.T) {
		wantCells(t, rowOf(t, rows, dName, "Nhân viên T"), map[int]string{dKind: "Giảm lao động", dEff: "31/05/2026", dFrom: "06/2026"})
	})
	t.Run("too few days", func(t *testing.T) {
		wantCells(t, rowOf(t, rows, dName, "Nhân viên U"), map[int]string{dKind: "Nghỉ không lương, ốm đau từ 14 ngày", dFrom: "06/2026", dTo: "06/2026"})
	})
	t.Run("adjustment", func(t *testing.T) {
		r := rowOf(t, rows, dName, "Nhân viên E")
		wantCells(t, r, map[int]string{dKind: "Điều chỉnh tiền lương", dEff: "01/06/2026", dSal: "25000000", dSI: "0123456789"})
		if !strings.Contains(r[dNote], "22.000.000") || !strings.Contains(r[dNote], "25.000.000") {
			t.Errorf("note %q", r[dNote])
		}
	})
	// Increases, then decreases, then adjustments.
	if got := []string{rows[1][dKind], rows[2][dKind], rows[4][dKind]}; got[0] != "Tăng lao động" || got[2] != "Điều chỉnh tiền lương" {
		t.Errorf("order %v", got)
	}

	// Back at work in July after a month without contributions.
	f.close(f.c, "2026-07", map[int64]map[int64][]string{f.a: {f.emp: full("2026-07"), joined: full("2026-07"), off: full("2026-07")}})
	rows, _ = f.exportReport(f.rep, "hrm.report.d02lt", `{"legal_entity_id":`+itoa(f.c)+`,"month":"2026-07"}`)
	wantCells(t, rowOf(t, rows, dName, "Nhân viên U"), map[int]string{dKind: "Đi làm lại", dEff: "01/07/2026", dFrom: "07/2026"})
}

// H joins on 25 May: too few days in May to contribute, so June is the increase, not a return.
func TestD02LTLateHire(t *testing.T) {
	f := newReportFixture(t)
	f.contract(f.emp, sample{salary: 22_000_000}, "2026-01-01", "2026-12-31")
	late := f.hire("H", f.a, "2026-05-25", 10_000_000)
	f.close(f.c, "2026-05", map[int64]map[int64][]string{f.a: {f.emp: full("2026-05"), late: workdays("2026-05-25", "2026-05-31")}})
	f.close(f.c, "2026-06", map[int64]map[int64][]string{f.a: {f.emp: full("2026-06"), late: full("2026-06")}})
	rows, _ := f.exportReport(f.rep, "hrm.report.d02lt", `{"legal_entity_id":`+itoa(f.c)+`,"month":"2026-06"}`)
	wantCells(t, rowOf(t, rows, dName, "Nhân viên H"), map[int]string{dKind: "Tăng lao động", dEff: "25/05/2026", dFrom: "06/2026"})
}

// G joins on 1 May but works too few days to contribute: June is an increase, not a return,
// since G was never insured here.
func TestD02LTFirstMonthWaived(t *testing.T) {
	f := newReportFixture(t)
	f.contract(f.emp, sample{salary: 22_000_000}, "2026-01-01", "2026-12-31")
	f.close(f.c, "2026-04", map[int64]map[int64][]string{f.a: {f.emp: full("2026-04")}})
	g := f.hire("G", f.a, "2026-05-01", 10_000_000)
	f.close(f.c, "2026-05", map[int64]map[int64][]string{f.a: {f.emp: full("2026-05"), g: workdays("2026-05-04", "2026-05-05")}})
	f.close(f.c, "2026-06", map[int64]map[int64][]string{f.a: {f.emp: full("2026-06"), g: full("2026-06")}})
	rows, _ := f.exportReport(f.rep, "hrm.report.d02lt", `{"legal_entity_id":`+itoa(f.c)+`,"month":"2026-06"}`)
	wantCells(t, rowOf(t, rows, dName, "Nhân viên G"), map[int]string{dKind: "Tăng lao động", dEff: "01/05/2026", dFrom: "06/2026"})
}

// E moves from C to C2 on 1 June: a decrease for C, an increase for C2.
func TestD02LTMoveBetweenEntities(t *testing.T) {
	f := newReportFixture(t)
	c2 := must(t)(f.iam.CreateOrgUnit(f.admin, iam.OrgUnitInput{Kind: "company", Name: "C2"}))
	d2 := must(t)(f.iam.CreateOrgUnit(f.admin, iam.OrgUnitInput{ParentID: &c2, Kind: "department", Name: "D2"}))
	if _, err := f.system.Exec(f.admin, `INSERT INTO hrm.payroll_locks (legal_entity_id) VALUES ($1)`, c2); err != nil {
		t.Fatal(err)
	}
	f.contract(f.emp, sample{salary: 22_000_000}, "2026-01-01", "2026-05-31")
	stay := f.hire("S", f.a, "2026-01-05", 9_000_000)
	other := f.hire("X", d2, "2026-01-05", 9_000_000)
	f.close(f.c, "2026-05", map[int64]map[int64][]string{f.a: {f.emp: full("2026-05"), stay: full("2026-05")}})
	f.close(c2, "2026-05", map[int64]map[int64][]string{d2: {other: full("2026-05")}})
	f.edit(f.emp, func(e *hrm.EmployeeFields) { e.OrgUnitID = d2; e.ManagerID = nil })
	f.contract(f.emp, sample{salary: 24_000_000}, "2026-06-01", "2026-12-31")
	f.close(f.c, "2026-06", map[int64]map[int64][]string{f.a: {stay: full("2026-06")}})
	f.close(c2, "2026-06", map[int64]map[int64][]string{d2: {other: full("2026-06"), f.emp: full("2026-06")}})

	rows, _ := f.exportReport(f.rep, "hrm.report.d02lt", `{"legal_entity_id":`+itoa(f.c)+`,"month":"2026-06"}`)
	wantCells(t, rowOf(t, rows, dName, "Nhân viên E"), map[int]string{dKind: "Giảm lao động", dNote: "Chuyển đơn vị", dEff: "31/05/2026"})
	rows, _ = f.exportReport(f.rep, "hrm.report.d02lt", `{"legal_entity_id":`+itoa(c2)+`,"month":"2026-06"}`)
	wantCells(t, rowOf(t, rows, dName, "Nhân viên E"), map[int]string{dKind: "Tăng lao động", dEff: "01/06/2026", dSal: "24000000"})
}

// U, hired before the first posted payroll and below the 14 days in April and May, starts
// contributing in June: with no history to show otherwise, U counts as insured before mmerp.
func TestD02LTReturnAtHistoryStart(t *testing.T) {
	f := newReportFixture(t)
	f.contract(f.emp, sample{salary: 22_000_000}, "2026-01-01", "2026-12-31")
	u := f.hire("U", f.a, "2026-01-05", 12_000_000)
	few := func(month string) []string { return workdays(month+"-04", month+"-05") }
	f.close(f.c, "2026-04", map[int64]map[int64][]string{f.a: {f.emp: full("2026-04"), u: workdays("2026-04-01", "2026-04-02")}})
	f.close(f.c, "2026-05", map[int64]map[int64][]string{f.a: {f.emp: full("2026-05"), u: few("2026-05")}})
	f.close(f.c, "2026-06", map[int64]map[int64][]string{f.a: {f.emp: full("2026-06"), u: full("2026-06")}})
	rows, _ := f.exportReport(f.rep, "hrm.report.d02lt", `{"legal_entity_id":`+itoa(f.c)+`,"month":"2026-06"}`)
	wantCells(t, rowOf(t, rows, dName, "Nhân viên U"), map[int]string{dKind: "Đi làm lại", dEff: "01/06/2026"})
}

// U contributed in February; March has no posted payroll; April and May are below the 14
// days. The walk back crosses the missing month and June is a return.
func TestD02LTReturnAcrossMissingMonth(t *testing.T) {
	f := newReportFixture(t)
	f.contract(f.emp, sample{salary: 22_000_000}, "2026-01-01", "2026-12-31")
	u := f.hire("U", f.a, "2026-01-05", 12_000_000)
	f.close(f.c, "2026-02", map[int64]map[int64][]string{f.a: {f.emp: full("2026-02"), u: full("2026-02")}})
	f.close(f.c, "2026-04", map[int64]map[int64][]string{f.a: {f.emp: full("2026-04"), u: workdays("2026-04-01", "2026-04-02")}})
	f.close(f.c, "2026-05", map[int64]map[int64][]string{f.a: {f.emp: full("2026-05"), u: workdays("2026-05-04", "2026-05-05")}})
	f.close(f.c, "2026-06", map[int64]map[int64][]string{f.a: {f.emp: full("2026-06"), u: full("2026-06")}})
	rows, _ := f.exportReport(f.rep, "hrm.report.d02lt", `{"legal_entity_id":`+itoa(f.c)+`,"month":"2026-06"}`)
	wantCells(t, rowOf(t, rows, dName, "Nhân viên U"), map[int]string{dKind: "Đi làm lại", dEff: "01/06/2026"})
}

// E contributed to C until April, joins C2 in May below the 14 days and contributes from June:
// for C2 that is an increase, since E was never insured there.
func TestD02LTMoverWaivedFirstMonth(t *testing.T) {
	f := newReportFixture(t)
	c2 := must(t)(f.iam.CreateOrgUnit(f.admin, iam.OrgUnitInput{Kind: "company", Name: "C2"}))
	d2 := must(t)(f.iam.CreateOrgUnit(f.admin, iam.OrgUnitInput{ParentID: &c2, Kind: "department", Name: "D2"}))
	if _, err := f.system.Exec(f.admin, `INSERT INTO hrm.payroll_locks (legal_entity_id) VALUES ($1)`, c2); err != nil {
		t.Fatal(err)
	}
	f.contract(f.emp, sample{salary: 22_000_000}, "2026-01-01", "2026-04-30")
	other := f.hire("X", d2, "2026-01-05", 9_000_000)
	f.close(f.c, "2026-04", map[int64]map[int64][]string{f.a: {f.emp: full("2026-04")}})
	f.close(c2, "2026-04", map[int64]map[int64][]string{d2: {other: full("2026-04")}})
	f.edit(f.emp, func(e *hrm.EmployeeFields) { e.OrgUnitID = d2; e.ManagerID = nil })
	f.contract(f.emp, sample{salary: 24_000_000}, "2026-05-01", "2026-12-31")
	f.close(c2, "2026-05", map[int64]map[int64][]string{d2: {other: full("2026-05"), f.emp: workdays("2026-05-04", "2026-05-05")}})
	f.close(c2, "2026-06", map[int64]map[int64][]string{d2: {other: full("2026-06"), f.emp: full("2026-06")}})
	rows, _ := f.exportReport(f.rep, "hrm.report.d02lt", `{"legal_entity_id":`+itoa(c2)+`,"month":"2026-06"}`)
	wantCells(t, rowOf(t, rows, dName, "Nhân viên E"), map[int]string{dKind: "Tăng lao động", dSal: "24000000"})
}

func TestD02LTPreviousMonthMissing(t *testing.T) {
	f := newReportFixture(t)
	f.contract(f.emp, sample{salary: 22_000_000}, "2026-01-01", "2026-12-31")
	f.close(f.c, "2026-06", map[int64]map[int64][]string{f.a: {f.emp: full("2026-06")}})
	job := f.reportJob(f.rep, "hrm.report.d02lt", `{"legal_entity_id":`+itoa(f.c)+`,"month":"2026-06"}`, "report_payroll_missing")
	if out := f.jobError(job); !strings.Contains(out, "05/2026") {
		t.Errorf("error %s", out)
	}
	f.reportJob(f.rep, "hrm.report.d02lt", `{"legal_entity_id":`+itoa(f.c)+`,"month":"2026-04"}`, "report_form_not_supported")
}

// jobError is the error a failed job stored, as JSON.
func (f *fixture) jobError(job int64) string {
	f.t.Helper()
	j, err := f.jobs.JobGet(f.admin, job)
	f.check(err)
	var out platform.JobOutput
	f.check(json.Unmarshal(j.Output(), &out))
	b, err := json.Marshal(out.Error)
	f.check(err)
	return string(b)
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// twoMonths closes May and June of C with E alone, whose salary rises in June, so June's
// D02-LT lists E.
func (f *reportFixture) twoMonths() {
	f.contract(f.emp, sample{salary: 22_000_000}, "2026-01-01", "2026-12-31")
	f.close(f.c, "2026-05", map[int64]map[int64][]string{f.a: {f.emp: full("2026-05")}})
	f.appendix(f.emp, "2026-06-01", 25_000_000)
	f.close(f.c, "2026-06", map[int64]map[int64][]string{f.a: {f.emp: full("2026-06")}})
}

func TestReportNeedsBothPermissions(t *testing.T) {
	f := newReportFixture(t)
	f.twoMonths()
	params := `{"legal_entity_id":` + itoa(f.c) + `,"month":"2026-06"}`
	f.reportJob(f.ctxFor("salary", "payroll@*"), "hrm.report.d02lt", params, "forbidden")
	f.reportJob(f.ctxFor("sensitive", "sensitive_viewer@*"), "hrm.report.d02lt", params, "forbidden")
}

// A file closes once the actor may no longer read an employee it holds.
func TestReportDownloadRechecks(t *testing.T) {
	f := newReportFixture(t)
	f.twoMonths()
	actor := f.ctxFor("narrow", "payroll@*", "sensitive_viewer@A")
	job := f.reportJob(actor, "hrm.report.d02lt", `{"legal_entity_id":`+itoa(f.c)+`,"month":"2026-06"}`, "completed")
	j, err := f.core.DataIO.Job(actor, job)
	f.check(err)
	if _, _, err := f.core.DataIO.File(actor, *j.FileID); err != nil {
		t.Fatal(err)
	}
	f.edit(f.emp, func(e *hrm.EmployeeFields) { e.OrgUnitID = f.b; e.ManagerID = nil })
	// dataio hides a file the actor may no longer read.
	if _, _, err := f.core.DataIO.File(actor, *j.FileID); errCode(err) != "not_found" {
		t.Fatalf("after the move: %v", err)
	}
}

func TestReportAuditsIdentifiers(t *testing.T) {
	f := newReportFixture(t)
	f.twoMonths()
	before := f.auditCount("hrm.employee_sensitive_exported")
	f.exportReport(f.rep, "hrm.report.d02lt", `{"legal_entity_id":`+itoa(f.c)+`,"month":"2026-06"}`)
	if got := f.auditCount("hrm.employee_sensitive_exported") - before; got != 1 {
		t.Fatalf("%d audit rows", got)
	}
	// D02-LT shows the social insurance number only; the audit names what the file holds.
	var data string
	f.check(f.system.QueryRow(f.admin, `SELECT data::text FROM audit.log WHERE action = 'hrm.employee_sensitive_exported' ORDER BY id DESC LIMIT 1`).Scan(&data))
	if !strings.Contains(data, "social_insurance_no") || strings.Contains(data, "tax_code") || strings.Contains(data, "national_id") {
		t.Fatalf("audited %s", data)
	}
}

// indicators maps each code of an indicator sheet to its value.
func indicators(t *testing.T, rows [][]string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, r := range rows {
		if r = pad(r, 4); strings.HasPrefix(r[2], "[") {
			out[r[2]] = r[3]
		}
	}
	return out
}

func wantIndicators(t *testing.T, got map[string]string, want map[string]int64) {
	t.Helper()
	for code, v := range want {
		if got[code] != itoa(v) {
			t.Errorf("%s = %q, want %d", code, got[code], v)
		}
	}
}

// Q3/2026 in C: E is taxed and recovers 1,000,000 in August; L earns too little to be taxed.
func TestPIT05KK(t *testing.T) {
	f := newReportFixture(t)
	f.contract(f.emp, sample{salary: 30_000_000}, "2026-01-01", "2026-12-31")
	low := f.hire("L", f.a, "2026-01-05", 6_000_000)
	var lines []hrm.PayrollLine
	for _, m := range []string{"2026-07", "2026-08", "2026-09"} {
		f.timesheet(f.a, m, map[int64][]string{f.emp: full(m), low: full(m)})
		p, err := f.hrm.CreatePayroll(f.pay, hrm.NewPayroll{LegalEntityID: f.c, Month: m})
		f.check(err)
		f.wantJob(p.JobID, "completed")
		if m == "2026-08" {
			job, err := f.hrm.SavePayrollAdjustments(f.pay, p.ID, hrm.PayrollAdjustments{Version: f.payrollDoc(p.ID).Version,
				Items: []hrm.PayrollAdjustmentInput{{EmployeeID: f.emp, Amount: -1_000_000, SourcePeriod: "2026-07", Reason: "Trả thừa"}}})
			f.check(err)
			f.wantJob(job, "completed")
		}
		f.check(f.send(p.ID))
		lines = append(lines, f.payrollDoc(p.ID).Lines...)
	}
	var tnct, exempt, taxed, assessable, tax int64
	for _, l := range lines {
		tnct += l.Gross - l.Exempt
		exempt += l.Exempt
		assessable += l.Taxable
		tax += l.IncomeTax
		if l.EmployeeID == f.emp {
			taxed += l.Gross - l.Exempt
		} else if l.IncomeTax != 0 {
			t.Fatalf("L is taxed: %+v", l)
		}
	}
	rows, name := f.exportReport(f.rep, "hrm.report.pit_05kk", `{"legal_entity_id":`+itoa(f.c)+`,"quarter":"2026-Q3"}`)
	if !strings.Contains(name, "05-KK-TNCN") || !strings.Contains(name, "Q3-2026") {
		t.Errorf("file name %q", name)
	}
	got := indicators(t, rows)
	wantIndicators(t, got, map[string]int64{"[15]": 2, "[16]": 2, "[17]": 1, "[18]": 1, "[19]": 0,
		"[20]": tnct, "[21]": tnct, "[22]": 0, "[23]": 0, "[24]": exempt, "[25]": 0, "[26]": 0, "[27]": 0, "[28]": exempt,
		"[29]": taxed, "[30]": taxed, "[31]": 0, "[32]": assessable, "[33]": assessable, "[34]": 0,
		"[35]": tax, "[36]": 0, "[37]": 0, "[38]": 0, "[39]": 0, "[40]": 0, "[41]": 0, "[42]": tax, "[43]": tax, "[44]": 0, "[45]": 0})
	// 31 indicators under 4 header fields: [01], [04], [05], [06].
	if tax == 0 || len(got) != 35 {
		t.Fatalf("tax %d, %d indicators", tax, len(got))
	}
}

func TestPIT05KKPeriods(t *testing.T) {
	f := newReportFixture(t)
	f.contract(f.emp, sample{salary: 22_000_000}, "2026-01-01", "2026-12-31")
	f.close(f.c, "2026-07", map[int64]map[int64][]string{f.a: {f.emp: full("2026-07")}})
	f.close(f.c, "2026-08", map[int64]map[int64][]string{f.a: {f.emp: full("2026-08")}})
	job := f.reportJob(f.rep, "hrm.report.pit_05kk", `{"legal_entity_id":`+itoa(f.c)+`,"quarter":"2026-Q3"}`, "report_payroll_missing")
	if out := f.jobError(job); !strings.Contains(out, "09/2026") {
		t.Errorf("error %s", out)
	}
	f.reportJob(f.rep, "hrm.report.pit_05kk", `{"legal_entity_id":`+itoa(f.c)+`,"quarter":"2026-Q2"}`, "report_form_not_supported")
}

// The header rows come from the legal entity: its legal name, tax code and address once set.
func TestReportEntityHeader(t *testing.T) {
	f := newReportFixture(t)
	f.contract(f.emp, sample{salary: 22_000_000}, "2026-01-01", "2026-12-31")
	for _, m := range []string{"2026-07", "2026-08", "2026-09"} {
		f.close(f.c, m, map[int64]map[int64][]string{f.a: {f.emp: full(m)}})
	}
	params := `{"legal_entity_id":` + itoa(f.c) + `,"quarter":"2026-Q3"}`
	head := func() map[string]string {
		rows, _ := f.exportReport(f.rep, "hrm.report.pit_05kk", params)
		return indicators(t, rows)
	}
	if h := head(); h["[04]"] != "C" || h["[05]"] != "" || h["[01]"] != "Quý 3 năm 2026" {
		t.Fatalf("before: %v", h)
	}
	f.check(f.iam.UpdateOrgUnit(f.admin, f.c, iam.OrgUnitInput{Kind: "company", Name: "C", LegalName: str("Công ty C"), TaxCode: str("0101234567"), Address: str("Hà Nội")}))
	if h := head(); h["[04]"] != "Công ty C" || h["[05]"] != "0101234567" || h["[06]"] != "Hà Nội" {
		t.Fatalf("after: %v", h)
	}
}
