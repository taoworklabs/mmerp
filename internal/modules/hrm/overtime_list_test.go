package hrm_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// overtimeListFixture adds to listFixture eight overtime requests in every status for
// M, E, E2 and E3, on the same dates so they tie across employees, and one draft of HR2's.
type overtimeListFixture struct {
	*listFixture
	otIDs map[string][]int64 // overtime request ids per employee code
}

func newOvertimeListFixture(t *testing.T) *overtimeListFixture {
	f := &overtimeListFixture{listFixture: newListFixture(t), otIDs: map[string][]int64{}}
	transition := func(id int64, to record.Status) {
		d, err := f.rec.Get(f.admin, record.Ref{Type: "hrm.overtime_request", ID: id})
		f.check(err)
		f.check(errOf(f.rec.Transition(f.hr, record.Ref{Type: "hrm.overtime_request", ID: id}, d.Version, to)))
	}
	var pending []int64
	for _, code := range []string{"M", "E", "E2", "E3"} {
		emp := f.emps[code]
		for j := range 8 {
			date := fmt.Sprintf("2026-03-%02d", 2+j)
			id := must(t)(f.hrm.CreateOvertime(f.hr, hrm.NewOvertime{EmployeeID: &emp, OvertimeFields: overtime(date, "2", "1")}))
			f.otIDs[code] = append(f.otIDs[code], id)
			switch j % 4 {
			case 1:
				transition(id, record.Posted)
			case 2:
				transition(id, record.Posted)
				transition(id, record.Cancelled)
			case 3:
				// M has no manager, so HR would be its only approver.
				if code != "M" {
					pending = append(pending, id)
				}
			}
		}
	}
	hr2 := f.emps["HR2"]
	f.otIDs["HR2"] = []int64{must(t)(f.hrm.CreateOvertime(f.hr, hrm.NewOvertime{EmployeeID: &hr2, OvertimeFields: overtime("2026-03-02", "2", "0")}))}
	f.check(f.appr.SaveRule(f.admin, "hrm.overtime_request", record.Post, approval.RuleInput{
		Flow: approval.Flow{{Step: &approval.StepDef{Approver: approval.Approver{Kind: "module"}}}}, MaxLevels: 3, FallbackProduct: "hrm", FallbackRole: "hr",
	}))
	for _, id := range pending {
		transition(id, record.Posted)
	}
	return f
}

func (f *overtimeListFixture) overtimes(ctx context.Context, fl hrm.OvertimeFilter) ([]int64, int64) {
	f.t.Helper()
	l, err := f.hrm.Overtimes(ctx, fl)
	f.check(err)
	ids := []int64{}
	for _, it := range l.Items {
		ids = append(ids, it.ID)
	}
	return ids, l.Total
}

func TestOvertimeListSearchLiteralPercent(t *testing.T) {
	f := newOvertimeListFixture(t)
	for _, q := range []string{"%", "_", `\`} {
		if _, total := f.overtimes(f.hr, hrm.OvertimeFilter{Q: q, Sort: "-date", Paging: platform.Paging{Page: 1, PageSize: 20}}); total != 0 {
			t.Errorf("q %q matched %d", q, total)
		}
	}
}

// A manager sees their own requests and their reports' at any depth, never a sibling's;
// the org-unit scope adds to that chain.
func TestOvertimeListVisibility(t *testing.T) {
	f := newOvertimeListFixture(t)
	idSet := func(codes ...string) []int64 {
		var ids []int64
		for _, c := range codes {
			ids = append(ids, f.otIDs[c]...)
		}
		slices.Sort(ids)
		return ids
	}
	for _, c := range []struct {
		actor string
		want  []int64
	}{
		{"M", idSet("M", "E", "E2", "E3")},
		{"E", idSet("E", "E3")},
		{"E2", idSet("E2")},
		{"E3", idSet("E3")},
		{"HR_A", idSet("M", "E", "HR2")},
		{"NOBODY", nil},
		{"ADMIN", idSet("M", "E", "E2", "E3", "HR2")},
	} {
		t.Run(c.actor, func(t *testing.T) {
			got, total := f.overtimes(f.actors[c.actor], hrm.OvertimeFilter{Sort: "-date", Paging: platform.Paging{Page: 1, PageSize: 50}})
			slices.Sort(got)
			if !slices.Equal(got, c.want) || total != int64(len(c.want)) {
				t.Fatalf("%s: got %v (total %d), want %v", c.actor, got, total, c.want)
			}
		})
	}
}

// reports=true keeps only the actor's reports, at any depth.
func TestOvertimeListReports(t *testing.T) {
	f := newOvertimeListFixture(t)
	for actor, reports := range map[string][]string{"M": {"E", "E2", "E3"}, "E": {"E3"}, "E2": nil, "HR_A": nil} {
		t.Run(actor, func(t *testing.T) {
			var want []int64
			for _, code := range reports {
				want = append(want, f.otIDs[code]...)
			}
			slices.Sort(want)
			got, total := f.overtimes(f.actors[actor], hrm.OvertimeFilter{Reports: true, Sort: "-date", Paging: platform.Paging{Page: 1, PageSize: 50}})
			slices.Sort(got)
			if !slices.Equal(got, want) || total != int64(len(want)) {
				t.Fatalf("got %v (total %d), want %v", got, total, want)
			}
		})
	}
}
