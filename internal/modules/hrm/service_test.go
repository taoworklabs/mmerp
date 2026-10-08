package hrm_test

import (
	"cmp"
	"context"
	"errors"
	"io"
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/dataio"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/notification"
	"github.com/taoworklabs/mmerp/internal/core/numbering"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/setting"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

type fixture struct {
	t    *testing.T
	pool platform.Conn
	iam  *iam.Service
	rec  *record.Service
	appr *approval.Service
	hrm  *hrm.Service
	jobs *river.Client[pgx.Tx]
	// started once work runs the jobs.
	started bool
	admin   context.Context
	c       int64 // company
	a, b    int64 // departments
	ctxFor  func(login string, grants ...string) context.Context
}

// newFixture builds company C with departments A and B. ctxFor creates a user
// with grants like "hr@A" or "viewer@*" and returns a ctx acting as them.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	pool := pgtest.New(t)
	ctx := platform.WithKeyring(platform.WithProducts(platform.WithDB(t.Context(), pool), []string{"hrm"}), pgtest.Keyring())
	ids := iam.NewService(iam.Deps{Setting: setting.NewService(), Audit: audit.NewService()})
	rec := record.NewService(record.Deps{IAM: ids, Numbering: numbering.NewService(), Audit: audit.NewService()})
	appr := approval.NewService(approval.Deps{IAM: ids, Record: rec, Audit: audit.NewService(), Notification: notification.NewService(notification.Deps{Record: rec, IAM: ids, Audit: audit.NewService(), Setting: setting.NewService()})})
	rec.SetApprovalGate(appr)
	h := hrm.NewService(hrm.Deps{IAM: ids, Record: rec, Audit: audit.NewService(), Setting: setting.NewService(),
		DataIO: dataio.NewService(dataio.Deps{IAM: ids, Audit: audit.NewService()}), Posting: posting.NewService(posting.Hooks{})})
	// Jobs are queued in ctx's transaction and worked only once work is called.
	workers := river.NewWorkers()
	hrm.Module(h).Workers(workers)
	var jobs *river.Client[pgx.Tx]
	base := func(c context.Context) context.Context {
		c = platform.WithJobNotifier(platform.WithKeyring(platform.WithProducts(platform.WithDB(c, pool), []string{"hrm"}), pgtest.Keyring()), notification.JobEnded)
		return platform.WithJobs(c, jobs)
	}
	jobs, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 2}}, Workers: workers,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Middleware: []rivertype.Middleware{platform.JobMiddleware(base)},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx = platform.WithJobs(ctx, jobs)
	adminID := must(t)(ids.CreateAdmin(ctx, "admin", "Admin", "long enough"))
	admin := platform.WithActor(ctx, adminID)
	c := must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{Kind: "company", Name: "C"}))
	f := &fixture{t: t, pool: pool, iam: ids, rec: rec, appr: appr, hrm: h, jobs: jobs, admin: admin, c: c}
	f.a = must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{ParentID: &c, Kind: "department", Name: "A"}))
	f.b = must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{ParentID: &c, Kind: "department", Name: "B"}))
	f.ctxFor = func(login string, grants ...string) context.Context {
		t.Helper()
		id := must(t)(ids.CreateUser(admin, login, login, "long enough"))
		for _, g := range grants {
			role, unit, _ := strings.Cut(g, "@")
			var u *int64
			switch unit {
			case "A":
				u = &f.a
			case "B":
				u = &f.b
			}
			must(t)(ids.GrantRole(admin, id, "hrm", role, u))
		}
		return platform.WithActor(ctx, id)
	}
	return f
}

