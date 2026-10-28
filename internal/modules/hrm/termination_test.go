package hrm_test

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/record/recordtest"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
	"github.com/taoworklabs/mmerp/internal/platform"
)

func quit(date, reason string) hrm.TerminationFields {
	return hrm.TerminationFields{TerminationDate: date, Reason: reason}
}

func terminationRef(id int64) record.Ref { return record.Ref{Type: "hrm.termination", ID: id} }

func (f *leaveFixture) terminationDate(emp int64) string {
	f.t.Helper()
	e, err := f.hrm.Employee(f.hr, emp)
	f.check(err)
	if e.TerminationDate == nil {
		return ""
	}
	return *e.TerminationDate
}

func TestTerminationRecordContract(t *testing.T) {
	recordtest.Run(t, func(t *testing.T) recordtest.Harness {
		f := newLeaveFixture(t)
		head := f.ctxFor("head", "hr@*")
		// One employee per document: an employee has one termination at a time.
		var mu sync.Mutex
		employees, byRequest := map[int64]int64{}, map[string]int64{}
		// A retried create, racing or not, is about the same employee.
		employeeFor := func(requestID string) (int64, error) {
			mu.Lock()
			defer mu.Unlock()
			if emp, ok := byRequest[requestID]; ok && requestID != "" {
				return emp, nil
			}
			emp, err := f.hrm.CreateEmployee(f.hr, employee(fmt.Sprintf("Q%d", len(byRequest)+len(employees)), f.a))
			byRequest[requestID] = emp
			return emp, err
		}
		return recordtest.Harness{
			Core: f.core, Actor: f.hr, Admin: f.admin, LegalEntity: f.c, Table: "hrm.terminations",
			Create: func(ctx context.Context, date, requestID string) (record.Ref, error) {
				emp, err := employeeFor(requestID)
				if err != nil {
					return record.Ref{}, err
				}
				id, err := f.hrm.CreateTermination(ctx, hrm.NewTermination{RequestID: requestID, EmployeeID: emp, TerminationFields: quit(date, "mutual_agreement")})
				mu.Lock()
				employees[id] = emp
				mu.Unlock()
				return terminationRef(id), err
			},
			Edit: func(ctx context.Context, r record.Ref, version int32, date string) error {
				return f.hrm.UpdateTermination(ctx, r.ID, hrm.TerminationUpdate{Version: version, TerminationFields: quit(date, "mutual_agreement")})
			},
			Get:    func(ctx context.Context, r record.Ref) (any, error) { return f.hrm.Termination(ctx, r.ID) },
			Reject: func(instance int64) error { return f.appr.Reject(head, instance, 1, "không") },
			RequireApproval: func(_ *testing.T, action record.Action) {
				f.check(f.appr.SaveRule(f.admin, "hrm.termination", action, approval.RuleInput{
					Flow: approval.Flow{{Step: &approval.StepDef{Approver: approval.Approver{Kind: "role", Product: "hrm", Role: "hr"}}}}, MaxLevels: 1, FallbackProduct: "hrm", FallbackRole: "hr",
				}))
			},
			Approve:      func(instance int64) error { return f.appr.Approve(head, instance, 1) },
			BreakPosting: func(_ *testing.T, r record.Ref) { f.closePeriod("2026-03-01", "2026-03-31", r.ID) },
			Effects: func(_ *testing.T, r record.Ref) int {
				mu.Lock()
				emp := employees[r.ID]
				mu.Unlock()
				if f.terminationDate(emp) == "" {
					return 0
				}
				return 1
			},
		}
	})
}

