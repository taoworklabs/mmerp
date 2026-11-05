package accounting_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// lockLedger wires the hook the way the app does and leaves one pending document dated 2026-03-31.
func lockLedger(t *testing.T) (*ledger, string) {
	t.Helper()
	l := newLedger(t)
	l.core.Record.SetBeforeLock(l.s.BeforeLock)
	l.record(t, l.doc(t, "2026-03-31"), l.payroll()...)
	return l, "2026-03-31"
}

func lockCode(t *testing.T, l *ledger, until string) string {
	t.Helper()
	err := l.core.Record.SetPeriodLock(l.core.Admin, l.le, &until)
	if err == nil {
		return ""
	}
	if e, ok := err.(*platform.Error); ok {
		return e.Code
	}
	t.Fatal(err)
	return ""
}

func TestLockRefusedWhilePending(t *testing.T) {
	l, until := lockLedger(t)
	if c := lockCode(t, l, until); c != "period_has_pending_documents" {
		t.Fatalf("code %q", c)
	}
	// A cutoff before the document leaves it outside the period.
	if c := lockCode(t, l, "2026-03-30"); c != "" {
		t.Fatalf("earlier cutoff %q", c)
	}
}

func TestLockAllowedOnceReconciled(t *testing.T) {
	l := newLedger(t)
	l.core.Record.SetBeforeLock(l.s.BeforeLock)
	ref := l.doc(t, "2026-03-31")
	l.record(t, ref, l.payroll()...)
	l.reconcile(t, ref)
	if c := lockCode(t, l, "2026-03-31"); c != "" {
		t.Fatalf("code %q", c)
	}
}

func TestLockAllowedWhenProductOff(t *testing.T) {
	l, until := lockLedger(t)
	ctx := platform.WithProducts(l.core.Admin, []string{"hrm"})
	if err := l.core.Record.SetPeriodLock(ctx, l.le, &until); err != nil {
		t.Fatal(err)
	}
}

func TestLockAllowedWithoutStartDate(t *testing.T) {
	l, until := lockLedger(t)
	if err := l.s.SetStart(l.core.Admin, l.le, nil); err != nil {
		t.Fatal(err)
	}
	if c := lockCode(t, l, until); c != "" {
		t.Fatalf("code %q", c)
	}
}

// hold runs fn in a transaction left open, so its locks stay held, until release commits it.
// The test's cleanup releases it too, so an early failure leaves no transaction open.
func hold(t *testing.T, ctx context.Context, fn func(context.Context) error) (release func() error, err error) {
	ready, free, done := make(chan error, 1), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- platform.InTx(ctx, func(ctx context.Context) error {
			ready <- fn(ctx)
			<-free
			return nil
		})
	}()
	release = sync.OnceValue(func() error { close(free); return <-done })
	t.Cleanup(func() { _ = release() })
	return release, <-ready
}

func async(fn func() error) <-chan error {
	ch := make(chan error, 1)
	go func() { ch <- fn() }()
	return ch
}

// waits fails unless ch is still blocked a moment later.
func waits(t *testing.T, ch <-chan error) {
	t.Helper()
	select {
	case err := <-ch:
		t.Fatalf("did not wait: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
}

// A lock racing the reconcile of a pending document in its period either waits for the entry
// and goes through, or is refused while the document is pending: never a lock over it.
func TestLockRacingReconcile(t *testing.T) {
	l := newLedger(t)
	l.core.Record.SetBeforeLock(l.s.BeforeLock)
	until := "2026-03-31"
	lock := func(ctx context.Context) error { return l.core.Record.SetPeriodLock(ctx, l.le, &until) }

	// Reconcile first: the lock waits for its entry, then goes through.
	ref := l.doc(t, until)
	l.record(t, ref, l.payroll()...)
	release, err := hold(t, l.core.Ctx, func(ctx context.Context) error { return l.s.Reconcile(ctx, ref) })
	if err != nil {
		t.Fatal(err)
	}
	locked := async(func() error { return lock(l.core.Admin) })
	waits(t, locked)
	if err := release(); err != nil {
		t.Fatal(err)
	}
	if err := <-locked; err != nil {
		t.Fatalf("lock after entry: %v", err)
	}
	if s := l.source(t, ref); s.posted == nil {
		t.Fatalf("source %+v", s)
	}

	// Lock first: refused while pending, and the reconcile waits for it.
	if err := l.core.Record.SetPeriodLock(l.core.Admin, l.le, nil); err != nil {
		t.Fatal(err)
	}
	ref = l.doc(t, until)
	l.record(t, ref, l.payroll()...)
	release, err = hold(t, l.core.Admin, lock)
	if !isCode(err, "period_has_pending_documents") {
		t.Fatalf("lock over pending: %v", err)
	}
	reconciled := async(func() error { return l.s.Reconcile(l.core.Ctx, ref) })
	waits(t, reconciled)
	if err := release(); err != nil {
		t.Fatal(err)
	}
	if err := <-reconciled; err != nil {
		t.Fatal(err)
	}
	if c := lockCode(t, l, until); c != "" {
		t.Fatalf("lock once reconciled: %q", c)
	}
}

// SetStart and a lock on the same legal entity serialize on the period row: whichever goes
// second sees the other's result.
func TestSetStartRacingLock(t *testing.T) {
	early, later, until := "2026-01-01", "2026-07-01", "2026-06-30"
	setup := func(t *testing.T) *ledger {
		l := newLedger(t)
		l.core.Record.SetBeforeLock(l.s.BeforeLock)
		if err := l.s.SetStart(l.core.Admin, l.le, &later); err != nil {
			t.Fatal(err)
		}
		l.record(t, l.doc(t, "2026-03-31"), l.payroll()...)
		return l
	}

	t.Run("start first", func(t *testing.T) {
		l := setup(t)
		release, err := hold(t, l.core.Admin, func(ctx context.Context) error { return l.s.SetStart(ctx, l.le, &early) })
		if err != nil {
			t.Fatal(err)
		}
		locked := async(func() error { return l.core.Record.SetPeriodLock(l.core.Admin, l.le, &until) })
		waits(t, locked)
		if err := release(); err != nil {
			t.Fatal(err)
		}
		if err := <-locked; !isCode(err, "period_has_pending_documents") {
			t.Fatalf("lock: %v", err)
		}
	})

	t.Run("lock first", func(t *testing.T) {
		l := setup(t)
		release, err := hold(t, l.core.Admin, func(ctx context.Context) error { return l.core.Record.SetPeriodLock(ctx, l.le, &until) })
		if err != nil {
			t.Fatal(err)
		}
		set := async(func() error { return l.s.SetStart(l.core.Admin, l.le, &early) })
		waits(t, set)
		if err := release(); err != nil {
			t.Fatal(err)
		}
		if err := <-set; !isCode(err, "start_date_in_locked_period") {
			t.Fatalf("start: %v", err)
		}
	})
}
