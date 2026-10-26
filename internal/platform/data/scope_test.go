package data

import (
	"reflect"
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// Internal test: Scope's fields are unexported, so only code in this package
// can build or inspect one outside GrantAll/GrantUnits.
func TestZeroScopeGrantsNothing(t *testing.T) {
	var s Scope
	if s.any() {
		t.Fatalf("zero Scope grants access: %+v", s)
	}
	if GrantAll().any() == false {
		t.Fatal("GrantAll grants nothing")
	}
	if GrantUnits([]int64{1}).any() == false {
		t.Fatal("GrantUnits grants nothing")
	}
	if !reflect.DeepEqual(GrantUnits(nil), Scope{}) {
		t.Fatal("GrantUnits(nil) should equal the zero Scope")
	}
	if !reflect.DeepEqual(GrantUnits([]int64{}), Scope{}) {
		t.Fatal("GrantUnits([]) should equal the zero Scope")
	}
}

// GrantUnits must copy its slice: mutating the caller's backing array afterward
// must not change the scope it already built.
func TestGrantUnitsCopiesSlice(t *testing.T) {
	units := []int64{1, 2}
	s := GrantUnits(units)
	units[0] = 999
	if !slices.Equal(s.units, []int64{1, 2}) {
		t.Fatalf("GrantUnits kept a reference to the caller's slice: %v", s.units)
	}
}

func TestViewerFromActor(t *testing.T) {
	if _, ok := viewerFrom(t.Context()); ok {
		t.Fatal("anonymous context reported a viewer")
	}
	v, ok := viewerFrom(platform.WithActor(t.Context(), 7))
	if !ok || v.Actor != 7 {
		t.Fatalf("viewerFrom = %+v, %v, want {Actor:7}, true", v, ok)
	}
}
