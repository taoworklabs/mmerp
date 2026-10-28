package hrm_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/taoworklabs/mmerp/internal/core/dataio"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

// exportEmployees runs the employee export as ctx's actor and returns the sheet's rows.
func (f *fixture) exportEmployees(ctx context.Context, params string) [][]string {
	f.t.Helper()
	job, err := f.core.DataIO.StartExport(ctx, "hrm.employees", []byte(params))
	f.check(err)
	f.wantJob(job, "completed")
	j, err := f.core.DataIO.Job(ctx, job)
	f.check(err)
	_, file, err := f.core.DataIO.File(ctx, *j.FileID)
	f.check(err)
	defer func() { f.check(file.Close()) }()
	b, err := io.ReadAll(file)
	f.check(err)
	x, err := excelize.OpenReader(bytes.NewReader(b))
	f.check(err)
	rows, err := x.GetRows("Sheet1", excelize.Options{RawCellValue: true})
	f.check(err)
	return rows
}

// exportEmployeesEnding starts an export that must end as want.
func (f *fixture) exportEmployeesEnding(ctx context.Context, params, want string) {
	f.t.Helper()
	job, err := f.core.DataIO.StartExport(ctx, "hrm.employees", []byte(params))
	f.check(err)
	f.wantJob(job, want)
}

func codes(rows [][]string) []string {
	var out []string
	for _, r := range rows[1:] {
		out = append(out, r[0])
	}
	return out
}

func TestEmployeeExportColumns(t *testing.T) {
	f := newFixture(t)
	hr := f.ctxFor("hr", "hr@*")
	e0 := must(t)(f.hrm.CreateEmployee(hr, employee("E0", f.b)))
	in := employee("E1", f.a)
	in.DateOfBirth, in.Gender, in.ManagerID = str("1990-02-03"), str("female"), &e0
	must(t)(f.hrm.CreateEmployee(hr, in))

	rows := f.exportEmployees(hr, `{"q":"E1"}`)
	want := []string{"Mã NV", "Họ tên", "Ngày sinh", "Giới tính", "SĐT", "Email", "Địa chỉ", "Phòng ban", "Mã người quản lý trực tiếp",
		"Ngày vào làm", "Ngày nghỉ việc", "CCCD/Hộ chiếu", "Số sổ BHXH", "Mã số thuế", "Số tài khoản"}
	if !slices.Equal(rows[0], want) {
		t.Fatalf("header %v", rows[0])
	}
	got := rows[1]
	for len(got) < len(want) {
		got = append(got, "")
	}
	exp := []string{"E1", "Nhân viên E1", "03/02/1990", "Nữ", "", "", "", "A", "E0", "05/01/2026", "", "", "", "", ""}
	if !slices.Equal(got, exp) {
		t.Fatalf("row %q, want %q", got, exp)
	}
}

func TestEmployeeExportFilters(t *testing.T) {
	f := newFixture(t)
	hr := f.ctxFor("hr", "hr@*")
	must(t)(f.hrm.CreateEmployee(hr, employee("E1", f.a)))
	must(t)(f.hrm.CreateEmployee(hr, employee("E2", f.b)))
	in := employee("E3", f.a)
	in.TerminationDate = str("2026-02-01")
	must(t)(f.hrm.CreateEmployee(hr, in))

	for params, want := range map[string][]string{
		fmt.Sprintf(`{"org_unit_id":%d}`, f.a): {"E1", "E3"},
		`{"status":"terminated"}`:              {"E3"},
		`{"q":"E2"}`:                           {"E2"},
		`{}`:                                   {"E1", "E2", "E3"},
	} {
		if got := codes(f.exportEmployees(hr, params)); !slices.Equal(got, want) {
			t.Errorf("%s: %v, want %v", params, got, want)
		}
	}
	if got := codes(f.exportEmployees(f.ctxFor("hra", "hr@A"), `{}`)); !slices.Equal(got, []string{"E1", "E3"}) {
		t.Errorf("scoped: %v", got)
	}
}

