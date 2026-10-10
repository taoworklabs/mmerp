package app_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	pdfread "github.com/ledongthuc/pdf"
	"github.com/xuri/excelize/v2"

	"github.com/taoworklabs/mmerp/internal/app"
)

// pdfText reads back the text of a PDF, page by page.
func pdfText(t *testing.T, b []byte) []string {
	t.Helper()
	r, err := pdfread.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatalf("not a PDF: %v", err)
	}
	var pages []string
	for i := 1; i <= r.NumPage(); i++ {
		s, err := r.Page(i).GetPlainText(nil)
		if err != nil {
			t.Fatal(err)
		}
		pages = append(pages, s)
	}
	return pages
}

// print runs a print as c and returns its job once ended.
func (f *jobsFixture) print(c *client, template string, params string) job {
	f.t.Helper()
	var ex struct {
		JobID int64 `json:"job_id"`
	}
	_ = json.Unmarshal(f.ok(c, "POST", "/api/exports/printing."+template, fmt.Sprintf(`{"params":%s}`, params), 202), &ex)
	return f.wait(c, ex.JobID)
}

// printed runs a print that must succeed and returns the text of its pages.
func (f *jobsFixture) printed(c *client, template string, params string) []string {
	f.t.Helper()
	j := f.print(c, template, params)
	if j.State != "completed" || j.FileID == nil {
		f.t.Fatalf("print: %+v", j)
	}
	rec := c.do("GET", "/api/files/"+*j.FileID, "")
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "application/pdf" {
		f.t.Fatalf("download = %d %s", rec.Code, rec.Header())
	}
	return pdfText(f.t, rec.Body.Bytes())
}

// draftContract adds a draft contract, with an allowance, for a new employee of P.
func (f *jobsFixture) draftContract(c *client) (contract int64, number string) {
	f.t.Helper()
	emp := f.id(f.head, "/api/hrm/employees", fmt.Sprintf(`{"code":"E9","full_name":"Nguyễn Thị Ánh Tuyết","date_of_birth":"1990-04-02","gender":null,
		"phone":null,"email":null,"address":"12 Lê Lợi, Huế","org_unit_id":%d,"manager_id":null,"user_login":null,"hire_date":"2026-01-05","termination_date":null}`, f.p))
	kind := f.id(f.head, "/api/hrm/contract-types", `{"name":"Xác định thời hạn","fixed_term":true,"active":true}`)
	contract = f.id(c, "/api/hrm/contracts", fmt.Sprintf(`{"employee_id":%d,"contract_type_id":%d,"start_date":"2026-02-01","end_date":"2027-01-31",
		"terms":{"salary":12345678,"lines":[{"kind":"support","name":"Hỗ trợ ăn trưa","amount":730000,"taxable":false}]}}`, emp, kind))
	var out struct {
		Number         string
		AllowedActions []string `json:"allowed_actions"`
	}
	_ = json.Unmarshal(f.ok(c, "GET", fmt.Sprintf("/api/hrm/contracts/%d", contract), "", 200), &out)
	return contract, out.Number
}

func allowedActions(f *jobsFixture, c *client, path string) []string {
	f.t.Helper()
	var out struct {
		AllowedActions []string `json:"allowed_actions"`
	}
	_ = json.Unmarshal(f.ok(c, "GET", path, "", 200), &out)
	return out.AllowedActions
}

