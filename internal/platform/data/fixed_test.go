package data_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

// fakeQueries stands in for a sqlc-generated *Queries: its methods run real SQL
// on whatever platform.Conn it was built with (pool or tx).
type fakeQueries struct {
	db platform.Conn
}

func newFakeQueries(c platform.Conn) *fakeQueries { return &fakeQueries{db: c} }

func (q *fakeQueries) GetValue(ctx context.Context, id int64) (string, error) {
	var v string
	err := q.db.QueryRow(ctx, `SELECT v FROM fixed_test_t WHERE id = $1`, id).Scan(&v)
	return v, err
}

func (q *fakeQueries) InsertUnique(ctx context.Context, code string) error {
	_, err := q.db.Exec(ctx, `INSERT INTO fixed_test_unique (code) VALUES ($1)`, code)
	return err
}

func TestOne1RunsInCtxTx(t *testing.T) {
	pool := pgtest.New(t)
	ctx := platform.WithDB(t.Context(), pool)

	getValue := data.OneArg("test.GetValue", newFakeQueries, (*fakeQueries).GetValue)

	err := platform.InTx(ctx, func(ctx context.Context) error {
		if _, err := platform.DBFrom(ctx).Exec(ctx, `CREATE TABLE fixed_test_t (id int primary key, v text)`); err != nil {
			return err
		}
		if _, err := platform.DBFrom(ctx).Exec(ctx, `INSERT INTO fixed_test_t (id, v) VALUES (1, 'hello')`); err != nil {
			return err
		}
		// The row above is only visible inside this same, uncommitted tx.
		got, err := getValue.Run(ctx, 1)
		if err != nil {
			return err
		}
		if got != "hello" {
			t.Errorf("Run() = %q, want %q", got, "hello")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunNormalizesErrors(t *testing.T) {
	errTaken := &platform.Error{Status: http.StatusConflict, Code: "fixed_test_code_taken"}
	data.RegisterConstraints(data.Constraints{"public.fixed_test_unique_code_key": errTaken})

	pool := pgtest.New(t)
	ctx := platform.WithDB(t.Context(), pool)
	if err := platform.InTx(ctx, func(ctx context.Context) error {
		_, err := platform.DBFrom(ctx).Exec(ctx, `CREATE TABLE fixed_test_unique (id serial primary key, code text unique)`)
		return err
	}); err != nil {
		t.Fatal(err)
	}

	insertUnique := data.ExecArg("test.InsertUnique", newFakeQueries, (*fakeQueries).InsertUnique)

	if err := platform.InTx(ctx, func(ctx context.Context) error {
		return insertUnique.Run(ctx, "dup")
	}); err != nil {
		t.Fatal(err)
	}

	err := platform.InTx(ctx, func(ctx context.Context) error {
		return insertUnique.Run(ctx, "dup")
	})

	got, ok := errors.AsType[*platform.Error](err)
	if !ok || got != errTaken {
		t.Fatalf("errors.AsType[*platform.Error](Run err) = %v, %v, want %v, true", got, ok, errTaken)
	}
}

type capturedRecord struct {
	attrs map[string]any
}

// logCapture records every log record handed to it, so the test can assert on
// the logged name without parsing JSON output.
type logCapture struct {
	mu      sync.Mutex
	records []capturedRecord
}

func (c *logCapture) Enabled(context.Context, slog.Level) bool { return true }

func (c *logCapture) Handle(_ context.Context, r slog.Record) error {
	attrs := map[string]any{}
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})
	c.mu.Lock()
	defer c.mu.Unlock()
	c.records = append(c.records, capturedRecord{attrs: attrs})
	return nil
}

func (c *logCapture) WithAttrs([]slog.Attr) slog.Handler { return c }
func (c *logCapture) WithGroup(string) slog.Handler      { return c }

func (c *logCapture) all() []capturedRecord {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]capturedRecord(nil), c.records...)
}

// tracedPool returns an AppPool on a fresh pgtest database with tracer wired in,
// so TraceQueryStart/End run on every query the test issues.
func tracedPool(t *testing.T, tracer pgx.QueryTracer) *pgxpool.Pool {
	t.Helper()
	cfg := pgtest.New(t).Config()
	cfg.ConnConfig.Tracer = tracer
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatalf("tracedPool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestRunSetsQueryName(t *testing.T) {
	c := &logCapture{}
	pool := tracedPool(t, platform.NewQueryTracer(slog.New(c), 0))
	ctx := platform.WithDB(t.Context(), pool)
	if err := platform.InTx(ctx, func(ctx context.Context) error {
		_, err := platform.DBFrom(ctx).Exec(ctx, `CREATE TABLE fixed_test_name (id int primary key, v text)`)
		if err != nil {
			return err
		}
		_, err = platform.DBFrom(ctx).Exec(ctx, `INSERT INTO fixed_test_name (id, v) VALUES (1, 'hi')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}

	getValue := data.OneArg("test.NamedGetValue", newFakeQueries, func(q *fakeQueries, ctx context.Context, id int64) (string, error) {
		var v string
		err := q.db.QueryRow(ctx, `SELECT v FROM fixed_test_name WHERE id = $1`, id).Scan(&v)
		return v, err
	})

	if _, err := getValue.Run(ctx, 1); err != nil {
		t.Fatal(err)
	}

	var found bool
	for _, r := range c.all() {
		if r.attrs["name"] == getValue.Name() {
			found = true
		}
	}
	if !found {
		t.Fatalf("no logged record with name %q, got %+v", getValue.Name(), c.all())
	}
}