func must(t *testing.T) func(int64, error) int64 {
	return func(v int64, err error) int64 {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

func str(s string) *string { return &s }

func employee(code string, unit int64) hrm.EmployeeInput {
	return hrm.EmployeeInput{EmployeeFields: hrm.EmployeeFields{Code: code, FullName: "Nhân viên " + code, OrgUnitID: unit, HireDate: "2026-01-05"}}
}

func TestScopeLimitsEmployees(t *testing.T) {
	f := newFixture(t)
	hrA := f.ctxFor("hr_a", "hr@A")
	viewer := f.ctxFor("viewer_all", "viewer@*")
	nobody := f.ctxFor("nobody")

	ea := must(t)(f.hrm.CreateEmployee(hrA, employee("A1", f.a)))
	if _, err := f.hrm.CreateEmployee(hrA, employee("B1", f.b)); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("create outside scope: %v", err)
	}
	eb := must(t)(f.hrm.CreateEmployee(f.ctxFor("hr_b", "hr@B"), employee("B1", f.b)))
	if _, err := f.hrm.CreateEmployee(hrA, employee("a1", f.a)); !errors.Is(err, hrm.ErrCodeTaken) {
		t.Fatalf("duplicate code: %v", err)
	}

	list, err := f.hrm.Employees(hrA, hrm.EmployeeFilter{Sort: "code", Page: 1, PageSize: 50})
	if err != nil || list.Total != 1 || list.Items[0].ID != ea {
		t.Fatalf("hr_a list = %+v %v", list, err)
	}
	if _, err := f.hrm.Employee(hrA, eb); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("hr_a reads B: %v", err)
	}
	if list, _ := f.hrm.Employees(viewer, hrm.EmployeeFilter{Sort: "-code", Page: 1, PageSize: 50}); list.Total != 2 || list.Items[0].Code != "B1" {
		t.Fatalf("viewer list = %+v", list)
	}
	if list, _ := f.hrm.Employees(nobody, hrm.EmployeeFilter{Sort: "code", Page: 1, PageSize: 50}); list.Total != 0 {
		t.Fatalf("nobody list = %+v", list)
	}

	e, err := f.hrm.Employee(viewer, ea)
	if err != nil || !slices.Equal(e.AllowedActions, []string{"view"}) {
		t.Fatalf("viewer actions = %v %v", e.AllowedActions, err)
	}
	if err := f.hrm.UpdateEmployee(viewer, ea, employee("A1", f.a)); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("viewer edits: %v", err)
	}
	moved := employee("A1", f.b)
	if err := f.hrm.UpdateEmployee(hrA, ea, moved); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("hr_a moves employee out of scope: %v", err)
	}

	e, _ = f.hrm.Employee(hrA, ea)
	if !slices.Equal(e.AllowedActions, []string{"view", "edit", "view_contracts"}) {
		t.Fatalf("hr_a actions = %v", e.AllowedActions)
	}
	disabled := platform.WithProducts(hrA, nil)
	if e, _ := f.hrm.Employee(disabled, ea); !slices.Equal(e.AllowedActions, []string{"view", "view_contracts"}) {
		t.Fatalf("product disabled: actions = %v", e.AllowedActions)
	}
}

func TestListFiltersSortAndPages(t *testing.T) {
	f := newFixture(t)
	hr := f.ctxFor("hr", "hr@*")
	for _, c := range []struct {
		code, name, term string
		unit             int64
	}{{"E3", "Cường", "", f.a}, {"E1", "An", "2026-02-01", f.a}, {"E2", "Bình", "", f.b}} {
		in := employee(c.code, c.unit)
		in.FullName = c.name
		if c.term != "" {
			in.TerminationDate = str(c.term)
		}
		must(t)(f.hrm.CreateEmployee(hr, in))
	}
	codes := func(fl hrm.EmployeeFilter) string {
		t.Helper()
		if fl.Sort == "" {
			fl.Sort = "code"
		}
		fl.Page, fl.PageSize = max(fl.Page, 1), cmp.Or(fl.PageSize, 50)
		list, err := f.hrm.Employees(hr, fl)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, it := range list.Items {
			if want := map[string]string{"E1": "terminated"}[it.Code]; it.Status != cmp.Or(want, "active") {
				t.Errorf("%s status = %s", it.Code, it.Status)
			}
			out = append(out, it.Code)
		}
		return strings.Join(out, ",") + "/" + string(rune('0'+list.Total))
	}
	for _, c := range []struct {
		f    hrm.EmployeeFilter
		want string
	}{
		{hrm.EmployeeFilter{}, "E1,E2,E3/3"},
		{hrm.EmployeeFilter{Sort: "-full_name"}, "E3,E2,E1/3"},
		{hrm.EmployeeFilter{Status: "active"}, "E2,E3/2"},
		{hrm.EmployeeFilter{Status: "terminated"}, "E1/1"},
		{hrm.EmployeeFilter{OrgUnitID: f.b}, "E2/1"},
		{hrm.EmployeeFilter{Q: "bìn"}, "E2/1"},
		{hrm.EmployeeFilter{PageSize: 20, Page: 2}, "/0"},
	} {
		if got := codes(c.f); got != c.want {
			t.Errorf("%+v: got %s, want %s", c.f, got, c.want)
		}
	}
}

