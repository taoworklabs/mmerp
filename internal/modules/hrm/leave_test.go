package hrm_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/record/recordtest"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

// leaveFixture adds to company C, department A: HR (hr and leave_admin tenant-wide),
// manager M and employee E reporting to M, both with accounts, and annual leave.
type leaveFixture struct {
	*fixture
	hr, m, e context.Context
	emp      int64 // E's employee id
	annual   int64
}

func newLeaveFixture(t *testing.T) *leaveFixture {
	f := &leaveFixture{fixture: newFixture(t)}
	f.hr = f.ctxFor("hr", "hr@*", "leave_admin@*")
	f.m, f.e = f.ctxFor("m"), f.ctxFor("e")
	mgr := employee("M", f.a)
	mgr.UserLogin = str("m")
	mgrID := must(t)(f.hrm.CreateEmployee(f.hr, mgr))
	e := employee("E", f.a)
	e.UserLogin, e.ManagerID = str("e"), &mgrID
	f.emp = must(t)(f.hrm.CreateEmployee(f.hr, e))
	f.annual = must(t)(f.hrm.SaveLeaveType(f.hr, 0, hrm.LeaveTypeInput{Name: "Phép năm", DeductsBalance: true, Active: true}))
	// The payroll lock row exists in any running company. Created on first use instead, its
	// insert would serialize the first two transactions of every concurrency test by itself.
	if _, err := f.pool.Exec(f.admin, `INSERT INTO hrm.payroll_locks (legal_entity_id) VALUES ($1)`, f.c); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *leaveFixture) fields(start, end, days string) hrm.LeaveFields {
	return hrm.LeaveFields{LeaveTypeID: f.annual, StartDate: start, EndDate: end, Days: days}
}

func (f *leaveFixture) grant(days string) {
	f.t.Helper()
	f.check(f.hrm.AdjustLeaveBalance(f.hr, f.emp, 2026, hrm.BalanceAdjustment{Delta: days, Reason: "đầu năm"}))
}

func (f *leaveFixture) balance() string {
	f.t.Helper()
	bs, err := f.hrm.LeaveBalances(f.hr, f.emp)
	f.check(err)
	for _, b := range bs {
		if b.Year == 2026 {
			return b.Days
		}
	}
	return "0"
}

func (f *leaveFixture) leave(id int64) hrm.Leave {
	f.t.Helper()
	l, err := f.hrm.Leave(f.hr, id)
	f.check(err)
	return l
}

func (f *leaveFixture) check(err error) {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
}

func (f *leaveFixture) send(ctx context.Context, id int64) error {
	return f.rec.Transition(ctx, record.Ref{Type: "hrm.leave_request", ID: id}, f.leave(id).Version, record.Posted)
}

// ruleManager makes every leave request need the direct manager's approval.
func (f *leaveFixture) ruleManager() {
	f.t.Helper()
	f.check(f.appr.SaveRule(f.admin, "hrm.leave_request", approval.RuleInput{
		Steps: []approval.Step{{Approver: approval.Approver{Kind: "module"}}}, MaxLevels: 3, FallbackProduct: "hrm", FallbackRole: "hr",
	}))
}

func errCode(err error) string {
	if e, ok := errors.AsType[*platform.Error](err); ok {
		return e.Code
	}
	return ""
}

func wantErr(t *testing.T, err error, code string) {
	t.Helper()
	if errCode(err) != code {
		t.Fatalf("got %v, want %s", err, code)
	}
}

