package record_test

import (
	"context"
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
)

func TestAllowedActionsAppliesProductGate(t *testing.T) {
	s := record.NewService(record.Deps{})
	s.Register(record.Type{Code: "hrm.employee", Product: "hrm", Kind: record.Catalog,
		Can: func(context.Context, int64, record.Action) (bool, error) { return true, nil }})
	all := []record.Action{record.View, record.Edit, record.Export}

	got, err := s.AllowedActions(platform.WithProducts(t.Context(), []string{"hrm"}), "hrm.employee", 1, all...)
	if err != nil || !slices.Equal(got, []string{"view", "edit", "export"}) {
		t.Fatalf("enabled: %v %v", got, err)
	}
	got, _ = s.AllowedActions(t.Context(), "hrm.employee", 1, all...)
	if !slices.Equal(got, []string{"view", "export"}) {
		t.Fatalf("disabled product keeps only read and export: %v", got)
	}
}

func TestParseNumberRefusesExponents(t *testing.T) {
	for _, s := range []string{"1e999999999", "1E5", "", "1.", ".5", "1234567890123456"} {
		if _, ok := record.ParseNumber(s); ok {
			t.Errorf("%q accepted", s)
		}
	}
	if d, ok := record.ParseNumber("-2.5"); !ok || d.String() != "-2.5" {
		t.Errorf("-2.5: %v %v", d, ok)
	}
}
