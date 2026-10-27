package iam_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool" //nolint:depguard // the test moves a lock into the past through the system pool

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/numbering"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/setting"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

// setup returns a service with a test product and a ctx acting as a fresh tenant admin.
func setup(t *testing.T) (*iam.Service, context.Context) {
	t.Helper()
	s, _, ctx := setupWithRecord(t)
	return s, ctx
}

// setupWithRecord also returns record, which has a document type test.doc of product test.
func setupWithRecord(t *testing.T) (*iam.Service, *record.Service, context.Context) {
	t.Helper()
	return setupWithPool(t, pgtest.New(t))
}

func setupWithPool(t *testing.T, pool *pgxpool.Pool) (*iam.Service, *record.Service, context.Context) {
	t.Helper()
	s := iam.NewService(iam.Deps{Setting: setting.NewService(), Audit: audit.NewService()})
	rec := record.NewService(record.Deps{IAM: s, Numbering: numbering.NewService(numbering.Deps{}), Audit: audit.NewService()})
	s.RegisterRoles("test", map[string][]string{"member": {"test.all"}})
	rec.Register(record.Type{Code: "test.doc", Product: "test", Kind: record.Document, NumberPrefix: "T",
		Perms: record.Perms{View: "test.all", Edit: "test.all"}, OnDelete: func(context.Context, int64) error { return nil }})
	s.SetDataProducts(rec)
	s.RegisterRoles("hrm", map[string][]string{"hr": {"hrm.employee.view", "hrm.employee.edit"}, "viewer": {"hrm.employee.view"}, "approval_admin": {"hrm.approval.manage"}}, "approval_admin")
	ctx := platform.WithDB(t.Context(), pool)
	id, err := s.CreateAdmin(ctx, "admin", "Admin", "long enough")
	if err != nil {
		t.Fatal(err)
	}
	return s, rec, platform.WithActor(ctx, id)
}

func must[T any](t *testing.T) func(T, error) T {
	return func(v T, err error) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

func TestCreateUserRules(t *testing.T) {
	s, ctx := setup(t)

	if _, err := s.CreateUser(ctx, "an", "An", "short"); !errors.Is(err, &platform.Error{Code: "password_weak"}) {
		t.Fatalf("short password: %v", err)
	}
	an := must[int64](t)(s.CreateUser(ctx, "an", "An", "long enough"))
	if _, err := s.CreateUser(ctx, "AN", "An", "long enough"); !errors.Is(err, iam.ErrLoginTaken) {
		t.Fatalf("duplicate login: %v", err)
	}
	if _, err := s.Login(ctx, "An", "long enough", ""); err != nil {
		t.Fatalf("login is case-insensitive: %v", err)
	}
	if _, err := s.CreateUser(platform.WithActor(ctx, an), "binh", "Bình", "long enough"); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("non-admin creates user: %v", err)
	}
}

func TestLoginPolicy(t *testing.T) {
	s, ctx := setup(t)

	got := must[iam.LoginPolicy](t)(s.LoginPolicy(ctx))
	if want := (iam.LoginPolicy{MinLength: 8, MaxFailures: 5, LockMinutes: 15}); got != want {
		t.Fatalf("defaults: %+v", got)
	}
	if err := s.SetLoginPolicy(ctx, iam.LoginPolicy{MinLength: 10, Digit: true, MaxFailures: 5, LockMinutes: 15}); err != nil {
		t.Fatal(err)
	}
	_, err := s.CreateUser(ctx, "an", "An", "longenough")
	var pe *platform.Error
	if !errors.As(err, &pe) || pe.Code != "password_weak" || !reflect.DeepEqual(pe.Params["rules"], []string{"digit"}) {
		t.Fatalf("weak password: %v", err)
	}
	if err := s.SetLoginPolicy(ctx, iam.LoginPolicy{MinLength: 7, MaxFailures: 5, LockMinutes: 15}); !errors.As(err, &pe) || pe.Code != "invalid_request" {
		t.Fatalf("out of bounds: %v", err)
	}
	an := must[int64](t)(s.CreateUser(ctx, "an", "An", "longenough1"))
	if err := s.SetLoginPolicy(platform.WithActor(ctx, an), got); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("non-admin: %v", err)
	}
}

