package record_test

import (
	"errors"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/coretest"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/record"
)

// Touch bumps a draft's version and keeps its header and approval fields.
func TestTouchKeepsTheHeader(t *testing.T) {
	core := coretest.New(t, "test")
	ids, rec, admin := core.IAM, core.Record, core.Admin
	adminDoc(core, record.Type{Fields: []record.Field{{Key: "days", Kind: record.Number, Label: "days"}}})
	c := must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{Kind: "company", Name: "C"}))
	amount := int64(500)
	h := record.Header{Date: "2026-03-10", OrgUnitID: c, Amount: &amount, Fields: map[string]string{"days": "2.5"}}
	d, err := rec.Create(admin, "test.doc", h)
	if err != nil {
		t.Fatal(err)
	}

	if v, err := rec.Touch(admin, d.Ref, 1); err != nil || v != 2 {
		t.Fatalf("touch: %d %v", v, err)
	}
	got, err := rec.Get(admin, d.Ref)
	if err != nil || got.Version != 2 || got.Date != "2026-03-10" || got.OrgUnitID != c ||
		got.Amount == nil || *got.Amount != 500 || got.Fields["days"] != "2.5" {
		t.Fatalf("after touch: %+v %v", got, err)
	}

	if _, err := rec.Touch(admin, d.Ref, 1); !errors.Is(err, record.ErrVersionConflict) {
		t.Fatalf("stale version: %v", err)
	}
	until := "2026-03-31"
	if err := rec.SetPeriodLock(admin, c, &until); err != nil {
		t.Fatal(err)
	}
	if _, err := rec.Touch(admin, d.Ref, 2); errorCode(err) != "period_locked" {
		t.Fatalf("locked period: %v", err)
	}
	if err := rec.SetPeriodLock(admin, c, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := rec.Transition(admin, d.Ref, 2, record.Posted); err != nil {
		t.Fatal(err)
	}
	if _, err := rec.Touch(admin, d.Ref, 3); !errors.Is(err, record.ErrNotEditable) {
		t.Fatalf("posted: %v", err)
	}
}
