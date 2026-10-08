package discussion_test

import (
	"context"
	"errors"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/discussion"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

// A draft's deletion that rolls back keeps its comments; one that commits removes them.
func TestDeletedWithTheDraft(t *testing.T) {
	pool := pgtest.New(t)
	ctx := platform.WithProducts(platform.WithDB(t.Context(), pool), []string{"test"})
	var actor int64
	if err := pool.QueryRow(ctx, `INSERT INTO iam.users (login, name, password_hash) VALUES ('u', 'U', '') RETURNING id`).Scan(&actor); err != nil {
		t.Fatal(err)
	}
	ctx = platform.WithActor(ctx, actor)
	rec := record.NewService(record.Deps{Audit: audit.NewService()})
	thing := record.Ref{Type: "test.thing", ID: 1}
	rec.Register(record.Type{Code: thing.Type, Product: "test", Kind: record.Catalog,
		Can: func(context.Context, int64, record.Action) (bool, error) { return true, nil }})
	s := discussion.NewService(discussion.Deps{Record: rec, Audit: audit.NewService()})
	if err := s.Add(ctx, thing, "nháp"); err != nil {
		t.Fatal(err)
	}

	errBoom := errors.New("boom")
	err := platform.InTx(ctx, func(ctx context.Context) error {
		if err := s.Deleted(ctx, thing); err != nil {
			return err
		}
		return errBoom
	})
	if !errors.Is(err, errBoom) {
		t.Fatal(err)
	}
	if d, err := s.List(ctx, thing); err != nil || len(d.Items) != 1 {
		t.Fatalf("after a rolled-back deletion: %+v %v", d, err)
	}
	if err := s.Deleted(ctx, thing); err != nil {
		t.Fatal(err)
	}
	if d, err := s.List(ctx, thing); err != nil || len(d.Items) != 0 {
		t.Fatalf("after the deletion: %+v %v", d, err)
	}
}
