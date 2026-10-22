package hrm_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// listFixture seeds leave requests in every status for M (unit A), E (A, under M),
// E2 (B, under M) and E3 (B, under E): eight each, on the same
// weekdays, so start dates tie across employees.
type listFixture struct {
	*leaveFixture
	emps map[string]int64
}

func newListFixture(t *testing.T) *listFixture {
	f := &listFixture{leaveFixture: newLeaveFixture(t), emps: map[string]int64{}}
	mgr := f.employeeID("M")
	e2 := employee("E2", f.b)
	e2.ManagerID = &mgr
	e3 := employee("E3", f.b)
	e3.ManagerID = &f.emp
	f.emps["M"], f.emps["E"] = mgr, f.emp
	f.emps["E2"] = must(t)(f.hrm.CreateEmployee(f.hr, e2))
	f.emps["E3"] = must(t)(f.hrm.CreateEmployee(f.hr, e3))
	unpaid := must(t)(f.hrm.SaveLeaveType(f.hr, 0, hrm.LeaveTypeInput{Name: "Không lương", Active: true}))

	var days []string
	for d := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC); len(days) < 16; d = d.AddDate(0, 0, 1) {
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			days = append(days, d.Format(time.DateOnly))
		}
	}
	var pending []int64
	for _, code := range []string{"M", "E", "E2", "E3"} {
		emp := f.emps[code]
		for j := range 8 {
			start, end, n := days[2*j], days[2*j], "1"
			if j%3 == 0 {
				end, n = days[2*j+1], "2"
			}
			id := must(t)(f.hrm.CreateLeave(f.hr, hrm.NewLeave{EmployeeID: &emp,
				LeaveFields: hrm.LeaveFields{LeaveTypeID: unpaid, StartDate: start, EndDate: end, Days: n}}))
			switch j % 4 {
			case 1:
				f.check(f.send(f.hr, id))
			case 2:
				f.check(f.send(f.hr, id))
				f.check(errOf(f.rec.Transition(f.hr, record.Ref{Type: "hrm.leave_request", ID: id}, f.leave(id).Version, record.Cancelled)))
			case 3:
				// M has no manager, so HR, who submits here, would be its only approver.
				if code != "M" {
					pending = append(pending, id)
				}
			}
		}
	}
	f.ruleManager()
	for _, id := range pending {
		f.check(f.send(f.hr, id))
	}
	return f
}

func (f *listFixture) employeeID(code string) int64 {
	f.t.Helper()
	l, err := f.hrm.Employees(f.hr, hrm.EmployeeFilter{Q: code, Sort: "code", Paging: platform.Paging{Page: 1, PageSize: 20}})
	f.check(err)
	for _, it := range l.Items {
		if it.Code == code {
			return it.ID
		}
	}
	f.t.Fatalf("no employee %s", code)
	return 0
}

func (f *listFixture) newLeaves(ctx context.Context, fl hrm.LeaveFilter) ([]int64, int64) {
	f.t.Helper()
	l, err := f.hrm.Leaves(ctx, fl)
	f.check(err)
	ids := []int64{}
	for _, it := range l.Items {
		ids = append(ids, it.ID)
	}
	return ids, l.Total
}

// A "%" in the search matches a literal percent sign, not everything.
func TestLeaveListSearchLiteralPercent(t *testing.T) {
	f := newListFixture(t)
	for _, q := range []string{"%", "_", `\`} {
		l, err := f.hrm.Leaves(f.hr, hrm.LeaveFilter{Q: q, Sort: "-start_date", Paging: platform.Paging{Page: 1, PageSize: 20}})
		f.check(err)
		if l.Total != 0 || len(l.Items) != 0 {
			t.Errorf("q %q matched %d", q, l.Total)
		}
	}
}

// A page past the end has no items but the true total.
func TestLeaveListPagePastEndTotal(t *testing.T) {
	f := newListFixture(t)
	l, err := f.hrm.Leaves(f.hr, hrm.LeaveFilter{Sort: "-start_date", Paging: platform.Paging{Page: 5, PageSize: 20}})
	f.check(err)
	if len(l.Items) != 0 || l.Total != 32 {
		t.Fatalf("page 5: %d items, total %d", len(l.Items), l.Total)
	}
}

// No sort gives the API's default order, newest start first.
func TestLeaveListDefaultSort(t *testing.T) {
	f := newListFixture(t)
	page := platform.Paging{Page: 1, PageSize: 50}
	got, _ := f.newLeaves(f.hr, hrm.LeaveFilter{Paging: page})
	want, _ := f.newLeaves(f.hr, hrm.LeaveFilter{Sort: "-start_date", Paging: page})
	if len(got) != 32 || !slices.Equal(got, want) {
		t.Fatalf("default order %v, want %v", got, want)
	}
}
