package hrm

import (
	"errors"
	"testing"
)

// A request's days are the working days of its range, less half a day per half flag; the
// halves fall on the first and last working dates, whatever the range starts or ends on.
func TestComputedDays(t *testing.T) {
	c := allocCalendar()
	cases := []struct {
		name, start, end   string
		startHalf, endHalf bool
		want               string
		alloc              map[string]string
	}{
		{"a working week", "2026-04-06", "2026-04-10", false, false, "5", nil},
		{"afternoon of the first day", "2026-04-06", "2026-04-10", true, false, "4.5",
			map[string]string{"2026-04-06": "0.5", "2026-04-07": "1", "2026-04-08": "1", "2026-04-09": "1", "2026-04-10": "1"}},
		{"both halves", "2026-04-06", "2026-04-10", true, true, "4", nil},
		{"the half skips the weekend", "2026-04-04", "2026-04-07", true, false, "1.5",
			map[string]string{"2026-04-06": "0.5", "2026-04-07": "1"}},
		{"morning of the last day", "2026-04-06", "2026-04-07", false, true, "1.5",
			map[string]string{"2026-04-06": "1", "2026-04-07": "0.5"}},
		{"a half day", "2026-04-06", "2026-04-06", true, false, "0.5", map[string]string{"2026-04-06": "0.5"}},
	}
	for _, tc := range cases {
		got, err := computedDays(c, tc.start, tc.end, tc.startHalf, tc.endHalf)
		if err != nil || got.String() != tc.want {
			t.Errorf("%s: got %s, %v; want %s", tc.name, got, err, tc.want)
			continue
		}
		if tc.alloc == nil {
			continue
		}
		a := halfAllocation(c, tc.start, tc.end, got, tc.startHalf)
		if len(a) != len(tc.alloc) {
			t.Errorf("%s: allocated %s", tc.name, days(a))
			continue
		}
		for d, want := range tc.alloc {
			if !a[d].Equal(dec(want)) {
				t.Errorf("%s: on %s got %s, want %s", tc.name, d, a[d], want)
			}
		}
	}
	if _, err := computedDays(c, "2026-04-06", "2026-04-06", true, true); !errors.Is(err, ErrLeaveHalfDays) {
		t.Errorf("both halves of one date: %v", err)
	}
	if _, err := computedDays(c, "2026-04-04", "2026-04-05", false, false); !errors.Is(err, ErrLeaveNoWorkingDays) {
		t.Errorf("a weekend: %v", err)
	}
}