func TestSensitiveFields(t *testing.T) {
	f := newFixture(t)
	hrA := f.ctxFor("hr_a", "hr@A")
	sens := f.ctxFor("sens_a", "hr@A", "sensitive_viewer@A")

	in := employee("A1", f.a)
	in.Sensitive = &hrm.SensitiveValues{NationalID: str("079123456789")}
	if _, err := f.hrm.CreateEmployee(hrA, in); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("hr without sensitive role writes CCCD: %v", err)
	}
	id := must(t)(f.hrm.CreateEmployee(sens, in))
	var createdFor int64
	if err := f.pool.QueryRow(t.Context(), `SELECT doc_id FROM audit.log WHERE action = 'hrm.employee_created'`).Scan(&createdFor); err != nil || createdFor != id {
		t.Fatalf("creation audited for %d, want %d (%v)", createdFor, id, err)
	}

	var raw []byte
	if err := f.pool.QueryRow(t.Context(), `SELECT national_id FROM hrm.employees WHERE id = $1`, id).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 || strings.Contains(string(raw), "079123456789") {
		t.Fatalf("national_id stored as %q", raw)
	}
	var auditText string
	if err := f.pool.QueryRow(t.Context(), `SELECT string_agg(data::text, ' ') FROM audit.log WHERE doc_type = 'hrm.employee'`).Scan(&auditText); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(auditText, "079123456789") || !strings.Contains(auditText, "national_id") {
		t.Fatalf("audit = %s", auditText)
	}

	e, _ := f.hrm.Employee(hrA, id)
	if !e.Sensitive.NationalID || e.Sensitive.TaxCode || slices.Contains(e.AllowedActions, "view_sensitive") {
		t.Fatalf("hr_a sees %+v %v", e.Sensitive, e.AllowedActions)
	}
	if _, err := f.hrm.RevealSensitive(hrA, id, "national_id"); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("hr_a reveals: %v", err)
	}
	if e, _ := f.hrm.Employee(sens, id); !slices.Contains(e.AllowedActions, "view_sensitive") {
		t.Fatalf("sens actions = %v", e.AllowedActions)
	}
	v, err := f.hrm.RevealSensitive(sens, id, "national_id")
	if err != nil || v == nil || *v != "079123456789" {
		t.Fatalf("reveal = %v %v", v, err)
	}
	if v, err := f.hrm.RevealSensitive(sens, id, "tax_code"); err != nil || v != nil {
		t.Fatalf("reveal empty = %v %v", v, err)
	}
	var views int
	if err := f.pool.QueryRow(t.Context(), `SELECT count(*) FROM audit.log WHERE action = 'hrm.employee_sensitive_viewed' AND doc_id = $1`, id).Scan(&views); err != nil || views != 2 {
		t.Fatalf("view audits = %d %v", views, err)
	}

	// Plain fields edited without touching sensitive ones keep them.
	upd := employee("A1", f.a)
	upd.FullName = "Tên mới"
	if err := f.hrm.UpdateEmployee(hrA, id, upd); err != nil {
		t.Fatal(err)
	}
	if v, _ := f.hrm.RevealSensitive(sens, id, "national_id"); v == nil || *v != "079123456789" {
		t.Fatalf("after plain update: %v", v)
	}
	upd.Sensitive = &hrm.SensitiveValues{NationalID: str("")}
	if err := f.hrm.UpdateEmployee(sens, id, upd); err != nil {
		t.Fatal(err)
	}
	if e, _ := f.hrm.Employee(sens, id); e.Sensitive.NationalID || e.FullName != "Tên mới" {
		t.Fatalf("after clearing: %+v", e)
	}
}