// TestLeaveSampleFlow walks steps 1–7 and 12 of the sample leave flow.
func TestLeaveSampleFlow(t *testing.T) {
	f := newLeaveFixture(t)
	// Two approvers for the department: the head and the deputy.
	f.ctxFor("head", "hr@A")
	f.ctxFor("deputy", "hr@A")
	f.check(f.appr.SaveRule(f.admin, "hrm.leave_request", approval.RuleInput{
		Steps: []approval.Step{{Approver: approval.Approver{Kind: "role", Product: "hrm", Role: "hr"}}}, MaxLevels: 3, FallbackProduct: "hrm", FallbackRole: "hr",
	}))
	head, deputy := f.loginCtx("head"), f.loginCtx("deputy")
	f.grant("12")

	// 1. E creates a 2-day request from 10/03.
	id := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-03-10", "2026-03-11", "2")}))
	if l := f.leave(id); l.Status != "draft" || l.Version != 1 || l.Number != "NP-2026-00001" || l.EmployeeID != f.emp {
		t.Fatalf("step 1: %+v", l)
	}
	// 2. Two tabs of E save version 1 at once: one waits for the row lock, then sees version 2.
	var wg sync.WaitGroup
	edits := make([]error, 2)
	for i := range edits {
		wg.Go(func() {
			edits[i] = f.hrm.UpdateLeave(f.e, id, hrm.LeaveUpdate{Version: 1, LeaveFields: f.fields("2026-03-10", "2026-03-12", "3")})
		})
	}
	wg.Wait()
	if (edits[0] == nil) == (edits[1] == nil) || errCode(edits[0])+errCode(edits[1]) != "version_conflict" {
		t.Fatalf("step 2: want one save and one version_conflict: %v, %v", edits[0], edits[1])
	}
	// 3. E sends it: pending, no more edits.
	f.check(f.send(f.e, id))
	i1 := f.leave(id)
	if i1.Status != "pending_approval" || i1.Version != 2 {
		t.Fatalf("step 3: %+v", i1)
	}
	wantErr(t, f.hrm.UpdateLeave(f.e, id, hrm.LeaveUpdate{Version: 2, LeaveFields: f.fields("2026-03-10", "2026-03-11", "2")}), "document_not_editable")
	ticket1 := f.ticket(id)
	// 4. E withdraws, edits to 5 days, sends again.
	f.check(f.rec.Transition(f.e, record.Ref{Type: "hrm.leave_request", ID: id}, 2, record.Draft))
	f.check(f.hrm.UpdateLeave(f.e, id, hrm.LeaveUpdate{Version: 3, LeaveFields: f.fields("2026-03-10", "2026-03-16", "5")}))
	f.check(f.send(f.e, id))
	if l := f.leave(id); l.Version != 4 {
		t.Fatalf("step 4: %+v", l)
	}
	ticket2 := f.ticket(id)
	// 5. The head approves on a stale screen of the first submission.
	wantErr(t, f.appr.Approve(head, ticket1, 1), "approval_closed")
	// 6. Head and deputy approve the second submission at once: deducted once.
	errs := make([]error, 2)
	for i, ctx := range []context.Context{head, deputy} {
		wg.Go(func() { errs[i] = f.appr.Approve(ctx, ticket2, 1) })
	}
	wg.Wait()
	if (errs[0] == nil) == (errs[1] == nil) {
		t.Fatalf("step 6: want one success: %v, %v", errs[0], errs[1])
	}
	if f.leave(id).Status != "posted" || f.balance() != "7" {
		t.Fatalf("step 6: %+v, balance %s", f.leave(id), f.balance())
	}
	// 7. An 8-day request cannot be approved with 7 days left; it stays pending.
	long := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-04-01", "2026-04-10", "8")}))
	f.check(f.send(f.e, long))
	wantErr(t, f.appr.Approve(head, f.ticket(long), 1), "insufficient_leave_balance")
	if f.leave(long).Status != "pending_approval" || f.balance() != "7" {
		t.Fatalf("step 7: %+v", f.leave(long))
	}

	// 12. Locking March while HR cancels the approved March request.
	ref := record.Ref{Type: "hrm.leave_request", ID: id}
	var cancelled error
	// The lock wins: the cancel waits, then reads the new lock date.
	f.check(platform.InTx(f.admin, func(ctx context.Context) error {
		f.check(f.rec.SetPeriodLock(ctx, f.c, str("2026-03-31")))
		wg.Go(func() { cancelled = f.rec.Transition(f.hr, ref, 5, record.Cancelled) })
		pgtest.AwaitLockWait(t, f.pool)
		return nil
	}))
	wg.Wait()
	wantErr(t, cancelled, "period_locked")
	f.check(f.rec.SetPeriodLock(f.admin, f.c, nil))
	// The cancel wins: the lock waits for it and then succeeds.
	var locked error
	f.check(platform.InTx(f.hr, func(ctx context.Context) error {
		f.check(f.rec.Transition(ctx, ref, 5, record.Cancelled))
		wg.Go(func() { locked = f.rec.SetPeriodLock(f.admin, f.c, str("2026-03-31")) })
		pgtest.AwaitLockWait(t, f.pool)
		return nil
	}))
	wg.Wait()
	f.check(locked)
	if f.leave(id).Status != "cancelled" || f.balance() != "12" {
		t.Fatalf("step 12: %+v, balance %s", f.leave(id), f.balance())
	}
}

