package hrm_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
)

// holiday adds 2026-04-30 to the company's calendar, so the leave tests below have a
// public holiday to run over. 04-04 and 04-05 are a Saturday and a Sunday.
func (f *leaveFixture) holiday() {
	f.t.Helper()
	f.check(f.hrm.SaveHoliday(f.ctxFor("cal", "payroll@*"), f.c, hrm.Holiday{Date: "2026-04-30", Name: "Giải phóng miền Nam"}))
}

// The entered day count may not exceed the working days of the range, and a range with no
// working day in it is refused outright.
func TestLeaveDaysCappedByWorkingDays(t *testing.T) {
	f := newLeaveFixture(t)
	f.holiday()
	f.grant("30")
	cases := []struct {
		name, start, end, days, want string
	}{
		{"the whole range is working days", "2026-04-06", "2026-04-07", "2", ""},
		{"a weekend in the middle does not count", "2026-04-03", "2026-04-06", "2", ""},
		{"half days are allowed up to the cap", "2026-04-03", "2026-04-06", "1.5", ""},
		{"more than the working days of the range", "2026-04-03", "2026-04-06", "4", "leave_days_exceed_working_days"},
		{"one more than the working days", "2026-04-06", "2026-04-07", "2.5", "leave_days_exceed_working_days"},
		{"the weekend only", "2026-04-04", "2026-04-05", "2", "leave_no_working_days"},
		{"the holiday only", "2026-04-30", "2026-04-30", "1", "leave_no_working_days"},
		{"the holiday and the weekend around it do not count", "2026-04-29", "2026-05-04", "6", "leave_days_exceed_working_days"},
		{"the same range, entered as the three working days it has", "2026-04-29", "2026-05-04", "3", ""},
	}
	for _, c := range cases {
		_, err := f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields(c.start, c.end, c.days)})
		if c.want == "" {
			f.check(err)
			continue
		}
		if got := errCode(err); got != c.want {
			t.Errorf("%s (%s→%s, %s days): error %q (%v), want %q", c.name, c.start, c.end, c.days, got, err, c.want)
		}
	}
}

// The cap reports how many days the range actually offers, so the employee can correct it.
func TestLeaveDaysErrorNamesTheWorkingDays(t *testing.T) {
	f := newLeaveFixture(t)
	f.grant("12")
	_, err := f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-04-03", "2026-04-06", "4")})
	if errCode(err) != "leave_days_exceed_working_days" {
		t.Fatalf("error %v", err)
	}
	if p := fmt.Sprint(errParams(err)["working_days"]); p != "2" {
		t.Fatalf("the error reports working_days = %v, want 2", p)
	}
}

// The ceiling the form reads is bounded the way the entry is: a range of centuries would
// otherwise walk every day of it, which is work anyone signed in could ask for repeatedly.
func TestLeaveWorkingDaysRejectsUnboundedRanges(t *testing.T) {
	f := newLeaveFixture(t)
	for _, c := range []struct{ start, end, want string }{
		{"0001-01-01", "9999-12-31", "leave_spans_years"},
		{"2026-01-01", "2027-01-01", "leave_spans_years"},
		{"2026-04-07", "2026-04-06", "leave_end_before_start"},
	} {
		_, err := f.hrm.LeaveWorkingDays(f.e, nil, c.start, c.end)
		if got := errCode(err); got != c.want {
			t.Errorf("%s → %s: error %q, want %q", c.start, c.end, got, c.want)
		}
	}
	// A whole year is the widest range a request can have, and it still answers.
	n, err := f.hrm.LeaveWorkingDays(f.e, nil, "2026-01-01", "2026-12-31")
	f.check(err)
	if n != 261 {
		t.Errorf("working days of 2026 = %d, want 261", n)
	}
}

// Editing a draft is held to the same cap as creating one.
func TestLeaveDaysCapAppliesToEdits(t *testing.T) {
	f := newLeaveFixture(t)
	f.grant("12")
	id := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-04-06", "2026-04-07", "2")}))
	err := f.hrm.UpdateLeave(f.e, id, hrm.LeaveUpdate{Version: f.leave(id).Version, LeaveFields: f.fields("2026-04-04", "2026-04-05", "2")})
	if errCode(err) != "leave_no_working_days" {
		t.Fatalf("editing onto a weekend: %v", err)
	}
}

