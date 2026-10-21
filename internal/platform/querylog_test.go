package platform_test

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

type capturedRecord struct {
	level slog.Level
	msg   string
	attrs map[string]any
}

// logCapture is a slog.Handler that records everything handed to it, so tests
// can assert on levels and attrs without parsing JSON output.
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
	c.records = append(c.records, capturedRecord{level: r.Level, msg: r.Message, attrs: attrs})
	return nil
}

func (c *logCapture) WithAttrs([]slog.Attr) slog.Handler { return c }
func (c *logCapture) WithGroup(string) slog.Handler      { return c }

func (c *logCapture) all() []capturedRecord {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]capturedRecord(nil), c.records...)
}

// tracedPool returns a pool on a fresh pgtest database with tracer wired in on
// ConnConfig, so TraceQueryStart/End run on every query the test issues.
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

func TestTracerLogsNameOnce(t *testing.T) {
	c := &logCapture{}
	pool := tracedPool(t, platform.NewQueryTracer(slog.New(c), 500*time.Millisecond))
	ctx := platform.WithQueryName(t.Context(), "hrm.GetEmployee", "")

	var one int
	if err := pool.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
		t.Fatal(err)
	}

	got := c.all()
	if len(got) != 1 {
		t.Fatalf("records = %+v, want exactly 1", got)
	}
	r := got[0]
	if r.level != slog.LevelDebug {
		t.Errorf("level = %v, want debug", r.level)
	}
	if r.attrs["name"] != "hrm.GetEmployee" {
		t.Errorf("name = %v, want hrm.GetEmployee", r.attrs["name"])
	}
	if _, ok := r.attrs["duration"]; !ok {
		t.Error("missing duration attr")
	}
	if _, ok := r.attrs["rows"]; !ok {
		t.Error("missing rows attr")
	}
}

func TestTracerNeverLogsSQLOrArgs(t *testing.T) {
	c := &logCapture{}
	pool := tracedPool(t, platform.NewQueryTracer(slog.New(c), 500*time.Millisecond))
	ctx := platform.WithQueryName(t.Context(), "test.secret", "")

	var s string
	if err := pool.QueryRow(ctx, "SELECT $1::text", "secret-value-123").Scan(&s); err != nil {
		t.Fatal(err)
	}

	for _, r := range c.all() {
		if strings.Contains(r.msg, "secret-value-123") || strings.Contains(r.msg, "SELECT") {
			t.Fatalf("message leaks SQL or args: %q", r.msg)
		}
		for k, v := range r.attrs {
			s := fmt.Sprintf("%v", v)
			if strings.Contains(s, "secret-value-123") || strings.Contains(s, "SELECT") {
				t.Fatalf("attr %s=%v leaks SQL or args", k, v)
			}
		}
	}
}

func TestTracerSlowWarn(t *testing.T) {
	c := &logCapture{}
	pool := tracedPool(t, platform.NewQueryTracer(slog.New(c), time.Nanosecond))
	ctx := platform.WithQueryName(t.Context(), "hrm.GetEmployee", "")

	var one int
	if err := pool.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
		t.Fatal(err)
	}

	got := c.all()
	if len(got) != 1 || got[0].level != slog.LevelWarn {
		t.Fatalf("records = %+v, want exactly 1 warn record", got)
	}
}

func TestTracerUnnamed(t *testing.T) {
	c := &logCapture{}
	pool := tracedPool(t, platform.NewQueryTracer(slog.New(c), time.Nanosecond))

	var one int
	if err := pool.QueryRow(t.Context(), "SELECT 1").Scan(&one); err != nil {
		t.Fatal(err)
	}

	got := c.all()
	if len(got) != 1 {
		t.Fatalf("records = %+v, want exactly 1", got)
	}
	if got[0].attrs["name"] != "unnamed" {
		t.Errorf("name = %v, want unnamed", got[0].attrs["name"])
	}
	if got[0].level != slog.LevelDebug {
		t.Errorf("level = %v, want debug even when slow", got[0].level)
	}
}

func TestTracerSQLState(t *testing.T) {
	c := &logCapture{}
	pool := tracedPool(t, platform.NewQueryTracer(slog.New(c), 500*time.Millisecond))
	ctx := platform.WithQueryName(t.Context(), "test.fail", "")

	if _, err := pool.Exec(ctx, "SELECT * FROM no_such_table_xyz"); err == nil {
		t.Fatal("want error")
	}

	got := c.all()
	if len(got) != 1 {
		t.Fatalf("records = %+v, want exactly 1", got)
	}
	if got[0].attrs["sqlstate"] != "42P01" {
		t.Errorf("sqlstate = %v, want 42P01", got[0].attrs["sqlstate"])
	}
}