func TestEmployeeExportSensitive(t *testing.T) {
	f := newFixture(t)
	both := f.ctxFor("both", "hr@*", "sensitive_viewer@*")
	in := employee("E1", f.a)
	in.Sensitive = &hrm.SensitiveValues{NationalID: str("012345678901"), TaxCode: str("8888888888"), BankAccount: str("123456")}
	id := must(t)(f.hrm.CreateEmployee(both, in))
	must(t)(f.hrm.CreateEmployee(both, employee("E2", f.b)))

	pad := func(r []string) []string { return append(r, make([]string, 15-len(r))...) }
	// Not asked for: empty for everyone, nothing audited.
	if got := pad(f.exportEmployees(both, `{"q":"E1"}`)[1])[11:]; !slices.Equal(got, []string{"", "", "", ""}) {
		t.Fatalf("not asked: %q", got)
	}
	if n := f.auditCount("hrm.employee_sensitive_exported"); n != 0 {
		t.Fatalf("audited %d", n)
	}
	// Asked for over employees the actor may not read in full: refused.
	mixed := f.ctxFor("mixed", "hr@*", "sensitive_viewer@A")
	f.exportEmployeesEnding(mixed, `{"include_sensitive":true}`, "forbidden")
	if got := pad(f.exportEmployees(mixed, fmt.Sprintf(`{"org_unit_id":%d,"include_sensitive":true}`, f.a))[1])[11:]; !slices.Equal(got, []string{"012345678901", "", "8888888888", "123456"}) {
		t.Fatalf("with permission: %q", got)
	}
	var n int
	var data []byte
	f.check(f.system.QueryRow(f.admin, `SELECT count(*), max(data::text) FROM audit.log WHERE action = 'hrm.employee_sensitive_exported' AND doc_type = 'hrm.employee' AND doc_id = $1`, id).Scan(&n, &data))
	if n != 1 {
		t.Fatalf("audited %d", n)
	}
	var d struct{ Fields []string }
	f.check(json.Unmarshal(data, &d))
	if !slices.Equal(d.Fields, []string{"national_id", "tax_code", "bank_account"}) {
		t.Fatalf("audit data %s", data)
	}
}

// A download asks again: the file with sensitive values closes when the permission is revoked.
func TestEmployeeExportSensitiveRevoked(t *testing.T) {
	f := newFixture(t)
	hr := f.ctxFor("hr", "hr@*")
	must(t)(f.hrm.CreateEmployee(hr, employee("E1", f.a)))
	uid, _ := platform.ActorFrom(hr)
	grant := must(t)(f.iam.GrantRole(f.admin, uid, "hrm", "sensitive_viewer", nil))
	job, err := f.core.DataIO.StartExport(hr, "hrm.employees", []byte(`{"include_sensitive":true}`))
	f.check(err)
	f.wantJob(job, "completed")
	j, err := f.core.DataIO.Job(hr, job)
	f.check(err)
	_, file, err := f.core.DataIO.File(hr, *j.FileID)
	f.check(err)
	f.check(file.Close())
	f.check(f.iam.RevokeRole(f.admin, uid, grant))
	// The download maps a permission error to not found, so the file's existence is not told.
	if _, _, err := f.core.DataIO.File(hr, *j.FileID); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("download after revoke: %v", err)
	}
}

func TestEmployeeExportNeedsView(t *testing.T) {
	f := newFixture(t)
	nobody := f.ctxFor("nobody")
	job, err := f.core.DataIO.StartExport(nobody, "hrm.employees", []byte(`{}`))
	if err != nil {
		if !errors.Is(err, platform.ErrForbidden) {
			t.Fatalf("start: %v", err)
		}
		return
	}
	f.wantJob(job, "forbidden")
}