func TestOrgTreeShape(t *testing.T) {
	s, ctx := setup(t)
	str := func(v string) *string { return &v }
	unit := func(parent *int64, kind string) (int64, error) {
		in := iam.OrgUnitInput{ParentID: parent, Kind: kind, Name: kind}
		if kind == "company" {
			in.TaxCode = str("0101234567")
		}
		return s.CreateOrgUnit(ctx, in)
	}
	group := must[int64](t)(unit(nil, "group"))
	company := must[int64](t)(unit(&group, "company"))
	dept := must[int64](t)(unit(&company, "department"))
	team := must[int64](t)(unit(&dept, "department"))

	for _, c := range []struct {
		name string
		err  error
		run  func() error
	}{
		{"company under company", iam.ErrOrgUnitMisplaced, func() error { _, err := unit(&dept, "company"); return err }},
		{"group under company", iam.ErrOrgUnitMisplaced, func() error { _, err := unit(&dept, "group"); return err }},
		{"department outside a company", iam.ErrOrgUnitMisplaced, func() error { _, err := unit(&group, "department"); return err }},
		{"legal fields on a department", iam.ErrLegalFieldsCompanyOnly, func() error {
			_, err := s.CreateOrgUnit(ctx, iam.OrgUnitInput{ParentID: &company, Kind: "department", Name: "x", TaxCode: str("1")})
			return err
		}},
		{"move under own descendant", iam.ErrOrgUnitCycle, func() error {
			return s.UpdateOrgUnit(ctx, dept, iam.OrgUnitInput{ParentID: &team, Kind: "department", Name: "d"})
		}},
		{"unknown parent", platform.ErrNotFound, func() error { _, err := unit(new(int64(999)), "department"); return err }},
	} {
		if err := c.run(); !errors.Is(err, c.err) {
			t.Errorf("%s: got %v, want %v", c.name, err, c.err)
		}
	}

	// A rejected write leaves nothing behind, its audit entry included.
	var audited int
	if err := platform.DBFrom(ctx).QueryRow(ctx, `SELECT count(*) FROM audit.log WHERE action LIKE 'iam.org_unit_%'`).Scan(&audited); err != nil || audited != 4 {
		t.Fatalf("org unit audit rows = %d (%v), want 4: only the accepted writes", audited, err)
	}

	before := must[iam.Me](t)(s.Me(ctx)).AuthzVersion
	if err := s.UpdateOrgUnit(ctx, team, iam.OrgUnitInput{ParentID: &company, Kind: "department", Name: "team"}); err != nil {
		t.Fatal(err)
	}
	if after := must[iam.Me](t)(s.Me(ctx)).AuthzVersion; after == before {
		t.Fatalf("moving a node kept authz_version %s", after)
	}
}

