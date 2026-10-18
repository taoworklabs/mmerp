package record_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/coretest"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// declared is a document type that declares its permissions: editors of a unit edit,
// the subject of a document views and prints it, and a refused id may not be edited by anyone.
type declared struct {
	core                      *coretest.Core
	rec                       *record.Service
	editor, subject, stranger context.Context
	subjectID                 int64
	a                         int64
	subjects                  map[int64]int64 // document id → its subject's account
	refused                   map[int64]bool
}

func newDeclared(t *testing.T) *declared {
	core := coretest.New(t, "test")
	ids, admin := core.IAM, core.Admin
	ids.RegisterRoles("test", map[string][]string{"editor": {"test.view", "test.edit"}})
	f := &declared{core: core, rec: core.Record, subjects: map[int64]int64{}, refused: map[int64]bool{}}
	f.rec.Register(record.Type{
		Code: "test.decl", Product: "test", Kind: record.Document, NumberPrefix: "D",
		Perms:          record.Perms{View: "test.view", Edit: "test.edit", Attach: "test.edit"},
		SubjectCreates: true, OnDelete: noRows,
		Grant: func(ctx context.Context, id int64, a record.Action) (bool, error) {
			actor, _ := platform.ActorFrom(ctx)
			return (a == record.View || a == record.Print) && f.subjects[id] == actor, nil
		},
		Refine: func(_ context.Context, id int64, a record.Action) (bool, error) {
			return a != record.Edit || !f.refused[id], nil
		},
	})
	c := must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{Kind: "company", Name: "C"}))
	f.a = must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{ParentID: &c, Kind: "department", Name: "A"}))
	user := func(login string) (int64, context.Context) {
		id := must(t)(ids.CreateUser(admin, login, login, "long enough"))
		return id, platform.WithActor(core.Ctx, id)
	}
	editorID, editor := user("editor")
	must(t)(ids.GrantRole(admin, editorID, "test", "editor", &f.a))
	f.editor = editor
	f.subjectID, f.subject = user("subject")
	_, f.stranger = user("stranger")
	return f
}

func (f *declared) create(t *testing.T, ctx context.Context, subject int64) record.Doc {
	t.Helper()
	d, err := f.rec.Create(ctx, "test.decl", record.Header{Date: "2026-03-10", OrgUnitID: f.a, Subject: subject})
	if err != nil {
		t.Fatal(err)
	}
	f.subjects[d.ID] = subject
	return d
}

