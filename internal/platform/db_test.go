package platform_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

func dbCtx(t *testing.T) context.Context {
	t.Helper()
	ctx := platform.WithDB(t.Context(), pgtest.New(t))
	if err := platform.InTx(ctx, func(ctx context.Context) error {
		_, err := platform.DBFrom(ctx).Exec(ctx, `CREATE TABLE items (id int)`)
		return err
	}); err != nil {
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

// Every form of write fails outside InTx, so a forgotten InTx cannot write silently.
func TestReadOnlyOutsideTx(t *testing.T) {
	ctx := dbCtx(t)
	if err := platform.InTx(ctx, func(ctx context.Context) error {
		_, err := platform.DBFrom(ctx).Exec(ctx, `INSERT INTO items VALUES (1);
			CREATE SEQUENCE seq;
			CREATE FUNCTION add_item() RETURNS void LANGUAGE plpgsql AS $$ BEGIN INSERT INTO items VALUES (2); END $$`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	forms := []struct {
		name, sql string
		writes    bool
	}{
		{"insert", `INSERT INTO items VALUES (3)`, true},
		{"update", `UPDATE items SET id = id`, true},
		{"delete", `DELETE FROM items WHERE id = 0`, true},
		{"data-modifying CTE", `WITH x AS (INSERT INTO items VALUES (4) RETURNING id) SELECT * FROM x`, true},
		{"for update", `SELECT id FROM items FOR UPDATE`, true},
		{"for share", `SELECT id FROM items FOR SHARE`, true},
		{"nextval", `SELECT nextval('seq')`, true},
		{"function that writes", `SELECT add_item()`, true},
		{"advisory xact lock", `SELECT pg_advisory_xact_lock(1)`, false},
		// Temp tables are DDL: refused too.
		{"create temp table", `CREATE TEMP TABLE tmp (id int)`, true},
	}
	run := func(ctx context.Context, sql string) error {
		rows, err := platform.DBFrom(ctx).Query(ctx, sql)
		if err != nil {
			return err
		}
		rows.Close()
		return rows.Err()
	}
	rollback := errors.New("rollback")
	for _, f := range forms {
		t.Run(f.name, func(t *testing.T) {
			err := run(ctx, f.sql)
			pgErr, _ := errors.AsType[*pgconn.PgError](err)
			if f.writes && (pgErr == nil || pgErr.Code != "25006") {
				t.Errorf("outside InTx: err = %v, want 25006", err)
			}
			if !f.writes && err != nil {
				t.Errorf("outside InTx: %v", err)
			}
			if err := platform.InTx(ctx, func(ctx context.Context) error {
				if err := run(ctx, f.sql); err != nil {
					return err
				}
				return rollback
			}); !errors.Is(err, rollback) {
				t.Errorf("inside InTx: %v", err)
			}
		})
	}
}
