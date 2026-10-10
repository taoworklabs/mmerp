package hrm_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/record/recordtest"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
)

// contractFixture adds to the leave fixture HR with the payroll role, a second approver,
// and two kinds of contract: kind without an end date, fixed with one.
type contractFixture struct {
	*leaveFixture
	pay, head   context.Context
	kind, fixed int64
}

func newContractFixture(t *testing.T) *contractFixture {
	f := &contractFixture{leaveFixture: newLeaveFixture(t)}
	f.pay = f.ctxFor("pay", "hr@*", "payroll@*")
	f.head = f.ctxFor("head", "hr@*")
	f.kind = must(t)(f.hrm.SaveContractType(f.hr, 0, hrm.ContractTypeInput{Name: "Không xác định thời hạn", Active: true}))
	f.fixed = must(t)(f.hrm.SaveContractType(f.hr, 0, hrm.ContractTypeInput{Name: "Xác định thời hạn", FixedTerm: true, Active: true}))
	return f
}

func terms(salary int64) hrm.ContractTerms {
	return hrm.ContractTerms{Salary: salary, Lines: []hrm.ContractLine{{Kind: "support", Name: "Ăn ca", Amount: 730_000}}}
}

// original creates a draft original of E from start, of the fixed-term kind when it has an end.
func (f *contractFixture) original(start, end string, salary int64) (int64, error) {
	kind := f.kind
	if end != "" {
		kind = f.fixed
	}
	return f.hrm.CreateContract(f.pay, hrm.NewContract{EmployeeID: f.emp, ContractFields: hrm.ContractFields{
		ContractTypeID: kind, StartDate: start, EndDate: optionalStr(end), Terms: terms(salary),
	}})
}

func (f *contractFixture) appendix(parent int64, start string, salary int64) (int64, error) {
	return f.hrm.CreateContract(f.pay, hrm.NewContract{EmployeeID: f.emp, ParentID: &parent, ContractFields: hrm.ContractFields{StartDate: start, Terms: terms(salary)}})
}

func (f *contractFixture) contract(id int64) hrm.Contract {
	f.t.Helper()
	c, err := f.hrm.Contract(f.pay, id)
	f.check(err)
	return c
}

func (f *contractFixture) post(id int64) error {
	return f.rec.Transition(f.pay, record.Ref{Type: "hrm.contract", ID: id}, f.contract(id).Version, record.Posted)
}

func (f *contractFixture) cancel(id int64) error {
	return f.rec.Transition(f.pay, record.Ref{Type: "hrm.contract", ID: id}, f.contract(id).Version, record.Cancelled)
}

func optionalStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func TestContractRecordContract(t *testing.T) {
	recordtest.Run(t, func(t *testing.T) recordtest.Harness {
		f := newContractFixture(t)
		ref := func(id int64, err error) (record.Ref, error) { return record.Ref{Type: "hrm.contract", ID: id}, err }
		return recordtest.Harness{
			Record: f.rec, Actor: f.pay, Admin: f.admin, LegalEntity: f.c,
			Create: func(ctx context.Context, date string) (record.Ref, error) {
				return ref(f.hrm.CreateContract(ctx, hrm.NewContract{EmployeeID: f.emp, ContractFields: hrm.ContractFields{ContractTypeID: f.kind, StartDate: date, Terms: terms(10_000_000)}}))
			},
			Edit: func(ctx context.Context, r record.Ref, version int32, date string) error {
				return f.hrm.UpdateContract(ctx, r.ID, hrm.ContractUpdate{Version: version, ContractFields: hrm.ContractFields{ContractTypeID: f.kind, StartDate: date, Terms: terms(10_000_000)}})
			},
			RequireApproval: func(*testing.T) {
				f.check(f.appr.SaveRule(f.admin, "hrm.contract", approval.RuleInput{
					Steps: []approval.Step{{Approver: approval.Approver{Kind: "role", Product: "hrm", Role: "hr"}}}, MaxLevels: 1, FallbackProduct: "hrm", FallbackRole: "hr",
				}))
			},
			Approve:      func(instance int64) error { return f.appr.Approve(f.head, instance, 1) },
			BreakPosting: func(_ *testing.T, r record.Ref) { f.closePeriod("2026-03-01", "2026-03-31", r.ID) },
			Effects: func(*testing.T, record.Ref) int {
				_, ok, err := f.hrm.ContractAt(f.admin, f.emp, "2026-12-31")
				f.check(err)
				if ok {
					return 1
				}
				return 0
			},
		}
	})
}

