package record_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/core/coretest"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
)

func TestParseNumberRefusesExponents(t *testing.T) {
	for _, s := range []string{"1e999999999", "1E5", "", "1.", ".5", "1234567890123456"} {
		if _, ok := record.ParseNumber(s); ok {
			t.Errorf("%q accepted", s)
		}
	}
	if d, ok := record.ParseNumber("-2.5"); !ok || d.String() != "-2.5" {
		t.Errorf("-2.5: %v %v", d, ok)
	}
}

func TestBeforeLockRefuses(t *testing.T) {
	core := coretest.New(t, "test")
	rec, admin := core.Record, core.Admin
	c := must(t)(core.IAM.CreateOrgUnit(admin, iam.OrgUnitInput{Kind: "company", Name: "C"}))
	calls, held := 0, true
	rec.SetBeforeLock(func(_ context.Context, _ int64, until pgtype.Date) ([]record.HeldDoc, error) {
		calls++
		if !held {
			return nil, nil
		}
		return []record.HeldDoc{{DocType: "x.doc", ID: 7, Number: "X-7", Date: until}}, nil
	})
	until := "2026-03-31"
	err := rec.SetPeriodLock(admin, c, &until)
	if e, ok := errors.AsType[*platform.Error](err); !ok || e.Code != "period_has_pending_documents" || len(e.Params["documents"].([]map[string]any)) != 1 {
		t.Fatalf("lock: %v", err)
	}
	if locks, err := rec.PeriodLocks(admin); err != nil || calls != 1 || locks[0].LockedUntil != nil {
		t.Fatalf("calls %d %+v %v", calls, locks, err)
	}
	held = false
	if err := rec.SetPeriodLock(admin, c, &until); err != nil || calls != 2 {
		t.Fatalf("lock: %v calls %d", err, calls)
	}
	// Moving earlier or unlocking is not moving later: the hook stays out of it.
	early := "2026-03-01"
	if err := rec.SetPeriodLock(admin, c, &early); err != nil {
		t.Fatal(err)
	}
	if err := rec.SetPeriodLock(admin, c, nil); err != nil || calls != 2 {
		t.Fatalf("unlock: %v calls %d", err, calls)
	}
}
