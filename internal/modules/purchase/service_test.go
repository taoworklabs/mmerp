package purchase_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/coretest"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/modules/purchase"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/partner"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
	"github.com/taoworklabs/mmerp/internal/shared/stock"
)

// fixture: company C with purchasing teams A and B and a staff user in A.
type fixture struct {
	t        *testing.T
	core     *coretest.Core
	ctx      context.Context
	admin    context.Context
	staff    context.Context // staff@A
	purchase *purchase.Service
	partners *partner.Service
	stock    *stock.Service
	c, a, b  int64
	ctxFor   func(login string, grants ...string) context.Context
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	core := coretest.New(t, "purchase")
	ctx, admin, ids, rec, num := core.Ctx, core.Admin, core.IAM, core.Record, core.Numbering
	partners := partner.NewService(partner.Deps{IAM: ids, Record: rec, Audit: core.Audit, Numbering: num})
	stk := stock.NewService(stock.Deps{IAM: ids, Audit: core.Audit, Numbering: num, Record: rec, Printing: core.Printing})
	s := purchase.NewService(purchase.Deps{IAM: ids, Record: rec, Audit: core.Audit, Setting: core.Setting, Printing: core.Printing,
		Numbering: num, Partner: partners, Stock: stk, Posting: posting.NewService(posting.Hooks{})})
	f := &fixture{t: t, core: core, ctx: ctx, admin: admin, purchase: s, partners: partners, stock: stk}
	f.c = must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{Kind: "company", Name: "C"}))
	f.a = must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{ParentID: &f.c, Kind: "department", Name: "A"}))
	f.b = must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{ParentID: &f.c, Kind: "department", Name: "B"}))
	f.ctxFor = func(login string, grants ...string) context.Context {
		t.Helper()
		id := must(t)(ids.CreateUser(admin, login, login, "long enough"))
		for _, g := range grants {
			role, unit, _ := strings.Cut(g, "@")
			u := map[string]*int64{"A": &f.a, "B": &f.b, "*": nil}[unit]
			product := "purchase"
			if p, r, ok := strings.Cut(role, "."); ok {
				product, role = p, r
			}
			must(t)(ids.GrantRole(admin, id, product, role, u))
		}
		return platform.WithActor(ctx, id)
	}
	f.staff = f.ctxFor("staff", "staff@A", "partner.editor@A")
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

func (f *fixture) check(err error) {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
}

func str(s string) *string { return &s }

func TestPurchaseRoles(t *testing.T) {
	f := newFixture(t)
	viewer := f.ctxFor("viewer", "viewer@A")
	for _, c := range []struct {
		ctx  context.Context
		perm string
		want bool
	}{
		{f.staff, purchase.PermOrderEdit, true},
		{f.staff, purchase.PermSupplierEdit, true},
		{viewer, purchase.PermOrderView, true},
		{viewer, purchase.PermOrderEdit, false},
	} {
		if got := f.core.IAM.Require(c.ctx, "purchase", c.perm, f.a) == nil; got != c.want {
			t.Errorf("%s = %v, want %v", c.perm, got, c.want)
		}
	}
}

func must2(_ int64, err error) error { return err }

func wantCode(t *testing.T, err error, code string) {
	t.Helper()
	if e, ok := errors.AsType[*platform.Error](err); !ok || e.Code != code {
		t.Fatalf("got %v, want %s", err, code)
	}
}
