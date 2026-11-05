package accounting_test

import (
	"context"
	"errors"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/modules/accounting"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

func TestSetStartPerLegalEntity(t *testing.T) {
	core, s := newAccounting(t)
	a, err := core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{Kind: "company", Name: "A"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{Kind: "company", Name: "B"})
	if err != nil {
		t.Fatal(err)
	}
	d := "2026-01-01"
	if err := s.SetStart(core.Admin, a, &d); err != nil {
		t.Fatal(err)
	}
	st, err := s.Starts(core.Admin)
	if err != nil || len(st) != 2 {
		t.Fatalf("starts %+v %v", st, err)
	}
	for _, x := range st {
		switch x.LegalEntityID {
		case a:
			if x.StartDate == nil || *x.StartDate != d {
				t.Errorf("a %+v", x)
			}
		case b:
			if x.StartDate != nil {
				t.Errorf("b %+v", x)
			}
		}
	}
	if err := s.SetStart(core.Admin, a, nil); err != nil {
		t.Fatal(err)
	}
	if st, _ = s.Starts(core.Admin); st[0].StartDate != nil || st[1].StartDate != nil {
		t.Fatalf("cleared %+v", st)
	}
}

func TestSetStartRefusesNonLegalEntity(t *testing.T) {
	core, s := newAccounting(t)
	c, _ := core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{Kind: "company", Name: "A"})
	dept, _ := core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{ParentID: &c, Kind: "department", Name: "D"})
	d := "2026-01-01"
	if err := s.SetStart(core.Admin, dept, &d); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("department: %v", err)
	}
}

func TestStartOf(t *testing.T) {
	core, s := newAccounting(t)
	c, _ := core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{Kind: "company", Name: "A"})
	if d, err := accounting.StartOfForTest(core.Admin, c); err != nil || d.Valid {
		t.Fatalf("unset %v %v", d, err)
	}
	day := "2026-03-01"
	if err := s.SetStart(core.Admin, c, &day); err != nil {
		t.Fatal(err)
	}
	if d, err := accounting.StartOfForTest(core.Admin, c); err != nil || !d.Valid || d.Time.Format("2006-01-02") != day {
		t.Fatalf("set %v %v", d, err)
	}
}

func startErr(t *testing.T, err error) (string, any) {
	t.Helper()
	if err == nil {
		return "", nil
	}
	var e *platform.Error
	if !errors.As(err, &e) {
		t.Fatal(err)
	}
	return e.Code, e.Params["locked_until"]
}

// A document recorded while it was before the start date, then locked, cannot become
// pending inside the locked period by moving the start earlier.
func TestSetStartRefusedIntoLockedPeriod(t *testing.T) {
	l := newLedger(t)
	l.core.Record.SetBeforeLock(l.s.BeforeLock)
	later := "2026-07-01"
	if err := l.s.SetStart(l.core.Admin, l.le, &later); err != nil {
		t.Fatal(err)
	}
	l.record(t, l.doc(t, "2026-03-31"), l.payroll()...)
	if c := lockCode(t, l, "2026-06-30"); c != "" {
		t.Fatalf("lock %q", c)
	}
	early := "2026-01-01"
	if c, p := startErr(t, l.s.SetStart(l.core.Admin, l.le, &early)); c != "start_date_in_locked_period" || p != "2026-06-30" {
		t.Fatalf("moved earlier: %q %v", c, p)
	}
	// Setting one for the first time after the lock is the same.
	if err := l.s.SetStart(l.core.Admin, l.le, nil); err != nil {
		t.Fatal(err)
	}
	if c, _ := startErr(t, l.s.SetStart(l.core.Admin, l.le, &early)); c != "start_date_in_locked_period" {
		t.Fatalf("first start: %q", c)
	}
	// After the lock date, or with nothing recorded in the locked part, is allowed.
	if err := l.s.SetStart(l.core.Admin, l.le, &later); err != nil {
		t.Fatalf("after lock: %v", err)
	}
	inside := "2026-04-01"
	if err := l.s.SetStart(l.core.Admin, l.le, &inside); err != nil {
		t.Fatalf("no lines in locked part: %v", err)
	}
}

// A start date change racing the first entry of its legal entity: either the entry is in
// first and the change is refused, or the change is in first and the entry follows the new
// date. Never an entry dated before the stored start.
func TestSetStartRacingFirstEntry(t *testing.T) {
	later := "2026-04-01"
	setStart := func(l *ledger) func(context.Context) error {
		return func(ctx context.Context) error { return l.s.SetStart(ctx, l.le, &later) }
	}
	before := func(t *testing.T, l *ledger) {
		t.Helper()
		var n int
		if err := l.core.Pool.QueryRow(l.core.Ctx, `SELECT count(*) FROM accounting.entries e
			LEFT JOIN accounting.starts s ON s.legal_entity_id = e.legal_entity_id
			WHERE s.start_date IS NULL OR e.date < s.start_date`).Scan(&n); err != nil || n != 0 {
			t.Fatalf("entries before the start %d %v", n, err)
		}
	}

	t.Run("entry first", func(t *testing.T) {
		l := newLedger(t)
		ref := l.doc(t, "2026-03-31")
		l.record(t, ref, l.payroll()...)
		release, err := hold(l.core.Ctx, func(ctx context.Context) error { return l.s.Reconcile(ctx, ref) })
		if err != nil {
			t.Fatal(err)
		}
		set := async(func() error { return setStart(l)(l.core.Admin) })
		waits(t, set)
		if err := release(); err != nil {
			t.Fatal(err)
		}
		if err := <-set; !errors.Is(err, accounting.ErrStartDateFixed) {
			t.Fatalf("set: %v", err)
		}
		if es := l.entries(t, ref); len(es) != 1 {
			t.Fatalf("entries %+v", es)
		}
		before(t, l)
	})

	t.Run("start first", func(t *testing.T) {
		l := newLedger(t)
		ref := l.doc(t, "2026-03-31")
		l.record(t, ref, l.payroll()...)
		release, err := hold(l.core.Admin, setStart(l))
		if err != nil {
			t.Fatal(err)
		}
		rec := async(func() error { return l.s.Reconcile(l.core.Ctx, ref) })
		waits(t, rec)
		if err := release(); err != nil {
			t.Fatal(err)
		}
		if err := <-rec; err != nil {
			t.Fatal(err)
		}
		if es := l.entries(t, ref); len(es) != 0 {
			t.Fatalf("entries %+v", es)
		}
		before(t, l)
	})
}

// A document voided while the tracking row still holds its recording is pending too, so it
// counts like a live one in the locked part.
func TestSetStartRefusedOverVoidedPending(t *testing.T) {
	l := newLedger(t)
	l.core.Record.SetBeforeLock(l.s.BeforeLock)
	ref := l.doc(t, "2026-03-31")
	// Nothing to book: the recording is tracked as posted with no entry.
	l.record(t, ref, posting.Line{Kind: posting.SalaryExpense, OrgUnitID: l.unit, Amount: 0})
	l.reconcile(t, ref)
	l.void(t, ref)
	later := "2026-07-01"
	if err := l.s.SetStart(l.core.Admin, l.le, &later); err != nil {
		t.Fatal(err)
	}
	if c := lockCode(t, l, "2026-06-30"); c != "" {
		t.Fatalf("lock %q", c)
	}
	early := "2026-01-01"
	if c, _ := startErr(t, l.s.SetStart(l.core.Admin, l.le, &early)); c != "start_date_in_locked_period" {
		t.Fatalf("start: %q", c)
	}
}
