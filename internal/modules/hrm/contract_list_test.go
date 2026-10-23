package hrm_test

import (
	"testing"
	"time"

	"github.com/taoworklabs/mmerp/internal/modules/hrm"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// contractAt creates a contract of emp; a parent makes it an appendix.
func (f *contractFixture) contractAt(emp int64, parent *int64, start, end string, post bool) int64 {
	f.t.Helper()
	kind := f.kind
	if end != "" {
		kind = f.fixed
	}
	id := must(f.t)(f.hrm.CreateContract(f.pay, hrm.NewContract{EmployeeID: emp, ParentID: parent, ContractFields: hrm.ContractFields{
		ContractTypeID: kind, StartDate: start, EndDate: optionalStr(end), Terms: terms(1),
	}}))
	if post {
		f.check(f.post(id))
	}
	return id
}

func (f *contractFixture) employeeIn(code string, unit int64) int64 {
	return must(f.t)(f.hrm.CreateEmployee(f.hr, employee(code, unit)))
}

func (f *contractFixture) today() time.Time {
	f.t.Helper()
	d, err := f.core.Setting.Today(f.pay)
	f.check(err)
	return d.Time
}

func TestContractListExpiringBounds(t *testing.T) {
	f := newContractFixture(t)
	today := f.today()
	day := func(n int) string { return today.AddDate(0, 0, n).Format(time.DateOnly) }
	emp := func(code string) int64 { return f.employeeIn(code, f.a) }
	in0 := f.contractAt(emp("B0"), nil, day(-100), day(0), true)
	in30 := f.contractAt(emp("B1"), nil, day(-100), day(30), true)
	f.contractAt(emp("B2"), nil, day(-100), day(31), true)
	f.contractAt(emp("B3"), nil, day(-100), day(-1), true)
	b4 := emp("B4")
	parent := f.contractAt(b4, nil, day(-100), day(100), true)
	f.contractAt(b4, &parent, day(-50), day(5), true)
	f.contractAt(emp("B5"), nil, day(-100), day(5), false)
	f.contractAt(emp("B6"), nil, day(-100), "", true)
	l, err := f.hrm.Contracts(f.pay, hrm.ContractFilter{Expiring: true, Sort: "end_date", Paging: platform.Paging{Page: 1, PageSize: 50}})
	f.check(err)
	var got []int64
	for _, it := range l.Items {
		got = append(got, it.ID)
	}
	if !equalIDs(got, in0, in30) || l.Total != 2 {
		t.Fatalf("expiring: %v of %d, want %d %d", got, l.Total, in0, in30)
	}
}

func TestContractListSearchLiteralPercent(t *testing.T) {
	f := newContractFixture(t)
	f.contractAt(f.emp, nil, "2026-01-01", "", false)
	l, err := f.hrm.Contracts(f.pay, hrm.ContractFilter{Q: "%", Sort: "-start_date", Paging: platform.Paging{Page: 1, PageSize: 20}})
	f.check(err)
	if l.Total != 0 || len(l.Items) != 0 {
		t.Fatalf("q %% matched %d", l.Total)
	}
}
