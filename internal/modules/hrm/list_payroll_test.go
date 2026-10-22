package hrm_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/coretest"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

// payrollListFixture pins Payrolls' current scope, filter, order and paging
// behaviour: two legal entities, each with one department below it, and seven
// payrolls spread across three months and three statuses.
//
// Creating each payroll through CreatePayroll would need a timesheet, a posted
// contract and a computation job per legal entity per month; the list only reads
// record.documents and hrm.payrolls, so rows are seeded directly through the
// fixture's writable system pool instead.
//
// LE1 ("Zulu") is created first, so it holds the lower id, but LE2 ("Alpha") is
// named to sort first: a sort by legal_entity_id instead of its name would give
// a different order than the one pinned below, so a port that drops the name
// join is caught here, not only in Task 2's own tests.
//
// Layout (period_start DESC, legal entity name, id DESC is the documented order):
//
//	p3 LE2(Alpha) 2026-03 cancelled  p2 LE1(Zulu) 2026-03 draft  p1 LE1(Zulu) 2026-03 posted
//	p5 LE2(Alpha) 2026-02 draft      p4 LE1(Zulu) 2026-02 posted
//	p7 LE2(Alpha) 2026-01 posted     p6 LE1(Zulu) 2026-01 cancelled
type payrollListFixture struct {
	t                          *testing.T
	core                       *coretest.Core
	system                     platform.Conn
	hrm                        *hrm.Service
	admin                      context.Context
	le1, le2                   int64
	dept1, dept2               int64
	p1, p2, p3, p4, p5, p6, p7 int64
}

func newPayrollListFixture(t *testing.T) *payrollListFixture {
	t.Helper()
	core := coretest.New(t, "hrm")
	h := hrm.NewService(hrm.Deps{IAM: core.IAM, Record: core.Record, Audit: core.Audit, Setting: core.Setting,
		DataIO: core.DataIO, Printing: core.Printing, Posting: posting.NewService(posting.Hooks{}), Numbering: core.Numbering})
	f := &payrollListFixture{t: t, core: core, system: core.System, hrm: h, admin: core.Admin}
	// Named so the alphabetical order of the name disagrees with the creation
	// (and so the id) order: see the comment on payrollListFixture.
	f.le1 = must(t)(core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{Kind: "company", Name: "Zulu"}))
	f.le2 = must(t)(core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{Kind: "company", Name: "Alpha"}))
	f.dept1 = must(t)(core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{ParentID: &f.le1, Kind: "department", Name: "LE1-Dept"}))
	f.dept2 = must(t)(core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{ParentID: &f.le2, Kind: "department", Name: "LE2-Dept"}))

	// p1 and p2 share LE1 ("Zulu") and 2026-03: p2, the later id, pins the final
	// id DESC tiebreak. p3 (LE2 "Alpha") shares only the period: it sorts before
	// both by name, ahead of its lower-id siblings.
	f.p1 = f.seed(f.le1, "2026-03", "posted")
	f.p2 = f.seed(f.le1, "2026-03", "draft")
	f.p3 = f.seed(f.le2, "2026-03", "cancelled")
	f.p4 = f.seed(f.le1, "2026-02", "posted")
	f.p5 = f.seed(f.le2, "2026-02", "draft")
	f.p6 = f.seed(f.le1, "2026-01", "cancelled")
	f.p7 = f.seed(f.le2, "2026-01", "posted")
	return f
}

// seed inserts a payroll document and its hrm.payrolls row directly, skipping
// the contract/timesheet/computation a real CreatePayroll needs.
func (f *payrollListFixture) seed(le int64, month, status string) int64 {
	f.t.Helper()
	start, end, err := platform.MonthPeriod(month)
	f.check(err)
	var id int64
	f.check(f.system.QueryRow(f.admin, `
		INSERT INTO record.documents (doc_type, number, status, version, date, legal_entity_id, org_unit_id)
		VALUES ('hrm.payroll', $1, $2, 1, $3, $4, $4) RETURNING id`,
		fmt.Sprintf("BL-SEED-%d-%s-%s", le, month, status), status, end, le).Scan(&id))
	_, err = f.system.Exec(f.admin, `INSERT INTO hrm.payrolls (id, period_start, period_end) VALUES ($1, $2, $3)`, id, start, end)
	f.check(err)
	return id
}

func (f *payrollListFixture) check(err error) {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
}

// actor creates a user and, if role is not empty, grants it at unit (nil for tenant-wide).
func (f *payrollListFixture) actor(login, role string, unit *int64) context.Context {
	f.t.Helper()
	id := must(f.t)(f.core.IAM.CreateUser(f.admin, login, login, "long enough"))
	if role != "" {
		must(f.t)(f.core.IAM.GrantRole(f.admin, id, "hrm", role, unit))
	}
	return platform.WithActor(f.core.Ctx, id)
}

