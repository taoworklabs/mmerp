package data

import "context"

// Base is a list query's fixed SQL and output columns, checked against its
// sqlc method so the row type follows the query.
type Base[R any] struct {
	name, sql string
	cols      []string
}

// NewBase records a base query; the method is taken only to infer R.
func NewBase[Q, R any](name, sql string, cols []string, _ func(Q, context.Context) ([]R, error)) Base[R] {
	return Base[R]{name: name, sql: sql, cols: cols}
}
