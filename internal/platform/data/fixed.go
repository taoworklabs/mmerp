package data

import (
	"context"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// run is the one place a no-parameter fixed query sets its name and normalizes
// its error: One, Many and ExecRows all call it, and Exec adapts to it.
func run[Q, R any](ctx context.Context, name string, newQ func(platform.Conn) Q, f func(Q, context.Context) (R, error)) (R, error) {
	ctx = platform.WithQueryName(ctx, name, "")
	r, err := f(newQ(platform.DBFrom(ctx)), ctx)
	return r, Normalize(err)
}

// runArg is the one place a one-parameter fixed query sets its name and normalizes
// its error: OneArg, ManyArg and ExecRowsArg all call it, and ExecArg adapts to it.
func runArg[Q, P, R any](ctx context.Context, name string, newQ func(platform.Conn) Q, p P, f func(Q, context.Context, P) (R, error)) (R, error) {
	ctx = platform.WithQueryName(ctx, name, "")
	r, err := f(newQ(platform.DBFrom(ctx)), ctx, p)
	return r, Normalize(err)
}

// OneQuery runs a fixed query with no parameters that returns one row.
type OneQuery[R any] struct {
	name string
	run  func(ctx context.Context) (R, error)
}

// One wraps a sqlc method with no parameters that returns one row.
func One[Q, R any](name string, newQ func(platform.Conn) Q, f func(Q, context.Context) (R, error)) OneQuery[R] {
	return OneQuery[R]{
		name: name,
		run:  func(ctx context.Context) (R, error) { return run(ctx, name, newQ, f) },
	}
}

// Run executes the query.
func (d OneQuery[R]) Run(ctx context.Context) (R, error) { return d.run(ctx) }

// Name reports the query's registered name.
func (d OneQuery[R]) Name() string { return d.name }

// OneArgQuery runs a fixed query with one parameter that returns one row.
type OneArgQuery[P, R any] struct {
	name string
	run  func(ctx context.Context, p P) (R, error)
}

// OneArg wraps a sqlc method with one parameter that returns one row.
func OneArg[Q, P, R any](name string, newQ func(platform.Conn) Q, f func(Q, context.Context, P) (R, error)) OneArgQuery[P, R] {
	return OneArgQuery[P, R]{
		name: name,
		run:  func(ctx context.Context, p P) (R, error) { return runArg(ctx, name, newQ, p, f) },
	}
}

// Run executes the query with p.
func (d OneArgQuery[P, R]) Run(ctx context.Context, p P) (R, error) { return d.run(ctx, p) }

// Name reports the query's registered name.
func (d OneArgQuery[P, R]) Name() string { return d.name }

// ManyQuery runs a fixed query with no parameters that returns many rows.
type ManyQuery[R any] struct {
	name string
	run  func(ctx context.Context) ([]R, error)
}

// Many wraps a sqlc method with no parameters that returns many rows.
func Many[Q, R any](name string, newQ func(platform.Conn) Q, f func(Q, context.Context) ([]R, error)) ManyQuery[R] {
	return ManyQuery[R]{
		name: name,
		run:  func(ctx context.Context) ([]R, error) { return run(ctx, name, newQ, f) },
	}
}

// Run executes the query.
func (d ManyQuery[R]) Run(ctx context.Context) ([]R, error) { return d.run(ctx) }

// Name reports the query's registered name.
func (d ManyQuery[R]) Name() string { return d.name }

// ManyArgQuery runs a fixed query with one parameter that returns many rows.
type ManyArgQuery[P, R any] struct {
	name string
	run  func(ctx context.Context, p P) ([]R, error)
}

// ManyArg wraps a sqlc method with one parameter that returns many rows.
func ManyArg[Q, P, R any](name string, newQ func(platform.Conn) Q, f func(Q, context.Context, P) ([]R, error)) ManyArgQuery[P, R] {
	return ManyArgQuery[P, R]{
		name: name,
		run:  func(ctx context.Context, p P) ([]R, error) { return runArg(ctx, name, newQ, p, f) },
	}
}

// Run executes the query with p.
func (d ManyArgQuery[P, R]) Run(ctx context.Context, p P) ([]R, error) { return d.run(ctx, p) }

// Name reports the query's registered name.
func (d ManyArgQuery[P, R]) Name() string { return d.name }

// ExecQuery runs a fixed query with no parameters and no result rows.
type ExecQuery struct {
	name string
	run  func(ctx context.Context) error
}

// Exec wraps a sqlc method with no parameters and no result rows, adapting it
// to run's shape with a throwaway result type.
func Exec[Q any](name string, newQ func(platform.Conn) Q, f func(Q, context.Context) error) ExecQuery {
	return ExecQuery{
		name: name,
		run: func(ctx context.Context) error {
			_, err := run(ctx, name, newQ, func(q Q, ctx context.Context) (struct{}, error) {
				return struct{}{}, f(q, ctx)
			})
			return err
		},
	}
}

// Run executes the query.
func (d ExecQuery) Run(ctx context.Context) error { return d.run(ctx) }

// Name reports the query's registered name.
func (d ExecQuery) Name() string { return d.name }

// ExecArgQuery runs a fixed query with one parameter and no result rows.
type ExecArgQuery[P any] struct {
	name string
	run  func(ctx context.Context, p P) error
}

// ExecArg wraps a sqlc method with one parameter and no result rows, adapting it
// to runArg's shape with a throwaway result type.
func ExecArg[Q, P any](name string, newQ func(platform.Conn) Q, f func(Q, context.Context, P) error) ExecArgQuery[P] {
	return ExecArgQuery[P]{
		name: name,
		run: func(ctx context.Context, p P) error {
			_, err := runArg(ctx, name, newQ, p, func(q Q, ctx context.Context, p P) (struct{}, error) {
				return struct{}{}, f(q, ctx, p)
			})
			return err
		},
	}
}

// Run executes the query with p.
func (d ExecArgQuery[P]) Run(ctx context.Context, p P) error { return d.run(ctx, p) }

// Name reports the query's registered name.
func (d ExecArgQuery[P]) Name() string { return d.name }

// ExecRowsQuery runs a fixed query with no parameters that reports rows affected.
type ExecRowsQuery struct {
	name string
	run  func(ctx context.Context) (int64, error)
}

// ExecRows wraps a sqlc method with no parameters that reports rows affected.
func ExecRows[Q any](name string, newQ func(platform.Conn) Q, f func(Q, context.Context) (int64, error)) ExecRowsQuery {
	return ExecRowsQuery{
		name: name,
		run:  func(ctx context.Context) (int64, error) { return run(ctx, name, newQ, f) },
	}
}

// Run executes the query.
func (d ExecRowsQuery) Run(ctx context.Context) (int64, error) { return d.run(ctx) }

// Name reports the query's registered name.
func (d ExecRowsQuery) Name() string { return d.name }

// ExecRowsArgQuery runs a fixed query with one parameter that reports rows affected.
type ExecRowsArgQuery[P any] struct {
	name string
	run  func(ctx context.Context, p P) (int64, error)
}

// ExecRowsArg wraps a sqlc method with one parameter that reports rows affected.
func ExecRowsArg[Q, P any](name string, newQ func(platform.Conn) Q, f func(Q, context.Context, P) (int64, error)) ExecRowsArgQuery[P] {
	return ExecRowsArgQuery[P]{
		name: name,
		run:  func(ctx context.Context, p P) (int64, error) { return runArg(ctx, name, newQ, p, f) },
	}
}

// Run executes the query with p.
func (d ExecRowsArgQuery[P]) Run(ctx context.Context, p P) (int64, error) { return d.run(ctx, p) }

// Name reports the query's registered name.
func (d ExecRowsArgQuery[P]) Name() string { return d.name }