func idsOf(list hrm.PayrollList) []int64 {
	out := make([]int64, len(list.Items))
	for i, it := range list.Items {
		out[i] = it.ID
	}
	return out
}

func page1(f hrm.PayrollFilter) hrm.PayrollFilter {
	f.Paging = platform.Paging{Page: 1, PageSize: 20}
	return f
}

func TestPayrollListScope(t *testing.T) {
	f := newPayrollListFixture(t)
	payLE1 := f.actor("pay_le1", "payroll", &f.le1)
	payDept := f.actor("pay_dept", "payroll", &f.dept1)
	nobody := f.actor("nobody", "", nil)

	if list, err := f.hrm.Payrolls(f.admin, page1(hrm.PayrollFilter{})); err != nil || list.Total != 7 {
		t.Fatalf("admin sees %+v, %v", list, err)
	}

	got, err := f.hrm.Payrolls(payLE1, page1(hrm.PayrollFilter{}))
	if err != nil {
		t.Fatal(err)
	}
	gotIDs, want := idsOf(got), []int64{f.p1, f.p2, f.p4, f.p6}
	slices.Sort(gotIDs)
	slices.Sort(want)
	if got.Total != 4 || !slices.Equal(gotIDs, want) {
		t.Fatalf("payLE1 sees %v total %d, want %v", gotIDs, got.Total, want)
	}

	if list, err := f.hrm.Payrolls(payDept, page1(hrm.PayrollFilter{})); err != nil || list.Total != 0 {
		t.Fatalf("payDept sees %+v, %v", list, err)
	}
	if list, err := f.hrm.Payrolls(nobody, page1(hrm.PayrollFilter{})); err != nil || list.Total != 0 {
		t.Fatalf("nobody sees %+v, %v", list, err)
	}
}

func TestPayrollListFilters(t *testing.T) {
	f := newPayrollListFixture(t)
	for _, c := range []struct {
		name string
		f    hrm.PayrollFilter
		want []int64
	}{
		{"status draft", hrm.PayrollFilter{Status: "draft"}, []int64{f.p2, f.p5}},
		{"status posted", hrm.PayrollFilter{Status: "posted"}, []int64{f.p1, f.p4, f.p7}},
		{"status cancelled", hrm.PayrollFilter{Status: "cancelled"}, []int64{f.p3, f.p6}},
		{"legal entity LE1", hrm.PayrollFilter{LegalEntityID: f.le1}, []int64{f.p2, f.p1, f.p4, f.p6}},
		{"legal entity LE2", hrm.PayrollFilter{LegalEntityID: f.le2}, []int64{f.p3, f.p5, f.p7}},
		{"month 2026-02", hrm.PayrollFilter{Month: "2026-02"}, []int64{f.p5, f.p4}},
		{"status and legal entity", hrm.PayrollFilter{Status: "posted", LegalEntityID: f.le1}, []int64{f.p1, f.p4}},
		{"status and month", hrm.PayrollFilter{Status: "draft", Month: "2026-03"}, []int64{f.p2}},
		{"legal entity and month", hrm.PayrollFilter{LegalEntityID: f.le2, Month: "2026-01"}, []int64{f.p7}},
		{"status, legal entity and month", hrm.PayrollFilter{Status: "cancelled", LegalEntityID: f.le1, Month: "2026-01"}, []int64{f.p6}},
	} {
		list, err := f.hrm.Payrolls(f.admin, page1(c.f))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got := idsOf(list); !slices.Equal(got, c.want) || int(list.Total) != len(c.want) {
			t.Errorf("%s: got %v total %d, want %v", c.name, got, list.Total, c.want)
		}
	}
}

func TestPayrollListOrder(t *testing.T) {
	f := newPayrollListFixture(t)
	list, err := f.hrm.Payrolls(f.admin, page1(hrm.PayrollFilter{}))
	if err != nil {
		t.Fatal(err)
	}
	want := []int64{f.p3, f.p2, f.p1, f.p5, f.p4, f.p7, f.p6}
	if got := idsOf(list); !slices.Equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestPayrollListPaging(t *testing.T) {
	f := newPayrollListFixture(t)
	// The fixture has 7 rows; a page of 20 holds all of them.
	list, err := f.hrm.Payrolls(f.admin, hrm.PayrollFilter{Paging: platform.Paging{Page: 1, PageSize: 20}})
	if err != nil || list.Total != 7 || len(list.Items) != 7 {
		t.Fatalf("page 1 = %+v, %v", list, err)
	}
	// A page past the end still counts the true total.
	past, err := f.hrm.Payrolls(f.admin, hrm.PayrollFilter{Paging: platform.Paging{Page: 2, PageSize: 20}})
	if err != nil || len(past.Items) != 0 || past.Total != 7 {
		t.Fatalf("page past the end = %+v, %v", past, err)
	}
}