func TestTermination(t *testing.T) {
	f := newLeaveFixture(t)
	id := must(t)(f.hrm.CreateTermination(f.hr, hrm.NewTermination{EmployeeID: f.emp, TerminationFields: quit("2026-04-15", "mutual_agreement")}))
	d, err := f.rec.Get(f.admin, terminationRef(id))
	f.check(err)
	if d.Number != "NV-2026-00001" || d.Fields["reason"] != "mutual_agreement" || d.Date != "2026-04-15" {
		t.Fatalf("header %+v", d)
	}
	got, err := f.hrm.Termination(f.hr, id)
	f.check(err)
	if got.EmployeeID != f.emp || got.Reason != "mutual_agreement" || len(got.AllowedActions) == 0 {
		t.Fatalf("termination %+v", got)
	}

	// One termination per employee while it stands.
	_, err = f.hrm.CreateTermination(f.hr, hrm.NewTermination{EmployeeID: f.emp, TerminationFields: quit("2026-04-20", "dismissal")})
	wantErr(t, err, "termination_exists")

	f.check(f.rec.Delete(f.hr, terminationRef(id), d.Version))
	_, err = f.hrm.CreateTermination(f.hr, hrm.NewTermination{EmployeeID: f.emp, TerminationFields: quit("2026-01-04", "dismissal")})
	wantErr(t, err, "termination_before_hire")
	_, err = f.hrm.CreateTermination(f.hr, hrm.NewTermination{EmployeeID: f.emp, TerminationFields: quit("2026-04-20", "retirement")})
	wantErr(t, err, "invalid_termination_reason")
	must(t)(f.hrm.CreateTermination(f.hr, hrm.NewTermination{EmployeeID: f.emp, TerminationFields: quit("2026-04-20", "dismissal")}))

	// Someone without the termination permission sees none.
	l, err := f.hrm.Terminations(f.e, hrm.TerminationFilter{Sort: "-date", Paging: platform.Paging{Page: 1, PageSize: 20}})
	f.check(err)
	if l.Total != 0 || len(l.AllowedActions) != 0 {
		t.Fatalf("employee sees %+v", l)
	}
	l, err = f.hrm.Terminations(f.hr, hrm.TerminationFilter{Sort: "-date", Paging: platform.Paging{Page: 1, PageSize: 20}})
	f.check(err)
	if l.Total != 1 || l.Items[0].Reason != "dismissal" || l.Items[0].TerminationDate != "2026-04-20" {
		t.Fatalf("list %+v", l)
	}
}

// post moves a termination straight to posted (no approval rule) and returns the error.
func (f *leaveFixture) postTermination(id int64) error {
	f.t.Helper()
	d, err := f.rec.Get(f.admin, terminationRef(id))
	f.check(err)
	return errOf(f.rec.Transition(f.hr, terminationRef(id), d.Version, record.Posted))
}

func (f *leaveFixture) cancelTermination(id int64) error {
	f.t.Helper()
	d, err := f.rec.Get(f.admin, terminationRef(id))
	f.check(err)
	return errOf(f.rec.Transition(f.hr, terminationRef(id), d.Version, record.Cancelled))
}

func TestTerminationPost(t *testing.T) {
	f := newLeaveFixture(t)
	id := must(t)(f.hrm.CreateTermination(f.hr, hrm.NewTermination{EmployeeID: f.emp, TerminationFields: quit("2026-04-15", "mutual_agreement")}))
	if got := f.terminationDate(f.emp); got != "" {
		t.Fatalf("a draft set the date: %s", got)
	}
	f.check(f.postTermination(id))
	if got := f.terminationDate(f.emp); got != "2026-04-15" {
		t.Fatalf("termination date %q", got)
	}
	// A second posted termination of the same employee (a draft slipped in before the first
	// was posted) is refused under the employee lock.
	other := must(t)(f.hrm.CreateEmployee(f.hr, employee("X", f.a)))
	twin := must(t)(f.hrm.CreateTermination(f.hr, hrm.NewTermination{EmployeeID: other, TerminationFields: quit("2026-04-20", "dismissal")}))
	if _, err := f.system.Exec(f.admin, `UPDATE hrm.terminations SET employee_id = $1 WHERE id = $2`, f.emp, twin); err != nil {
		t.Fatal(err)
	}
	wantErr(t, f.postTermination(twin), "termination_exists")
	_, err := f.hrm.CreateTermination(f.hr, hrm.NewTermination{EmployeeID: f.emp, TerminationFields: quit("2026-04-20", "dismissal")})
	wantErr(t, err, "termination_exists")
}