func (f *leaveFixture) ticket(id int64) int64 {
	f.t.Helper()
	d, err := f.rec.Get(f.admin, record.Ref{Type: "hrm.leave_request", ID: id})
	f.check(err)
	return *d.ApprovalTicket
}

// loginCtx acts as a user already created by ctxFor.
func (f *leaveFixture) loginCtx(login string) context.Context {
	id, err := f.iam.UserIDByLogin(f.admin, login)
	f.check(err)
	return platform.WithActor(f.admin, id)
}

// Step 12' with a leave request: a pending March request blocks locking March until withdrawn.
func TestPendingLeaveBlocksPeriodLock(t *testing.T) {
	f := newLeaveFixture(t)
	f.ruleManager()
	f.grant("12")
	id := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-03-10", "2026-03-10", "1")}))
	f.check(f.send(f.e, id))
	err := f.rec.SetPeriodLock(f.admin, f.c, str("2026-03-31"))
	wantErr(t, err, "period_has_pending_documents")
	docs := err.(*platform.Error).Params["documents"].([]map[string]any)
	if len(docs) != 1 || docs[0]["id"] != id {
		t.Fatalf("listed %v", docs)
	}
	f.check(f.rec.Transition(f.e, record.Ref{Type: "hrm.leave_request", ID: id}, f.leave(id).Version, record.Draft))
	f.check(f.rec.SetPeriodLock(f.admin, f.c, str("2026-03-31")))
}

// Adjusting the balance while a request is approved never loses an adjustment nor goes negative.
func TestBalanceAdjustmentsRaceApproval(t *testing.T) {
	f := newLeaveFixture(t)
	f.ruleManager()
	f.grant("2")
	id := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-03-10", "2026-03-11", "2")}))
	f.check(f.send(f.e, id))
	ticket := f.ticket(id)
	var wg sync.WaitGroup
	errs := make([]error, 6)
	wg.Go(func() { errs[0] = f.appr.Approve(f.m, ticket, 1) })
	for i := 1; i < len(errs); i++ {
		wg.Go(func() {
			errs[i] = f.hrm.AdjustLeaveBalance(f.hr, f.emp, 2026, hrm.BalanceAdjustment{Delta: "1", Reason: "bù"})
		})
	}
	wg.Wait()
	for _, err := range errs {
		f.check(err)
	}
	if f.balance() != "5" {
		t.Fatalf("balance %s, want 2 + 5 - 2", f.balance())
	}
	// Taking away more than is left is refused.
	wantErr(t, f.hrm.AdjustLeaveBalance(f.hr, f.emp, 2026, hrm.BalanceAdjustment{Delta: "-5.5", Reason: "sai"}), "leave_balance_negative")
}

