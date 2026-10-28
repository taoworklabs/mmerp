package hrm_test

import (
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
)

// The count comes from the calendar; a client that sends one must send the same.
func TestLeaveDaysComputedOnSave(t *testing.T) {
	f := newLeaveFixture(t)
	in := f.fields("2026-04-06", "2026-04-10", "")
	in.EndHalf = true
	id := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: in}))
	if l := f.leave(id); l.Days != "4.5" || !l.EndHalf || l.StartHalf {
		t.Fatalf("saved %s days, halves %v %v", l.Days, l.StartHalf, l.EndHalf)
	}
	_, err := f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-04-06", "2026-04-10", "3")})
	wantErr(t, err, "leave_days_changed")
	if p := errParams(err)["days"]; p != "5" {
		t.Fatalf("the error reports %v days, want 5", p)
	}
	one := f.fields("2026-04-06", "2026-04-06", "")
	one.StartHalf, one.EndHalf = true, true
	_, err = f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: one})
	wantErr(t, err, "leave_half_days")
}

// A holiday added while the request waited changes its count: approval refuses rather than
// approve a figure the requester never saw.
func TestApprovalRefusesChangedCount(t *testing.T) {
	f := newLeaveFixture(t)
	f.ruleManager()
	f.grant("12")
	id := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-04-27", "2026-05-01", "")}))
	f.check(f.send(f.e, id))
	f.holiday()
	wantErr(t, f.appr.Approve(f.m, f.ticket(id), 1), "leave_days_changed")
	if f.balance() != "12" {
		t.Fatalf("balance %s", f.balance())
	}
}

// A request saved before counts were computed keeps its typed count and the calendar cap.
func TestLegacyRequestApprovedUnderOldRules(t *testing.T) {
	f := newLeaveFixture(t)
	f.grant("12")
	id := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-04-06", "2026-04-08", "")}))
	if _, err := f.system.Exec(f.admin, `UPDATE hrm.leave_requests SET days = 1, computed = false WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	f.check(f.send(f.e, id))
	if f.leave(id).Status != string(record.Posted) || f.balance() != "11" {
		t.Fatalf("status %s, balance %s", f.leave(id).Status, f.balance())
	}
}
