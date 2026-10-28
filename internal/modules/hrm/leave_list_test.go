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
// weekdays, so start dates tie across employees. HR2 (unit B, no reports) belongs
// to the A-scoped HR actor, outside that actor's scoped units, to pin the OR with
// their own manager chain. actors holds one context per case in TestLeaveListVisibility.
type listFixture struct {
	*leaveFixture
	emps     map[string]int64
	leaveIDs map[string][]int64 // leave request ids per employee code
	actors   map[string]context.Context
}

func newListFixture(t *testing.T) *listFixture {
	f := &listFixture{leaveFixture: newLeaveFixture(t), emps: map[string]int64{}, leaveIDs: map[string][]int64{}}
	mgr := f.employeeID("M")
	e2ctx, e3ctx := f.ctxFor("e2"), f.ctxFor("e3")
	hrACtx := f.ctxFor("hr_a", "hr@A")
	e2 := employee("E2", f.b)
	e2.ManagerID, e2.UserLogin = &mgr, str("e2")
	e3 := employee("E3", f.b)
	e3.ManagerID, e3.UserLogin = &f.emp, str("e3")
	hr2 := employee("HR2", f.b)
	hr2.UserLogin = str("hr_a")
	f.emps["M"], f.emps["E"] = mgr, f.emp
	f.emps["E2"] = must(t)(f.hrm.CreateEmployee(f.hr, e2))
	f.emps["E3"] = must(t)(f.hrm.CreateEmployee(f.hr, e3))
	f.emps["HR2"] = must(t)(f.hrm.CreateEmployee(f.hr, hr2))
	f.actors = map[string]context.Context{
		"M": f.m, "E": f.e, "E2": e2ctx, "E3": e3ctx, "HR_A": hrACtx, "NOBODY": f.ctxFor("outsider"), "ADMIN": f.admin,
	}
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
			f.leaveIDs[code] = append(f.leaveIDs[code], id)
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
	// One request of HR2's own, outside the A-scoped HR's unit: visible only via the chain, not the scope.
	hr2Emp := f.emps["HR2"]
	hr2Leave := must(t)(f.hrm.CreateLeave(f.hr, hrm.NewLeave{EmployeeID: &hr2Emp,
		LeaveFields: hrm.LeaveFields{LeaveTypeID: unpaid, StartDate: days[0], EndDate: days[0], Days: "1"}}))
	f.leaveIDs["HR2"] = append(f.leaveIDs["HR2"], hr2Leave)
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
	if len(l.Items) != 0 || l.Total != 33 {
		t.Fatalf("page 5: %d items, total %d", len(l.Items), l.Total)
	}
}

// No sort gives the API's default order, newest start first.
func TestLeaveListDefaultSort(t *testing.T) {
	f := newListFixture(t)
	page := platform.Paging{Page: 1, PageSize: 50}
	got, _ := f.newLeaves(f.hr, hrm.LeaveFilter{Paging: page})
	want, _ := f.newLeaves(f.hr, hrm.LeaveFilter{Sort: "-start_date", Paging: page})
	if len(got) != 33 || !slices.Equal(got, want) {
		t.Fatalf("default order %v, want %v", got, want)
	}
}

// TestLeaveListVisibility pins who sees which leave requests: an actor's own chain
// (their employee plus every report below it, at any depth) OR'd with their
// org-unit scope — never a sibling's, and never the whole company without a
// tenant-wide grant.
func TestLeaveListVisibility(t *testing.T) {
	f := newListFixture(t)
	idSet := func(codes ...string) []int64 {
		var ids []int64
		for _, c := range codes {
			ids = append(ids, f.leaveIDs[c]...)
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
			got, total := f.newLeaves(f.actors[c.actor], hrm.LeaveFilter{Sort: "-start_date", Paging: platform.Paging{Page: 1, PageSize: 50}})
			slices.Sort(got)
			if !slices.Equal(got, c.want) || total != int64(len(c.want)) {
				t.Fatalf("%s: got %v (total %d), want %v", c.actor, got, total, c.want)
			}
		})
	}
}

// reports=true keeps only the actor's reports, at any depth: never their own requests,
// nor those their unit scope adds; has_reports says whether there are any.
func TestLeaveListReports(t *testing.T) {
	f := newListFixture(t)
	for _, c := range []struct {
		actor   string
		reports []string
	}{
		{"M", []string{"E", "E2", "E3"}},
		{"E", []string{"E3"}},
		{"E2", nil},
		{"HR_A", nil},
		{"ADMIN", nil},
	} {
		t.Run(c.actor, func(t *testing.T) {
			var want []int64
			for _, code := range c.reports {
				want = append(want, f.leaveIDs[code]...)
			}
			slices.Sort(want)
			got, total := f.newLeaves(f.actors[c.actor], hrm.LeaveFilter{Reports: true, Sort: "-start_date", Paging: platform.Paging{Page: 1, PageSize: 50}})
			slices.Sort(got)
			if !slices.Equal(got, want) || total != int64(len(want)) {
				t.Fatalf("got %v (total %d), want %v", got, total, want)
			}
			acts, err := f.hrm.LeaveActions(f.actors[c.actor])
			f.check(err)
			if acts.HasReports != (len(c.reports) > 0) {
				t.Fatalf("has_reports %v", acts.HasReports)
			}
		})
	}
}

// Several statuses filter as a union; none filters nothing.
func TestLeaveListSeveralStatuses(t *testing.T) {
	f := newListFixture(t)
	paging := platform.Paging{Page: 1, PageSize: 100}
	list := func(status ...string) ([]int64, int64) {
		return f.newLeaves(f.hr, hrm.LeaveFilter{Status: status, Sort: "-start_date", Paging: paging})
	}
	_, all := list()
	_, draft := list("draft")
	_, pending := list("pending_approval")
	_, posted := list("posted")
	_, cancelled := list("cancelled")
	if draft == 0 || pending == 0 || posted+cancelled == 0 || draft+pending+posted+cancelled != all {
		t.Fatalf("single statuses %d+%d+%d+%d vs all %d", draft, pending, posted, cancelled, all)
	}
	ids, total := list("pending_approval", "draft")
	if total != draft+pending || int64(len(ids)) != total {
		t.Fatalf("union: %d ids, total %d, want %d", len(ids), total, draft+pending)
	}
}