func TestScopeAndGrants(t *testing.T) {
	s, ctx := setup(t)
	company := must[int64](t)(s.CreateOrgUnit(ctx, iam.OrgUnitInput{Kind: "company", Name: "C"}))
	a := must[int64](t)(s.CreateOrgUnit(ctx, iam.OrgUnitInput{ParentID: &company, Kind: "department", Name: "A"}))
	a1 := must[int64](t)(s.CreateOrgUnit(ctx, iam.OrgUnitInput{ParentID: &a, Kind: "department", Name: "A1"}))
	b := must[int64](t)(s.CreateOrgUnit(ctx, iam.OrgUnitInput{ParentID: &company, Kind: "department", Name: "B"}))

	an := must[int64](t)(s.CreateUser(ctx, "an", "An", "long enough"))
	asAn := platform.WithActor(ctx, an)
	v0 := must[iam.Me](t)(s.Me(asAn)).AuthzVersion

	grant := must[int64](t)(s.GrantRole(ctx, an, "hrm", "viewer", &a))
	if _, err := s.GrantRole(ctx, an, "hrm", "viewer", &a); !errors.Is(err, iam.ErrRoleAlreadyGranted) {
		t.Fatalf("duplicate grant: %v", err)
	}
	for _, c := range []struct {
		product, role string
		unit          *int64
		err           error
	}{
		{"hrm", "boss", nil, iam.ErrUnknownRole},
		{"core", "admin", &a, iam.ErrRoleTenantWide},
		{"hrm", "approval_admin", &a, iam.ErrRoleTenantWide},
	} {
		if _, err := s.GrantRole(ctx, an, c.product, c.role, c.unit); !errors.Is(err, c.err) {
			t.Errorf("grant %s.%s: %v, want %v", c.product, c.role, err, c.err)
		}
	}

	me := must[iam.Me](t)(s.Me(asAn))
	if me.AuthzVersion == v0 || !slices.Equal(me.Permissions["hrm"], []string{"hrm.employee.view"}) {
		t.Fatalf("after grant: %+v", me)
	}
	sc := must[iam.Scope](t)(s.Scope(asAn, "hrm", "hrm.employee.view"))
	if sc.All || !sc.Has(a) || !sc.Has(a1) || sc.Has(b) || sc.Has(company) {
		t.Fatalf("scope = %+v (a=%d a1=%d b=%d)", sc, a, a1, b)
	}
	if sc := must[iam.Scope](t)(s.Scope(asAn, "hrm", "hrm.employee.edit")); sc.Has(a) {
		t.Fatalf("viewer can edit: %+v", sc)
	}
	// Allowed needs every unit; tenant-wide needs a grant above every unit.
	if ok, err := s.Allowed(asAn, "hrm", "hrm.employee.view", a, a1); err != nil || !ok {
		t.Fatalf("allowed at a and a1: %v %v", ok, err)
	}
	if err := s.Require(asAn, "hrm", "hrm.employee.view", a, b); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("required at a and b: %v", err)
	}
	if err := s.RequireTenantWide(asAn, "hrm", "hrm.employee.view"); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("tenant-wide from a grant at a: %v", err)
	}
	if units := must[[]iam.OrgUnit](t)(s.OrgUnits(asAn, "", "")); len(units) != 4 {
		t.Fatalf("whole tree for a signed-in user = %+v", units)
	}
	if _, err := s.CreateOrgUnit(asAn, iam.OrgUnitInput{ParentID: &company, Kind: "department", Name: "X"}); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("non-admin creates a unit: %v", err)
	}
	if err := s.UpdateOrgUnit(asAn, b, iam.OrgUnitInput{ParentID: &a, Kind: "department", Name: "B"}); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("non-admin moves a unit: %v", err)
	}
	if units := must[[]iam.OrgUnit](t)(s.OrgUnits(asAn, "hrm", "hrm.employee.view")); len(units) != 2 {
		t.Fatalf("scoped units = %+v", units)
	}

	must[int64](t)(s.GrantRole(ctx, an, "hrm", "hr", nil))
	must[int64](t)(s.GrantRole(ctx, an, "hrm", "approval_admin", nil))
	for _, r := range s.Roles() {
		if want := r.Product == "core" || r.Role == "approval_admin"; r.TenantWide != want {
			t.Errorf("role %s.%s tenant_wide = %v", r.Product, r.Role, r.TenantWide)
		}
	}
	if sc := must[iam.Scope](t)(s.Scope(asAn, "hrm", "hrm.employee.edit")); !sc.All {
		t.Fatalf("tenant-wide grant: %+v", sc)
	}

	v1 := must[iam.Me](t)(s.Me(asAn)).AuthzVersion
	if err := s.RevokeRole(ctx, an, grant); err != nil {
		t.Fatal(err)
	}
	if err := s.RevokeRole(ctx, an, grant); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("revoke twice: %v", err)
	}
	if v2 := must[iam.Me](t)(s.Me(asAn)).AuthzVersion; v2 == v1 {
		t.Fatal("revoke kept authz_version")
	}
}

// DataScope must agree with Scope for an admin, a unit-scoped user and a user
// with no role: it only converts, it never decides access on its own.
func TestDataScopeMatchesScope(t *testing.T) {
	s, ctx := setup(t)
	company := must[int64](t)(s.CreateOrgUnit(ctx, iam.OrgUnitInput{Kind: "company", Name: "C"}))
	a := must[int64](t)(s.CreateOrgUnit(ctx, iam.OrgUnitInput{ParentID: &company, Kind: "department", Name: "A"}))

	viewer := must[int64](t)(s.CreateUser(ctx, "viewer", "Viewer", "long enough"))
	must[int64](t)(s.GrantRole(ctx, viewer, "hrm", "viewer", &a))
	noRole := must[int64](t)(s.CreateUser(ctx, "norole", "No Role", "long enough"))

	for _, c := range []struct {
		name string
		ctx  context.Context
	}{
		{"admin", ctx},
		{"unit-scoped", platform.WithActor(ctx, viewer)},
		{"no role", platform.WithActor(ctx, noRole)},
	} {
		sc := must[iam.Scope](t)(s.Scope(c.ctx, "hrm", "hrm.employee.view"))
		ds := must[data.Scope](t)(s.DataScope(c.ctx, "hrm", "hrm.employee.view"))
		want := data.Scope{}
		switch {
		case sc.All:
			want = data.GrantAll()
		case len(sc.Units) > 0:
			want = data.GrantUnits(sc.Units)
		}
		if !reflect.DeepEqual(ds, want) {
			t.Errorf("%s: DataScope = %+v, want %+v (Scope = %+v)", c.name, ds, want, sc)
		}
	}
}