// A draft contract prints from its current data with a draft watermark, for whoever may
// see its salary; others may neither print it nor fetch the file.
func TestPrintDraftContract(t *testing.T) {
	f := newJobsFixture(t)
	pay := f.payrollSetup()
	f.work([]string{"hrm"})
	id, number := f.draftContract(pay)
	path := fmt.Sprintf("/api/hrm/contracts/%d", id)
	if a := allowedActions(f, pay, path); !slices.Contains(a, "print") {
		t.Fatalf("pay's actions: %v", a)
	}
	// head and hr see the contract but not its salary.
	if a := allowedActions(f, f.head, path); slices.Contains(a, "print") {
		t.Fatalf("head's actions: %v", a)
	}

	params := fmt.Sprintf(`{"id":%d}`, id)
	pages := f.printed(pay, "hrm.contract", params)
	if len(pages) != 1 {
		t.Fatalf("%d pages", len(pages))
	}
	for _, want := range []string{number, "Nguyễn Thị Ánh Tuyết", "12.345.678", "Hỗ trợ ăn trưa", "730.000", "BẢN NHÁP"} {
		if !strings.Contains(pages[0], want) {
			t.Errorf("no %q in %q", want, pages[0])
		}
	}

	for _, c := range []*client{f.head, f.hr} {
		if j := f.print(c, "hrm.contract", params); j.Code == nil || *j.Code != "forbidden" {
			t.Fatalf("print without salary: %+v", j)
		}
	}
	// The administrator sees every contract but holds no salary permission without a role.
	if j := f.print(f.admin, "hrm.contract", params); j.Code == nil || *j.Code != "forbidden" {
		t.Fatalf("admin's print: %+v", j)
	}
	if j := f.print(pay, "hrm.contract", `{"id":999999}`); j.Code == nil || *j.Code != "not_found" {
		t.Fatalf("missing contract: %+v", j)
	}
	// The file is pay's alone.
	j := f.print(pay, "hrm.contract", params)
	f.ok(f.head, "GET", "/api/files/"+*j.FileID, "", 404)

	// Audited on the contract, shown only to who may print it.
	var printed int
	if err := f.pool.QueryRow(t.Context(), `SELECT count(*) FROM audit.log WHERE action = 'printing.printed' AND doc_id = $1`, id).Scan(&printed); err != nil || printed != 2 {
		t.Fatalf("%d prints audited, %v", printed, err)
	}
	history := fmt.Sprintf("/api/documents/hrm.contract/%d/history", id)
	if h := string(f.ok(pay, "GET", history, "", 200)); !strings.Contains(h, "printing.printed") {
		t.Fatalf("pay's history: %s", h)
	}
	if h := string(f.ok(f.head, "GET", history, "", 200)); strings.Contains(h, "printing.printed") {
		t.Fatalf("head's history: %s", h)
	}
}

// With HRM off, a contract still prints: printing is an export.
func TestPrintWithProductOff(t *testing.T) {
	f := newJobsFixture(t)
	pay := f.payrollSetup()
	id, _ := f.draftContract(pay)
	f.work(nil)
	pages := f.printed(pay, "hrm.contract", fmt.Sprintf(`{"id":%d}`, id))
	if !strings.Contains(pages[0], "Nguyễn Thị Ánh Tuyết") {
		t.Fatalf("page: %q", pages[0])
	}
}

// moveContract moves a contract as c, at the version it has now.
func (f *jobsFixture) moveContract(c *client, id int64, to string) {
	f.t.Helper()
	var d struct{ Version int32 }
	_ = json.Unmarshal(f.ok(c, "GET", fmt.Sprintf("/api/hrm/contracts/%d", id), "", 200), &d)
	f.ok(c, "POST", fmt.Sprintf("/api/documents/hrm.contract/%d/transitions", id), fmt.Sprintf(`{"to":%q,"version":%d}`, to, d.Version), 204)
}

// rename changes the full name of the employee coded code.
func (f *jobsFixture) rename(code, name string) {
	f.t.Helper()
	if _, err := f.pool.Exec(f.t.Context(), `UPDATE hrm.employees SET full_name = $2 WHERE code = $1`, code, name); err != nil {
		f.t.Fatal(err)
	}
}