func TestLeavePermissions(t *testing.T) {
	f := newLeaveFixture(t)
	f.ruleManager()
	f.grant("12")
	other := f.ctxFor("other")
	id := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-03-10", "2026-03-10", "1")}))

	if l, err := f.hrm.Leave(f.e, id); err != nil || l.Balance == nil || *l.Balance != "12" {
		t.Fatalf("own request: %+v %v", l, err)
	}
	if l, err := f.hrm.Leave(f.m, id); err != nil || l.Balance != nil || len(l.AllowedActions) != 0 {
		t.Fatalf("manager sees it read-only, without the balance: %+v %v", l, err)
	}
	if _, err := f.hrm.Leave(other, id); errCode(err) != "not_found" {
		t.Fatalf("other: %v", err)
	}
	if _, err := f.hrm.CreateLeave(other, hrm.NewLeave{LeaveFields: f.fields("2026-03-10", "2026-03-10", "1")}); errCode(err) != "not_an_employee" {
		t.Fatalf("other creates: %v", err)
	}
	if _, err := f.hrm.CreateLeave(f.m, hrm.NewLeave{EmployeeID: &f.emp, LeaveFields: f.fields("2026-03-10", "2026-03-10", "1")}); errCode(err) != "not_found" {
		t.Fatalf("manager creates for E: %v", err)
	}
	for login, want := range map[string]int{"e": 1, "m": 1, "hr": 1, "other": 0} {
		l, err := f.hrm.Leaves(f.loginCtx(login), hrm.LeaveFilter{Sort: "-start_date", Paging: platform.Paging{Page: 1, PageSize: 20}})
		if err != nil || l.Total != int64(want) {
			t.Fatalf("%s lists %d: %v", login, l.Total, err)
		}
	}
	for name, c := range map[string]struct {
		ctx  context.Context
		want []string
	}{
		"e": {f.e, []string{"create"}}, "m": {f.m, []string{"create"}}, "hr": {f.hr, []string{"create", "create_for_others"}},
		"other": {other, nil}, "hr, product off": {platform.WithProducts(f.hr, nil), nil},
	} {
		if a, err := f.hrm.LeaveActions(c.ctx); err != nil || !equal(a.AllowedActions, c.want...) {
			t.Fatalf("%s before a request: %v %v", name, a.AllowedActions, err)
		}
	}
	if acts := f.leave(id).AllowedActions; !equal(acts, "edit", "delete", "submit") {
		t.Fatalf("draft actions %v", acts)
	}
	f.check(f.send(f.e, id))
	if l, _ := f.hrm.Leave(f.e, id); !equal(l.AllowedActions, "withdraw") {
		t.Fatalf("pending actions %v", l.AllowedActions)
	}
	// Only the sender withdraws, not HR.
	if acts := f.leave(id).AllowedActions; len(acts) != 0 {
		t.Fatalf("HR pending actions %v", acts)
	}
	wantErr(t, f.rec.Transition(f.hr, record.Ref{Type: "hrm.leave_request", ID: id}, f.leave(id).Version, record.Draft), "forbidden")
	f.check(f.appr.Approve(f.m, f.ticket(id), 1))
	if l, _ := f.hrm.Leave(f.e, id); len(l.AllowedActions) != 0 {
		t.Fatalf("E cannot cancel an approved request: %v", l.AllowedActions)
	}
	if acts := f.leave(id).AllowedActions; !equal(acts, "cancel") {
		t.Fatalf("HR posted actions %v", acts)
	}
}

func TestLeaveDaysAndDates(t *testing.T) {
	f := newLeaveFixture(t)
	for _, c := range []struct{ start, end, days, code string }{
		{"2026-03-10", "2026-03-09", "1", "leave_end_before_start"},
		{"2026-12-31", "2027-01-01", "2", "leave_spans_years"},
		{"2026-03-10", "2026-03-11", "2.5", "leave_days_exceed_working_days"},
		{"2026-03-10", "2026-03-11", "1.3", "invalid_leave_days"},
		{"2026-03-10", "2026-03-11", "0", "invalid_leave_days"},
	} {
		_, err := f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields(c.start, c.end, c.days)})
		wantErr(t, err, c.code)
	}
	must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-03-10", "2026-03-11", "1.5")}))
}

