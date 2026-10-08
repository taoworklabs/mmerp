// Package recordtest is the contract every document type must pass: the module
// calls record before writing its tables, so record's guarantees hold for it.
package recordtest

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// Harness drives one document type through its module's service, on a fresh database.
type Harness struct {
	Record *record.Service
	// Actor may create, edit, send, withdraw and cancel documents of the type.
	Actor context.Context
	// Admin manages period locks.
	Admin       context.Context
	LegalEntity int64
	// Create adds a draft dated date (YYYY-MM-DD); the module may already have edited it,
	// so the suite reads its version rather than assuming 1.
	Create func(ctx context.Context, date string) (record.Ref, error)
	// Edit moves a draft to date.
	Edit func(ctx context.Context, ref record.Ref, version int32, date string) error
	// RequireApproval makes sending need one step that Approve can decide.
	RequireApproval func(t *testing.T)
	// Approve approves the current step of an approval instance as its approver.
	Approve func(instance int64) error
	// BreakPosting makes the type's OnTransition into posted fail for ref.
	BreakPosting func(t *testing.T, ref record.Ref)
	// Effects counts how many times posting ref has taken effect in the module.
	Effects func(t *testing.T, ref record.Ref) int
}

// Run checks the record guarantees for a document type; setup builds a fresh harness per case.
func Run(t *testing.T, setup func(t *testing.T) Harness) {
	t.Run("pending and posted documents cannot be edited", func(t *testing.T) {
		h := setup(t)
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
	})

	t.Run("documents in a locked period are frozen", func(t *testing.T) {
		h := setup(t)
		march := create(t, h, "2026-03-10")
		april := create(t, h, "2026-04-10")
		lock(t, h, h.Admin, "2026-03-31")
		mv, av := get(t, h, march).Version, get(t, h, april).Version
		wantCode(t, h.Edit(h.Actor, march, mv, "2026-04-15"), "period_locked")
		wantCode(t, h.Edit(h.Actor, april, av, "2026-03-20"), "period_locked")
		_, err := h.Create(h.Actor, "2026-03-15")
		wantCode(t, err, "period_locked")
		wantCode(t, h.Record.Transition(h.Actor, march, mv, record.Posted), "period_locked")
		if err := h.Edit(h.Actor, april, av, "2026-04-11"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("two concurrent posts take effect once", func(t *testing.T) {
		h := setup(t)
		ref := create(t, h, "2026-03-10")
		v := get(t, h, ref).Version
		var wg sync.WaitGroup
		errs := make([]error, 2)
		for i := range errs {
			wg.Go(func() { errs[i] = h.Record.Transition(h.Actor, ref, v, record.Posted) })
		}
		wg.Wait()
		if (errs[0] == nil) == (errs[1] == nil) {
			t.Fatalf("want exactly one success: %v, %v", errs[0], errs[1])
		}
		if n := h.Effects(t, ref); n != 1 {
			t.Fatalf("posting took effect %d times", n)
		}
	})

	t.Run("a stale version is refused", func(t *testing.T) {
		h := setup(t)
		ref := create(t, h, "2026-03-10")
		v := get(t, h, ref).Version
		if err := h.Edit(h.Actor, ref, v, "2026-03-11"); err != nil {
			t.Fatal(err)
		}
		wantCode(t, h.Edit(h.Actor, ref, v, "2026-03-12"), "version_conflict")
		wantCode(t, h.Record.Transition(h.Actor, ref, v, record.Posted), "version_conflict")
	})

	t.Run("approving a withdrawn submission is refused", func(t *testing.T) {
		h := setup(t)
		h.RequireApproval(t)
		ref := create(t, h, "2026-03-10")
		send(t, h, ref)
		first := *get(t, h, ref).ApprovalTicket
		if err := h.Record.Transition(h.Actor, ref, get(t, h, ref).Version, record.Draft); err != nil {
			t.Fatal(err)
		}
		send(t, h, ref)
		second := *get(t, h, ref).ApprovalTicket
		wantCode(t, h.Approve(first), "approval_closed")
		if d := get(t, h, ref); d.Status != record.PendingApproval || *d.ApprovalTicket != second {
			t.Fatalf("the new submission must stay pending: %+v", d)
		}
		if err := h.Approve(second); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("a failed completion leaves the document pending and the instance open", func(t *testing.T) {
		h := setup(t)
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
	})

	t.Run("a period with pending documents cannot be locked", func(t *testing.T) {
		h := setup(t)
		h.RequireApproval(t)
		ref := create(t, h, "2026-03-10")
		send(t, h, ref)
		wantCode(t, h.Record.SetPeriodLock(h.Admin, h.LegalEntity, new("2026-03-31")), "period_has_pending_documents")
		if err := h.Record.Transition(h.Actor, ref, get(t, h, ref).Version, record.Draft); err != nil {
			t.Fatal(err)
		}
		lock(t, h, h.Admin, "2026-03-31")
	})

	t.Run("a write racing a period lock never lands in the locked period", func(t *testing.T) {
		h := setup(t)
		// The lock commits first: the create waits for it, then sees the new date.
		var created error
		var wg sync.WaitGroup
		if err := platform.InTx(h.Admin, func(ctx context.Context) error {
			lock(t, h, ctx, "2026-03-31")
			wg.Go(func() { _, created = h.Create(h.Actor, "2026-03-15") })
			time.Sleep(100 * time.Millisecond)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		wg.Wait()
		wantCode(t, created, "period_locked")

		// The send commits first: the lock waits for it, then sees the pending document.
		h = setup(t)
		h.RequireApproval(t)
		ref := create(t, h, "2026-03-10")
		var locked error
		if err := platform.InTx(h.Actor, func(ctx context.Context) error {
			if err := h.Record.Transition(ctx, ref, get(t, h, ref).Version, record.Posted); err != nil {
				return err
			}
			wg.Go(func() { locked = h.Record.SetPeriodLock(h.Admin, h.LegalEntity, new("2026-03-31")) })
			time.Sleep(100 * time.Millisecond)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		wg.Wait()
		wantCode(t, locked, "period_has_pending_documents")
	})
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
	d, err := h.Record.Get(h.Admin, ref)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func send(t *testing.T, h Harness, ref record.Ref) {
	t.Helper()
	if err := h.Record.Transition(h.Actor, ref, get(t, h, ref).Version, record.Posted); err != nil {
		t.Fatal(err)
	}
}

func lock(t *testing.T, h Harness, ctx context.Context, date string) {
	t.Helper()
	if err := h.Record.SetPeriodLock(ctx, h.LegalEntity, &date); err != nil {
		t.Fatal(err)
	}
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
