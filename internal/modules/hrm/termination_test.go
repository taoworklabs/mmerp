package hrm_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/approval"
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
