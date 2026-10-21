package platform

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type queryNameKey struct{}

type queryName struct {
	name  string
	shape string
}

// WithQueryName attaches a query's name, and for list queries its shape (active
// filters and sort key, from code only), so the pool's tracer can log it. The
// store never logs; only NewQueryTracer does, reading these from ctx.
func WithQueryName(ctx context.Context, name, shape string) context.Context {
	return context.WithValue(ctx, queryNameKey{}, queryName{name: name, shape: shape})
}

type queryStartKey struct{}

type queryTracer struct {
	log  *slog.Logger
	slow time.Duration
}

// NewQueryTracer logs every query once, by name and duration: never by SQL text or
// parameter values, which pgx hands it on TraceQueryStartData but it never reads.
func NewQueryTracer(log *slog.Logger, slow time.Duration) pgx.QueryTracer {
	return &queryTracer{log: log, slow: slow}
}

func (t *queryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, queryStartKey{}, time.Now())
}

func (t *queryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	name, _ := ctx.Value(queryNameKey{}).(queryName)
	if name.name == "" {
		name.name = "unnamed"
	}
	var dur time.Duration
	if start, ok := ctx.Value(queryStartKey{}).(time.Time); ok {
		dur = time.Since(start)
	}

	attrs := []slog.Attr{
		slog.String("name", name.name),
		slog.Duration("duration", dur),
		slog.Int64("rows", data.CommandTag.RowsAffected()),
	}
	if name.shape != "" {
		attrs = append(attrs, slog.String("shape", name.shape))
	}

	level := slog.LevelDebug
	if data.Err != nil {
		attrs = append(attrs, slog.String("sqlstate", pgSQLState(data.Err)))
	} else if name.name != "unnamed" && t.slow > 0 && dur >= t.slow {
		level = slog.LevelWarn
	}

	t.log.LogAttrs(ctx, level, "query", attrs...)
}

func pgSQLState(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}
