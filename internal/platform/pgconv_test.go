package platform_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/platform"
)

func TestPlainDecimal(t *testing.T) {
	for in, want := range map[string]string{"1.0": "1", "2.50": "2.5", "-0.000": "0", "10": "10", "": "", "abc": "abc"} {
		if got := platform.PlainDecimal(in); got != want {
			t.Errorf("PlainDecimal(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestOptional(t *testing.T) {
	for _, c := range []struct {
		in   string
		want *string
	}{{"", nil}, {"x", new("x")}, {" ", new(" ")}} {
		got := platform.Optional(c.in)
		if (got == nil) != (c.want == nil) || got != nil && *got != *c.want {
			t.Errorf("Optional(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestInstant(t *testing.T) {
	hcm := time.FixedZone("ICT", 7*3600)
	for _, c := range []struct {
		in   pgtype.Timestamptz
		want *string
	}{
		{pgtype.Timestamptz{}, nil},
		{pgtype.Timestamptz{Time: time.Date(2026, 3, 1, 8, 30, 0, 0, hcm), Valid: true}, new("2026-03-01T01:30:00Z")},
		{pgtype.Timestamptz{Time: time.Date(2026, 3, 1, 8, 30, 0, 999, time.UTC), Valid: true}, new("2026-03-01T08:30:00Z")},
	} {
		got := platform.Instant(c.in)
		if (got == nil) != (c.want == nil) || got != nil && *got != *c.want {
			t.Errorf("Instant(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestMonthPeriod(t *testing.T) {
	for _, c := range []struct {
		month, start, end string
		ok                bool
	}{
		{"2026-01", "2026-01-01", "2026-01-31", true},
		{"2026-02", "2026-02-01", "2026-02-28", true},
		{"2028-02", "2028-02-01", "2028-02-29", true},
		{"2026-12", "2026-12-01", "2026-12-31", true},
		{"2026-13", "", "", false},
		{"2026-1", "", "", false},
		{"", "", "", false},
	} {
		start, end, err := platform.MonthPeriod(c.month)
		if !c.ok {
			var pe *platform.Error
			if !errors.As(err, &pe) || pe.Status != http.StatusUnprocessableEntity {
				t.Errorf("MonthPeriod(%q) err = %v, want a 422", c.month, err)
			}
			continue
		}
		if err != nil || *platform.DatePtr(start) != c.start || *platform.DatePtr(end) != c.end {
			t.Errorf("MonthPeriod(%q) = %v, %v, %v, want %s..%s", c.month, start, end, err, c.start, c.end)
		}
	}
}