func TestEmployeeExportWhileProductDisabled(t *testing.T) {
	f := newFixture(t)
	hr := f.ctxFor("hr", "hr@*")
	must(t)(f.hrm.CreateEmployee(hr, employee("E1", f.a)))
	if got := codes(f.exportEmployees(platform.WithProducts(hr, nil), `{}`)); !slices.Equal(got, []string{"E1"}) {
		t.Fatalf("disabled: %v", got)
	}
}

func TestUnitPaths(t *testing.T) {
	id := func(i int64) *int64 { return &i }
	u := func(i int64, parent *int64, name string) iam.OrgUnit {
		return iam.OrgUnit{ID: i, OrgUnitInput: iam.OrgUnitInput{ParentID: parent, Name: name}}
	}
	units := []iam.OrgUnit{u(1, nil, "C"), u(2, id(1), "HN"), u(3, id(1), "HCM"), u(4, id(2), "Kinh doanh"), u(5, id(3), "Kinh doanh"), u(6, id(1), "Kế toán")}
	paths := hrm.UnitPaths(units)
	for i, want := range map[int64]string{6: "Kế toán", 4: "HN / Kinh doanh", 5: "HCM / Kinh doanh", 1: "C"} {
		if paths[i] != want {
			t.Errorf("unit %d: %q, want %q", i, paths[i], want)
		}
	}
	for path, want := range map[string]int{"Kinh doanh": 2, "HN / Kinh doanh": 1, "C / HN / Kinh doanh": 1, "X": 0, " HN /  Kinh doanh ": 1} {
		if _, n := hrm.ResolveUnit(units, path); n != want {
			t.Errorf("%q: %d matches, want %d", path, n, want)
		}
	}
}

var employeeHeader = []any{"Mã NV", "Họ tên", "Ngày sinh", "Giới tính", "SĐT", "Email", "Địa chỉ", "Phòng ban", "Mã người quản lý trực tiếp",
	"Ngày vào làm", "Ngày nghỉ việc", "CCCD/Hộ chiếu", "Số sổ BHXH", "Mã số thuế", "Số tài khoản"}

// eRow is an import row with the code and the given cells, by column index.
func eRow(code string, cells map[int]any) []any {
	r := make([]any, 15)
	r[0] = code
	for i, v := range cells {
		r[i] = v
	}
	return r
}

// asRows turns an export's data rows back into import rows.
func asRows(rows [][]string) [][]any {
	var out [][]any
	for _, r := range rows[1:] {
		row := make([]any, len(r))
		for i, c := range r {
			row[i] = c
		}
		out = append(out, row)
	}
	return out
}

// importEmployees imports rows under the employee header as ctx's actor; the job must end as want.
func (f *fixture) importEmployees(ctx context.Context, want string, rows ...[]any) dataio.Job {
	f.t.Helper()
	job := f.startImport(ctx, rows...)
	f.wantJob(job, want)
	j, err := f.core.DataIO.Job(ctx, job)
	f.check(err)
	return j
}

// startImport queues an import of rows under the employee header as ctx's actor.
func (f *fixture) startImport(ctx context.Context, rows ...[]any) int64 {
	f.t.Helper()
	x := excelize.NewFile()
	for r, row := range append([][]any{employeeHeader}, rows...) {
		cell, _ := excelize.CoordinatesToCellName(1, r+1)
		f.check(x.SetSheetRow("Sheet1", cell, &row))
	}
	b, err := x.WriteToBuffer()
	f.check(err)
	job, err := f.core.DataIO.StartImport(ctx, "hrm.employees", "e.xlsx", b, nil)
	f.check(err)
	return job
}

func wantRowError(t *testing.T, j dataio.Job, row int, part string) {
	t.Helper()
	for _, e := range j.RowErrors {
		if e.Row == row && strings.Contains(e.Message, part) {
			return
		}
	}
	t.Fatalf("no row %d error with %q: %+v", row, part, j.RowErrors)
}

func (f *fixture) employeeByCode(ctx context.Context, code string) hrm.Employee {
	f.t.Helper()
	var id int64
	f.check(f.system.QueryRow(f.admin, `SELECT id FROM hrm.employees WHERE code = $1`, code).Scan(&id))
	e, err := f.hrm.Employee(ctx, id)
	f.check(err)
	return e
}

