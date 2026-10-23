package item_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/coretest"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/item"
)

type fixture struct {
	core  *coretest.Core
	items *item.Service
	unit  int64
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	core := coretest.New(t)
	f := &fixture{core: core, items: item.NewService(item.Deps{IAM: core.IAM, Audit: core.Audit, Numbering: core.Numbering})}
	f.unit = must(t)(core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{Kind: "company", Name: "C"}))
	return f
}

// user creates a user holding item roles: "admin" tenant-wide, "viewer" at the company.
func (f *fixture) user(t *testing.T, login string, roles ...string) context.Context {
	t.Helper()
	id := must(t)(f.core.IAM.CreateUser(f.core.Admin, login, login, "long enough"))
	for _, r := range roles {
		var unit *int64
		if r == "viewer" {
			unit = &f.unit
		}
		must(t)(f.core.IAM.GrantRole(f.core.Admin, id, "item", r, unit))
	}
	return platform.WithActor(f.core.Ctx, id)
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

func fields(name string, active bool) item.ItemFields {
	return item.ItemFields{Name: name, Unit: "cái", Price: 1000, VatRate: "8", Active: active}
}

var page = platform.Paging{Page: 1, PageSize: 50}

func TestItemCRUD(t *testing.T) {
	f := newFixture(t)
	admin := f.user(t, "cat", "admin")
	id := must(t)(f.items.CreateItem(admin, item.NewItem{ItemFields: fields(" Giấy ", true)}))
	must(t)(f.items.CreateItem(admin, item.NewItem{Code: "OLD", ItemFields: fields("Cũ", false)}))
	if err := f.items.UpdateItem(admin, id, fields("Giấy A4", true)); err != nil {
		t.Fatal(err)
	}
	viewer := f.user(t, "viewer", "viewer")
	l, err := f.items.Items(viewer, item.ItemFilter{Active: "true", Paging: page})
	if err != nil || l.Total != 1 || l.Items[0].ID != id || l.Items[0].Code != "HH00001" || l.Items[0].Name != "Giấy A4" {
		t.Fatalf("active items %+v %v", l, err)
	}
	if l, err := f.items.Items(viewer, item.ItemFilter{Active: "false", Paging: page}); err != nil || l.Total != 1 || l.Items[0].Code != "OLD" {
		t.Fatalf("retired items %+v %v", l, err)
	}
	// The engine escapes LIKE metacharacters: "%" matches a literal percent sign only.
	if l, err := f.items.Items(viewer, item.ItemFilter{Q: "%", Paging: page}); err != nil || l.Total != 0 {
		t.Fatalf("q %%: %+v %v", l, err)
	}
	if _, err := f.items.Items(f.user(t, "nobody"), item.ItemFilter{Paging: page}); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("nobody lists: %v", err)
	}
	if _, err := f.items.CreateItem(viewer, item.NewItem{ItemFields: fields("X", true)}); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("viewer creates: %v", err)
	}
	if err := f.items.UpdateItem(admin, id+100, fields("X", true)); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("update missing: %v", err)
	}
	codes := func(sort string) []string {
		l, err := f.items.Items(viewer, item.ItemFilter{Sort: sort, Paging: page})
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, it := range l.Items {
			out = append(out, it.Code)
		}
		return out
	}
	// Names: "Giấy A4" (HH00001), "Cũ" (OLD).
	for sort, want := range map[string][]string{"": {"HH00001", "OLD"}, "-code": {"OLD", "HH00001"}, "name": {"OLD", "HH00001"}} {
		if got := codes(sort); !slices.Equal(got, want) {
			t.Fatalf("sort %q: %v", sort, got)
		}
	}
}

// Managing needs item.manage tenant-wide; held at an org unit it is not enough.
func TestItemManageTenantWide(t *testing.T) {
	f := newFixture(t)
	ctx := f.user(t, "unit_admin")
	id, _ := platform.ActorFrom(ctx)
	// The admin role cannot be granted at an org unit through iam, so plant one.
	if _, err := f.core.System.Exec(f.core.Ctx, `INSERT INTO iam.user_roles (user_id, product, role, org_unit_id) VALUES ($1, 'item', 'admin', $2)`, id, f.unit); err != nil {
		t.Fatal(err)
	}
	if _, err := f.items.CreateItem(ctx, item.NewItem{ItemFields: fields("X", true)}); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("admin at an org unit creates: %v", err)
	}
}

// Items belong to no product: they stay writable with every product disabled.
func TestItemWriteWithProductsDisabled(t *testing.T) {
	f := newFixture(t)
	ctx := platform.WithProducts(f.user(t, "cat", "admin"), nil)
	id := must(t)(f.items.CreateItem(ctx, item.NewItem{ItemFields: fields("X", true)}))
	if err := f.items.UpdateItem(ctx, id, fields("Y", false)); err != nil {
		t.Fatal(err)
	}
}

func TestItemCodeTaken(t *testing.T) {
	f := newFixture(t)
	admin := f.user(t, "cat", "admin")
	must(t)(f.items.CreateItem(admin, item.NewItem{Code: "BUT", ItemFields: fields("Bút", true)}))
	_, err := f.items.CreateItem(admin, item.NewItem{Code: "but", ItemFields: fields("Bút 2", true)})
	if e, ok := errors.AsType[*platform.Error](err); !ok || e.Code != "item_code_taken" {
		t.Fatalf("got %v, want item_code_taken", err)
	}
}
