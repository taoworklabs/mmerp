package data

import (
	"context"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// run0 is the one place a no-parameter fixed query sets its name and normalizes
// its error: One0, Many0 and ExecRows0 all call it, and Exec0 adapts to it.
func run0[Q, R any](ctx context.Context, name string, newQ func(platform.Conn) Q, f func(Q, context.Context) (R, error)) (R, error) {
	ctx = platform.WithQueryName(ctx, name, "")
	r, err := f(newQ(platform.DBFrom(ctx)), ctx)
	return r, Normalize(err)
}

// run1 is the one place a one-parameter fixed query sets its name and normalizes
// its error: One1, Many1 and ExecRows1 all call it, and Exec1 adapts to it.
func run1[Q, P, R any](ctx context.Context, name string, newQ func(platform.Conn) Q, p P, f func(Q, context.Context, P) (R, error)) (R, error) {
	ctx = platform.WithQueryName(ctx, name, "")
	r, err := f(newQ(platform.DBFrom(ctx)), ctx, p)
	return r, Normalize(err)
}

// One0Query runs a fixed query with no parameters that returns one row.
type One0Query[R any] struct {
	name string
	run  func(ctx context.Context) (R, error)
}

// One0 wraps a sqlc method with no parameters that returns one row.
func One0[Q, R any](name string, newQ func(platform.Conn) Q, f func(Q, context.Context) (R, error)) One0Query[R] {
	return One0Query[R]{
		name: name,
		run:  func(ctx context.Context) (R, error) { return run0(ctx, name, newQ, f) },
	}
}

// Run executes the query.
func (d One0Query[R]) Run(ctx context.Context) (R, error) { return d.run(ctx) }

// Name reports the query's registered name.
func (d One0Query[R]) Name() string { return d.name }

// One1Query runs a fixed query with one parameter that returns one row.
type One1Query[P, R any] struct {
	name string
	run  func(ctx context.Context, p P) (R, error)
}

// One1 wraps a sqlc method with one parameter that returns one row.
func One1[Q, P, R any](name string, newQ func(platform.Conn) Q, f func(Q, context.Context, P) (R, error)) One1Query[P, R] {
	return One1Query[P, R]{
		name: name,
		run:  func(ctx context.Context, p P) (R, error) { return run1(ctx, name, newQ, p, f) },
	}
}

// Run executes the query with p.
func (d One1Query[P, R]) Run(ctx context.Context, p P) (R, error) { return d.run(ctx, p) }

// Name reports the query's registered name.
func (d One1Query[P, R]) Name() string { return d.name }

// Many0Query runs a fixed query with no parameters that returns many rows.
type Many0Query[R any] struct {
	name string
	run  func(ctx context.Context) ([]R, error)
}

// Many0 wraps a sqlc method with no parameters that returns many rows.
func Many0[Q, R any](name string, newQ func(platform.Conn) Q, f func(Q, context.Context) ([]R, error)) Many0Query[R] {
	return Many0Query[R]{
		name: name,
		run:  func(ctx context.Context) ([]R, error) { return run0(ctx, name, newQ, f) },
	}
}

// Run executes the query.
func (d Many0Query[R]) Run(ctx context.Context) ([]R, error) { return d.run(ctx) }

// Name reports the query's registered name.
func (d Many0Query[R]) Name() string { return d.name }

// Many1Query runs a fixed query with one parameter that returns many rows.
type Many1Query[P, R any] struct {
	name string
	run  func(ctx context.Context, p P) ([]R, error)
}

// Many1 wraps a sqlc method with one parameter that returns many rows.
func Many1[Q, P, R any](name string, newQ func(platform.Conn) Q, f func(Q, context.Context, P) ([]R, error)) Many1Query[P, R] {
	return Many1Query[P, R]{
		name: name,
		run:  func(ctx context.Context, p P) ([]R, error) { return run1(ctx, name, newQ, p, f) },
	}
}

// Run executes the query with p.
func (d Many1Query[P, R]) Run(ctx context.Context, p P) ([]R, error) { return d.run(ctx, p) }

// Name reports the query's registered name.
func (d Many1Query[P, R]) Name() string { return d.name }

// Exec0Query runs a fixed query with no parameters and no result rows.
type Exec0Query struct {
	name string
	run  func(ctx context.Context) error
}

// Exec0 wraps a sqlc method with no parameters and no result rows, adapting it
// to run0's shape with a throwaway result type.
func Exec0[Q any](name string, newQ func(platform.Conn) Q, f func(Q, context.Context) error) Exec0Query {
	return Exec0Query{
		name: name,
		run: func(ctx context.Context) error {
			_, err := run0(ctx, name, newQ, func(q Q, ctx context.Context) (struct{}, error) {
				return struct{}{}, f(q, ctx)
			})
			return err
		},
	}
}

// Run executes the query.
func (d Exec0Query) Run(ctx context.Context) error { return d.run(ctx) }

// Name reports the query's registered name.
func (d Exec0Query) Name() string { return d.name }

// Exec1Query runs a fixed query with one parameter and no result rows.
type Exec1Query[P any] struct {
	name string
	run  func(ctx context.Context, p P) error
}

// Exec1 wraps a sqlc method with one parameter and no result rows, adapting it
// to run1's shape with a throwaway result type.
func Exec1[Q, P any](name string, newQ func(platform.Conn) Q, f func(Q, context.Context, P) error) Exec1Query[P] {
	return Exec1Query[P]{
		name: name,
		run: func(ctx context.Context, p P) error {
			_, err := run1(ctx, name, newQ, p, func(q Q, ctx context.Context, p P) (struct{}, error) {
				return struct{}{}, f(q, ctx, p)
			})
			return err
		},
	}
}

// Run executes the query with p.
func (d Exec1Query[P]) Run(ctx context.Context, p P) error { return d.run(ctx, p) }

// Name reports the query's registered name.
func (d Exec1Query[P]) Name() string { return d.name }

// ExecRows0Query runs a fixed query with no parameters that reports rows affected.
type ExecRows0Query struct {
	name string
	run  func(ctx context.Context) (int64, error)
}

// ExecRows0 wraps a sqlc method with no parameters that reports rows affected.
func ExecRows0[Q any](name string, newQ func(platform.Conn) Q, f func(Q, context.Context) (int64, error)) ExecRows0Query {
	return ExecRows0Query{
		name: name,
		run:  func(ctx context.Context) (int64, error) { return run0(ctx, name, newQ, f) },
	}
}

// Run executes the query.
func (d ExecRows0Query) Run(ctx context.Context) (int64, error) { return d.run(ctx) }

// Name reports the query's registered name.
func (d ExecRows0Query) Name() string { return d.name }

// ExecRows1Query runs a fixed query with one parameter that reports rows affected.
type ExecRows1Query[P any] struct {
	name string
	run  func(ctx context.Context, p P) (int64, error)
}

// ExecRows1 wraps a sqlc method with one parameter that reports rows affected.
func ExecRows1[Q, P any](name string, newQ func(platform.Conn) Q, f func(Q, context.Context, P) (int64, error)) ExecRows1Query[P] {
	return ExecRows1Query[P]{
		name: name,
		run:  func(ctx context.Context, p P) (int64, error) { return run1(ctx, name, newQ, p, f) },
	}
}

// Run executes the query with p.
func (d ExecRows1Query[P]) Run(ctx context.Context, p P) (int64, error) { return d.run(ctx, p) }

// Name reports the query's registered name.
func (d ExecRows1Query[P]) Name() string { return d.name }