func TestEmployeeImportCreatesAndEdits(t *testing.T) {
	f := newFixture(t)
	hr := f.ctxFor("hr", "hr@*")
	in := employee("E1", f.a)
	in.Address, in.Gender = str("Hà Nội"), str("male")
	e1 := must(t)(f.hrm.CreateEmployee(hr, in))
	f.importEmployees(hr, "completed",
		eRow("E1", map[int]any{4: "0901"}),
		eRow("E9", map[int]any{1: "Chín", 7: "B", 9: "01/02/2026"}),
		eRow("", map[int]any{1: "Không mã", 7: "A", 9: "2026-02-01"}))
	e, err := f.hrm.Employee(hr, e1)
	f.check(err)
	if *e.Phone != "0901" || *e.Address != "Hà Nội" || *e.Gender != "male" || e.FullName != "Nhân viên E1" || e.OrgUnitID != f.a || e.HireDate != "2026-01-05" {
		t.Fatalf("E1 %+v", e.EmployeeFields)
	}
	if e9 := f.employeeByCode(hr, "E9"); e9.OrgUnitID != f.b || e9.HireDate != "2026-02-01" || e9.FullName != "Chín" {
		t.Fatalf("E9 %+v", e9.EmployeeFields)
	}
	if e := f.employeeByCode(hr, "NV00001"); e.FullName != "Không mã" {
		t.Fatalf("issued %+v", e.EmployeeFields)
	}
	if u, c := f.auditCount("hrm.employee_updated"), f.auditCount("hrm.employee_created"); u != 1 || c != 3 {
		t.Fatalf("audit: %d updated, %d created", u, c)
	}
}

func TestEmployeeImportAllOrNothing(t *testing.T) {
	f := newFixture(t)
	hr := f.ctxFor("hr", "hr@*")
	j := f.importEmployees(hr, "import_rows_invalid",
		eRow("E1", map[int]any{1: "Một", 7: "A", 9: "01/02/2026"}),
		eRow("E2", map[int]any{1: "Hai", 7: "Z", 9: "01/02/2026"}))
	wantRowError(t, j, 3, "Không có phòng ban \"Z\"")
	if n := f.auditCount("hrm.employee_created"); n != 0 {
		t.Fatalf("created %d", n)
	}
}

// A file exported by someone who may view more than they may edit goes back as is.
func TestEmployeeImportUnchangedExportWritesNothing(t *testing.T) {
	f := newFixture(t)
	admin := f.ctxFor("boss", "hr@*")
	e0 := must(t)(f.hrm.CreateEmployee(admin, employee("E0", f.b)))
	in := employee("E1", f.a)
	in.DateOfBirth, in.Gender, in.Phone, in.ManagerID, in.TerminationDate = str("1990-02-03"), str("female"), str("09"), &e0, str("2026-12-31")
	must(t)(f.hrm.CreateEmployee(admin, in))
	must(t)(f.hrm.CreateEmployee(admin, employee("E2", f.b)))
	hr := f.ctxFor("hr", "viewer@*", "hr@A")
	rows := f.exportEmployees(hr, `{}`)
	count := func() (n int) {
		f.check(f.system.QueryRow(f.admin, `SELECT count(*) FROM audit.log WHERE action LIKE 'hrm.%'`).Scan(&n))
		return n
	}
	before := count()
	f.importEmployees(hr, "completed", asRows(rows)...)
	if n := count(); n != before {
		t.Fatalf("audit rows %d, was %d", n, before)
	}
}