func TestLeaveRecordContract(t *testing.T) {
	recordtest.Run(t, func(t *testing.T) recordtest.Harness {
		f := newLeaveFixture(t)
		// The contract suite picks its own dates and is about the lifecycle, not the
		// calendar, so no day of the week is off here and every date can hold leave.
		f.check(f.hrm.SaveWorkWeek(f.ctxFor("cal", "payroll@*"), f.c, hrm.WorkWeek{EffectiveFrom: "2020-01-01", OffDays: []int{}}))
		f.grant("10")
		granted := int64(10) // posting a 1-day request takes one day from here
		ref := func(id int64, err error) (record.Ref, error) {
			return record.Ref{Type: "hrm.leave_request", ID: id}, err
		}
		return recordtest.Harness{
			Core: f.core, Actor: f.hr, Admin: f.admin, LegalEntity: f.c,
			Create: func(ctx context.Context, date string) (record.Ref, error) {
				return ref(f.hrm.CreateLeave(ctx, hrm.NewLeave{EmployeeID: &f.emp, LeaveFields: f.fields(date, date, "1")}))
			},
			Edit: func(ctx context.Context, r record.Ref, version int32, date string) error {
				return f.hrm.UpdateLeave(ctx, r.ID, hrm.LeaveUpdate{Version: version, LeaveFields: f.fields(date, date, "1")})
			},
			Get: func(ctx context.Context, r record.Ref) (any, error) { return f.hrm.Leave(ctx, r.ID) },
			Delete: func(ctx context.Context, r record.Ref, version int32) error {
				return f.hrm.DeleteLeave(ctx, r.ID, version)
			},
			RequireApproval: func(*testing.T) { f.ruleManager() },
			Approve:         func(instance int64) error { return f.appr.Approve(f.m, instance, 1) },
			Reject:          func(instance int64) error { return f.appr.Reject(f.m, instance, 1, "không") },
			// The leave type comes to need a supporting file.
			BreakSubmit: func(*testing.T, record.Ref) {
				must(t)(f.hrm.SaveLeaveType(f.hr, f.annual, hrm.LeaveTypeInput{Name: "Phép năm", DeductsBalance: true, Active: true, RequiresAttachment: true}))
			},
			BreakPosting: func(*testing.T, record.Ref) {
				f.check(f.hrm.AdjustLeaveBalance(f.hr, f.emp, 2026, hrm.BalanceAdjustment{Delta: "-" + f.balance(), Reason: "hết"}))
				granted = 0
			},
			Effects: func(*testing.T, record.Ref) int {
				return int(decimal.NewFromInt(granted).Sub(decimal.RequireFromString(f.balance())).IntPart())
			},
		}
	})
}

