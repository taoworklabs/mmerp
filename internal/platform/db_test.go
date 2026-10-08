package platform_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

func dbCtx(t *testing.T) context.Context {
	t.Helper()
	ctx := platform.WithDB(t.Context(), pgtest.New(t))
	if _, err := platform.DBFrom(ctx).Exec(ctx, `CREATE TABLE items (id int)`); err != nil {
		t.Fatal(err)
	}
	return ctx
}

func count(t *testing.T, ctx context.Context) int {
	t.Helper()
	var n int
	if err := platform.DBFrom(ctx).QueryRow(ctx, `SELECT count(*) FROM items`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestInTxNestedSharesTransaction(t *testing.T) {
	ctx := dbCtx(t)
	boom := errors.New("boom")
	err := platform.InTx(ctx, func(ctx context.Context) error {
		if _, ok := platform.DBFrom(ctx).(pgx.Tx); !ok {
			t.Fatal("DBFrom inside InTx is not the transaction")
		}
		if err := platform.InTx(ctx, func(ctx context.Context) error {
			_, err := platform.DBFrom(ctx).Exec(ctx, `INSERT INTO items VALUES (1)`)
			return err
		}); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	// The inner insert must roll back with the outer transaction.
	if n := count(t, ctx); n != 0 {
		t.Fatalf("rows = %d, want 0", n)
	}
}

func TestInTxCommits(t *testing.T) {
	ctx := dbCtx(t)
	if err := platform.InTx(ctx, func(ctx context.Context) error {
		_, err := platform.DBFrom(ctx).Exec(ctx, `INSERT INTO items VALUES (1)`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if n := count(t, ctx); n != 1 {
		t.Fatalf("rows = %d, want 1", n)
	}
}
