package accounting_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/taoworklabs/mmerp/internal/modules/accounting"
	"github.com/taoworklabs/mmerp/internal/platform"
)

func TestPendingListsUnreconciled(t *testing.T) {
	l := newLedger(t)
	ref := l.doc(t, "2026-03-31")
	l.record(t, ref, l.payroll()...)
	got, err := l.s.Pending(l.core.Admin, l.le)
	if err != nil || len(got) != 1 || got[0].DocID != ref.ID || got[0].Reason != "queued" || got[0].Number == "" || got[0].Date != "2026-03-31" {
		t.Fatalf("pending %+v %v", got, err)
	}
	l.reconcile(t, ref)
	if got, err = l.s.Pending(l.core.Admin, l.le); err != nil || len(got) != 0 {
		t.Fatalf("after reconcile %+v %v", got, err)
	}
	// A void leaves the ledger holding a recording that is gone.
	l.void(t, ref)
	if got, err = l.s.Pending(l.core.Admin, l.le); err != nil || len(got) != 1 {
		t.Fatalf("after void %+v %v", got, err)
	}
}

func TestPendingShowsUnmapped(t *testing.T) {
	l := newLedger(t)
	if err := l.s.DeleteRule(l.core.Admin, "pit_payable"); err != nil {
		t.Fatal(err)
	}
	ref := l.doc(t, "2026-03-31")
	l.record(t, ref, l.payroll()...)
	l.reconcile(t, ref)
	got, err := l.s.Pending(l.core.Admin, l.le)
	if err != nil || len(got) != 1 || got[0].Reason != "unmapped" || !slices.Equal(got[0].MissingKinds, l.source(t, ref).missing) || len(got[0].MissingKinds) != 1 {
		t.Fatalf("pending %+v %v", got, err)
	}
}

func TestPendingDocumentsFromBeforeEnable(t *testing.T) {
	l := newLedger(t)
	ref := l.doc(t, "2026-03-31")
	off := platform.WithProducts(l.core.Ctx, []string{"hrm"})
	if err := platform.InTx(off, func(ctx context.Context) error { return l.pst.Record(ctx, ref, l.payroll()) }); err != nil {
		t.Fatal(err)
	}
	if got, err := l.s.Pending(l.core.Admin, l.le); err != nil || len(got) != 1 {
		t.Fatalf("pending %+v %v", got, err)
	}
}

func TestPendingIgnoresBeforeStartAndUnset(t *testing.T) {
	l := newLedger(t)
	l.record(t, l.doc(t, "2025-12-31"), l.payroll()...)
	if got, err := l.s.Pending(l.core.Admin, l.le); err != nil || len(got) != 0 {
		t.Fatalf("before start %+v %v", got, err)
	}
	l.record(t, l.doc(t, "2026-03-31"), l.payroll()...)
	if err := l.s.SetStart(l.core.Admin, l.le, nil); err != nil {
		t.Fatal(err)
	}
	if got, err := l.s.Pending(l.core.Admin, l.le); err != nil || len(got) != 0 {
		t.Fatalf("unset %+v %v", got, err)
	}
}

func TestPostPendingEnqueuesOnePerDocument(t *testing.T) {
	l := newLedger(t)
	l.record(t, l.doc(t, "2026-03-30"), l.payroll()...)
	l.record(t, l.doc(t, "2026-03-31"), l.payroll()...)
	client, err := river.NewClient(riverpgxv5.New(l.core.Pool), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	n, err := l.s.PostPending(platform.WithJobs(l.core.Admin, client), l.le)
	if err != nil || n != 2 {
		t.Fatalf("queued %d %v", n, err)
	}
	var jobs int
	if err := l.core.Pool.QueryRow(l.core.Ctx, `SELECT count(*) FROM river_job WHERE kind = 'accounting.reconcile'`).Scan(&jobs); err != nil || jobs != 2 {
		t.Fatalf("jobs %d %v", jobs, err)
	}
}

func TestPostPendingNeedsPost(t *testing.T) {
	l := newLedger(t)
	id, err := l.core.IAM.CreateUser(l.core.Admin, "view", "View", "long enough")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.core.IAM.GrantRole(l.core.Admin, id, "accounting", "viewer", nil); err != nil {
		t.Fatal(err)
	}
	ctx := platform.WithActor(l.core.Ctx, id)
	if _, err := l.s.Pending(ctx, l.le); err != nil {
		t.Fatalf("viewer reads: %v", err)
	}
	if _, err := l.s.PostPending(ctx, l.le); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("got %v", err)
	}
}

func TestPendingUntilBound(t *testing.T) {
	l := newLedger(t)
	l.record(t, l.doc(t, "2026-03-31"), l.payroll()...)
	l.record(t, l.doc(t, "2026-04-01"), l.payroll()...)
	until := pgtype.Date{Time: time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC), Valid: true}
	got, err := accounting.PendingInForTest(l.core.Admin, l.le, &until)
	if err != nil || len(got) != 1 || got[0].Date != "2026-03-31" {
		t.Fatalf("until %+v %v", got, err)
	}
	if got, err = accounting.PendingInForTest(l.core.Admin, l.le, nil); err != nil || len(got) != 2 {
		t.Fatalf("all %+v %v", got, err)
	}
}
