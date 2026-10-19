package hrm_test

import (
	"fmt"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
)

// Probes for leave situations no rule settles yet: a type that neither deducts nor pays, a
// missing approver, and HR filing for someone without an account. They only observe and log;
// they assert nothing, because the expected result is still being agreed. The NP codes label
// the observations. Leave on non-working days and two requests over the same days were probed
// here too until a rule settled them; leave_overlap_test.go now asserts both.

func (f *leaveFixture) balanceOf(emp int64) string {
	f.t.Helper()
	bs, err := f.hrm.LeaveBalances(f.hr, emp)
	f.check(err)
	for _, b := range bs {
		if b.Year == 2026 {
			return b.Days
		}
	}
	return "0"
}

func (f *leaveFixture) steps(id int64) string {
	f.t.Helper()
	inst, err := f.appr.DocumentInstance(f.admin, record.Ref{Type: "hrm.leave_request", ID: id})
	f.check(err)
	if inst == nil {
		return "(no approval instance)"
	}
	out := ""
	for _, s := range inst.Steps {
		var names []string
		for _, v := range s.Voters {
			names = append(names, v.Name)
		}
		out += fmt.Sprintf("[step %d approvers=%v fallback=%v]", s.Position, names, s.Fallback)
	}
	return fmt.Sprintf("%s %s", inst.Status, out)
}

// NP-10: a type that neither deducts the balance nor is company-paid.
func TestProbeNP10UnpaidBalance(t *testing.T) {
	f := newLeaveFixture(t)
	f.ruleManager()
	f.grant("12")
	unpaid := must(t)(f.hrm.SaveLeaveType(f.hr, 0, hrm.LeaveTypeInput{Name: "Nghỉ không lương", DeductsBalance: false, Paid: false, Active: true}))

	id := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: hrm.LeaveFields{
		LeaveTypeID: unpaid, StartDate: "2026-03-10", EndDate: "2026-03-11", Days: "2"}}))
	f.check(f.send(f.e, id))
	f.check(f.appr.Approve(f.m, f.ticket(id), 1))
	t.Logf("NP-10 unpaid, non-deducting type approved: status %s, balance %s (12 granted)",
		f.leave(id).Status, f.balance())

	// Cancelling it must not hand back days it never took.
	l := f.leave(id)
	_, err := f.rec.Transition(f.hr, record.Ref{Type: "hrm.leave_request", ID: id}, l.Version, record.Cancelled)
	t.Logf("NP-10 cancelling it: %v → balance %s", err, f.balance())
}

// NP-12: the employee has no direct manager, or the manager has no account.
func TestProbeNP12NoManager(t *testing.T) {
	f := newLeaveFixture(t)
	f.ruleManager()

	// X has an account but no manager.
	x := f.ctxFor("x")
	ex := employee("X", f.a)
	ex.UserLogin = str("x")
	xid := must(t)(f.hrm.CreateEmployee(f.hr, ex))
	f.check(f.hrm.AdjustLeaveBalance(f.hr, xid, 2026, hrm.BalanceAdjustment{Delta: "12", Reason: "đầu năm"}))
	idx := must(t)(f.hrm.CreateLeave(x, hrm.NewLeave{LeaveFields: f.fields("2026-03-10", "2026-03-10", "1")}))
	errx := f.send(x, idx)
	t.Logf("NP-12 no manager at all: submit %v → %s", errx, f.steps(idx))

	// Y reports to a manager whose employee record has no account.
	noAccount := must(t)(f.hrm.CreateEmployee(f.hr, employee("NOACC", f.a)))
	y := f.ctxFor("y")
	ey := employee("Y", f.a)
	ey.UserLogin, ey.ManagerID = str("y"), &noAccount
	yid := must(t)(f.hrm.CreateEmployee(f.hr, ey))
	f.check(f.hrm.AdjustLeaveBalance(f.hr, yid, 2026, hrm.BalanceAdjustment{Delta: "12", Reason: "đầu năm"}))
	idy := must(t)(f.hrm.CreateLeave(y, hrm.NewLeave{LeaveFields: f.fields("2026-03-12", "2026-03-12", "1")}))
	erry := f.send(y, idy)
	t.Logf("NP-12 manager without an account: submit %v → %s", erry, f.steps(idy))
}

// NP-15: HR files a request for an employee who has no account.
func TestProbeNP15HRForEmployeeWithoutAccount(t *testing.T) {
	f := newLeaveFixture(t)
	f.ruleManager()

	// A manager with an account, and Z reporting to them with no account of their own.
	f.ctxFor("m2")
	m2 := employee("M2", f.a)
	m2.UserLogin = str("m2")
	mgrID := must(t)(f.hrm.CreateEmployee(f.hr, m2))
	z := employee("Z", f.a)
	z.ManagerID = &mgrID
	zid := must(t)(f.hrm.CreateEmployee(f.hr, z))
	f.check(f.hrm.AdjustLeaveBalance(f.hr, zid, 2026, hrm.BalanceAdjustment{Delta: "12", Reason: "đầu năm"}))

	id := must(t)(f.hrm.CreateLeave(f.hr, hrm.NewLeave{EmployeeID: &zid, LeaveFields: f.fields("2026-03-10", "2026-03-11", "2")}))
	l := f.leave(id)
	t.Logf("NP-15 HR created for Z: employee %d, number %s, status %s", l.EmployeeID, l.Number, l.Status)
	err := f.send(f.hr, id)
	t.Logf("NP-15 HR submits it: %v → %s", err, f.steps(id))
	if err == nil {
		e := f.appr.Approve(f.loginCtx("m2"), f.ticket(id), 1)
		t.Logf("NP-15 the manager approves: %v → status %s, Z's balance %s (12 granted)",
			e, f.leave(id).Status, f.balanceOf(zid))
	}
}