// One original from 2026-01-01 and appendices from 03-01 (posted), 05-01 (draft),
// 06-01 (posted) and 07-01 (cancelled).
func TestContractAt(t *testing.T) {
	f := newContractFixture(t)
	orig := must(t)(f.original("2026-01-01", "2026-12-31", 10_000_000))
	f.check(f.post(orig))
	march := must(t)(f.appendix(orig, "2026-03-01", 11_000_000))
	f.check(f.post(march))
	must(t)(f.appendix(orig, "2026-05-01", 99_000_000))
	june := must(t)(f.appendix(orig, "2026-06-01", 12_000_000))
	f.check(f.post(june))
	july := must(t)(f.appendix(orig, "2026-07-01", 98_000_000))
	f.check(f.post(july))
	f.check(f.cancel(july))
	for date, want := range map[string]int64{
		"2026-01-01": 10_000_000, "2026-02-28": 10_000_000, "2026-03-01": 11_000_000, "2026-05-15": 11_000_000,
		"2026-06-01": 12_000_000, "2026-07-15": 12_000_000, "2026-12-31": 12_000_000,
	} {
		got, ok, err := f.hrm.ContractAt(f.admin, f.emp, date)
		if err != nil || !ok || got.Salary != want || len(got.Lines) != 1 {
			t.Fatalf("%s: %+v %v %v, want %d", date, got, ok, err, want)
		}
	}
	for _, date := range []string{"2025-12-31", "2027-01-01"} {
		if _, ok, err := f.hrm.ContractAt(f.admin, f.emp, date); err != nil || ok {
			t.Fatalf("%s: in force %v %v", date, ok, err)
		}
	}
	// Reading terms for a computation is a read like any other: audited on the contract it came from.
	before := f.auditCount("hrm.contract_terms_viewed")
	_, _, err := f.hrm.ContractAt(f.admin, f.emp, "2026-06-15")
	f.check(err)
	var doc int64
	f.check(f.pool.QueryRow(f.admin, `SELECT doc_id FROM audit.log WHERE action = 'hrm.contract_terms_viewed' ORDER BY id DESC LIMIT 1`).Scan(&doc))
	if n := f.auditCount("hrm.contract_terms_viewed"); n != before+1 || doc != june {
		t.Fatalf("audited %d reads of %d, want 1 of %d", n-before, doc, june)
	}
	if cs, err := f.hrm.EmployeeContracts(f.pay, f.emp); err != nil || len(cs) != 5 || cs[0].ID != orig || cs[1].ID != march {
		t.Fatalf("list %+v %v", cs, err)
	}
}

func TestContractRules(t *testing.T) {
	f := newContractFixture(t)
	_, err := f.original("2026-02-01", "2026-01-31", 1)
	wantErr(t, err, "contract_end_before_start")
	_, err = f.hrm.CreateContract(f.pay, hrm.NewContract{EmployeeID: f.emp, ContractFields: hrm.ContractFields{
		ContractTypeID: f.kind, StartDate: "2026-01-01", Terms: hrm.ContractTerms{Lines: []hrm.ContractLine{{Kind: "allowance", Name: " ", Amount: 1}}},
	}})
	wantErr(t, err, "invalid_contract_terms")
	// A salary allowance is always taxable; only a support may be tax-free.
	_, err = f.hrm.CreateContract(f.pay, hrm.NewContract{EmployeeID: f.emp, ContractFields: hrm.ContractFields{
		ContractTypeID: f.kind, StartDate: "2026-01-01", Terms: hrm.ContractTerms{Lines: []hrm.ContractLine{{Kind: "allowance", Name: "Trách nhiệm", Amount: 1}}},
	}})
	wantErr(t, err, "invalid_contract_terms")

	a := must(t)(f.original("2026-01-01", "2026-06-30", 10_000_000))
	_, err = f.appendix(a, "2026-07-01", 1)
	wantErr(t, err, "appendix_outside_contract")
	app := must(t)(f.appendix(a, "2026-03-01", 11_000_000))
	_, err = f.appendix(app, "2026-04-01", 1)
	wantErr(t, err, "invalid_contract_parent")
	// The appendix cannot go in force before its original.
	wantErr(t, f.post(app), "contract_parent_not_posted")
	wantErr(t, f.hrm.DeleteContract(f.pay, a, 1), "contract_has_appendices")
	f.check(f.post(a))
	f.check(f.post(app))

	// Originals of one employee may not overlap; one without an end date runs forever.
	b := must(t)(f.original("2026-06-01", "", 12_000_000))
	wantErr(t, f.post(b), "contract_overlaps")
	c := must(t)(f.original("2026-07-01", "", 12_000_000))
	f.check(f.post(c))

	// An original with a posted appendix stays until the appendix is cancelled.
	wantErr(t, f.cancel(a), "contract_has_appendices")
	f.check(f.cancel(app))
	f.check(f.cancel(a))
}