func TestEmployeeImportKeepsSensitiveAndAccount(t *testing.T) {
	f := newFixture(t)
	must(t)(f.iam.CreateUser(f.admin, "u1", "u1", "long enough"))
	both := f.ctxFor("both", "hr@*", "sensitive_viewer@*")
	in := employee("E1", f.a)
	in.UserLogin = str("u1")
	in.Sensitive = &hrm.SensitiveValues{TaxCode: str("123")}
	id := must(t)(f.hrm.CreateEmployee(both, in))
	hr := f.ctxFor("hr", "hr@*")
	rows := asRows(f.exportEmployees(hr, `{}`))
	rows[0] = append(rows[0], make([]any, 15-len(rows[0]))...)
	rows[0][6] = "Đà Nẵng"
	f.importEmployees(hr, "completed", rows...)
	e, err := f.hrm.Employee(both, id)
	f.check(err)
	tax, err := f.hrm.RevealSensitive(both, id, "tax_code")
	f.check(err)
	if *e.Address != "Đà Nẵng" || *tax != "123" || e.UserLogin == nil || *e.UserLogin != "u1" {
		t.Fatalf("after import: %+v tax %v", e.EmployeeFields, *tax)
	}
	rows[0][13] = "123"
	wantRowError(t, f.importEmployees(hr, "import_rows_invalid", rows...), 2, "thông tin nhạy cảm của nhân viên E1")
	updated := f.auditCount("hrm.employee_updated")
	rows[0][13] = "456"
	f.importEmployees(both, "completed", rows...)
	if tax, err := f.hrm.RevealSensitive(both, id, "tax_code"); err != nil || *tax != "456" {
		t.Fatalf("tax %v %v", *tax, err)
	}
	if n := f.auditCount("hrm.employee_updated"); n != updated+1 {
		t.Fatalf("updated %d, was %d", n, updated)
	}
}

func TestEmployeeImportScope(t *testing.T) {
	f := newFixture(t)
	must(t)(f.hrm.CreateEmployee(f.ctxFor("boss", "hr@*"), employee("E2", f.b)))
	hr := f.ctxFor("hr", "hr@A")
	wantRowError(t, f.importEmployees(hr, "import_rows_invalid", eRow("E2", map[int]any{4: "09"})), 2, "E2 đã được dùng")
	j := f.importEmployees(hr, "import_rows_invalid", eRow("E3", map[int]any{1: "Ba", 7: "B", 9: "01/02/2026"}))
	if len(j.RowErrors) != 1 || j.RowErrors[0].Message != "Bạn không có quyền ghi nhân viên E3" {
		t.Fatalf("%+v", j.RowErrors)
	}
}

func TestEmployeeImportDuplicateCode(t *testing.T) {
	f := newFixture(t)
	hr := f.ctxFor("hr", "hr@*")
	j := f.importEmployees(hr, "import_rows_invalid",
		eRow("e1", map[int]any{1: "Một", 7: "A", 9: "01/02/2026"}),
		eRow("E1", map[int]any{1: "Một", 7: "A", 9: "01/02/2026"}))
	if len(j.RowErrors) != 1 {
		t.Fatalf("%+v", j.RowErrors)
	}
	wantRowError(t, j, 3, "nhiều dòng")
}

func TestEmployeeImportDatesAndGender(t *testing.T) {
	f := newFixture(t)
	hr := f.ctxFor("hr", "hr@*")
	base := map[int]any{1: "X", 3: "nam", 7: "A", 9: "01/02/2026"}
	with := func(dob any) map[int]any {
		m := maps.Clone(base)
		m[2] = dob
		return m
	}
	f.importEmployees(hr, "completed", eRow("D1", with("03/02/1990")), eRow("D2", with("1990-02-03")), eRow("D3", with(32907)))
	for _, c := range []string{"D1", "D2", "D3"} {
		if e := f.employeeByCode(hr, c); *e.DateOfBirth != "1990-02-03" || *e.Gender != "male" {
			t.Fatalf("%s: %v %v", c, *e.DateOfBirth, *e.Gender)
		}
	}
	wantRowError(t, f.importEmployees(hr, "import_rows_invalid", eRow("D4", with("31/02/1990"))), 2, "Ngày sinh: \"31/02/1990\"")
	// A year typed alone is not a day count.
	wantRowError(t, f.importEmployees(hr, "import_rows_invalid", eRow("D5", with(1990))), 2, "Ngày sinh: \"1990\"")
}