func TestMoveCannotGrantSensitiveAccess(t *testing.T) {
	f := newFixture(t)
	mover := f.ctxFor("mover", "hr@A", "hr@B", "sensitive_viewer@B")
	id := must(t)(f.hrm.CreateEmployee(mover, employee("A1", f.a)))
	if err := f.hrm.UpdateEmployee(mover, id, employee("A1", f.b)); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("move into own sensitive scope: %v", err)
	}
	both := f.ctxFor("both", "hr@A", "hr@B", "sensitive_viewer@A", "sensitive_viewer@B")
	if err := f.hrm.UpdateEmployee(both, id, employee("A1", f.b)); err != nil {
		t.Fatalf("move with sensitive access on both sides: %v", err)
	}
}

func TestManagerAndAccountLinks(t *testing.T) {
	f := newFixture(t)
	hr := f.ctxFor("hr", "hr@*")
	f.ctxFor("an")
	boss := must(t)(f.hrm.CreateEmployee(hr, employee("M", f.a)))
	in := employee("E", f.a)
	in.ManagerID, in.UserLogin = &boss, str("AN")
	e := must(t)(f.hrm.CreateEmployee(hr, in))
	got, _ := f.hrm.Employee(hr, e)
	if got.ManagerName == nil || *got.ManagerName != "Nhân viên M" || got.UserLogin == nil || *got.UserLogin != "an" {
		t.Fatalf("links = %+v", got)
	}
	other := employee("F", f.a)
	other.UserLogin = str("an")
	if _, err := f.hrm.CreateEmployee(hr, other); !errors.Is(err, hrm.ErrUserAlreadyLinked) {
		t.Fatalf("second link: %v", err)
	}
	other.UserLogin = str("ghost")
	if _, err := f.hrm.CreateEmployee(hr, other); !errors.Is(err, hrm.ErrUserNotFound) {
		t.Fatalf("unknown login: %v", err)
	}
	outside := must(t)(f.hrm.CreateEmployee(hr, employee("B", f.b)))
	hrA := f.ctxFor("hr_a", "hr@A")
	probe := employee("G", f.a)
	probe.ManagerID = &outside
	if _, err := f.hrm.CreateEmployee(hrA, probe); !errors.Is(err, hrm.ErrManagerNotFound) {
		t.Fatalf("manager outside scope: %v", err)
	}
	cyc := employee("M", f.a)
	cyc.ManagerID = &e
	var perr *platform.Error
	if err := f.hrm.UpdateEmployee(hr, boss, cyc); !errors.Is(err, hrm.ErrManagerCycle) || !errors.As(err, &perr) || perr.Params["name"] != "E · Nhân viên E" {
		t.Fatalf("manager cycle: %v %+v", err, perr)
	}
	cyc.ManagerID = &boss
	if err := f.hrm.UpdateEmployee(hr, boss, cyc); !errors.Is(err, hrm.ErrManagerSelf) {
		t.Fatalf("own manager: %v", err)
	}
	// The manager picker for the boss offers neither the boss nor anyone under them.
	list, err := f.hrm.Employees(hr, hrm.EmployeeFilter{ManagerOf: boss, Sort: "code", Page: 1, PageSize: 50})
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range list.Items {
		if it.ID == boss || it.ID == e {
			t.Fatalf("manager_of offers %s", it.Code)
		}
	}
	if list.Total != 1 || list.Items[0].ID != outside {
		t.Fatalf("manager_of = %+v", list.Items)
	}
}