// A posted contract prints the same whatever changes after: the employee's name, the
// employer's details, the language of whoever prints it. Cancelled, it is marked so.
func TestPrintPostedContract(t *testing.T) {
	f := newJobsFixture(t)
	pay := f.payrollSetup()
	f.work([]string{"hrm"})
	id, _ := f.draftContract(pay)
	f.moveContract(pay, id, "posted")
	params := fmt.Sprintf(`{"id":%d}`, id)
	first := f.printed(pay, "hrm.contract", params)
	if strings.Contains(first[0], "BẢN NHÁP") || !strings.Contains(first[0], "Nguyễn Thị Ánh Tuyết") || !strings.Contains(first[0], "HỢP ĐỒNG LAO ĐỘNG") {
		t.Fatalf("posted print: %q", first[0])
	}

	f.rename("E9", "Trần Thị Ánh Tuyết")
	f.ok(f.admin, "PUT", fmt.Sprintf("/api/org-units/%d", f.legalEntity()),
		`{"parent_id":null,"kind":"company","name":"C","tax_code":"0101234567","legal_name":"Công ty TNHH C","address":"1 Tràng Tiền, Hà Nội"}`, 204)
	f.ok(pay, "PATCH", "/api/me", `{"locale":"en"}`, 204)
	if again := f.printed(pay, "hrm.contract", params); !slices.Equal(again, first) {
		t.Fatalf("reprint differs:\n%q\n%q", again, first)
	}

	f.moveContract(pay, id, "cancelled")
	cancelled := f.printed(pay, "hrm.contract", params)
	if !strings.Contains(cancelled[0], "ĐÃ HỦY") || !strings.Contains(cancelled[0], "Nguyễn Thị Ánh Tuyết") {
		t.Fatalf("cancelled print: %q", cancelled[0])
	}
}

