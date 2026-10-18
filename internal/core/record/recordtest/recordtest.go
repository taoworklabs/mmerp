// Package recordtest is the contract every document type must pass: the module
// calls record before writing its tables, so record's guarantees hold for it.
package recordtest

import (
	"context"
	"errors"
	"maps"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/coretest"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

// Harness drives one document type through its module's service, on a fresh database.
type Harness struct {
	Core *coretest.Core
	// Actor may create, edit, delete, send, withdraw and cancel documents of the type.
	Actor context.Context
	// Admin manages period locks.
	Admin       context.Context
	LegalEntity int64
	// Create adds a draft dated date (YYYY-MM-DD); the module may already have edited it,
	// so the suite reads its version rather than assuming 1.
	Create func(ctx context.Context, date string) (record.Ref, error)
	// Edit moves a draft to date.
	Edit func(ctx context.Context, ref record.Ref, version int32, date string) error
	// Get reads the document through the module's detail route; the result's
	// AllowedActions field is what the API returns as allowed_actions.
	Get func(ctx context.Context, ref record.Ref) (any, error)
	// Delete deletes a draft through the module's route.
	Delete func(ctx context.Context, ref record.Ref, version int32) error
	// Routes are the type's other module routes on one document, by name.
	Routes map[string]func(ctx context.Context, ref record.Ref) error
	// RequireApproval makes sending need one step that Approve or Reject can decide.
	RequireApproval func(t *testing.T)
	// Approve approves the current step of an approval instance as its approver.
	Approve func(instance int64) error
	// Reject rejects the current step of an approval instance as its approver.
	Reject func(instance int64) error
	// BreakPosting makes the type's OnTransition into posted fail for ref.
	BreakPosting func(t *testing.T, ref record.Ref)
	// Effects counts how many times posting ref has taken effect in the module.
	Effects func(t *testing.T, ref record.Ref) int
	// BreakSubmit makes the type's BeforeSubmit refuse ref; nil for a type without one.
	BreakSubmit func(t *testing.T, ref record.Ref)
	// PostingLines counts ref's posting lines not voided; nil for a type that writes none.
	PostingLines func(t *testing.T, ref record.Ref) int
	// Gaps skips the named cases, each with why: defects the suite found and that are
	// still open.
	Gaps map[string]string
}

// readActions are the allowed_actions that write nothing.
var readActions = []string{"print", "export"}

type testCase struct {
	name string
	run  func(t *testing.T, h Harness)
}

// Run checks the record guarantees for a document type; setup builds a fresh harness per case.
func Run(t *testing.T, setup func(t *testing.T) Harness) {
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := setup(t)
			if why, ok := h.Gaps[c.name]; ok {
				t.Skip("known gap: " + why)
			}
			c.run(t, h)
		})
	}
}