func TestDependents(t *testing.T) {
	f := newFixture(t)
	hr := f.ctxFor("hr", "hr@A")
	sens := f.ctxFor("sens", "hr@A", "sensitive_viewer@A")
	id := must(t)(f.hrm.CreateEmployee(hr, employee("A1", f.a)))

	d := hrm.DependentInput{FullName: "Con", Relationship: "child", DeductionFrom: str("2026-01")}
	if _, err := f.hrm.SaveDependent(hr, id, 0, d); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("hr without sensitive: %v", err)
	}
	did := must(t)(f.hrm.SaveDependent(sens, id, 0, d))
	d.FullName = "Con út"
	must(t)(f.hrm.SaveDependent(sens, id, did, d))
	ds, err := f.hrm.Dependents(sens, id)
	if err != nil || len(ds) != 1 || ds[0].FullName != "Con út" || ds[0].ID != did {
		t.Fatalf("dependents = %+v %v", ds, err)
	}
	if _, err := f.hrm.Dependents(hr, id); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("hr reads dependents: %v", err)
	}
	if err := f.hrm.DeleteDependent(sens, id, did); err != nil {
		t.Fatal(err)
	}
	if err := f.hrm.DeleteDependent(sens, id, did); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("delete twice: %v", err)
	}
	var stored string
	_ = f.pool.QueryRow(t.Context(), `SELECT string_agg(data::text, ' ') FROM audit.log WHERE doc_type = 'hrm.employee'`).Scan(&stored)
	if strings.Contains(stored, "Con") {
		t.Fatalf("dependent in clear in audit: %s", stored)
	}
}

func TestEmployeeActions(t *testing.T) {
	f := newFixture(t)
	for _, c := range []struct {
		ctx  context.Context
		want []string
	}{
		{f.ctxFor("viewer", "viewer@*"), []string{}},
		{f.ctxFor("hr_a", "hr@A"), []string{"create"}},
		{f.ctxFor("sens_a", "hr@A", "sensitive_viewer@B"), []string{"create", "view_sensitive"}},
	} {
		got, err := f.hrm.EmployeeActions(c.ctx)
		if err != nil || !slices.Equal(got, c.want) {
			t.Errorf("actions = %v %v, want %v", got, err, c.want)
		}
	}
	got, _ := f.hrm.EmployeeActions(platform.WithProducts(f.ctxFor("hr_off", "hr@A", "sensitive_viewer@A"), nil))
	if !slices.Equal(got, []string{"view_sensitive"}) {
		t.Errorf("product disabled: %v", got)
	}
}

func TestRevealFailsWhenItsAuditFails(t *testing.T) {
	f := newFixture(t)
	sens := f.ctxFor("sens", "hr@A", "sensitive_viewer@A")
	in := employee("A1", f.a)
	in.Sensitive = &hrm.SensitiveValues{NationalID: str("079123456789")}
	id := must(t)(f.hrm.CreateEmployee(sens, in))

	if _, err := f.pool.Exec(t.Context(), `
		CREATE FUNCTION audit.refuse() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'audit down'; END $$;
		CREATE TRIGGER refuse BEFORE INSERT ON audit.log FOR EACH ROW EXECUTE FUNCTION audit.refuse();`); err != nil {
		t.Fatal(err)
	}
	if v, err := f.hrm.RevealSensitive(sens, id, "national_id"); err == nil || !strings.Contains(err.Error(), "audit down") || v != nil {
		t.Fatalf("reveal without audit = %v, %v; want an error and no value", v, err)
	}
	if _, err := f.hrm.Dependents(sens, id); err == nil || !strings.Contains(err.Error(), "audit down") {
		t.Fatal("dependents listed without audit")
	}
}