func TestMeListsProductsWithData(t *testing.T) {
	s, rec, ctx := setupWithRecord(t)
	off := platform.WithProducts(ctx, nil)
	if me := must[iam.Me](t)(s.Me(off)); len(me.ProductsWithData) != 0 {
		t.Fatalf("no records yet: %v", me.ProductsWithData)
	}
	company := must[int64](t)(s.CreateOrgUnit(ctx, iam.OrgUnitInput{Kind: "company", Name: "C"}))
	if _, err := rec.Create(platform.WithProducts(ctx, []string{"test"}), "test.doc", record.Header{Date: "2026-03-10", OrgUnitID: company}); err != nil {
		t.Fatal(err)
	}
	// The product is off, yet its data shows.
	if me := must[iam.Me](t)(s.Me(off)); !slices.Equal(me.ProductsWithData, []string{"test"}) || len(me.Products) != 0 {
		t.Fatalf("after a record: %+v", me)
	}
	// Master data counts too, without any document.
	s.RegisterRoles("cat", map[string][]string{"member": {"cat.all"}})
	rec.Register(record.Type{Code: "cat.customer", Product: "cat", Kind: record.Catalog,
		Perms:   record.Perms{View: "cat.all"},
		Locate:  func(context.Context, int64) (int64, error) { return 0, platform.ErrNotFound },
		HasData: func(context.Context) (bool, error) { return true, nil }})
	if me := must[iam.Me](t)(s.Me(off)); !slices.Equal(me.ProductsWithData, []string{"cat", "test"}) {
		t.Fatalf("with master data: %+v", me.ProductsWithData)
	}
}

func TestListSettingsArePersonal(t *testing.T) {
	s, ctx := setup(t)
	if got := must[iam.ListSettings](t)(s.ListSettings(ctx, "sales.customers")); len(got.Views) != 0 || len(got.Columns) != 0 {
		t.Fatalf("nothing saved should read empty, got %+v", got)
	}

	width := 240
	in := iam.ListSettings{
		Views:   []iam.ListView{{Name: " Mine ", Query: "q=an&sort=-code"}},
		Columns: []iam.ListColumn{{Key: "phone", Hidden: true}, {Key: "name", Width: &width}},
	}
	if err := s.SetListSettings(ctx, "sales.customers", in); err != nil {
		t.Fatal(err)
	}
	got := must[iam.ListSettings](t)(s.ListSettings(ctx, "sales.customers"))
	if len(got.Views) != 1 || got.Views[0].Name != "Mine" || got.Views[0].Query != "q=an&sort=-code" {
		t.Fatalf("views: %+v", got.Views)
	}
	if len(got.Columns) != 2 || !got.Columns[0].Hidden || got.Columns[1].Width == nil || *got.Columns[1].Width != 240 {
		t.Fatalf("columns: %+v", got.Columns)
	}

	// Another list of the same user and the same list of another user stay apart.
	if got := must[iam.ListSettings](t)(s.ListSettings(ctx, "sales.items")); len(got.Views) != 0 {
		t.Fatalf("other list leaked: %+v", got)
	}
	other := must[int64](t)(s.CreateUser(ctx, "binh", "Bình", "long enough"))
	if got := must[iam.ListSettings](t)(s.ListSettings(platform.WithActor(ctx, other), "sales.customers")); len(got.Views) != 0 {
		t.Fatalf("other user sees my views: %+v", got)
	}

	dup := iam.ListSettings{Views: []iam.ListView{{Name: "Mine", Query: "a=1"}, {Name: "mine ", Query: "a=2"}}}
	if err := s.SetListSettings(ctx, "sales.customers", dup); !errors.Is(err, iam.ErrListViewNameTaken) {
		t.Fatalf("duplicate names: %v", err)
	}
}

func TestAdminHoldsSharedNamespace(t *testing.T) {
	s, ctx := setup(t)
	s.RegisterRoles("widget", map[string][]string{"editor": {"widget.view", "widget.edit"}})
	if sc := must[iam.Scope](t)(s.Scope(ctx, "widget", "widget.edit")); !sc.All {
		t.Fatalf("admin scope on widget.edit = %+v, want all", sc)
	}
}

