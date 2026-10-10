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

// InTx runs fn in a transaction, joining the one already in ctx if present.
func InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if txFrom(ctx) != nil {
		return fn(ctx)
	}
	return pgx.BeginFunc(ctx, mustPool(ctx), func(tx pgx.Tx) error {
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