func TestTerminationCancel(t *testing.T) {
	f := newLeaveFixture(t)
	id := must(t)(f.hrm.CreateTermination(f.hr, hrm.NewTermination{EmployeeID: f.emp, TerminationFields: quit("2026-04-15", "mutual_agreement")}))
	f.check(f.postTermination(id))
	f.check(f.cancelTermination(id))
	if got := f.terminationDate(f.emp); got != "" {
		t.Fatalf("cancelled, date still %q", got)
	}
	// A cancelled one no longer counts: a new termination can be made and posted.
	again := must(t)(f.hrm.CreateTermination(f.hr, hrm.NewTermination{EmployeeID: f.emp, TerminationFields: quit("2026-04-15", "mutual_agreement")}))
	f.check(f.postTermination(again))
	// Once April's payroll is closed, it paid the termination: cancelling is refused.
	f.closePeriod("2026-04-01", "2026-04-30", again)
	wantErr(t, f.cancelTermination(again), "payroll_period_closed")
	if got := f.terminationDate(f.emp); got != "2026-04-15" {
		t.Fatalf("refused cancel changed the date to %q", got)
	}
}

// The common late case: the employee left on 31/03, March's payroll is closed, then HR
// enters the termination. Posting is refused until March's payroll is cancelled, so the
// allowance cannot be paid through the system after pay-out.
func TestTerminationLateEntry(t *testing.T) {
	f := newLeaveFixture(t)
	id := must(t)(f.hrm.CreateTermination(f.hr, hrm.NewTermination{EmployeeID: f.emp, TerminationFields: quit("2026-03-31", "employee_resignation")}))
	f.closePeriod("2026-03-01", "2026-03-31", id) // any document stands in for March's payroll
	err := f.postTermination(id)
	wantErr(t, err, "payroll_period_closed")
	if p := fmt.Sprint(err.(*platform.Error).Params["periods"]); p != "[{2026-03-01 2026-03-31}]" {
		t.Fatalf("periods %s", p)
	}
	if got := f.terminationDate(f.emp); got != "" {
		t.Fatalf("refused post set the date to %q", got)
	}
}

// An employee given a termination date before terminations existed keeps it through a
// termination posted and cancelled.
func TestTerminationLegacyDate(t *testing.T) {
	f := newLeaveFixture(t)
	e := employee("L", f.a)
	e.TerminationDate = str("2026-05-31")
	emp := must(t)(f.hrm.CreateEmployee(f.hr, e))
	id := must(t)(f.hrm.CreateTermination(f.hr, hrm.NewTermination{EmployeeID: emp, TerminationFields: quit("2026-04-15", "contract_expired")}))
	f.check(f.postTermination(id))
	if got := f.terminationDate(emp); got != "2026-04-15" {
		t.Fatalf("posted: %q", got)
	}
	f.check(f.cancelTermination(id))
	if got := f.terminationDate(emp); got != "2026-05-31" {
		t.Fatalf("cancelled: %q", got)
	}
}

func TestTerminationLegalEntityChanged(t *testing.T) {
	f := newLeaveFixture(t)
	id := must(t)(f.hrm.CreateTermination(f.hr, hrm.NewTermination{EmployeeID: f.emp, TerminationFields: quit("2026-04-15", "mutual_agreement")}))
	other := must(t)(f.iam.CreateOrgUnit(f.admin, iam.OrgUnitInput{Kind: "company", Name: "D"}))
	e, err := f.hrm.Employee(f.hr, f.emp)
	f.check(err)
	moved := employee("E", other)
	moved.UserLogin, moved.ManagerID = str("e"), e.ManagerID
	f.check(f.hrm.UpdateEmployee(f.hr, f.emp, moved.EmployeeInput))
	wantErr(t, f.postTermination(id), "termination_legal_entity_changed")
}

