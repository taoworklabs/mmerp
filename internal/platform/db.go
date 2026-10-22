package platform

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Conn is satisfied by both the pool and a transaction; sqlc stores accept it.
type Conn interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	// SendBatch runs several statements in one round trip; the data engine uses it
	// to run a list's page and its count together.
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

// PoolRole says what a pool may do outside a transaction.
type PoolRole int

const (
	// AppPool sessions are read-only by default: only InTx can write, so a write
	// that forgot it fails with 25006 instead of committing on its own.
	AppPool PoolRole = iota
	// SystemPool sessions are writable: migrations and River, which manage their own transactions.
	SystemPool
)

// NewPool opens a pool for role; tracer may be nil.
func NewPool(ctx context.Context, url string, role PoolRole, tracer pgx.QueryTracer) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	if role == AppPool {
		cfg.ConnConfig.RuntimeParams["default_transaction_read_only"] = "on"
	}
	if tracer != nil {
		cfg.ConnConfig.Tracer = tracer
	}
	return pgxpool.NewWithConfig(ctx, cfg)
}

type poolKey struct{}
type txKey struct{}

// WithDB puts the tenant database into ctx; only cmd/server and internal/app call it.
func WithDB(ctx context.Context, pool *pgxpool.Pool) context.Context {
	return context.WithValue(ctx, poolKey{}, pool)
}

// DBFrom returns the transaction in ctx if any, otherwise the pool.
func DBFrom(ctx context.Context) Conn {
	if tx := txFrom(ctx); tx != nil {
		return tx
	}
	return mustPool(ctx)
}

// InTx runs fn in a transaction, joining the one already in ctx if present. It is the
// only way to write on an AppPool.
func InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if txFrom(ctx) != nil {
		return fn(ctx)
	}
	return pgx.BeginTxFunc(ctx, mustPool(ctx), pgx.TxOptions{AccessMode: pgx.ReadWrite}, func(tx pgx.Tx) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

func txFrom(ctx context.Context) pgx.Tx {
	tx, _ := ctx.Value(txKey{}).(pgx.Tx)
	return tx
}

var errNoDB = errors.New("platform: no database in context")

func mustPool(ctx context.Context) *pgxpool.Pool {
	pool, ok := ctx.Value(poolKey{}).(*pgxpool.Pool)
	if !ok {
		// Wiring bug, not a runtime condition.
		panic(errNoDB)
	}
	return pool
}

// Violates reports whether err breaks the named constraint (unique, check or foreign key).
func Violates(err error, constraint string) bool {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	return ok && pgErr.ConstraintName == constraint
}
