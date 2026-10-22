// Package data normalizes database errors into the engine's coded API errors.
package data

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// Class groups a database error by what caused it, read off the SQLSTATE, so
// callers can react without matching codes themselves.
type Class int

const (
	None Class = iota
	Unique
	ForeignKey
	Check
	NotNull
	Serialization
	Deadlock
	LockTimeout
	Canceled
	ReadOnly
)

var classBySQLState = map[string]Class{
	"23505": Unique,
	"23503": ForeignKey,
	"23514": Check,
	"23502": NotNull,
	"40001": Serialization,
	"40P01": Deadlock,
	"55P03": LockTimeout,
	"57014": Canceled,
	"25006": ReadOnly,
}

// Classify reports the Class of err, or None if it did not come from Postgres
// or carries a SQLSTATE this package does not group.
func Classify(err error) Class {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	if !ok {
		return None
	}
	return classBySQLState[pgErr.Code]
}

// IsRetryable reports whether err is a transient conflict (serialization or
// deadlock) that a caller may retry the whole transaction for. There is no
// automatic retry.
func IsRetryable(err error) bool {
	switch Classify(err) {
	case Serialization, Deadlock:
		return true
	default:
		return false
	}
}

// Constraints maps a database constraint, keyed "schema.constraint", to the
// *platform.Error it means everywhere it fires. Only constraints with one
// meaning everywhere belong here; context-dependent ones stay unmapped and
// their call sites keep platform.Violates.
type Constraints map[string]*platform.Error

var (
	registryMu sync.Mutex
	registry   = Constraints{}
)

// RegisterConstraints adds c to the registry Normalize uses. Modules call it at
// startup; re-registering the same key with an equal error is a no-op (test
// binaries build services more than once), but registering a different error
// for a key already taken is a wiring bug and panics.
func RegisterConstraints(c Constraints) {
	registryMu.Lock()
	defer registryMu.Unlock()
	for key, e := range c {
		if existing, ok := registry[key]; ok {
			if !reflect.DeepEqual(existing, e) {
				panic(fmt.Sprintf("data: constraint %q already registered to a different error", key))
			}
			continue
		}
		registry[key] = e
	}
}

// Normalize maps err through the constraint registry. A mapped constraint
// becomes a value wrapping both the *platform.Error and the underlying
// *pgconn.PgError, so both platform.Violates and errors.AsType still match.
// Anything else — an unmapped violation, pgx.ErrNoRows, a non-database error —
// is returned unchanged.
func Normalize(err error) error {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	if !ok {
		return err
	}
	registryMu.Lock()
	mapped, ok := registry[pgErr.SchemaName+"."+pgErr.ConstraintName]
	registryMu.Unlock()
	if !ok {
		return err
	}
	return &mappedError{apiErr: mapped, pgErr: pgErr}
}

// mappedError is what Normalize returns for a mapped constraint.
type mappedError struct {
	apiErr *platform.Error
	pgErr  *pgconn.PgError
}

func (e *mappedError) Error() string   { return e.apiErr.Error() }
func (e *mappedError) Unwrap() []error { return []error{e.apiErr, e.pgErr} }