func ok[T any](t *testing.T) func(T, error) T {
	return func(v T, err error) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

// noRows is the delete hook of a type that keeps no rows of its own.
func noRows(context.Context, int64) error { return nil }

func errorCode(err error) string {
	if e, ok := errors.AsType[*platform.Error](err); ok {
		return e.Code
	}
	return ""
}

func TestDeclaredCreateChecksThePermission(t *testing.T) {
	f := newDeclared(t)
	h := record.Header{Date: "2026-03-10", OrgUnitID: f.a}
	if _, err := f.rec.Create(f.stranger, "test.decl", h); errorCode(err) != "forbidden" {
		t.Fatalf("stranger creates: %v", err)
	}
	// Naming a subject one may not file for hides whether it exists.
	h.Subject = f.subjectID
	if _, err := f.rec.Create(f.stranger, "test.decl", h); errorCode(err) != "not_found" {
		t.Fatalf("stranger creates for the subject: %v", err)
	}
	f.create(t, f.editor, f.subjectID)
	f.create(t, f.subject, f.subjectID)
}

func TestDeclaredCan(t *testing.T) {
	f := newDeclared(t)
	d := f.create(t, f.editor, f.subjectID)
	can := func(ctx context.Context, id int64, a record.Action) bool {
		t.Helper()
		ok, err := f.rec.Can(ctx, "test.decl", id, a)
		if err != nil {
			t.Fatal(err)
		}
		return ok
	}
	for a, want := range map[record.Action]bool{
		record.View: true, record.Edit: true, record.Post: true, record.Cancel: true, record.Attach: true,
		record.ViewFiles: false, // undeclared: the type has no files to see
		record.Print:     false, // no template
		record.Export:    false, // no export
	} {
		if got := can(f.editor, d.ID, a); got != want {
			t.Errorf("editor %s: %v, want %v", a, got, want)
		}
	}
	if !can(f.subject, d.ID, record.View) || can(f.subject, d.ID, record.Edit) {
		t.Error("the relation grant gives the subject view only")
	}
	if can(f.stranger, d.ID, record.View) {
		t.Error("stranger views")
	}
	f.refused[d.ID] = true
	if can(f.editor, d.ID, record.Edit) || !can(f.editor, d.ID, record.View) {
		t.Error("refine narrows edit only")
	}
	f.rec.Offer("test.decl", record.Print)
	if !can(f.editor, d.ID, record.Print) || !can(f.subject, d.ID, record.Print) {
		t.Error("print follows view once offered")
	}
	if can(platform.WithProducts(f.editor, nil), d.ID, record.Edit) {
		t.Error("product off still edits")
	}
	f.rec.Register(record.Type{Code: "test.other", Product: "test", Kind: record.Document, NumberPrefix: "O",
		Perms: record.Perms{View: "test.view", Edit: "test.edit"}, OnDelete: noRows})
	other := ok[record.Doc](t)(f.rec.Create(f.editor, "test.other", record.Header{Date: "2026-03-10", OrgUnitID: f.a}))
	if can(f.editor, other.ID, record.View) {
		t.Error("an id of another type is visible")
	}
}

func TestDeclaredReads(t *testing.T) {
	f := newDeclared(t)
	d := f.create(t, f.editor, f.subjectID)
	ref := d.Ref
	if _, err := f.rec.Read(f.stranger, ref); errorCode(err) != "not_found" {
		t.Fatalf("stranger reads: %v", err)
	}
	if got, err := f.rec.Read(f.subject, ref); err != nil || got.Number != d.Number {
		t.Fatalf("subject reads: %+v %v", got, err)
	}
	if _, err := f.rec.Read(f.editor, record.Ref{Type: "test.decl", ID: d.ID + 1000}); errorCode(err) != "not_found" {
		t.Fatalf("missing id: %v", err)
	}
	f.rec.Offer("test.decl", record.Print)
	acts := ok[[]string](t)(f.rec.Actions(f.editor, d, "extra"))
	if !slices.Equal(acts, []string{"edit", "delete", "submit", "print", "extra"}) {
		t.Fatalf("editor actions %v", acts)
	}
	if acts := ok[[]string](t)(f.rec.Actions(f.subject, d)); !slices.Equal(acts, []string{"print"}) {
		t.Fatalf("subject actions %v", acts)
	}
	if acts := ok[[]string](t)(f.rec.Actions(platform.WithProducts(f.editor, nil), d)); !slices.Equal(acts, []string{"print"}) {
		t.Fatalf("product off actions %v", acts)
	}

	for ctx, want := range map[context.Context][]string{
		f.editor: {"create"}, f.stranger: {}, platform.WithProducts(f.editor, nil): {},
	} {
		if got := ok[[]string](t)(f.rec.TypeActions(ctx, "test.decl")); !slices.Equal(got, want) {
			t.Errorf("type actions %v, want %v", got, want)
		}
	}
}

func TestDeclarationsAreCheckedAtRegistration(t *testing.T) {
	f := newDeclared(t)
	for name, typ := range map[string]record.Type{
		"no view":             {Code: "test.a", Kind: record.Document, NumberPrefix: "A", Perms: record.Perms{Edit: "test.edit"}, OnDelete: noRows},
		"unknown permission":  {Code: "test.b", Kind: record.Document, NumberPrefix: "B", Perms: record.Perms{View: "test.nope"}, OnDelete: noRows},
		"document, no delete": {Code: "test.e", Kind: record.Document, NumberPrefix: "E", Perms: record.Perms{View: "test.view"}},
		"catalog, no locate":  {Code: "test.c", Kind: record.Catalog, Perms: record.Perms{View: "test.view"}},
		"catalog with number": {Code: "test.d", Kind: record.Catalog, NumberPrefix: "D", Perms: record.Perms{View: "test.view"}, Locate: func(context.Context, int64) (int64, error) { return 0, nil }},
	} {
		typ.Product = "test"
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: registered", name)
				}
			}()
			f.rec.Register(typ)
		}()
	}
}
