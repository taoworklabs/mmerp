package hrm_test

import (
	"context"
	"fmt"
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
