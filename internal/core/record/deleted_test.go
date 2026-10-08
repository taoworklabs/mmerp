package record_test

import (
	"context"
	"errors"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/numbering"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/setting"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

// A keeper's Deleted runs in the deletion's transaction, once the draft's row is gone,
// and its failure keeps the draft.
func TestKeepersGoWithTheDraft(t *testing.T) {
	pool := pgtest.New(t)
	ctx := platform.WithKeyring(platform.WithProducts(platform.WithDB(t.Context(), pool), []string{"test"}), pgtest.Keyring())
	ids := iam.NewService(iam.Deps{Setting: setting.NewService(), Audit: audit.NewService()})
	rec := record.NewService(record.Deps{IAM: ids, Numbering: numbering.NewService(), Audit: audit.NewService()})
	ids.SetTreeHook(rec)
	rec.Register(record.Type{Code: "test.doc", Product: "test", Kind: record.Document, NumberPrefix: "T",
		Can: func(context.Context, int64, record.Action) (bool, error) { return true, nil }})
	admin := platform.WithActor(ctx, must(t)(ids.CreateAdmin(ctx, "admin", "Admin", "long enough")))
	c := must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{Kind: "company", Name: "C"}))
	d, err := rec.Create(admin, "test.doc", record.Header{Date: "2026-03-10", OrgUnitID: c})
	if err != nil {
		t.Fatal(err)
	}

	errBoom := errors.New("boom")
	k := &keeper{t: t, rec: rec, fail: errBoom}
	rec.OnDeleted(k)
	if err := rec.Delete(admin, d.Ref, d.Version); !errors.Is(err, errBoom) {
		t.Fatalf("delete with a failing keeper = %v", err)
	}
	if _, err := rec.Get(admin, d.Ref); err != nil {
		t.Fatalf("draft lost after a failed keeper: %v", err)
	}
	k.fail = nil
	if err := rec.Delete(admin, d.Ref, d.Version); err != nil {
		t.Fatal(err)
	}
	if len(k.seen) != 2 || k.seen[1] != d.Ref {
		t.Fatalf("keeper saw %v", k.seen)
	}
}

type keeper struct {
	t    *testing.T
	rec  *record.Service
	fail error
	seen []record.Ref
}

func (k *keeper) Deleted(ctx context.Context, ref record.Ref) error {
	k.seen = append(k.seen, ref)
	if _, err := k.rec.Get(ctx, ref); !errors.Is(err, platform.ErrNotFound) {
		k.t.Errorf("keeper sees the draft: %v", err)
	}
	return k.fail
}