// A contract posted before printing existed is frozen at its first print.
func TestPrintContractPostedBeforePrinting(t *testing.T) {
	f := newJobsFixture(t)
	pay := f.payrollSetup()
	f.work([]string{"hrm"})
	var id int64
	if err := f.pool.QueryRow(t.Context(), `SELECT c.id FROM hrm.contracts c JOIN hrm.employees e ON e.id = c.employee_id WHERE e.code = 'E1'`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(t.Context(), `DELETE FROM printing.snapshots WHERE doc_id = $1`, id); err != nil {
		t.Fatal(err)
	}
	params := fmt.Sprintf(`{"id":%d}`, id)
	first := f.printed(pay, "hrm.contract", params)
	if !strings.Contains(first[0], "Nhân viên E1") || strings.Contains(first[0], "BẢN NHÁP") {
		t.Fatalf("first print: %q", first[0])
	}
	f.rename("E1", "Tên mới")
	if again := f.printed(pay, "hrm.contract", params); !slices.Equal(again, first) {
		t.Fatalf("reprint differs:\n%q\n%q", again, first)
	}
}

// money writes an amount as a Vietnamese print does.
func money(n int64) string {
	s := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Payslips print a page per employee, with the amounts of the payroll and of its Excel
// export to the đồng; only for who sees salaries, and unchanged once the payroll is posted.
func TestPrintPayslips(t *testing.T) {
	f := newJobsFixture(t)
	f.work([]string{"hrm"})
	f.wait(f.hr, f.upload(f.hr, "hrm.timesheet", f.params(), march()...))
	f.ok(f.hr, "POST", fmt.Sprintf("/api/documents/hrm.timesheet/%d/transitions", f.timesheet), `{"to":"posted","version":2}`, 204)
	f.approveAll(204)
	pay := f.payrollSetup()
	id := f.createPayroll(pay)
	path := fmt.Sprintf("/api/hrm/payrolls/%d", id)
	if a := allowedActions(f, pay, path); !slices.Contains(a, "print") {
		t.Fatalf("pay's actions: %v", a)
	}
	var p struct {
		Lines []struct {
			EmployeeID   int64  `json:"employee_id"`
			EmployeeName string `json:"employee_name"`
			Gross, Net   int64
			IncomeTax    int64 `json:"income_tax"`
			Social       int64 `json:"social_insurance"`
		}
	}
	_ = json.Unmarshal(f.ok(pay, "GET", path, "", 200), &p)
	params := fmt.Sprintf(`{"id":%d}`, id)

	pages := f.printed(pay, "hrm.payslip", params)
	if len(pages) != 2 {
		t.Fatalf("%d pages", len(pages))
	}
	for i, l := range p.Lines {
		for _, want := range []string{"PHIẾU LƯƠNG", l.EmployeeName, money(l.Gross), money(l.Net), money(l.Social), "BẢN NHÁP"} {
			if !strings.Contains(pages[i], want) {
				t.Errorf("page %d: no %q in %q", i+1, want, pages[i])
			}
		}
	}
	// The Excel export says the same: gross and net of E1.
	var ex struct {
		JobID int64 `json:"job_id"`
	}
	_ = json.Unmarshal(f.ok(pay, "POST", "/api/exports/hrm.payroll", fmt.Sprintf(`{"params":{"payroll_id":%d}}`, id), 202), &ex)
	x, err := excelize.OpenReader(bytes.NewReader(f.ok(pay, "GET", "/api/files/"+*f.wait(pay, ex.JobID).FileID, "", 200)))
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := x.GetRows("Sheet1", excelize.Options{RawCellValue: true})
	for _, col := range []string{"Tổng thu nhập", "Thực lĩnh"} {
		c := slices.Index(rows[0], col)
		v, _ := strconv.ParseInt(rows[1][c], 10, 64)
		if c < 0 || !strings.Contains(pages[0], money(v)) {
			t.Errorf("%s of E1 in Excel (%v) is not on the payslip", col, rows[1])
		}
	}

	// One payslip; an employee not on the payroll is refused.
	one := f.printed(pay, "hrm.payslip", fmt.Sprintf(`{"id":%d,"parts":[%d]}`, id, f.e2))
	if len(one) != 1 || !strings.Contains(one[0], "Nhân viên E2") {
		t.Fatalf("E2's payslip: %q", one)
	}
	if j := f.print(pay, "hrm.payslip", fmt.Sprintf(`{"id":%d,"parts":[999999]}`, id)); j.Code == nil || *j.Code != "invalid_request" {
		t.Fatalf("unknown employee: %+v", j)
	}
	var parts string
	if err := f.pool.QueryRow(t.Context(), `SELECT data->>'parts' FROM audit.log WHERE action = 'printing.printed' ORDER BY id DESC LIMIT 1`).Scan(&parts); err != nil || parts != fmt.Sprintf("[%d]", f.e2) {
		t.Fatalf("audited parts %q %v", parts, err)
	}

	// Department totals only: no print.
	viewer := f.id(f.admin, "/api/users", `{"login":"viewer","name":"Kế toán","password":"correct horse"}`)
	f.id(f.admin, fmt.Sprintf("/api/users/%d/roles", viewer), `{"product":"hrm","role":"payroll_viewer","org_unit_id":null}`)
	v := f.login("viewer")
	if a := allowedActions(f, v, path); slices.Contains(a, "print") {
		t.Fatalf("viewer's actions: %v", a)
	}
	if j := f.print(v, "hrm.payslip", params); j.Code == nil || *j.Code != "forbidden" {
		t.Fatalf("viewer's print: %+v", j)
	}

	// Posted: one snapshot per line, and payslips reprint the same after a rename.
	var version int32
	if err := f.pool.QueryRow(t.Context(), `SELECT version FROM record.documents WHERE id = $1`, id).Scan(&version); err != nil {
		t.Fatal(err)
	}
	f.ok(pay, "POST", fmt.Sprintf("/api/documents/hrm.payroll/%d/transitions", id), fmt.Sprintf(`{"to":"posted","version":%d}`, version), 204)
	var snapshots int
	if err := f.pool.QueryRow(t.Context(), `SELECT count(*) FROM printing.snapshots WHERE doc_type = 'hrm.payroll' AND doc_id = $1`, id).Scan(&snapshots); err != nil || snapshots != 2 {
		t.Fatalf("%d snapshots %v", snapshots, err)
	}
	posted := f.printed(pay, "hrm.payslip", params)
	f.rename("E1", "Tên mới")
	if again := f.printed(pay, "hrm.payslip", params); !slices.Equal(again, posted) || strings.Contains(posted[0], "BẢN NHÁP") {
		t.Fatalf("reprint differs:\n%q\n%q", again, posted)
	}

	// Without salary rights the file printed before is gone.
	j := f.print(pay, "hrm.payslip", params)
	f.ok(f.admin, "DELETE", fmt.Sprintf("/api/users/%d/roles/%d", f.payID, f.payGrant), "", 204)
	f.ok(pay, "GET", "/api/files/"+*j.FileID, "", 404)
}

type printTemplate struct {
	Code, Name, Product string
	Version             int
	Blocks              []struct {
		Key, Label   string
		Placeholders []string
		Vi, En       string
	}
}

// Administrators edit a template's text blocks: each save is a version; a posted document
// keeps the text of its first print, one printed first afterwards takes the new text.
func TestPrintTextBlocks(t *testing.T) {
	f := newJobsFixture(t)
	pay := f.payrollSetup()
	f.work([]string{"hrm"})
	var list []printTemplate
	_ = json.Unmarshal(f.ok(f.admin, "GET", "/api/print-templates", "", 200), &list)
	if i := slices.IndexFunc(list, func(p printTemplate) bool { return p.Code == "hrm.contract" }); i < 0 || list[i].Version != 0 || list[i].Name != "Hợp đồng lao động" {
		t.Fatalf("templates: %+v", list)
	}
	var tpl printTemplate
	_ = json.Unmarshal(f.ok(f.admin, "GET", "/api/print-templates/hrm.contract", "", 200), &tpl)
	clauses := slices.IndexFunc(tpl.Blocks, func(b struct {
		Key, Label   string
		Placeholders []string
		Vi, En       string
	}) bool {
		return b.Key == "clauses"
	})
	if clauses < 0 || tpl.Blocks[clauses].Vi == "" || !slices.Contains(tpl.Blocks[clauses].Placeholders, "employer_name") {
		t.Fatalf("template: %+v", tpl)
	}
	oldClause := strings.SplitN(tpl.Blocks[clauses].Vi, "{", 2)[0]
	wantCode(t, pay.do("GET", "/api/print-templates", ""), 403, "forbidden")

	id, _ := f.draftContract(pay)
	f.moveContract(pay, id, "posted")
	params := fmt.Sprintf(`{"id":%d}`, id)
	first := f.printed(pay, "hrm.contract", params)
	if !strings.Contains(first[0], oldClause) {
		t.Fatalf("no default clause %q in %q", oldClause, first[0])
	}

	blocks := func(clause string) string {
		var bs []string
		for _, b := range tpl.Blocks {
			vi := b.Vi
			if b.Key == "clauses" {
				vi = clause
			}
			v, _ := json.Marshal(map[string]string{"key": b.Key, "vi": vi, "en": b.En})
			bs = append(bs, string(v))
		}
		return `{"blocks":[` + strings.Join(bs, ",") + `]}`
	}
	wantCode(t, f.admin.do("PUT", "/api/print-templates/hrm.contract/blocks", blocks("Lương {salary}")), 422, "print_placeholder_unknown")
	f.ok(f.admin, "PUT", "/api/print-templates/hrm.contract/blocks", blocks("Điều khoản mới của {employer_name}."), 204)
	_ = json.Unmarshal(f.ok(f.admin, "GET", "/api/print-templates/hrm.contract", "", 200), &tpl)
	if tpl.Version != 1 {
		t.Fatalf("version %d", tpl.Version)
	}

	if again := f.printed(pay, "hrm.contract", params); !slices.Equal(again, first) {
		t.Fatalf("reprint differs:\n%q\n%q", again, first)
	}
	// E1's contract, posted before the change but never printed, takes the new text.
	var e1 int64
	if err := f.pool.QueryRow(t.Context(), `SELECT c.id FROM hrm.contracts c JOIN hrm.employees e ON e.id = c.employee_id WHERE e.code = 'E1'`).Scan(&e1); err != nil {
		t.Fatal(err)
	}
	if p := f.printed(pay, "hrm.contract", fmt.Sprintf(`{"id":%d}`, e1)); !strings.Contains(p[0], "Điều khoản mới của C.") {
		t.Fatalf("E1's contract: %q", p[0])
	}
	var saved int
	if err := f.pool.QueryRow(t.Context(), `SELECT count(*) FROM audit.log WHERE action = 'printing.blocks_saved'`).Scan(&saved); err != nil || saved != 1 {
		t.Fatalf("%d saves audited %v", saved, err)
	}

	// With HRM off, its templates read but take no change.
	off := &client{t: t, h: app.New(env(t, f.pool, nil), app.Modules(), nil)}
	off.cookie = off.do("POST", "/api/auth/login", `{"login":"admin","password":"correct horse"}`).Result().Cookies()[0]
	f.ok(off, "GET", "/api/print-templates/hrm.contract", "", 200)
	wantCode(t, off.do("PUT", "/api/print-templates/hrm.contract/blocks", blocks("x")), 403, "product_not_enabled")
}