func TestContractPermissions(t *testing.T) {
	f := newContractFixture(t)
	id := must(t)(f.original("2026-01-01", "", 10_000_000))
	// HR without payroll sees the contract but not its terms, and may not write one.
	c, err := f.hrm.Contract(f.hr, id)
	if err != nil || c.Terms != nil || !equal(c.AllowedActions, "submit") {
		t.Fatalf("hr: %+v %v", c, err)
	}
	_, err = f.hrm.CreateContract(f.hr, hrm.NewContract{EmployeeID: f.emp, ContractFields: hrm.ContractFields{ContractTypeID: f.kind, StartDate: "2027-01-01", Terms: terms(1)}})
	wantErr(t, err, "forbidden")
	wantErr(t, f.hrm.UpdateContract(f.hr, id, hrm.ContractUpdate{Version: 1, ContractFields: hrm.ContractFields{ContractTypeID: f.kind, StartDate: "2026-01-01", Terms: terms(1)}}), "forbidden")
	if _, err := f.hrm.Contract(f.m, id); errCode(err) != "not_found" {
		t.Fatalf("manager: %v", err)
	}
	// HR sends a contract without seeing its terms, and takes it back the same way.
	f.check(f.appr.SaveRule(f.admin, "hrm.contract", approval.RuleInput{
		Steps: []approval.Step{{Approver: approval.Approver{Kind: "role", Product: "hrm", Role: "hr"}}}, MaxLevels: 1, FallbackProduct: "hrm", FallbackRole: "hr",
	}))
	ref := record.Ref{Type: "hrm.contract", ID: id}
	f.check(f.rec.Transition(f.hr, ref, 1, record.Posted))
	sent, err := f.hrm.Contract(f.hr, id)
	if err != nil || !equal(sent.AllowedActions, "withdraw") {
		t.Fatalf("hr after sending: %+v %v", sent.AllowedActions, err)
	}
	f.check(f.rec.Transition(f.hr, ref, sent.Version, record.Draft))
	// With payroll every read of the terms is audited, and the contract may be printed.
	before := f.auditCount("hrm.contract_terms_viewed")
	if c := f.contract(id); c.Terms == nil || c.Terms.Salary != 10_000_000 || !equal(c.AllowedActions, "edit", "delete", "submit", "print") {
		t.Fatalf("payroll: %+v", c)
	}
	f.contract(id)
	if n := f.auditCount("hrm.contract_terms_viewed"); n != before+2 {
		t.Fatalf("audited %d reads, want 2", n-before)
	}
	// The audit of a write keeps the terms encrypted.
	var data string
	f.check(f.pool.QueryRow(f.admin, `SELECT data::text FROM audit.log WHERE action = 'hrm.contract_created' AND doc_id = $1`, id).Scan(&data))
	if strings.Contains(data, "10000000") || !strings.Contains(data, `"sensitive": true`) {
		t.Fatalf("audit %s", data)
	}
}

func (f *fixture) auditCount(action string) int {
	f.t.Helper()
	var n int
	if err := f.pool.QueryRow(f.admin, `SELECT count(*) FROM audit.log WHERE action = $1`, action).Scan(&n); err != nil {
		f.t.Fatal(err)
	}
	return n
}

// A contract without an end date touches every later period; an appendix runs to its original's end.
func TestContractAffectedRange(t *testing.T) {
	f := newContractFixture(t)
	open := must(t)(f.original("2025-01-01", "", 10_000_000))
	ended := must(t)(f.original("2026-01-01", "2026-12-31", 10_000_000))
	f.check(f.post(ended))
	app := must(t)(f.appendix(ended, "2026-03-01", 11_000_000))
	f.closePeriod("2030-04-01", "2030-04-30", ended)
	wantErr(t, f.post(open), "payroll_period_closed")
	f.check(f.post(app)) // the original ends in 2026, so 2030 is not affected
	f.closePeriod("2026-11-01", "2026-11-30", ended)
	wantErr(t, f.cancel(app), "payroll_period_closed")
	if c := f.contract(app); c.Status != "posted" {
		t.Fatalf("appendix %s", c.Status)
	}
}

