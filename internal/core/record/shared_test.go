package record_test

import (
	"context"
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/coretest"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// widget registers a shared, anywhere catalog of namespace widget, which no product enables.
func widget(t *testing.T) *coretest.Core {
	core := coretest.New(t)
	core.IAM.RegisterRoles("widget", map[string][]string{"viewer": {"widget.view"}, "editor": {"widget.view", "widget.edit"}})
	core.Record.Register(record.Type{
		Code: "widget.widget", Product: "widget", Kind: record.Catalog, Shared: true, Anywhere: true,
		Perms:   record.Perms{View: "widget.view", Edit: "widget.edit"},
		Locate:  func(context.Context, int64) (int64, error) { return 0, nil },
		HasData: func(context.Context) (bool, error) { return true, nil },
	})
	return core
}

// userWith creates a user holding role of widget at a fresh department.
func userWith(t *testing.T, core *coretest.Core, role string) context.Context {
	c := must(t)(core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{Kind: "company", Name: "C"}))
	a := must(t)(core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{ParentID: &c, Kind: "department", Name: "A"}))
	id := must(t)(core.IAM.CreateUser(core.Admin, "u", "U", "long enough"))
	if role != "" {
		must(t)(core.IAM.GrantRole(core.Admin, id, "widget", role, &a))
	}
	return platform.WithActor(core.Ctx, id)
}

func TestSharedTypeNotProductGated(t *testing.T) {
	core := widget(t)
	ctx := platform.WithProducts(userWith(t, core, "editor"), nil)
	if ok, err := core.Record.Can(ctx, "widget.widget", 1, record.Edit); err != nil || !ok {
		t.Fatalf("edit with no product enabled = %v, %v; want true", ok, err)
	}
	if acts, err := core.Record.TypeActions(ctx, "widget.widget"); err != nil || !slices.Contains(acts, "create") {
		t.Fatalf("type actions = %v, %v; want create", acts, err)
	}
	if err := core.Record.WriteGate(ctx, record.Ref{Type: "widget.widget", ID: 1}); err != nil {
		t.Fatalf("write gate: %v", err)
	}
}

func TestAnywhereGrantCoversEveryRecord(t *testing.T) {
	core := widget(t)
	if ok, err := core.Record.Can(userWith(t, core, "viewer"), "widget.widget", 1, record.View); err != nil || !ok {
		t.Fatalf("viewer at A sees a record located nowhere: %v, %v", ok, err)
	}
	none := must(t)(core.IAM.CreateUser(core.Admin, "none", "None", "long enough"))
	if ok, err := core.Record.Can(platform.WithActor(core.Ctx, none), "widget.widget", 1, record.View); err != nil || ok {
		t.Fatalf("user without grants = %v, %v; want false", ok, err)
	}
}

func TestSharedTypeNotProductWithData(t *testing.T) {
	core := widget(t)
	got, err := core.Record.ProductsWithData(core.Admin)
	if err != nil || slices.Contains(got, "widget") {
		t.Fatalf("products with data = %v, %v; want no widget", got, err)
	}
}

func TestAnywhereDocumentRefused(t *testing.T) {
	core := widget(t)
	defer func() {
		if recover() == nil {
			t.Fatal("anywhere document registered")
		}
	}()
	core.Record.Register(record.Type{Code: "widget.doc", Product: "widget", Kind: record.Document, NumberPrefix: "W", Anywhere: true,
		Perms: record.Perms{View: "widget.view", Edit: "widget.edit"}, OnDelete: noRows})
}