// Only a termination changes the date on a profile; creating one may carry it, for former
// employees loaded without documents.
func TestEmployeeTerminationDateReadOnly(t *testing.T) {
	f := newLeaveFixture(t)
	e := employee("L", f.a)
	e.TerminationDate = str("2026-05-31")
	emp := must(t)(f.hrm.CreateEmployee(f.hr, e))
	f.check(f.hrm.UpdateEmployee(f.hr, emp, e.EmployeeInput))
	for _, date := range []*string{str("2026-06-30"), nil} {
		e.TerminationDate = date
		wantErr(t, f.hrm.UpdateEmployee(f.hr, emp, e.EmployeeInput), "termination_via_document")
	}
	if got := f.terminationDate(emp); got != "2026-05-31" {
		t.Fatalf("date %q", got)
	}
}

func TestEmployeeImportTerminationDate(t *testing.T) {
	f := newFixture(t)
	hr := f.ctxFor("hr", "hr@*")
	e := employee("E1", f.a)
	e.TerminationDate = str("2026-05-31")
	must(t)(f.hrm.CreateEmployee(hr, e))
	j := f.importEmployees(hr, "import_rows_invalid", eRow("E1", map[int]any{10: "30/06/2026"}))
	wantRowError(t, j, 2, "chỉ đổi qua quyết định nghỉ việc")
	// Empty and unchanged cells pass; a new employee may carry a date.
	f.importEmployees(hr, "completed",
		eRow("E1", map[int]any{4: "0901"}),
		eRow("E1B", map[int]any{1: "Hai", 7: "A", 9: "01/02/2026", 10: "31/03/2026"}))
	f.importEmployees(hr, "completed", eRow("E1", map[int]any{10: "31/05/2026"}))
	if got := f.employeeByCode(hr, "E1"); *got.TerminationDate != "2026-05-31" || *got.Phone != "0901" {
		t.Fatalf("E1 %+v", got.EmployeeFields)
	}
	if got := f.employeeByCode(hr, "E1B"); *got.TerminationDate != "2026-03-31" {
		t.Fatalf("E1B %+v", got.EmployeeFields)
	}
}

// Replacing or restoring an earlier date changes the pay from that earlier date on, so a
// closed period there refuses both.
func TestTerminationEarlierDateInClosedPeriod(t *testing.T) {
	f := newLeaveFixture(t)
	e := employee("L", f.a)
	e.TerminationDate = str("2026-02-28")
	emp := must(t)(f.hrm.CreateEmployee(f.hr, e))
	id := must(t)(f.hrm.CreateTermination(f.hr, hrm.NewTermination{EmployeeID: emp, TerminationFields: quit("2026-04-15", "contract_expired")}))
	f.closePeriod("2026-03-01", "2026-03-31", id)
	wantErr(t, f.postTermination(id), "payroll_period_closed")
	if got := f.terminationDate(emp); got != "2026-02-28" {
		t.Fatalf("refused post changed the date to %q", got)
	}
}

func TestTerminationCancelAfterLegalEntityChanged(t *testing.T) {
	f := newLeaveFixture(t)
	id := must(t)(f.hrm.CreateTermination(f.hr, hrm.NewTermination{EmployeeID: f.emp, TerminationFields: quit("2026-04-15", "mutual_agreement")}))
	f.check(f.postTermination(id))
	other := must(t)(f.iam.CreateOrgUnit(f.admin, iam.OrgUnitInput{Kind: "company", Name: "D"}))
	e, err := f.hrm.Employee(f.hr, f.emp)
	f.check(err)
	moved := employee("E", other)
	moved.UserLogin, moved.ManagerID, moved.TerminationDate = str("e"), e.ManagerID, e.TerminationDate
	f.check(f.hrm.UpdateEmployee(f.hr, f.emp, moved.EmployeeInput))
	wantErr(t, f.cancelTermination(id), "termination_legal_entity_changed")
}