// A holiday added after the draft was saved must not let a stale day count through approval.
func TestLeaveDaysRecheckedOnApproval(t *testing.T) {
	f := newLeaveFixture(t)
	f.ruleManager()
	f.grant("12")
	id := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-04-29", "2026-04-30", "2")}))
	f.check(f.send(f.e, id))
	f.holiday() // 2026-04-30 becomes a holiday, leaving one working day in the range
	if got := errCode(f.appr.Approve(f.m, f.ticket(id), 1)); got != "leave_days_exceed_working_days" {
		t.Fatalf("approving a request the calendar has overtaken: %q", got)
	}
	if s := f.leave(id).Status; s != "pending_approval" {
		t.Fatalf("status %s, want it left pending", s)
	}
	if f.balance() != "12" {
		t.Fatalf("balance %s, want 12 untouched", f.balance())
	}
}

// Approving a request whose dates already hold a day of approved absence is refused.
func TestLeaveOverlapRefusedOnApproval(t *testing.T) {
	f := newLeaveFixture(t)
	f.ruleManager()
	f.grant("30")
	first := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-04-06", "2026-04-07", "2")}))
	f.check(f.send(f.e, first))
	f.check(f.appr.Approve(f.m, f.ticket(first), 1))

	// 2026-04-07 is in both.
	second := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-04-07", "2026-04-08", "2")}))
	f.check(f.send(f.e, second))
	if got := errCode(f.appr.Approve(f.m, f.ticket(second), 1)); got != "leave_overlaps" {
		t.Fatalf("approving the overlapping request: %q", got)
	}
	if s := f.leave(second).Status; s != "pending_approval" {
		t.Fatalf("status %s, want it left pending", s)
	}
	if f.balance() != "28" {
		t.Fatalf("balance %s, want only the first request deducted", f.balance())
	}

	// A request over days the first one does not touch is fine.
	clear := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-04-08", "2026-04-09", "2")}))
	f.check(f.send(f.e, clear))
	f.check(f.appr.Approve(f.m, f.ticket(clear), 1))
	if f.balance() != "26" {
		t.Fatalf("balance %s, want 26", f.balance())
	}
}

// Half days compose: two halves on one date make a whole day and are allowed; a half plus
// a whole day are not.
func TestLeaveOverlapCountsHalfDays(t *testing.T) {
	f := newLeaveFixture(t)
	f.ruleManager()
	f.grant("30")
	morning := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-04-06", "2026-04-06", "0.5")}))
	f.check(f.send(f.e, morning))
	f.check(f.appr.Approve(f.m, f.ticket(morning), 1))

	afternoon := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-04-06", "2026-04-06", "0.5")}))
	f.check(f.send(f.e, afternoon))
	f.check(f.appr.Approve(f.m, f.ticket(afternoon), 1))
	if f.balance() != "29" {
		t.Fatalf("balance %s, want both halves deducted", f.balance())
	}

	third := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-04-06", "2026-04-06", "0.5")}))
	f.check(f.send(f.e, third))
	if got := errCode(f.appr.Approve(f.m, f.ticket(third), 1)); got != "leave_overlaps" {
		t.Fatalf("a third half day on the same date: %q", got)
	}

	// A whole day on a date already holding half a day is refused too.
	f.check(f.hrm.AdjustLeaveBalance(f.hr, f.emp, 2026, hrm.BalanceAdjustment{Delta: "1", Reason: "test"}))
	half := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-04-08", "2026-04-08", "0.5")}))
	f.check(f.send(f.e, half))
	f.check(f.appr.Approve(f.m, f.ticket(half), 1))
	whole := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-04-08", "2026-04-08", "1")}))
	f.check(f.send(f.e, whole))
	if got := errCode(f.appr.Approve(f.m, f.ticket(whole), 1)); got != "leave_overlaps" {
		t.Fatalf("a whole day over a half day: %q", got)
	}
}