func TestEmployeeImportRequiredInOrder(t *testing.T) {
	f := newFixture(t)
	hr := f.ctxFor("hr", "hr@*")
	j := f.importEmployees(hr, "import_rows_invalid", eRow("X1", nil), eRow("X2", nil), eRow("X3", nil))
	var got []string
	for _, e := range j.RowErrors {
		got = append(got, e.Message)
	}
	want := []string{"Thiếu họ tên", "Thiếu phòng ban", "Thiếu ngày vào làm"}
	if !slices.Equal(got, slices.Concat(want, want, want)) {
		t.Fatalf("%q", got)
	}
}

// A typed code the series would issue next is written before the rows that take an issued one.
func TestEmployeeImportTypedCodesFirst(t *testing.T) {
	f := newFixture(t)
	hr := f.ctxFor("hr", "hr@*")
	f.importEmployees(hr, "completed",
		eRow("", map[int]any{1: "Không mã", 7: "A", 9: "01/02/2026"}),
		eRow("NV00001", map[int]any{1: "Có mã", 7: "A", 9: "01/02/2026"}))
	if a, b := f.employeeByCode(hr, "NV00001"), f.employeeByCode(hr, "NV00002"); a.FullName != "Có mã" || b.FullName != "Không mã" {
		t.Fatalf("%s, %s", a.FullName, b.FullName)
	}
}

// A constraint failure aborts the transaction, so the rows after it are not written but the error is the row's.
func TestEmployeeImportStopsAtConstraintFailure(t *testing.T) {
	f := newFixture(t)
	hr := f.ctxFor("hr", "hr@*")
	_, err := f.system.Exec(f.admin, `CREATE FUNCTION hrm.test_boom() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.full_name = 'Boom' THEN RAISE unique_violation USING CONSTRAINT = 'employees_code_key'; END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER boom BEFORE INSERT ON hrm.employees FOR EACH ROW EXECUTE FUNCTION hrm.test_boom();`)
	f.check(err)
	j := f.importEmployees(hr, "import_rows_invalid",
		eRow("B1", map[int]any{1: "Boom", 7: "A", 9: "01/02/2026"}),
		eRow("B2", map[int]any{1: "Fine", 7: "A", 9: "01/02/2026"}))
	if len(j.RowErrors) != 1 {
		t.Fatalf("%+v", j.RowErrors)
	}
	wantRowError(t, j, 2, "B1 đã được dùng")
}

// A form edit committed while the import waits is kept: the import reads the profile under its lock.
func TestEmployeeImportLocksBeforeReading(t *testing.T) {
	f := newFixture(t)
	hr := f.ctxFor("hr", "hr@*")
	id := must(t)(f.hrm.CreateEmployee(hr, employee("E1", f.a)))
	job := f.startImport(hr, eRow("E1", map[int]any{6: "Đà Nẵng"}))
	tx, err := pgtest.System(t, f.core.Pool).Begin(t.Context())
	f.check(err)
	defer func() { _ = tx.Rollback(context.Background()) }()
	_, err = tx.Exec(t.Context(), `SELECT 1 FROM hrm.employees WHERE id = $1 FOR UPDATE`, id)
	f.check(err)
	f.check(f.jobs.Start(t.Context()))
	pgtest.AwaitLockWait(t, f.system)
	_, err = tx.Exec(t.Context(), `UPDATE hrm.employees SET phone = '0999' WHERE id = $1`, id)
	f.check(err)
	f.check(tx.Commit(t.Context()))
	f.check(f.jobs.Stop(t.Context()))
	f.wantJob(job, "completed")
	e, err := f.hrm.Employee(hr, id)
	f.check(err)
	if e.Phone == nil || *e.Phone != "0999" || *e.Address != "Đà Nẵng" {
		t.Fatalf("%+v", e.EmployeeFields)
	}
}

