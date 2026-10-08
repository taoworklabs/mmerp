package app_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	pdfread "github.com/ledongthuc/pdf"
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
	// Nobody may probe a contract they cannot see.
	if j := f.print(f.admin, "hrm.contract", params); j.Code == nil || *j.Code != "not_found" {
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