func TestLockout(t *testing.T) {
	pool := pgtest.New(t)
	s, _, ctx := setupWithPool(t, pool)
	must[int64](t)(s.CreateUser(ctx, "an", "An", "right password"))
	locked := func(err error) bool { return errors.Is(err, iam.ErrLoginLocked) }

	for range 4 {
		if _, err := s.Login(ctx, "an", "wrong", ""); !errors.Is(err, iam.ErrInvalidCredentials) {
			t.Fatalf("wrong password: %v", err)
		}
	}
	must[string](t)(s.Login(ctx, "an", "right password", ""))
	// The success reset the count: four more wrong ones do not lock.
	for range 4 {
		_, _ = s.Login(ctx, "AN", "wrong", "")
	}
	must[string](t)(s.Login(ctx, "an", "right password", ""))

	for i := range 5 {
		_, err := s.Login(ctx, "an", "wrong", "")
		if !errors.Is(err, iam.ErrInvalidCredentials) {
			t.Fatalf("attempt %d (the 5th sets the lock): %v", i+1, err)
		}
	}
	count := func(action string) (n int) {
		if err := platform.DBFrom(ctx).QueryRow(ctx, `SELECT count(*) FROM audit.log WHERE action = $1`, action).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	failed := count("iam.login_failed")
	if _, err := s.Login(ctx, "AN", "right password", ""); !locked(err) {
		t.Fatalf("locked login: %v", err)
	}
	if got := count("iam.login_failed"); got != failed {
		t.Fatalf("a locked attempt wrote login_failed rows: %d -> %d", failed, got)
	}
	for range 5 {
		_, _ = s.Login(ctx, "ghost", "wrong", "")
	}
	if _, err := s.Login(ctx, "ghost", "wrong", ""); !locked(err) {
		t.Fatalf("unknown login: %v", err)
	}

	sys := pgtest.System(t, pool)
	if _, err := sys.Exec(ctx, `UPDATE iam.login_failures SET locked_until = now() - interval '1 second'`); err != nil {
		t.Fatal(err)
	}
	must[string](t)(s.Login(ctx, "an", "right password", ""))
}

func TestLockoutCountsAtomically(t *testing.T) {
	s, ctx := setup(t)
	must[int64](t)(s.CreateUser(ctx, "an", "An", "right password"))

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = s.Login(ctx, "an", "wrong", "")
		}()
	}
	wg.Wait()
	var n int
	if err := platform.DBFrom(ctx).QueryRow(ctx, `SELECT count(*) FROM audit.log WHERE action = 'iam.login_failed'`).Scan(&n); err != nil || n != 5 {
		t.Fatalf("login_failed rows = %d (%v), want 5: the rest are refused before hashing", n, err)
	}
	if err := platform.DBFrom(ctx).QueryRow(ctx, `SELECT count(*) FROM audit.log WHERE action = 'iam.login_locked'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("login_locked rows = %d (%v)", n, err)
	}
	if _, err := s.Login(ctx, "an", "right password", ""); !errors.Is(err, iam.ErrLoginLocked) {
		t.Fatalf("after lock: %v", err)
	}
}

func TestLockUser(t *testing.T) {
	s, ctx := setup(t)
	id := must[int64](t)(s.CreateUser(ctx, "an", "An", "right password"))
	token := must[string](t)(s.Login(ctx, "an", "right password", ""))

	if err := s.LockUser(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.Authenticate(ctx, token); ok {
		t.Fatal("session survived the lock")
	}
	if _, err := s.Login(ctx, "an", "wrong", ""); !errors.Is(err, iam.ErrInvalidCredentials) {
		t.Fatalf("wrong password: %v", err)
	}
	if _, err := s.Login(ctx, "an", "right password", ""); !errors.Is(err, iam.ErrAccountLocked) {
		t.Fatalf("right password: %v", err)
	}
	if u := must[iam.User](t)(s.User(ctx, id)); !u.Locked {
		t.Fatal("user not reported locked")
	}

	// A temporary lock built up meanwhile is cleared by unlock too.
	for range 5 {
		_, _ = s.Login(ctx, "an", "wrong", "")
	}
	if err := s.UnlockUser(ctx, id); err != nil {
		t.Fatal(err)
	}
	must[string](t)(s.Login(ctx, "an", "right password", ""))
	if err := s.LockUser(ctx, 999999); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("unknown id: %v", err)
	}
}

func TestLockedUserSessionIsDead(t *testing.T) {
	pool := pgtest.New(t)
	s, _, ctx := setupWithPool(t, pool)
	id := must[int64](t)(s.CreateUser(ctx, "an", "An", "right password"))
	if err := s.LockUser(ctx, id); err != nil {
		t.Fatal(err)
	}
	sys := pgtest.System(t, pool)
	if _, err := sys.Exec(ctx, `INSERT INTO iam.sessions (token_hash, user_id, expires_at) VALUES (sha256('tok'::bytea), $1, now() + interval '1 day')`, id); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := s.Authenticate(ctx, "tok"); err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

// A sign-in that verified the old password while a password change commits must not open a session.
func TestLoginRacingPasswordChange(t *testing.T) {
	pool := pgtest.New(t)
	s, _, ctx := setupWithPool(t, pool)
	id := must[int64](t)(s.CreateUser(ctx, "an", "An", "right password"))
	sys := pgtest.System(t, pool)
	// The uncommitted change holds the row lock; the sign-in still sees the old hash.
	updated, release, changed := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		changed <- platform.InTx(platform.WithDB(ctx, sys), func(ctx context.Context) error {
			if _, err := platform.DBFrom(ctx).Exec(ctx, `UPDATE iam.users SET password_hash = 'changed' WHERE id = $1`, id); err != nil {
				return err
			}
			close(updated)
			<-release
			return nil
		})
	}()
	select {
	case <-updated:
	case err := <-changed:
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := s.Login(ctx, "an", "right password", ""); done <- err }()
	for waiting := false; !waiting; {
		if err := sys.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE wait_event_type = 'Lock' AND datname = current_database())`).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		select {
		case err := <-done:
			close(release)
			t.Fatalf("login finished without waiting for the change: %v", err)
		default:
		}
	}
	close(release)
	if err := <-changed; err != nil {
		t.Fatal(err)
	}
	if err := <-done; !errors.Is(err, iam.ErrInvalidCredentials) {
		t.Fatalf("login after the change committed: %v", err)
	}
	var sessions int
	if err := sys.QueryRow(ctx, `SELECT count(*) FROM iam.sessions WHERE user_id = $1`, id).Scan(&sessions); err != nil || sessions != 0 {
		t.Fatalf("sessions=%d err=%v", sessions, err)
	}
}

