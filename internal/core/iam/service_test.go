package iam_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/numbering"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/setting"
	"github.com/taoworklabs/mmerp/internal/platform"
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
	s := iam.NewService(iam.Deps{Setting: setting.NewService(), Audit: audit.NewService()})
	rec := record.NewService(record.Deps{IAM: s, Numbering: numbering.NewService(), Audit: audit.NewService()})
	rec.Register(record.Type{Code: "test.doc", Product: "test", Kind: record.Document, NumberPrefix: "T",
		Can: func(context.Context, int64, record.Action) (bool, error) { return true, nil }})
	s.SetDataProducts(rec)
	s.RegisterRoles("hrm", map[string][]string{"hr": {"hrm.employee.view", "hrm.employee.edit"}, "viewer": {"hrm.employee.view"}, "approval_admin": {"hrm.approval.manage"}}, "approval_admin")
	ctx := platform.WithDB(t.Context(), pgtest.New(t))
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

	if _, err := s.CreateUser(ctx, "an", "An", "short"); !errors.Is(err, iam.ErrPasswordTooShort) {
		t.Fatalf("short password: %v", err)
	}
	an := must[int64](t)(s.CreateUser(ctx, "an", "An", "long enough"))
	if _, err := s.CreateUser(ctx, "AN", "An", "long enough"); !errors.Is(err, iam.ErrLoginTaken) {
		t.Fatalf("duplicate login: %v", err)
	}
	if _, err := s.Login(ctx, "An", "long enough"); err != nil {
		t.Fatalf("login is case-insensitive: %v", err)
	}
	if _, err := s.CreateUser(platform.WithActor(ctx, an), "binh", "Bình", "long enough"); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("non-admin creates user: %v", err)
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
	rec.Register(record.Type{Code: "cat.customer", Product: "cat", Kind: record.Catalog,
		Can:     func(context.Context, int64, record.Action) (bool, error) { return true, nil },
		HasData: func(context.Context) (bool, error) { return true, nil }})
	if me := must[iam.Me](t)(s.Me(off)); !slices.Equal(me.ProductsWithData, []string{"cat", "test"}) {
		t.Fatalf("with master data: %+v", me.ProductsWithData)
	}
}
