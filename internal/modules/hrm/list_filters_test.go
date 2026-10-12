package hrm_test

import (
	"testing"

	"github.com/taoworklabs/mmerp/internal/modules/hrm"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// Lists of requests search by number or employee and filter by org unit; period documents sort.
func TestListsSearchAndFilter(t *testing.T) {
	f := newLeaveFixture(t)
	page := platform.Paging{Page: 1, PageSize: 20}
	leave := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-03-10", "2026-03-10", "1")}))
	number := f.leave(leave).Number
	for name, c := range map[string]struct {
		filter hrm.LeaveFilter
		want   int64
	}{
		"number":        {hrm.LeaveFilter{Q: number}, 1},
		"employee code": {hrm.LeaveFilter{Q: "e"}, 1},
		"employee name": {hrm.LeaveFilter{Q: "nhân viên e"}, 1},
		"no match":      {hrm.LeaveFilter{Q: "zzz"}, 0},
		"its unit":      {hrm.LeaveFilter{OrgUnitID: f.a}, 1},
		"another unit":  {hrm.LeaveFilter{OrgUnitID: f.b}, 0},
	} {
		c.filter.Paging = page
		if l, err := f.hrm.Leaves(f.hr, c.filter); err != nil || l.Total != c.want {
			t.Fatalf("leaves %s: %d %v", name, l.Total, err)
		}
	}

	must(t)(f.hrm.CreateOvertime(f.hr, hrm.NewOvertime{EmployeeID: &f.emp, OvertimeFields: overtime("2026-03-11", "2", "0")}))
	for name, c := range map[string]struct {
		filter hrm.OvertimeFilter
		want   int64
	}{
		"employee name": {hrm.OvertimeFilter{Q: "Nhân viên E"}, 1},
		"no match":      {hrm.OvertimeFilter{Q: "zzz"}, 0},
		"its unit":      {hrm.OvertimeFilter{OrgUnitID: f.a}, 1},
		"another unit":  {hrm.OvertimeFilter{OrgUnitID: f.b}, 0},
	} {
		c.filter.Paging = page
		if l, err := f.hrm.Overtimes(f.hr, c.filter); err != nil || l.Total != c.want {
			t.Fatalf("overtimes %s: %d %v", name, l.Total, err)
		}
	}

	march := must(t)(f.hrm.CreateTimesheet(f.hr, hrm.NewTimesheet{OrgUnitID: f.a, Month: "2026-03"}))
	april := must(t)(f.hrm.CreateTimesheet(f.hr, hrm.NewTimesheet{OrgUnitID: f.a, Month: "2026-04"}))
	for sort, want := range map[string][]int64{"": {april, march}, "-period": {april, march}, "period": {march, april}} {
		l, err := f.hrm.Timesheets(f.hr, hrm.TimesheetFilter{Sort: sort, Paging: page})
		f.check(err)
		if len(l.Items) != 2 || l.Items[0].ID != want[0] || l.Items[1].ID != want[1] {
			t.Fatalf("timesheets sort %q: %+v", sort, l.Items)
		}
	}
}
