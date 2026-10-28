package hrm_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
	"github.com/taoworklabs/mmerp/internal/platform"
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
