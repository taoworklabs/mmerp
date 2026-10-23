package partner_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/coretest"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/partner"
)

type fixture struct {
	core     *coretest.Core
	partners *partner.Service
	a, b     int64
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	core := coretest.New(t)
	f := &fixture{core: core, partners: partner.NewService(partner.Deps{IAM: core.IAM, Record: core.Record, Audit: core.Audit, Numbering: core.Numbering})}
	c := must(t)(core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{Kind: "company", Name: "C"}))
	f.a = must(t)(core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{ParentID: &c, Kind: "department", Name: "A"}))
	f.b = must(t)(core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{ParentID: &c, Kind: "department", Name: "B"}))
	return f
}

// user creates a user holding a partner role at unit.
func (f *fixture) user(t *testing.T, login, role string, unit int64) context.Context {
	t.Helper()
	id := must(t)(f.core.IAM.CreateUser(f.core.Admin, login, login, "long enough"))
	if role != "" {
		must(t)(f.core.IAM.GrantRole(f.core.Admin, id, "partner", role, &unit))
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

func str(s string) *string { return &s }

func wantCode(t *testing.T, err error, code string) *platform.Error {
	t.Helper()
	e, ok := errors.AsType[*platform.Error](err)
	if !ok || e.Code != code {
		t.Fatalf("got %v, want %s", err, code)
	}
	return e
}

var page = platform.Paging{Page: 1, PageSize: 50}

func TestPartnerCRUD(t *testing.T) {
	f := newFixture(t)
	editor := f.user(t, "editor", "editor", f.a)
	id := must(t)(f.partners.Create(editor, "", partner.PartnerFields{Name: " Công ty Ánh Dương ", TaxCode: str(" 0312345678 "), Address: str(" "), ContactName: str("Chị Hương")}))
	must(t)(f.partners.Create(editor, "own", partner.PartnerFields{Name: "Bình Minh"}))
	p, err := f.partners.Get(editor, id)
	if err != nil || p.Code != "DT00001" || p.Name != "Công ty Ánh Dương" || *p.TaxCode != "0312345678" || p.Address != nil || !slices.Equal(p.AllowedActions, []string{"edit"}) {
		t.Fatalf("created %+v %v", p, err)
	}
	in := p.PartnerFields
	in.Phone = str("0909")
	if err := f.partners.Update(editor, id, in); err != nil {
		t.Fatal(err)
	}
	viewer := f.user(t, "viewer", "viewer", f.a)
	if p, err := f.partners.Get(viewer, id); err != nil || *p.Phone != "0909" || len(p.AllowedActions) != 0 {
		t.Fatalf("viewer reads %+v %v", p, err)
	}
	// An unchanged identity saves without edit; a changed one needs it.
	if err := f.partners.Update(viewer, id, in); err != nil {
		t.Fatalf("viewer saves unchanged: %v", err)
	}
	// The route always needs edit, so it tells no outsider which partners exist.
	if err := f.partners.Edit(f.user(t, "outsider", "", 0), id, in); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("outsider saves unchanged through the route: %v", err)
	}
	if err := f.partners.Edit(editor, id, in); err != nil {
		t.Fatal(err)
	}
	in.Name = "Đổi"
	if err := f.partners.Update(viewer, id, in); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("viewer edits: %v", err)
	}
	if _, err := f.partners.Create(viewer, "", partner.PartnerFields{Name: "X"}); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("viewer creates: %v", err)
	}
	if err := f.partners.Update(editor, id+100, in); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("update missing: %v", err)
	}
	wantCode(t, must2(f.partners.Create(editor, "OWN", partner.PartnerFields{Name: "Y"})), "partner_code_taken")

	l, err := f.partners.List(viewer, partner.PartnerFilter{Q: "ánh", Paging: page})
	if err != nil || l.Total != 1 || l.Items[0].ID != id {
		t.Fatalf("search %+v %v", l, err)
	}
	if l, err := f.partners.List(viewer, partner.PartnerFilter{TaxCode: " 0312345678", Paging: page}); err != nil || l.Total != 1 || l.Items[0].ID != id {
		t.Fatalf("by tax code %+v %v", l, err)
	}
	if l, err := f.partners.List(viewer, partner.PartnerFilter{Sort: "-code", Paging: page}); err != nil || l.Total != 2 || l.Items[0].Code != "own" {
		t.Fatalf("sorted %+v %v", l, err)
	}
	nobody := f.user(t, "nobody", "", 0)
	if _, err := f.partners.List(nobody, partner.PartnerFilter{Paging: page}); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("nobody lists: %v", err)
	}
	if _, err := f.partners.Get(nobody, id); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("nobody reads: %v", err)
	}

	// The partner is a catalog record: files and history follow its permissions.
	ref := record.Ref{Type: "partner.partner", ID: id}
	if ok, err := f.core.Record.Can(viewer, ref.Type, id, record.Attach); err != nil || ok {
		t.Fatalf("viewer attaches: %v %v", ok, err)
	}
	if ok, err := f.core.Record.Can(editor, ref.Type, id, record.Attach); err != nil || !ok {
		t.Fatalf("editor attaches: %v %v", ok, err)
	}
	h, err := f.core.Record.History(viewer, ref)
	if err != nil || len(h) != 2 {
		t.Fatalf("history %+v %v", h, err)
	}
}