// Cancelling the first request frees its dates again.
func TestLeaveOverlapFreedByCancelling(t *testing.T) {
	f := newLeaveFixture(t)
	f.ruleManager()
	f.grant("30")
	first := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-04-06", "2026-04-07", "2")}))
	f.check(f.send(f.e, first))
	f.check(f.appr.Approve(f.m, f.ticket(first), 1))
	f.check(f.rec.Transition(f.hr, record.Ref{Type: "hrm.leave_request", ID: first}, f.leave(first).Version, record.Cancelled))

	second := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-04-06", "2026-04-07", "2")}))
	f.check(f.send(f.e, second))
	f.check(f.appr.Approve(f.m, f.ticket(second), 1))
	if f.balance() != "28" {
		t.Fatalf("balance %s, want the refund and then the new deduction", f.balance())
	}
}

// Overlap is about absence, not about pay: a type that neither deducts nor is paid still
// occupies its dates.
func TestLeaveOverlapCountsEveryType(t *testing.T) {
	f := newLeaveFixture(t)
	f.ruleManager()
	f.grant("30")
	unpaid := must(t)(f.hrm.SaveLeaveType(f.hr, 0, hrm.LeaveTypeInput{Name: "Nghỉ không lương", DeductsBalance: false, Paid: false, Active: true}))
	first := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: hrm.LeaveFields{
		LeaveTypeID: unpaid, StartDate: "2026-04-06", EndDate: "2026-04-07", Days: "2"}}))
	f.check(f.send(f.e, first))
	f.check(f.appr.Approve(f.m, f.ticket(first), 1))

	second := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-04-07", "2026-04-08", "2")}))
	f.check(f.send(f.e, second))
	if got := errCode(f.appr.Approve(f.m, f.ticket(second), 1)); got != "leave_overlaps" {
		t.Fatalf("annual leave over approved unpaid leave: %q", got)
	}
}

// One employee's dates do not block another's.
func TestLeaveOverlapIsPerEmployee(t *testing.T) {
	f := newLeaveFixture(t)
	f.ruleManager()
	f.grant("30")
	mine := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-04-06", "2026-04-07", "2")}))
	f.check(f.send(f.e, mine))
	f.check(f.appr.Approve(f.m, f.ticket(mine), 1))

	// A colleague reporting to the same manager, so the same approver decides both.
	mine2, err := f.hrm.Employee(f.hr, f.emp)
	f.check(err)
	other := employee("OTHER", f.a)
	other.UserLogin, other.ManagerID = str("other"), mine2.ManagerID
	otherCtx := f.ctxFor("other")
	otherID := must(t)(f.hrm.CreateEmployee(f.hr, other))
	f.check(f.hrm.AdjustLeaveBalance(f.hr, otherID, 2026, hrm.BalanceAdjustment{Delta: "12", Reason: "đầu năm"}))
	theirs := must(t)(f.hrm.CreateLeave(otherCtx, hrm.NewLeave{LeaveFields: f.fields("2026-04-06", "2026-04-07", "2")}))
	f.check(f.send(otherCtx, theirs))
	f.check(f.appr.Approve(f.m, f.ticket(theirs), 1))
}

// Two overlapping requests approved at the same time: the employee lock makes one of them
// lose, so a date never ends up holding two days.
func TestOverlappingRequestsApprovedAtOnce(t *testing.T) {
	f := newLeaveFixture(t)
	f.ruleManager()
	f.grant("30")
	a := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-04-06", "2026-04-07", "2")}))
	b := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-04-07", "2026-04-08", "2")}))
	f.check(f.send(f.e, a))
	f.check(f.send(f.e, b))
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, id := range []int64{a, b} {
		wg.Go(func() { errs[i] = f.appr.Approve(f.m, f.ticket(id), 1) })
	}
	wg.Wait()
	posted, refused := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			posted++
		case errCode(err) == "leave_overlaps":
			refused++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if posted != 1 || refused != 1 {
		t.Fatalf("%d posted and %d refused, want exactly one of each (%v, %v)", posted, refused, errs[0], errs[1])
	}
	if f.balance() != "28" {
		t.Fatalf("balance %s, want only one request deducted", f.balance())
	}
}