func TestEmployeeImportManagerInFile(t *testing.T) {
	f := newFixture(t)
	hr := f.ctxFor("hr", "hr@*")
	must(t)(f.hrm.CreateEmployee(hr, employee("E1", f.a)))
	f.importEmployees(hr, "completed",
		eRow("R1", map[int]any{1: "R", 7: "A", 8: "m1", 9: "01/02/2026"}),
		eRow("E1", map[int]any{8: "M1"}),
		eRow("", map[int]any{1: "Không mã", 7: "A", 8: "M1", 9: "01/02/2026"}),
		eRow("M1", map[int]any{1: "M", 7: "A", 9: "01/02/2026"}))
	m1 := f.employeeByCode(hr, "M1")
	for _, c := range []string{"R1", "E1", "NV00001"} {
		if e := f.employeeByCode(hr, c); e.ManagerID == nil || *e.ManagerID != m1.ID {
			t.Fatalf("%s manager %v", c, e.ManagerID)
		}
	}
}

// The cycle rule holds for the end state, so a file may swap managers in any row order.
func TestEmployeeImportManagerSwap(t *testing.T) {
	f := newFixture(t)
	hr := f.ctxFor("hr", "hr@*")
	b := must(t)(f.hrm.CreateEmployee(hr, employee("B", f.a)))
	in := employee("A", f.a)
	in.ManagerID = &b
	a := must(t)(f.hrm.CreateEmployee(hr, in))
	c := must(t)(f.hrm.CreateEmployee(hr, employee("C", f.a)))
	f.importEmployees(hr, "completed", eRow("B", map[int]any{8: "A"}), eRow("A", map[int]any{8: "C"}))
	if ea, eb := f.employeeByCode(hr, "A"), f.employeeByCode(hr, "B"); *ea.ManagerID != c || *eb.ManagerID != a {
		t.Fatalf("A→%d B→%d", *ea.ManagerID, *eb.ManagerID)
	}
}

func TestEmployeeImportManagerCycle(t *testing.T) {
	f := newFixture(t)
	hr := f.ctxFor("hr", "hr@*")
	b := must(t)(f.hrm.CreateEmployee(hr, employee("B", f.a)))
	in := employee("A", f.a)
	in.ManagerID = &b
	must(t)(f.hrm.CreateEmployee(hr, in))
	j := f.importEmployees(hr, "import_rows_invalid", eRow("B", map[int]any{8: "A"}))
	wantRowError(t, j, 2, "Người quản lý của nhân viên B tạo thành vòng lặp qua A · Nhân viên A")
	if e := f.employeeByCode(hr, "B"); e.ManagerID != nil {
		t.Fatalf("B manager %v", *e.ManagerID)
	}
	j = f.importEmployees(hr, "import_rows_invalid",
		eRow("N1", map[int]any{1: "Một", 7: "A", 8: "N2", 9: "01/02/2026"}),
		eRow("N2", map[int]any{1: "Hai", 7: "A", 8: "N1", 9: "01/02/2026"}))
	if len(j.RowErrors) != 1 || !strings.Contains(j.RowErrors[0].Message, "vòng lặp") {
		t.Fatalf("%+v", j.RowErrors)
	}
	if n := f.auditCount("hrm.employee_created"); n != 2 {
		t.Fatalf("created %d", n)
	}
}

func TestEmployeeImportManagerSelf(t *testing.T) {
	f := newFixture(t)
	hr := f.ctxFor("hr", "hr@*")
	must(t)(f.hrm.CreateEmployee(hr, employee("E1", f.a)))
	j := f.importEmployees(hr, "import_rows_invalid",
		eRow("E1", map[int]any{8: "e1"}),
		eRow("N1", map[int]any{1: "Một", 7: "A", 8: "N1", 9: "01/02/2026"}))
	wantRowError(t, j, 2, "E1 không thể là người quản lý")
	wantRowError(t, j, 3, "N1 không thể là người quản lý")
}
