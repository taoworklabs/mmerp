package data

import (
	"context"
	"slices"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// Scope is where a caller may see rows: everywhere, specific org units (their
// subtrees included), or nowhere. The zero value grants nothing. Only
// iam.(*Service).DataScope builds a non-zero Scope, so a list caller cannot
// widen its own visibility.
type Scope struct {
	all   bool
	units []int64
}

// GrantAll returns a scope with no restriction.
func GrantAll() Scope { return Scope{all: true} }

// GrantUnits returns a scope limited to units. An empty or nil slice equals
// the zero Scope. The slice is copied so the caller cannot mutate it afterward.
func GrantUnits(units []int64) Scope {
	if len(units) == 0 {
		return Scope{}
	}
	return Scope{units: slices.Clone(units)}
}

// any reports whether the scope grants access anywhere.
func (s Scope) any() bool { return s.all || len(s.units) > 0 }

// Viewer is the acting user as a declarative Visibility rule sees it.
type Viewer struct{ Actor int64 }

// viewerFrom builds a Viewer from the request actor; ok is false for an anonymous request.
func viewerFrom(ctx context.Context) (Viewer, bool) {
	id, ok := platform.ActorFrom(ctx)
	return Viewer{Actor: id}, ok
}
