package record_test

import (
	"context"
	"errors"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/coretest"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// Touch bumps a draft's version and keeps its header and approval fields.
func TestTouchKeepsTheHeader(t *testing.T) {
	core := coretest.New(t, "test")
	ids, rec, admin := core.IAM, core.Record, core.Admin
	rec.Register(record.Type{Code: "test.doc", Product: "test", Kind: record.Document, NumberPrefix: "T",
		Can:    func(context.Context, int64, record.Action) (bool, error) { return true, nil },
		Fields: []record.Field{{Key: "days", Kind: record.Number, Label: "days"}}})
	c := must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{Kind: "company", Name: "C"}))
	amount := int64(500)
	h := record.Header{Date: "2026-03-10", OrgUnitID: c, Amount: &amount, Fields: map[string]string{"days": "2.5"}}
	d, err := rec.Create(admin, "test.doc", h)
	if err != nil {
		t.Fatal(err)
	}

	if err := rec.Touch(admin, d.Ref, 1); err != nil {
		t.Fatal(err)
	}
	got, err := rec.Get(admin, d.Ref)
	if err != nil || got.Version != 2 || got.Date != "2026-03-10" || got.OrgUnitID != c ||
		got.Amount == nil || *got.Amount != 500 || got.Fields["days"] != "2.5" {
		t.Fatalf("after touch: %+v %v", got, err)
	}

	if err := rec.Touch(admin, d.Ref, 1); !errors.Is(err, record.ErrVersionConflict) {
		t.Fatalf("stale version: %v", err)
	}
	until := "2026-03-31"
	if err := rec.SetPeriodLock(admin, c, &until); err != nil {
		t.Fatal(err)
	}
	if e, ok := errors.AsType[*platform.Error](rec.Touch(admin, d.Ref, 2)); !ok || e.Code != "period_locked" {
		t.Fatalf("locked period: %v", e)
	}
	if err := rec.SetPeriodLock(admin, c, nil); err != nil {
		t.Fatal(err)
	}
	if err := rec.Transition(admin, d.Ref, 2, record.Posted); err != nil {
		t.Fatal(err)
	}
	if err := rec.Touch(admin, d.Ref, 3); !errors.Is(err, record.ErrNotEditable) {
		t.Fatalf("posted: %v", err)
	}
}