func equal(got []string, want ...string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// HR files a request for E; E holds the approving role too, yet never approves their own leave.
func TestTheEmployeeNeverApprovesTheirOwnLeave(t *testing.T) {
	f := newLeaveFixture(t)
	f.grant("12")
	eID, err := f.iam.UserIDByLogin(f.admin, "e")
	f.check(err)
	must(t)(f.iam.GrantRole(f.admin, eID, "hrm", "hr", &f.a))
	f.ctxFor("head", "hr@A")
	f.check(f.appr.SaveRule(f.admin, "hrm.leave_request", approval.RuleInput{
		Steps: []approval.Step{{Approver: approval.Approver{Kind: "role", Product: "hrm", Role: "hr"}}}, MaxLevels: 3, FallbackProduct: "hrm", FallbackRole: "hr",
	}))
	id := must(t)(f.hrm.CreateLeave(f.hr, hrm.NewLeave{EmployeeID: &f.emp, LeaveFields: f.fields("2026-03-10", "2026-03-10", "1")}))
	f.check(f.send(f.hr, id))
	wantErr(t, f.appr.Approve(f.e, f.ticket(id), 1), "forbidden")
	wantErr(t, f.appr.Reassign(f.hr, f.ticket(id), 1, "e"), "self_approval")
	// Linking head's account to E after the step started does not let head approve E's leave.
	e := employee("E", f.a)
	e.UserLogin = str("head")
	f.check(f.hrm.UpdateEmployee(f.hr, f.emp, e.EmployeeInput))
	wantErr(t, f.appr.Approve(f.loginCtx("head"), f.ticket(id), 1), "self_approval")
}

// Step 7 at once: two requests that do not both fit are approved together; one waits on
// the balance row, finds too little left and stays pending.
func TestTwoRequestsApprovedAtOnce(t *testing.T) {
	f := newLeaveFixture(t)
	f.ruleManager()
	f.grant("12")
	five := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-03-10", "2026-03-16", "5")}))
	eight := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-04-01", "2026-04-10", "8")}))
	f.check(f.send(f.e, five))
	f.check(f.send(f.e, eight))
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, id := range []int64{five, eight} {
		wg.Go(func() { errs[i] = f.appr.Approve(f.m, f.ticket(id), 1) })
	}
	wg.Wait()
	switch {
	case errs[0] == nil && errCode(errs[1]) == "insufficient_leave_balance":
		if f.balance() != "7" || f.leave(eight).Status != "pending_approval" {
			t.Fatalf("5 first: balance %s, 8-day %s", f.balance(), f.leave(eight).Status)
		}
	case errs[1] == nil && errCode(errs[0]) == "insufficient_leave_balance":
		if f.balance() != "4" || f.leave(five).Status != "pending_approval" {
			t.Fatalf("8 first: balance %s, 5-day %s", f.balance(), f.leave(five).Status)
		}
	default:
		t.Fatalf("want one approval and one insufficient_leave_balance: %v, %v", errs[0], errs[1])
	}
}

// Taking days away while a request is approved never makes the balance negative: whichever
// commits second finds too little left.
func TestNegativeAdjustmentRacesApproval(t *testing.T) {
	f := newLeaveFixture(t)
	f.ruleManager()
	f.grant("2")
	id := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-03-10", "2026-03-11", "2")}))
	f.check(f.send(f.e, id))
	ticket := f.ticket(id)
	var approved, adjusted error
	var wg sync.WaitGroup
	wg.Go(func() { approved = f.appr.Approve(f.m, ticket, 1) })
	wg.Go(func() {
		adjusted = f.hrm.AdjustLeaveBalance(f.hr, f.emp, 2026, hrm.BalanceAdjustment{Delta: "-1", Reason: "sai"})
	})
	wg.Wait()
	switch {
	case approved == nil && errCode(adjusted) == "leave_balance_negative":
		if f.balance() != "0" {
			t.Fatalf("balance %s", f.balance())
		}
	case adjusted == nil && errCode(approved) == "insufficient_leave_balance":
		if f.balance() != "1" {
			t.Fatalf("balance %s", f.balance())
		}
	default:
		t.Fatalf("want exactly one to succeed: approve %v, adjust %v", approved, adjusted)
	}
}

// Cancelling gives back what approving took, even after the kind of leave stopped deducting.
func TestCancelRefundsWhatWasDeducted(t *testing.T) {
	f := newLeaveFixture(t)
	f.grant("12")
	id := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-03-10", "2026-03-12", "3")}))
	f.check(f.send(f.e, id)) // no rule: posted at once
	if f.balance() != "9" {
		t.Fatalf("after approval %s", f.balance())
	}
	must(t)(f.hrm.SaveLeaveType(f.hr, f.annual, hrm.LeaveTypeInput{Name: "Phép năm", DeductsBalance: false, Active: true}))
	f.check(f.rec.Transition(f.hr, record.Ref{Type: "hrm.leave_request", ID: id}, f.leave(id).Version, record.Cancelled))
	if f.balance() != "12" {
		t.Fatalf("after cancel %s", f.balance())
	}
}

// The employee's timeline shows a viewer neither balance adjustments nor reads of sensitive data;
// whoever sees sensitive data sees those reads.
func TestEmployeeHistoryHidesBalancesAndSensitiveReads(t *testing.T) {
	f := newLeaveFixture(t)
	f.grant("12")
	sens := f.ctxFor("sens", "viewer@*", "sensitive_viewer@*")
	if _, err := f.hrm.Dependents(sens, f.emp); err != nil {
		t.Fatal(err)
	}
	actions := func(ctx context.Context) []string {
		t.Helper()
		h, err := f.rec.History(ctx, record.Ref{Type: "hrm.employee", ID: f.emp})
		f.check(err)
		var out []string
		for _, e := range h {
			out = append(out, e.Action)
		}
		return out
	}
	if a := actions(f.ctxFor("viewer", "viewer@*")); slices.Contains(a, "hrm.leave_balance_adjusted") || slices.Contains(a, "hrm.dependents_viewed") {
		t.Fatalf("viewer sees %v", a)
	}
	if a := actions(sens); !slices.Contains(a, "hrm.dependents_viewed") || slices.Contains(a, "hrm.leave_balance_adjusted") {
		t.Fatalf("sensitive viewer sees %v", a)
	}
}

// The reason is checked by the service, not only by the API: imports reach it directly.
func TestAdjustBalanceNeedsReason(t *testing.T) {
	f := newLeaveFixture(t)
	wantErr(t, f.hrm.AdjustLeaveBalance(f.hr, f.emp, 2026, hrm.BalanceAdjustment{Delta: "1", Reason: "  "}), "reason_required")
}

// "2.0" over a stored 2 is no change in the audit diff.
func TestLeaveAuditIgnoresNumberSpelling(t *testing.T) {
	f := newLeaveFixture(t)
	id := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-03-10", "2026-03-11", "2")}))
	in := f.fields("2026-03-10", "2026-03-11", "2.0")
	in.Reason = str("ốm")
	f.check(f.hrm.UpdateLeave(f.e, id, hrm.LeaveUpdate{Version: 1, LeaveFields: in}))
	var data string
	f.check(f.pool.QueryRow(f.admin, `SELECT data::text FROM audit.log WHERE action = 'hrm.leave_request_updated' AND doc_id = $1`, id).Scan(&data))
	if strings.Contains(data, `"days"`) || !strings.Contains(data, `"reason"`) {
		t.Fatalf("diff %s", data)
	}
}

// Adjusting a balance needs leave_admin, not HR alone; a cut within what is left goes
// through, and each adjustment writes who, how much, the result and why on the employee.
func TestAdjustBalanceIsGuardedAndAudited(t *testing.T) {
	f := newLeaveFixture(t)
	hrOnly := f.ctxFor("hronly", "hr@*")
	wantErr(t, f.hrm.AdjustLeaveBalance(hrOnly, f.emp, 2026, hrm.BalanceAdjustment{Delta: "1", Reason: "x"}), "not_found")
	f.grant("12")
	f.check(f.hrm.AdjustLeaveBalance(f.hr, f.emp, 2026, hrm.BalanceAdjustment{Delta: "-2.5", Reason: "trừ nhầm"}))
	if f.balance() != "9.5" {
		t.Fatalf("balance %s, want 12 - 2.5", f.balance())
	}
	hrID, err := f.iam.UserIDByLogin(f.admin, "hr")
	f.check(err)
	rows, err := f.pool.Query(f.admin, `SELECT actor_id, data->>'delta', data->>'days', data->>'reason' FROM audit.log
		WHERE action = 'hrm.leave_balance_adjusted' AND doc_type = 'hrm.employee' AND doc_id = $1 ORDER BY id`, f.emp)
	f.check(err)
	var got []string
	for rows.Next() {
		var actor int64
		var delta, days, reason string
		f.check(rows.Scan(&actor, &delta, &days, &reason))
		if actor != hrID {
			t.Fatalf("actor %d, want %d", actor, hrID)
		}
		got = append(got, delta+"→"+days+" "+reason)
	}
	f.check(rows.Err())
	if !equal(got, "12→12 đầu năm", "-2.5→9.5 trừ nhầm") {
		t.Fatalf("audit %v", got)
	}
}

// A rejected request goes back to draft without touching the balance, and E may edit and send it again.
func TestRejectedLeaveReturnsToDraft(t *testing.T) {
	f := newLeaveFixture(t)
	f.ruleManager()
	f.grant("12")
	id := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-03-10", "2026-03-11", "2")}))
	f.check(f.send(f.e, id))
	ticket := f.ticket(id)
	wantErr(t, f.appr.Reject(f.e, ticket, 1, "tự từ chối"), "forbidden")
	f.check(f.appr.Reject(f.m, ticket, 1, "trùng lịch họp"))
	if l := f.leave(id); l.Status != "draft" || f.balance() != "12" {
		t.Fatalf("after reject: %+v, balance %s", l, f.balance())
	}
	if l, _ := f.hrm.Leave(f.e, id); !equal(l.AllowedActions, "edit", "delete", "submit") {
		t.Fatalf("E's actions after reject %v", l.AllowedActions)
	}
	wantErr(t, f.appr.Approve(f.m, ticket, 1), "approval_closed")
	f.check(f.hrm.UpdateLeave(f.e, id, hrm.LeaveUpdate{Version: f.leave(id).Version, LeaveFields: f.fields("2026-03-10", "2026-03-10", "1")}))
	f.check(f.send(f.e, id))
	f.check(f.appr.Approve(f.m, f.ticket(id), 1))
	if f.leave(id).Status != "posted" || f.balance() != "11" {
		t.Fatalf("resent: %+v, balance %s", f.leave(id), f.balance())
	}
}