// Only an original that is not cancelled takes an appendix, and only from someone who may write contracts.
func TestAddAppendixAction(t *testing.T) {
	f := newContractFixture(t)
	a := must(t)(f.original("2026-01-01", "", 10_000_000))
	f.check(f.post(a))
	app := must(t)(f.appendix(a, "2026-03-01", 11_000_000))
	b := must(t)(f.original("2025-01-01", "2025-12-31", 9_000_000))
	f.check(f.post(b))
	f.check(f.cancel(b))
	actions := func(ctx context.Context) map[int64][]string {
		cs, err := f.hrm.EmployeeContracts(ctx, f.emp)
		f.check(err)
		out := map[int64][]string{}
		for _, c := range cs {
			out[c.ID] = c.AllowedActions
		}
		return out
	}
	pay := actions(f.pay)
	if !equal(pay[a], "add_appendix") || !equal(pay[app]) || !equal(pay[b]) {
		t.Fatalf("payroll: %v", pay)
	}
	if hr := actions(f.hr); !equal(hr[a]) {
		t.Fatalf("hr without payroll: %v", hr)
	}
}

// A fixed-term kind needs an end date and a kind without a term refuses one; appendices have none.
func TestEndDateFollowsTheKind(t *testing.T) {
	f := newContractFixture(t)
	_, err := f.hrm.CreateContract(f.pay, hrm.NewContract{EmployeeID: f.emp, ContractFields: hrm.ContractFields{ContractTypeID: f.fixed, StartDate: "2026-01-01", Terms: terms(1)}})
	wantErr(t, err, "contract_end_required")
	_, err = f.hrm.CreateContract(f.pay, hrm.NewContract{EmployeeID: f.emp, ContractFields: hrm.ContractFields{ContractTypeID: f.kind, StartDate: "2026-01-01", EndDate: str("2026-12-31"), Terms: terms(1)}})
	wantErr(t, err, "contract_end_not_allowed")
	id := must(t)(f.original("2026-01-01", "2026-12-31", 1))
	wantErr(t, f.hrm.UpdateContract(f.pay, id, hrm.ContractUpdate{Version: 1, ContractFields: hrm.ContractFields{ContractTypeID: f.fixed, StartDate: "2026-01-01", Terms: terms(1)}}), "contract_end_required")
	must(t)(f.appendix(id, "2026-03-01", 2))
}

func TestContractList(t *testing.T) {
	f := newContractFixture(t)
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	f.check(err)
	today := time.Now().In(loc)
	day := func(n int) string { return today.AddDate(0, 0, n).Format(time.DateOnly) }
	ended := must(t)(f.original(day(-400), day(-301), 1))
	soon := must(t)(f.original(day(-300), day(10), 1))
	later := must(t)(f.original(day(11), day(40), 1))
	for _, id := range []int64{ended, soon, later} {
		f.check(f.post(id))
	}
	app := must(t)(f.appendix(soon, day(-100), 2))
	f.check(f.post(app))
	draft := must(t)(f.original(day(-500), day(5), 1))

	list := func(ctx context.Context, filter hrm.ContractFilter) []int64 {
		filter.Sort, filter.Page, filter.PageSize = "-start_date", 1, 50
		l, err := f.hrm.Contracts(ctx, filter)
		f.check(err)
		ids := []int64{}
		for _, c := range l.Items {
			ids = append(ids, c.ID)
		}
		if l.Total != int64(len(ids)) {
			t.Fatalf("total %d of %d items", l.Total, len(ids))
		}
		return ids
	}
	if got := list(f.pay, hrm.ContractFilter{}); !equalIDs(got, later, app, soon, ended, draft) {
		t.Fatalf("all: %v", got)
	}
	// Expiring: a posted original ending within 30 days; not a draft, not one already ended.
	if got := list(f.pay, hrm.ContractFilter{Expiring: true}); !equalIDs(got, soon) {
		t.Fatalf("expiring: %v", got)
	}
	if got := list(f.pay, hrm.ContractFilter{Status: "draft"}); !equalIDs(got, draft) {
		t.Fatalf("drafts: %v", got)
	}
	if got := list(f.pay, hrm.ContractFilter{ContractTypeID: f.kind}); !equalIDs(got) {
		t.Fatalf("kind without term: %v", got)
	}
	// Outside the contract permission there is nothing to list.
	if got := list(f.m, hrm.ContractFilter{}); !equalIDs(got) {
		t.Fatalf("manager: %v", got)
	}
	l, err := f.hrm.Contracts(f.pay, hrm.ContractFilter{EmployeeID: f.emp, Sort: "start_date", Page: 1, PageSize: 50})
	f.check(err)
	if c := l.Items[0]; c.ID != draft || c.EmployeeName != "Nhân viên E" || c.ParentNumber != nil {
		t.Fatalf("row %+v", c)
	}
}

func equalIDs(got []int64, want ...int64) bool {
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