// leaver is E hired on 01/07/2007 with 20,000,000 from 2025 and 24,000,000 from 2026.
func leaver(t *testing.T) *contractFixture {
	f := newContractFixture(t)
	e, err := f.hrm.Employee(f.hr, f.emp)
	f.check(err)
	in := employee("E", f.a)
	in.UserLogin, in.ManagerID, in.HireDate = str("e"), e.ManagerID, "2007-07-01"
	f.check(f.hrm.UpdateEmployee(f.hr, f.emp, in.EmployeeInput))
	orig := must(t)(f.original("2025-01-01", "", 20_000_000))
	f.check(f.post(orig))
	f.check(f.post(must(t)(f.appendix(orig, "2026-01-01", 24_000_000))))
	return f
}

func TestTerminationAllowancePreview(t *testing.T) {
	f := leaver(t)
	in := quit("2026-04-15", "mutual_agreement")
	in.InsuredMonths = 207
	id := must(t)(f.hrm.CreateTermination(f.hr, hrm.NewTermination{EmployeeID: f.emp, TerminationFields: in}))
	before := f.auditCount("hrm.contract_terms_viewed")
	got, err := f.hrm.Termination(f.hr, id)
	f.check(err)
	a := got.Allowance
	if a.Kind != "severance" || a.MonthsWorked != 225 || a.MonthsCounted != 18 || a.Years != "1.5" || a.Average != nil || a.Amount != nil {
		t.Fatalf("without salary view %+v", a)
	}
	if f.auditCount("hrm.contract_terms_viewed") != before {
		t.Fatal("terms read without the salary permission")
	}
	got, err = f.hrm.Termination(f.pay, id)
	f.check(err)
	a = got.Allowance
	if a.Average == nil || *a.Average != 22_000_000 || *a.Amount != 16_500_000 {
		t.Fatalf("with salary view %+v", a)
	}
	if f.auditCount("hrm.contract_terms_viewed") == before {
		t.Fatal("terms read unaudited")
	}
}

func TestTerminationWarnings(t *testing.T) {
	f := leaver(t)
	f.grant("4")
	leave := must(t)(f.hrm.CreateLeave(f.hr, hrm.NewLeave{EmployeeID: &f.emp, LeaveFields: f.fields("2026-04-20", "2026-04-21", "2")}))
	gone := must(t)(f.hrm.CreateLeave(f.hr, hrm.NewLeave{EmployeeID: &f.emp, LeaveFields: f.fields("2026-04-23", "2026-04-23", "1")}))
	f.check(errOf(f.rec.Transition(f.hr, record.Ref{Type: "hrm.leave_request", ID: gone}, 1, record.Posted)))
	f.check(errOf(f.rec.Transition(f.hr, record.Ref{Type: "hrm.leave_request", ID: gone}, 2, record.Cancelled)))
	ot := must(t)(f.hrm.CreateOvertime(f.hr, hrm.NewOvertime{EmployeeID: &f.emp, OvertimeFields: overtime("2026-04-22", "2", "0")}))
	f.check(errOf(f.rec.Transition(f.hr, record.Ref{Type: "hrm.overtime_request", ID: ot}, 1, record.Posted)))
	id := must(t)(f.hrm.CreateTermination(f.hr, hrm.NewTermination{EmployeeID: f.emp, TerminationFields: quit("2026-04-15", "mutual_agreement")}))
	got, err := f.hrm.Termination(f.hr, id)
	f.check(err)
	codes := map[string]map[string]any{}
	for _, w := range got.Warnings {
		codes[w.Code] = w.Params
	}
	if len(codes) != 4 || codes["contract_in_force_after"]["number"] != "HD-2025-00001" ||
		codes["leave_after_termination"]["number"] != f.leave(leave).Number || codes["leave_after_termination"]["status"] != "draft" ||
		codes["overtime_after_termination"]["date"] != "2026-04-22" || codes["leave_balance_to_pay"]["days"] != "4" {
		t.Fatalf("warnings %+v", got.Warnings)
	}
}