var cases = []testCase{
	{"pending and posted documents cannot be edited", func(t *testing.T, h Harness) {
		h.RequireApproval(t)
		ref := create(t, h, "2026-03-10")
		send(t, h, ref)
		d := get(t, h, ref)
		if d.Status != record.PendingApproval {
			t.Fatalf("status %s, want pending_approval", d.Status)
		}
		wantCode(t, h.Edit(h.Actor, ref, d.Version, "2026-03-11"), "document_not_editable")
		if err := h.Approve(*d.ApprovalTicket); err != nil {
			t.Fatal(err)
		}
		d = get(t, h, ref)
		if d.Status != record.Posted {
			t.Fatalf("status %s, want posted", d.Status)
		}
		wantCode(t, h.Edit(h.Actor, ref, d.Version, "2026-03-11"), "document_not_editable")
	}},

	{"whoever may print a document may view it, in every status", func(t *testing.T, h Harness) {
		h.RequireApproval(t)
		stranger := newUser(t, h, "print_stranger")
		ref := create(t, h, "2026-03-10")
		steps := []func(){
			func() {},
			func() { send(t, h, ref) },
			func() { approve(t, h, ref) },
			func() { move(t, h, ref, record.Cancelled) },
		}
		for _, step := range steps {
			step()
			for _, ctx := range []context.Context{h.Actor, stranger} {
				if can(t, h, ctx, ref.Type, ref.ID, record.Print) && !can(t, h, ctx, ref.Type, ref.ID, record.View) {
					t.Fatalf("%s: print without view", get(t, h, ref).Status)
				}
			}
		}
	}},

	{"delete removes a draft with its files and comments", func(t *testing.T, h Harness) {
		ref := create(t, h, "2026-03-10")
		files := platform.WithFiles(h.Actor, platform.Files{Dir: t.TempDir()})
		attach := can(t, h, h.Actor, ref.Type, ref.ID, record.Attach)
		if attach {
			if err := h.Core.Attachment.Add(files, ref, "a.pdf", strings.NewReader("%PDF-1.4\n")); err != nil {
				t.Fatal(err)
			}
		}
		if err := h.Core.Discussion.Add(h.Actor, ref, "note"); err != nil {
			t.Fatal(err)
		}
		if n := attached(t, h, ref, "attachment.files"); attach && n != 1 {
			t.Fatalf("%d attachments", n)
		}
		if n := attached(t, h, ref, "discussion.comments"); n != 1 {
			t.Fatalf("%d comments", n)
		}
		v := get(t, h, ref).Version
		wantCode(t, h.Delete(h.Actor, ref, v+1), "version_conflict")
		if err := h.Delete(h.Actor, ref, v); err != nil {
			t.Fatal(err)
		}
		if _, err := h.Get(h.Admin, ref); code(err) != "not_found" {
			t.Fatalf("module read after delete: %v", err)
		}
		if _, err := h.Core.Record.Get(h.Admin, ref); code(err) != "not_found" {
			t.Fatalf("record read after delete: %v", err)
		}
		for _, table := range []string{"attachment.files", "discussion.comments"} {
			if n := attached(t, h, ref, table); n != 0 {
				t.Fatalf("%d rows left in %s", n, table)
			}
		}
	}},

	{"only a draft outside a locked period is deleted", func(t *testing.T, h Harness) {
		posted := create(t, h, "2026-04-10")
		send(t, h, posted)
		wantCode(t, h.Delete(h.Actor, posted, get(t, h, posted).Version), "document_not_editable")
		march := create(t, h, "2026-03-10")
		lock(t, h, h.Admin, "2026-03-31")
		wantCode(t, h.Delete(h.Actor, march, get(t, h, march).Version), "period_locked")
		for _, ref := range []record.Ref{posted, march} {
			if _, err := h.Get(h.Admin, ref); err != nil {
				t.Fatalf("refused delete removed %v: %v", ref, err)
			}
		}
	}},

	{"cancel undoes the posting and is refused in a locked period", func(t *testing.T, h Harness) {
		ref := create(t, h, "2026-03-10")
		send(t, h, ref)
		if n := h.Effects(t, ref); n != 1 {
			t.Fatalf("posting took effect %d times", n)
		}
		if h.PostingLines != nil && h.PostingLines(t, ref) == 0 {
			t.Fatal("posted without posting lines")
		}
		lock(t, h, h.Admin, "2026-03-31")
		wantCode(t, h.Core.Record.Transition(h.Actor, ref, get(t, h, ref).Version, record.Cancelled), "period_locked")
		if n := h.Effects(t, ref); n != 1 {
			t.Fatalf("a refused cancel changed the effect to %d", n)
		}
		if err := h.Core.Record.SetPeriodLock(h.Admin, h.LegalEntity, nil); err != nil {
			t.Fatal(err)
		}
		move(t, h, ref, record.Cancelled)
		if n := h.Effects(t, ref); n != 0 {
			t.Fatalf("cancelled, the posting still takes effect %d times", n)
		}
		if h.PostingLines != nil {
			if n := h.PostingLines(t, ref); n != 0 {
				t.Fatalf("%d posting lines not voided", n)
			}
		}
	}},

	{"a rejected submission returns the document to draft", func(t *testing.T, h Harness) {
		h.RequireApproval(t)
		ref := create(t, h, "2026-03-10")
		send(t, h, ref)
		if err := h.Reject(*get(t, h, ref).ApprovalTicket); err != nil {
			t.Fatal(err)
		}
		d := get(t, h, ref)
		if d.Status != record.Draft || d.ApprovalTicket != nil || d.SubmittedBy != nil {
			t.Fatalf("after reject: %+v", d)
		}
		if n := h.Effects(t, ref); n != 0 {
			t.Fatalf("a rejected document took effect %d times", n)
		}
		if a := actions(t, h, h.Actor, ref); !slices.Contains(a, "edit") || !slices.Contains(a, "submit") {
			t.Fatalf("rejected draft offers %v", a)
		}
		send(t, h, ref)
		if d := get(t, h, ref); d.Status != record.PendingApproval {
			t.Fatalf("resent: %s", d.Status)
		}
	}},

	{"a refused submit check keeps the draft, with or without approval", func(t *testing.T, h Harness) {
		if h.BreakSubmit == nil {
			t.Skip("the type has no submit check")
		}
		direct, approved := create(t, h, "2026-03-10"), create(t, h, "2026-03-11")
		h.BreakSubmit(t, direct)
		h.BreakSubmit(t, approved)
		refused := func(ref record.Ref) string {
			before := get(t, h, ref)
			err := h.Core.Record.Transition(h.Actor, ref, before.Version, record.Posted)
			if err == nil {
				t.Fatal("sent past the submit check")
			}
			if d := get(t, h, ref); d.Status != record.Draft || d.Version != before.Version || d.ApprovalTicket != nil {
				t.Fatalf("after a refused send: %+v", d)
			}
			return code(err)
		}
		first := refused(direct)
		h.RequireApproval(t)
		if second := refused(approved); first == "" || second != first {
			t.Fatalf("refused with %q without approval and %q with it", first, second)
		}
	}},

	{"a user who may not view a document gets not found on every route", func(t *testing.T, h Harness) {
		stranger := newUser(t, h, "route_stranger")
		files := platform.WithFiles(stranger, platform.Files{Dir: t.TempDir()})
		ref := create(t, h, "2026-03-10")
		v := get(t, h, ref).Version
		type route struct {
			name string
			call func() error
		}
		// Reads first, so a route that wrongly lets the stranger write does not hide the others.
		routes := []route{
			{"get", func() error { _, err := h.Get(stranger, ref); return err }},
			{"history", func() error { _, err := h.Core.Record.History(stranger, ref); return err }},
			{"attachments", func() error { _, err := h.Core.Attachment.List(stranger, ref); return err }},
			{"comments", func() error { _, err := h.Core.Discussion.List(stranger, ref); return err }},
			{"attach", func() error { return h.Core.Attachment.Add(files, ref, "a.pdf", strings.NewReader("%PDF-1.4\n")) }},
			{"comment", func() error { return h.Core.Discussion.Add(stranger, ref, "note") }},
			{"edit", func() error { return h.Edit(stranger, ref, v, "2026-03-11") }},
		}
		for _, name := range slices.Sorted(maps.Keys(h.Routes)) {
			routes = append(routes, route{name, func() error { return h.Routes[name](stranger, ref) }})
		}
		routes = append(routes,
			route{"send", func() error { return h.Core.Record.Transition(stranger, ref, v, record.Posted) }},
			route{"delete", func() error { return h.Delete(stranger, ref, v) }})
		for _, r := range routes {
			if err := r.call(); code(err) != "not_found" {
				t.Errorf("%s: %v, want not_found", r.name, err)
			}
		}
		send(t, h, ref)
		if err := h.Core.Record.Transition(stranger, ref, get(t, h, ref).Version, record.Cancelled); code(err) != "not_found" {
			t.Errorf("cancel: %v, want not_found", err)
		}
		if _, err := h.Get(stranger, ref); code(err) != "not_found" {
			t.Errorf("get posted: %v, want not_found", err)
		}
	}},

	{"Can is false for an id of another type", func(t *testing.T, h Harness) {
		ref := create(t, h, "2026-03-10")
		h.Core.Record.Register(record.Type{Code: "recordtest.other", Product: "recordtest", Kind: record.Document, NumberPrefix: "RT",
			Can: func(context.Context, int64, record.Action) (bool, error) { return true, nil }})
		other, err := h.Core.Record.Create(platform.WithProducts(h.Admin, []string{"recordtest"}), "recordtest.other",
			record.Header{Date: "2026-03-10", OrgUnitID: h.LegalEntity})
		if err != nil {
			t.Fatal(err)
		}
		for _, ctx := range []context.Context{h.Actor, h.Admin} {
			for _, a := range []record.Action{record.View, record.Edit, record.Post, record.Cancel, record.Export,
				record.ViewFiles, record.Attach, record.Print} {
				if can(t, h, ctx, ref.Type, other.ID, a) {
					t.Errorf("%s on the id of a %s", a, other.Type)
				}
			}
		}
	}},

	{"write actions leave allowed_actions when the product is off or the period locked", func(t *testing.T, h Harness) {
		draft := create(t, h, "2026-03-10")
		posted := create(t, h, "2026-02-10") // another month, for types that allow one per month
		send(t, h, posted)
		if a := actions(t, h, h.Actor, draft); !slices.Contains(a, "edit") {
			t.Fatalf("the actor may not edit the draft: %v", a)
		}
		readOnly := func(when string, ctx context.Context) {
			for _, ref := range []record.Ref{draft, posted} {
				for _, a := range actions(t, h, ctx, ref) {
					if !slices.Contains(readActions, a) {
						t.Errorf("%s, %s offers %q", when, get(t, h, ref).Status, a)
					}
				}
			}
		}
		readOnly("product off", platform.WithProducts(h.Actor, nil))
		lock(t, h, h.Admin, "2026-03-31")
		readOnly("period locked", h.Actor)
	}},

	{"a failing record call rolls back the module's writes in its transaction", func(t *testing.T, h Harness) {
		ref := create(t, h, "2026-03-10")
		v := get(t, h, ref).Version
		before, err := h.Get(h.Admin, ref)
		if err != nil {
			t.Fatal(err)
		}
		err = platform.InTx(h.Actor, func(ctx context.Context) error {
			if err := h.Edit(ctx, ref, v, "2026-03-11"); err != nil {
				return err
			}
			return h.Core.Record.Transition(ctx, ref, v, record.Posted) // v is stale now
		})
		wantCode(t, err, "version_conflict")
		after, err := h.Get(h.Admin, ref)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(before, after) {
			t.Fatalf("the module kept writes of a failed transaction:\nbefore %+v\nafter  %+v", before, after)
		}
	}},

	{"documents in a locked period are frozen", func(t *testing.T, h Harness) {
		march := create(t, h, "2026-03-10")
		april := create(t, h, "2026-04-10")
		lock(t, h, h.Admin, "2026-03-31")
		mv, av := get(t, h, march).Version, get(t, h, april).Version
		wantCode(t, h.Edit(h.Actor, march, mv, "2026-04-15"), "period_locked")
		wantCode(t, h.Edit(h.Actor, april, av, "2026-03-20"), "period_locked")
		_, err := h.Create(h.Actor, "2026-03-15")
		wantCode(t, err, "period_locked")
		wantCode(t, h.Core.Record.Transition(h.Actor, march, mv, record.Posted), "period_locked")
		if err := h.Edit(h.Actor, april, av, "2026-04-11"); err != nil {
			t.Fatal(err)
		}
	}},

	{"two concurrent posts take effect once", func(t *testing.T, h Harness) {
		ref := create(t, h, "2026-03-10")
		v := get(t, h, ref).Version
		var wg sync.WaitGroup
		errs := make([]error, 2)
		for i := range errs {
			wg.Go(func() { errs[i] = h.Core.Record.Transition(h.Actor, ref, v, record.Posted) })
		}
		wg.Wait()
		if (errs[0] == nil) == (errs[1] == nil) {
			t.Fatalf("want exactly one success: %v, %v", errs[0], errs[1])
		}
		if n := h.Effects(t, ref); n != 1 {
			t.Fatalf("posting took effect %d times", n)
		}
	}},

	{"a stale version is refused", func(t *testing.T, h Harness) {
		ref := create(t, h, "2026-03-10")
		v := get(t, h, ref).Version
		if err := h.Edit(h.Actor, ref, v, "2026-03-11"); err != nil {
			t.Fatal(err)
		}
		wantCode(t, h.Edit(h.Actor, ref, v, "2026-03-12"), "version_conflict")
		wantCode(t, h.Core.Record.Transition(h.Actor, ref, v, record.Posted), "version_conflict")
	}},

	{"approving a withdrawn submission is refused", func(t *testing.T, h Harness) {
		h.RequireApproval(t)
		ref := create(t, h, "2026-03-10")
		send(t, h, ref)
		first := *get(t, h, ref).ApprovalTicket
		move(t, h, ref, record.Draft)
		send(t, h, ref)
		second := *get(t, h, ref).ApprovalTicket
		wantCode(t, h.Approve(first), "approval_closed")
		if d := get(t, h, ref); d.Status != record.PendingApproval || *d.ApprovalTicket != second {
			t.Fatalf("the new submission must stay pending: %+v", d)
		}
		if err := h.Approve(second); err != nil {
			t.Fatal(err)
		}
	}},

	{"a failed completion leaves the document pending and the instance open", func(t *testing.T, h Harness) {
		h.RequireApproval(t)
		ref := create(t, h, "2026-03-10")
		send(t, h, ref)
		ticket := *get(t, h, ref).ApprovalTicket
		h.BreakPosting(t, ref)
		for range 2 { // the instance stays open, so the second try fails the same way
			err := h.Approve(ticket)
			if err == nil || code(err) == "approval_closed" || code(err) == "approval_stale" {
				t.Fatalf("want the posting error, got %v", err)
			}
		}
		if d := get(t, h, ref); d.Status != record.PendingApproval || *d.ApprovalTicket != ticket {
			t.Fatalf("want pending with the same instance: %+v", d)
		}
		if n := h.Effects(t, ref); n != 0 {
			t.Fatalf("posting took effect %d times", n)
		}
	}},

	{"a period with pending documents cannot be locked", func(t *testing.T, h Harness) {
		h.RequireApproval(t)
		ref := create(t, h, "2026-03-10")
		send(t, h, ref)
		wantCode(t, h.Core.Record.SetPeriodLock(h.Admin, h.LegalEntity, new("2026-03-31")), "period_has_pending_documents")
		move(t, h, ref, record.Draft)
		lock(t, h, h.Admin, "2026-03-31")
	}},

	{"a create racing a period lock waits and sees the lock", func(t *testing.T, h Harness) {
		var created error
		var wg sync.WaitGroup
		if err := platform.InTx(h.Admin, func(ctx context.Context) error {
			lock(t, h, ctx, "2026-03-31")
			wg.Go(func() { _, created = h.Create(h.Actor, "2026-03-15") })
			pgtest.AwaitLockWait(t, platform.DBFrom(h.Admin))
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		wg.Wait()
		wantCode(t, created, "period_locked")
	}},

	{"a period lock racing a send waits and sees the pending document", func(t *testing.T, h Harness) {
		h.RequireApproval(t)
		ref := create(t, h, "2026-03-10")
		var locked error
		var wg sync.WaitGroup
		if err := platform.InTx(h.Actor, func(ctx context.Context) error {
			if err := h.Core.Record.Transition(ctx, ref, get(t, h, ref).Version, record.Posted); err != nil {
				return err
			}
			wg.Go(func() { locked = h.Core.Record.SetPeriodLock(h.Admin, h.LegalEntity, new("2026-03-31")) })
			pgtest.AwaitLockWait(t, platform.DBFrom(h.Admin))
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		wg.Wait()
		wantCode(t, locked, "period_has_pending_documents")
	}},
}

func create(t *testing.T, h Harness, date string) record.Ref {
	t.Helper()
	ref, err := h.Create(h.Actor, date)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func get(t *testing.T, h Harness, ref record.Ref) record.Doc {
	t.Helper()
	d, err := h.Core.Record.Get(h.Admin, ref)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func move(t *testing.T, h Harness, ref record.Ref, to record.Status) {
	t.Helper()
	if err := h.Core.Record.Transition(h.Actor, ref, get(t, h, ref).Version, to); err != nil {
		t.Fatal(err)
	}
}

func send(t *testing.T, h Harness, ref record.Ref) {
	t.Helper()
	move(t, h, ref, record.Posted)
}

func approve(t *testing.T, h Harness, ref record.Ref) {
	t.Helper()
	if err := h.Approve(*get(t, h, ref).ApprovalTicket); err != nil {
		t.Fatal(err)
	}
}

func lock(t *testing.T, h Harness, ctx context.Context, date string) {
	t.Helper()
	if err := h.Core.Record.SetPeriodLock(ctx, h.LegalEntity, &date); err != nil {
		t.Fatal(err)
	}
}

func can(t *testing.T, h Harness, ctx context.Context, code string, id int64, a record.Action) bool {
	t.Helper()
	ok, err := h.Core.Record.Can(ctx, code, id, a)
	if err != nil {
		t.Fatal(err)
	}
	return ok
}

// newUser is a user with no role at all.
func newUser(t *testing.T, h Harness, login string) context.Context {
	t.Helper()
	id, err := h.Core.IAM.CreateUser(h.Admin, login, login, "long enough")
	if err != nil {
		t.Fatal(err)
	}
	return platform.WithActor(h.Actor, id)
}

func actions(t *testing.T, h Harness, ctx context.Context, ref record.Ref) []string {
	t.Helper()
	v, err := h.Get(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	f := reflect.Indirect(reflect.ValueOf(v)).FieldByName("AllowedActions")
	if !f.IsValid() || f.Type() != reflect.TypeFor[[]string]() {
		t.Fatalf("%T has no AllowedActions []string", v)
	}
	return f.Interface().([]string)
}

// attached counts the rows a core module keeps for ref in table; nothing else reads them
// once the document is gone.
func attached(t *testing.T, h Harness, ref record.Ref, table string) int {
	t.Helper()
	var n int
	if err := platform.DBFrom(h.Admin).QueryRow(h.Admin, `SELECT count(*) FROM `+table+` WHERE doc_type = $1 AND doc_id = $2`,
		ref.Type, ref.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func code(err error) string {
	if e, ok := errors.AsType[*platform.Error](err); ok {
		return e.Code
	}
	return ""
}

func wantCode(t *testing.T, err error, want string) {
	t.Helper()
	if code(err) != want {
		t.Fatalf("got %v, want %s", err, want)
	}
}
