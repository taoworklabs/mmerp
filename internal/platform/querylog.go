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
	t.logQuery(ctx, data.CommandTag.RowsAffected(), data.Err)
}

// TraceBatchStart marks when a pgx.Batch (page + count sent together) began, so
// each query traced inside it can report a duration.
func (t *queryTracer) TraceBatchStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceBatchStartData) context.Context {
	return context.WithValue(ctx, queryStartKey{}, time.Now())
}

// TraceBatchQuery fires once per query inside the batch, so a batched page and
// count each log their own record, by name and shape, never SQL or args.
func (t *queryTracer) TraceBatchQuery(ctx context.Context, _ *pgx.Conn, data pgx.TraceBatchQueryData) {
	t.logQuery(ctx, data.CommandTag.RowsAffected(), data.Err)
}

// TraceBatchEnd only logs when the batch failed before any query in it was
// traced (e.g. preparing it); per-query failures already went through TraceBatchQuery.
func (t *queryTracer) TraceBatchEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceBatchEndData) {
	if data.Err == nil {
		return
	}
	t.logQuery(ctx, 0, data.Err)
}

func (t *queryTracer) logQuery(ctx context.Context, rowsAffected int64, err error) {
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
		slog.Int64("rows", rowsAffected),
	}
	if name.shape != "" {
		attrs = append(attrs, slog.String("shape", name.shape))
	}

	level := slog.LevelDebug
	if err != nil {
		attrs = append(attrs, slog.String("sqlstate", pgSQLState(err)))
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