func TestRenameUser(t *testing.T) {
	s, ctx := setup(t)
	id := must[int64](t)(s.CreateUser(ctx, "an", "An", "right password"))
	if err := s.RenameUser(ctx, id, "  "); err == nil {
		t.Fatal("blank name accepted")
	} else if e, ok := errors.AsType[*platform.Error](err); !ok || e.Code != "invalid_request" {
		t.Fatalf("blank name: %v", err)
	}
	tok := must[string](t)(s.Login(ctx, "an", "right password", ""))
	before, _, _ := s.Authenticate(ctx, tok)
	if err := s.RenameUser(ctx, id, " Anh "); err != nil {
		t.Fatal(err)
	}
	if after, _, _ := s.Authenticate(ctx, tok); after.AuthzVersion == before.AuthzVersion {
		t.Fatal("rename did not bump authz version")
	}
	if u := must[iam.User](t)(s.User(ctx, id)); u.Name != "Anh" {
		t.Fatalf("name = %q", u.Name)
	}
	var n int
	if err := platform.DBFrom(ctx).QueryRow(ctx, `SELECT count(*) FROM audit.log WHERE action = 'iam.user_renamed'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("audit rows = %d (%v)", n, err)
	}
}

func TestChangePassword(t *testing.T) {
	s, ctx := setup(t)
	id := must[int64](t)(s.CreateUser(ctx, "an", "An", "right password"))
	t1 := must[string](t)(s.Login(ctx, "an", "right password", ""))
	t2 := must[string](t)(s.Login(ctx, "an", "right password", ""))
	me := platform.WithActor(ctx, id)

	if err := s.ChangePassword(me, t1, "nope", "new password 1"); !errors.Is(err, iam.ErrWrongPassword) {
		t.Fatalf("wrong current: %v", err)
	}
	if e, ok := errors.AsType[*platform.Error](s.ChangePassword(me, t1, "right password", "short")); !ok || e.Code != "password_weak" {
		t.Fatalf("weak: %v", e)
	}
	if err := s.ChangePassword(me, t1, "right password", "new password 1"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.Authenticate(ctx, t1); !ok {
		t.Fatal("own session dropped")
	}
	if _, ok, _ := s.Authenticate(ctx, t2); ok {
		t.Fatal("other session survived")
	}
	if _, err := s.Login(ctx, "an", "right password", ""); !errors.Is(err, iam.ErrInvalidCredentials) {
		t.Fatalf("old password: %v", err)
	}
	must[string](t)(s.Login(ctx, "an", "new password 1", ""))
}

func TestResetPassword(t *testing.T) {
	s, ctx := setup(t)
	id := must[int64](t)(s.CreateUser(ctx, "an", "An", "right password"))
	token := must[string](t)(s.Login(ctx, "an", "right password", ""))
	if err := s.ResetPassword(platform.WithActor(ctx, id), id, "temporary 1"); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("non-admin: %v", err)
	}
	if err := s.ResetPassword(ctx, id, "temporary 1"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.Authenticate(ctx, token); ok {
		t.Fatal("session survived the reset")
	}
	token = must[string](t)(s.Login(ctx, "an", "temporary 1", ""))
	sess, ok, _ := s.Authenticate(ctx, token)
	if !ok || !sess.MustChangePassword {
		t.Fatalf("session = %+v", sess)
	}
	if !must[iam.Me](t)(s.Me(platform.WithActor(ctx, id))).MustChangePassword {
		t.Fatal("Me does not report the forced change")
	}
	if err := s.ChangePassword(platform.WithActor(ctx, id), token, "temporary 1", "chosen password"); err != nil {
		t.Fatal(err)
	}
	if sess, _, _ := s.Authenticate(ctx, token); sess.MustChangePassword {
		t.Fatal("flag not cleared")
	}
}

func TestMySessions(t *testing.T) {
	s, ctx := setup(t)
	id := must[int64](t)(s.CreateUser(ctx, "an", "An", "right password"))
	other := must[int64](t)(s.CreateUser(ctx, "bo", "Bo", "right password"))
	ta := must[string](t)(s.Login(ctx, "an", "right password", "A"))
	tb := must[string](t)(s.Login(ctx, "an", "right password", "B"))
	tother := must[string](t)(s.Login(ctx, "bo", "right password", "C"))
	me := platform.WithActor(ctx, id)

	list := must[[]iam.SessionInfo](t)(s.MySessions(me, tb))
	if len(list) != 2 || !list[0].Current || list[0].UserAgent != "B" || list[1].Current {
		t.Fatalf("list = %+v", list)
	}
	theirs := must[[]iam.SessionInfo](t)(s.MySessions(platform.WithActor(ctx, other), tother))
	if err := s.EndMySession(me, theirs[0].ID); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("other user's session: %v", err)
	}
	if err := s.EndMyOtherSessions(me, tb); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.Authenticate(ctx, ta); ok {
		t.Fatal("session A still valid")
	}
	list = must[[]iam.SessionInfo](t)(s.MySessions(me, tb))
	if len(list) != 1 || !list[0].Current {
		t.Fatalf("list = %+v", list)
	}
	if err := s.EndMySession(me, list[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.Authenticate(ctx, tb); ok {
		t.Fatal("session B still valid")
	}
}

func TestChangePasswordWeakDoesNotLock(t *testing.T) {
	s, ctx := setup(t)
	id := must[int64](t)(s.CreateUser(ctx, "an", "An", "right password"))
	token := must[string](t)(s.Login(ctx, "an", "right password", ""))
	me := platform.WithActor(ctx, id)
	for range 5 {
		err := s.ChangePassword(me, token, "right password", "short")
		if e, ok := errors.AsType[*platform.Error](err); !ok || e.Code != "password_weak" {
			t.Fatalf("weak next: %v", err)
		}
	}
	if _, err := s.Login(ctx, "an", "right password", ""); err != nil {
		t.Fatalf("login after weak tries: %v", err)
	}
}

func TestChangePasswordThrottled(t *testing.T) {
	s, ctx := setup(t)
	id := must[int64](t)(s.CreateUser(ctx, "an", "An", "right password"))
	token := must[string](t)(s.Login(ctx, "an", "right password", ""))
	me := platform.WithActor(ctx, id)
	for range 5 {
		if err := s.ChangePassword(me, token, "nope", "new password 1"); !errors.Is(err, iam.ErrWrongPassword) {
			t.Fatalf("wrong current: %v", err)
		}
	}
	if err := s.ChangePassword(me, token, "right password", "new password 1"); !errors.Is(err, iam.ErrLoginLocked) {
		t.Fatalf("right current while locked: %v", err)
	}
	if _, err := s.Login(ctx, "an", "right password", ""); !errors.Is(err, iam.ErrLoginLocked) {
		t.Fatalf("login while locked: %v", err)
	}
}

func TestPasswordAuditAndResetPolicy(t *testing.T) {
	s, ctx := setup(t)
	id := must[int64](t)(s.CreateUser(ctx, "minhan", "An", "right password"))
	if e, ok := errors.AsType[*platform.Error](s.ResetPassword(ctx, id, "xx-minhan-secret")); !ok || e.Code != "password_weak" {
		t.Fatalf("temporary with login: %v", e)
	}
	token := must[string](t)(s.Login(ctx, "minhan", "right password", ""))
	if err := s.ChangePassword(platform.WithActor(ctx, id), token, "right password", "new password 1"); err != nil {
		t.Fatal(err)
	}
	if err := s.ResetPassword(ctx, id, "temporary 1"); err != nil {
		t.Fatal(err)
	}
	for _, a := range []string{"iam.password_changed", "iam.password_reset"} {
		var n int
		if err := platform.DBFrom(ctx).QueryRow(ctx, `SELECT count(*) FROM audit.log WHERE action = $1`, a).Scan(&n); err != nil || n != 1 {
			t.Fatalf("%s rows = %d (%v)", a, n, err)
		}
	}
}

func TestLoginUserAgentStored(t *testing.T) {
	s, ctx := setup(t)
	id := must[int64](t)(s.CreateUser(ctx, "an", "An", "right password"))
	me := platform.WithActor(ctx, id)
	must[string](t)(s.Login(ctx, "an", "right password", "bad\xff\xfeua"))
	must[string](t)(s.Login(ctx, "an", "right password", strings.Repeat("é", 400)))
	list := must[[]iam.SessionInfo](t)(s.MySessions(me, ""))
	if len(list) != 2 || list[0].UserAgent != strings.Repeat("é", 300) || list[1].UserAgent != "badua" {
		t.Fatalf("user agents = %+v", list)
	}
}

func TestRoot(t *testing.T) {
	pool := pgtest.New(t)
	s, _, ctx := setupWithPool(t, pool)
	sys := platform.WithDB(t.Context(), pool)

	if err := s.EnsureRoot(sys, ""); err == nil {
		t.Fatal("no password: created")
	}
	if err := s.EnsureRoot(sys, "root1234"); err == nil || !strings.Contains(err.Error(), "contains_login") {
		t.Fatalf("weak password: %v", err)
	}
	if err := s.EnsureRoot(sys, "long enough"); err != nil {
		t.Fatal(err)
	}
	if err := s.EnsureRoot(sys, "another password"); err != nil {
		t.Fatalf("second call: %v", err)
	}
	root := must[int64](t)(s.UserIDByLogin(ctx, iam.RootLogin))
	if u := must[iam.User](t)(s.User(ctx, root)); !u.Root || u.Name != "root" {
		t.Fatalf("root = %+v", u)
	}
	if !slices.ContainsFunc(must[[]iam.User](t)(s.Users(ctx)), func(u iam.User) bool { return u.Root && u.ID == root }) {
		t.Fatal("Users does not mark root")
	}
	token := must[string](t)(s.Login(ctx, "ROOT", "long enough", ""))
	rootCtx := platform.WithActor(ctx, root)
	if !must[iam.Me](t)(s.Me(rootCtx)).MustChangePassword {
		t.Fatal("root must change its first password")
	}

	protected := func(what string, err error) {
		t.Helper()
		if !errors.Is(err, iam.ErrRootProtected) {
			t.Fatalf("%s: %v", what, err)
		}
	}
	protected("lock", s.LockUser(ctx, root))
	protected("reset", s.ResetPassword(ctx, root, "temporary 1"))
	grants := must[[]iam.Grant](t)(s.UserRoles(ctx, root))
	adminGrant := grants[slices.IndexFunc(grants, func(g iam.Grant) bool { return g.Product == "core" && g.Role == "admin" })].ID
	protected("revoke", s.RevokeRole(ctx, root, adminGrant))
	protected("revoke own", s.RevokeRole(rootCtx, root, adminGrant))
	if _, err := s.CreateUser(ctx, "Root", "Someone", "long enough"); !errors.Is(err, iam.ErrLoginTaken) {
		t.Fatalf("create root: %v", err)
	}
	if err := s.RenameUser(ctx, root, "Superuser"); err != nil {
		t.Fatalf("rename root: %v", err)
	}

	for range 5 {
		_, _ = s.Login(ctx, "root", "wrong", "")
	}
	if _, err := s.Login(ctx, "root", "long enough", ""); !errors.Is(err, iam.ErrLoginLocked) {
		t.Fatalf("root is throttled too: %v", err)
	}
	if err := s.RecoverPassword(sys, "root", "another long one"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.Authenticate(ctx, token); ok {
		t.Fatal("session survived the recovery")
	}
	token = must[string](t)(s.Login(ctx, "root", "another long one", ""))
	if sess, _, _ := s.Authenticate(ctx, token); sess.MustChangePassword {
		t.Fatal("forced change not cleared")
	}
	if err := s.RecoverPassword(sys, "nobody", "another long one"); err == nil || !strings.Contains(err.Error(), "nobody") {
		t.Fatalf("unknown login: %v", err)
	}
}

func TestRecoverPasswordUnlocks(t *testing.T) {
	pool := pgtest.New(t)
	s, _, ctx := setupWithPool(t, pool)
	id := must[int64](t)(s.CreateUser(ctx, "an", "An", "right password"))
	if err := s.LockUser(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := s.RecoverPassword(platform.WithDB(t.Context(), pool), "AN", "new password"); err != nil {
		t.Fatal(err)
	}
	must[string](t)(s.Login(ctx, "an", "new password", ""))
}

func TestEnsureRootRefusesPlainRoot(t *testing.T) {
	pool := pgtest.New(t)
	s, _, _ := setupWithPool(t, pool)
	sys := platform.WithDB(t.Context(), pool)
	if _, err := s.CreateAdmin(sys, "Root", "Someone", "long enough"); !errors.Is(err, iam.ErrLoginTaken) {
		t.Fatalf("CreateAdmin root: %v", err)
	}
	if _, err := pgtest.System(t, pool).Exec(sys, `INSERT INTO iam.users (login, name, password_hash) VALUES ('Root', 'Someone', 'x')`); err != nil {
		t.Fatal(err)
	}
	if err := s.EnsureRoot(sys, "long enough"); err == nil {
		t.Fatal("a plain user named root became the root")
	}
}