func must2(_ int64, err error) error { return err }

func TestPartnerTaxCodeTaken(t *testing.T) {
	f := newFixture(t)
	editor := f.user(t, "editor", "editor", f.a)
	id := must(t)(f.partners.Create(editor, "", partner.PartnerFields{Name: "A", TaxCode: str(" 0101234567 ")}))
	e := wantCode(t, must2(f.partners.Create(editor, "", partner.PartnerFields{Name: "B", TaxCode: str("0101234567")})), "partner_tax_code_taken")
	if e.Params["partner_id"] != id {
		t.Fatalf("params %v", e.Params)
	}
	other := must(t)(f.partners.Create(editor, "", partner.PartnerFields{Name: "C"}))
	e = wantCode(t, f.partners.Update(editor, other, partner.PartnerFields{Name: "C", TaxCode: str("0101234567")}), "partner_tax_code_taken")
	if e.Params["partner_id"] != id {
		t.Fatalf("params %v", e.Params)
	}
	// Its own tax code is no conflict.
	if err := f.partners.Update(editor, id, partner.PartnerFields{Name: "A2", TaxCode: str("0101234567")}); err != nil {
		t.Fatal(err)
	}
}

// Partner view held at one org unit sees every partner.
func TestPartnerVisibleAnywhere(t *testing.T) {
	f := newFixture(t)
	id := must(t)(f.partners.Create(f.user(t, "editor_b", "editor", f.b), "", partner.PartnerFields{Name: "Của B"}))
	viewer := f.user(t, "viewer_a", "viewer", f.a)
	if p, err := f.partners.Get(viewer, id); err != nil || p.Name != "Của B" {
		t.Fatalf("viewer of A reads B's partner: %+v %v", p, err)
	}
	if l, err := f.partners.List(viewer, partner.PartnerFilter{Paging: page}); err != nil || l.Total != 1 {
		t.Fatalf("viewer of A lists: %+v %v", l, err)
	}
	if err := f.partners.Update(f.user(t, "editor_a", "editor", f.a), id, partner.PartnerFields{Name: "Đổi"}); err != nil {
		t.Fatalf("editor of A edits B's partner: %v", err)
	}
}

// Partners belong to no product: they stay writable with every product disabled.
func TestPartnerWriteWithSalesDisabled(t *testing.T) {
	f := newFixture(t)
	ctx := platform.WithProducts(f.user(t, "editor", "editor", f.a), nil)
	id := must(t)(f.partners.Create(ctx, "", partner.PartnerFields{Name: "X"}))
	if err := f.partners.Update(ctx, id, partner.PartnerFields{Name: "Y"}); err != nil {
		t.Fatal(err)
	}
	if p, err := f.partners.Get(ctx, id); err != nil || !slices.Equal(p.AllowedActions, []string{"edit"}) {
		t.Fatalf("actions with products off: %+v %v", p, err)
	}
}
