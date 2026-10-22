package data_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgconn/ctxwatch"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

// cancelCtxPool returns a pool configured to actually send a cancel request to the
// server and wait for its response, instead of pgx's default of giving up locally.
// Only this way does a cancelled ctx surface the server's real SQLSTATE.
func cancelCtxPool(t *testing.T, app *pgxpool.Pool) *pgxpool.Pool {
	t.Helper()
	cfg, err := pgxpool.ParseConfig(app.Config().ConnString())
	if err != nil {
		t.Fatalf("cancelCtxPool: %v", err)
	}
	cfg.ConnConfig.BuildContextWatcherHandler = func(pgConn *pgconn.PgConn) ctxwatch.Handler {
		return &pgconn.CancelRequestContextWatcherHandler{Conn: pgConn, DeadlineDelay: 2 * time.Second}
	}
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatalf("cancelCtxPool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestClassify(t *testing.T) {
	pool := pgtest.New(t)
	ctx := platform.WithDB(t.Context(), pool)

	t.Run("unique", func(t *testing.T) {
		if err := platform.InTx(ctx, func(ctx context.Context) error {
			_, err := platform.DBFrom(ctx).Exec(ctx, `CREATE TABLE t_unique (id int primary key, code text unique)`)
			if err != nil {
				return err
			}
			_, err = platform.DBFrom(ctx).Exec(ctx, `INSERT INTO t_unique (id, code) VALUES (1, 'x')`)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		err := platform.InTx(ctx, func(ctx context.Context) error {
			_, err := platform.DBFrom(ctx).Exec(ctx, `INSERT INTO t_unique (id, code) VALUES (2, 'x')`)
			return err
		})
		if got := data.Classify(err); got != data.Unique {
			t.Fatalf("Classify() = %v, want Unique (err: %v)", got, err)
		}
	})

	t.Run("foreign key", func(t *testing.T) {
		if err := platform.InTx(ctx, func(ctx context.Context) error {
			_, err := platform.DBFrom(ctx).Exec(ctx, `CREATE TABLE t_fk (id int primary key, parent_id int references t_fk(id))`)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		err := platform.InTx(ctx, func(ctx context.Context) error {
			_, err := platform.DBFrom(ctx).Exec(ctx, `INSERT INTO t_fk (id, parent_id) VALUES (1, 999)`)
			return err
		})
		if got := data.Classify(err); got != data.ForeignKey {
			t.Fatalf("Classify() = %v, want ForeignKey (err: %v)", got, err)
		}
	})

	t.Run("check", func(t *testing.T) {
		if err := platform.InTx(ctx, func(ctx context.Context) error {
			_, err := platform.DBFrom(ctx).Exec(ctx, `CREATE TABLE t_check (id int primary key, n int check (n > 0))`)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		err := platform.InTx(ctx, func(ctx context.Context) error {
			_, err := platform.DBFrom(ctx).Exec(ctx, `INSERT INTO t_check (id, n) VALUES (1, -1)`)
			return err
		})
		if got := data.Classify(err); got != data.Check {
			t.Fatalf("Classify() = %v, want Check (err: %v)", got, err)
		}
	})

	t.Run("not null", func(t *testing.T) {
		if err := platform.InTx(ctx, func(ctx context.Context) error {
			_, err := platform.DBFrom(ctx).Exec(ctx, `CREATE TABLE t_notnull (id int primary key, req text not null)`)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		err := platform.InTx(ctx, func(ctx context.Context) error {
			_, err := platform.DBFrom(ctx).Exec(ctx, `INSERT INTO t_notnull (id, req) VALUES (1, NULL)`)
			return err
		})
		if got := data.Classify(err); got != data.NotNull {
			t.Fatalf("Classify() = %v, want NotNull (err: %v)", got, err)
		}
	})

	t.Run("deadlock", func(t *testing.T) {
		if err := platform.InTx(ctx, func(ctx context.Context) error {
			_, err := platform.DBFrom(ctx).Exec(ctx, `CREATE TABLE t_deadlock (id int primary key)`)
			if err != nil {
				return err
			}
			_, err = platform.DBFrom(ctx).Exec(ctx, `INSERT INTO t_deadlock VALUES (1), (2)`)
			return err
		}); err != nil {
			t.Fatal(err)
		}

		readyA := make(chan struct{})
		done := make(chan error, 2)
		go func() {
			done <- platform.InTx(ctx, func(ctx context.Context) error {
				if _, err := platform.DBFrom(ctx).Exec(ctx, `SELECT id FROM t_deadlock WHERE id = 1 FOR UPDATE`); err != nil {
					return err
				}
				close(readyA)
				time.Sleep(300 * time.Millisecond)
				_, err := platform.DBFrom(ctx).Exec(ctx, `SELECT id FROM t_deadlock WHERE id = 2 FOR UPDATE`)
				return err
			})
		}()
		go func() {
			<-readyA
			done <- platform.InTx(ctx, func(ctx context.Context) error {
				if _, err := platform.DBFrom(ctx).Exec(ctx, `SELECT id FROM t_deadlock WHERE id = 2 FOR UPDATE`); err != nil {
					return err
				}
				_, err := platform.DBFrom(ctx).Exec(ctx, `SELECT id FROM t_deadlock WHERE id = 1 FOR UPDATE`)
				return err
			})
		}()

		var deadlockErr error
		for range 2 {
			if err := <-done; err != nil {
				deadlockErr = err
			}
		}
		if deadlockErr == nil {
			t.Fatal("want one of the two transactions to fail with a deadlock")
		}
		if got := data.Classify(deadlockErr); got != data.Deadlock {
			t.Fatalf("Classify() = %v, want Deadlock (err: %v)", got, deadlockErr)
		}
		if !data.IsRetryable(deadlockErr) {
			t.Error("IsRetryable() = false, want true for a deadlock")
		}
	})

	t.Run("lock timeout", func(t *testing.T) {
		if err := platform.InTx(ctx, func(ctx context.Context) error {
			_, err := platform.DBFrom(ctx).Exec(ctx, `CREATE TABLE t_locktimeout (id int primary key)`)
			if err != nil {
				return err
			}
			_, err = platform.DBFrom(ctx).Exec(ctx, `INSERT INTO t_locktimeout VALUES (1)`)
			return err
		}); err != nil {
			t.Fatal(err)
		}

		locked := make(chan struct{})
		release := make(chan struct{})
		holderErr := make(chan error, 1)
		go func() {
			holderErr <- platform.InTx(ctx, func(ctx context.Context) error {
				if _, err := platform.DBFrom(ctx).Exec(ctx, `SELECT id FROM t_locktimeout WHERE id = 1 FOR UPDATE`); err != nil {
					return err
				}
				close(locked)
				<-release
				return nil
			})
		}()
		<-locked
		err := platform.InTx(ctx, func(ctx context.Context) error {
			if _, err := platform.DBFrom(ctx).Exec(ctx, `SET LOCAL lock_timeout = '10ms'`); err != nil {
				return err
			}
			_, err := platform.DBFrom(ctx).Exec(ctx, `SELECT id FROM t_locktimeout WHERE id = 1 FOR UPDATE`)
			return err
		})
		close(release)
		if e := <-holderErr; e != nil {
			t.Fatalf("holder: %v", e)
		}
		if got := data.Classify(err); got != data.LockTimeout {
			t.Fatalf("Classify() = %v, want LockTimeout (err: %v)", got, err)
		}
	})

	t.Run("canceled", func(t *testing.T) {
		cancelPool := cancelCtxPool(t, pool)
		cctx, cancel := context.WithCancel(t.Context())
		errCh := make(chan error, 1)
		go func() {
			_, err := cancelPool.Exec(cctx, `SELECT pg_sleep(5)`)
			errCh <- err
		}()
		time.Sleep(200 * time.Millisecond)
		cancel()
		err := <-errCh
		if got := data.Classify(err); got != data.Canceled {
			t.Fatalf("Classify() = %v, want Canceled (err: %v)", got, err)
		}
	})

	t.Run("read-only", func(t *testing.T) {
		// Outside InTx, the AppPool session is read-only: the write itself is the violation.
		_, err := platform.DBFrom(ctx).Exec(ctx, `CREATE TABLE t_readonly (id int)`)
		if got := data.Classify(err); got != data.ReadOnly {
			t.Fatalf("Classify() = %v, want ReadOnly (err: %v)", got, err)
		}
	})
}

func TestNormalizeMapped(t *testing.T) {
	errTaken := &platform.Error{Status: http.StatusConflict, Code: "t_code_taken"}
	data.RegisterConstraints(data.Constraints{"public.t_code_key": errTaken})

	pool := pgtest.New(t)
	ctx := platform.WithDB(t.Context(), pool)
	if err := platform.InTx(ctx, func(ctx context.Context) error {
		_, err := platform.DBFrom(ctx).Exec(ctx, `CREATE TABLE t (id int primary key, code text unique)`)
		if err != nil {
			return err
		}
		_, err = platform.DBFrom(ctx).Exec(ctx, `INSERT INTO t (id, code) VALUES (1, 'x')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	err := platform.InTx(ctx, func(ctx context.Context) error {
		_, err := platform.DBFrom(ctx).Exec(ctx, `INSERT INTO t (id, code) VALUES (2, 'x')`)
		return err
	})

	normalized := data.Normalize(err)
	got, ok := errors.AsType[*platform.Error](normalized)
	if !ok || got != errTaken {
		t.Fatalf("errors.AsType[*platform.Error](Normalize(err)) = %v, %v, want %v, true", got, ok, errTaken)
	}
	if !platform.Violates(normalized, "t_code_key") {
		t.Error("platform.Violates(Normalize(err), \"t_code_key\") = false, want true")
	}
}

func TestNormalizeUnmapped(t *testing.T) {
	pool := pgtest.New(t)
	ctx := platform.WithDB(t.Context(), pool)
	if err := platform.InTx(ctx, func(ctx context.Context) error {
		_, err := platform.DBFrom(ctx).Exec(ctx, `CREATE TABLE t_unmapped (id int primary key, code text unique)`)
		if err != nil {
			return err
		}
		_, err = platform.DBFrom(ctx).Exec(ctx, `INSERT INTO t_unmapped (id, code) VALUES (1, 'x')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	err := platform.InTx(ctx, func(ctx context.Context) error {
		_, err := platform.DBFrom(ctx).Exec(ctx, `INSERT INTO t_unmapped (id, code) VALUES (2, 'x')`)
		return err
	})

	if _, ok := errors.AsType[*platform.Error](data.Normalize(err)); ok {
		t.Error("errors.AsType[*platform.Error] on an unmapped violation = true, want false")
	}
}

func TestRegisterConstraintsIdempotent(t *testing.T) {
	e := &platform.Error{Status: http.StatusConflict, Code: "idempotent_taken"}
	data.RegisterConstraints(data.Constraints{"public.t_idempotent_key": e})
	data.RegisterConstraints(data.Constraints{"public.t_idempotent_key": e})
	data.RegisterConstraints(data.Constraints{"public.t_idempotent_key": &platform.Error{Status: http.StatusConflict, Code: "idempotent_taken"}})
}

func TestRegisterConstraintsConflictPanics(t *testing.T) {
	data.RegisterConstraints(data.Constraints{"public.t_conflict_key": &platform.Error{Status: http.StatusConflict, Code: "conflict_a"}})
	defer func() {
		if recover() == nil {
			t.Fatal("RegisterConstraints did not panic on a conflicting re-registration")
		}
	}()
	data.RegisterConstraints(data.Constraints{"public.t_conflict_key": &platform.Error{Status: http.StatusConflict, Code: "conflict_b"}})
}

func TestNoRowsPassesThrough(t *testing.T) {
	if !errors.Is(data.Normalize(pgx.ErrNoRows), pgx.ErrNoRows) {
		t.Error("Normalize(pgx.ErrNoRows) lost pgx.ErrNoRows")
	}
}
