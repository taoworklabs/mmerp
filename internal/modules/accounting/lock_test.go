package accounting_test

import (
	"testing"

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