// The profile offers a termination until one stands, then links to it.
func TestEmployeeTerminationLink(t *testing.T) {
	f := newLeaveFixture(t)
	e, err := f.hrm.Employee(f.hr, f.emp)
	f.check(err)
	if !slices.Contains(e.AllowedActions, "create_termination") || e.Termination != nil {
		t.Fatalf("before %v %+v", e.AllowedActions, e.Termination)
	}
	id := must(t)(f.hrm.CreateTermination(f.hr, hrm.NewTermination{EmployeeID: f.emp, TerminationFields: quit("2026-04-15", "mutual_agreement")}))
	e, err = f.hrm.Employee(f.hr, f.emp)
	f.check(err)
	if slices.Contains(e.AllowedActions, "create_termination") || e.Termination == nil || e.Termination.ID != id || e.Termination.Number != "NV-2026-00001" {
		t.Fatalf("after %v %+v", e.AllowedActions, e.Termination)
	}
	// A profile viewer without the termination permission sees neither.
	self, err := f.hrm.Employee(f.ctxFor("viewer", "viewer@*"), f.emp)
	f.check(err)
	if self.Termination != nil || slices.Contains(self.AllowedActions, "create_termination") {
		t.Fatalf("self %v %+v", self.AllowedActions, self.Termination)
	}
}

// A termination stays filed under the unit it was made in: HR of the employee's new unit
// neither sees it on the profile nor gets to create a second one.
func TestEmployeeTerminationLinkAfterMove(t *testing.T) {
	f := newLeaveFixture(t)
	must(t)(f.hrm.CreateTermination(f.hr, hrm.NewTermination{EmployeeID: f.emp, TerminationFields: quit("2026-04-15", "mutual_agreement")}))
	e, err := f.hrm.Employee(f.hr, f.emp)
	f.check(err)
	moved := employee("E", f.b)
	moved.UserLogin, moved.ManagerID = str("e"), e.ManagerID
	f.check(f.hrm.UpdateEmployee(f.hr, f.emp, moved.EmployeeInput))
	got, err := f.hrm.Employee(f.ctxFor("hrb", "hr@B"), f.emp)
	f.check(err)
	if got.Termination != nil || slices.Contains(got.AllowedActions, "create_termination") {
		t.Fatalf("new unit's HR %v %+v", got.AllowedActions, got.Termination)
	}
}

// The decision prints who leaves, when and why, and never an amount.
func TestTerminationPrint(t *testing.T) {
	f := newLeaveFixture(t)
	in := quit("2026-04-15", "restructuring")
	in.DecisionDate = str("2026-04-01")
	id := must(t)(f.hrm.CreateTermination(f.hr, hrm.NewTermination{EmployeeID: f.emp, TerminationFields: in}))
	parts, err := f.hrm.TerminationPrint(f.hr, id)
	f.check(err)
	var got map[string]any
	f.check(json.Unmarshal(parts[0].Data, &got))
	if got["number"] != "NV-2026-00001" || got["employee_name"] != "Nhân viên E" || got["termination_date"] != "2026-04-15" ||
		got["decision_date"] != "2026-04-01" || got["reason"] != "restructuring" || got["allowance"] != "none" { // under 12 months worked
		t.Fatalf("print data %v", got)
	}
	for k := range got {
		if strings.Contains(k, "amount") || strings.Contains(k, "average") {
			t.Fatalf("print carries %s", k)
		}
	}
	job, err := f.core.DataIO.StartExport(f.hr, "printing.hrm.termination", fmt.Appendf(nil, `{"id":%d}`, id))
	f.check(err)
	f.wantJob(job, "completed")
}
