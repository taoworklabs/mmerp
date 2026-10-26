package hrm_test

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/modules/hrm"
	"github.com/taoworklabs/mmerp/internal/platform"
)

func dateStr(d pgtype.Date) *string {
	s := d.Time.Format("2006-01-02")
	return &s
}

// Terminated today is still active: the boundary is termination_date < today.
func TestEmployeeListTerminatedTodayIsActive(t *testing.T) {
	f := newFixture(t)
	hr := f.ctxFor("hr", "hr@*")
	today, err := f.core.Setting.Today(hr)
	if err != nil {
		t.Fatal(err)
	}
	in := employee("T1", f.a)
	in.TerminationDate = dateStr(today)
	id := must(t)(f.hrm.CreateEmployee(hr, in))
	list := func(status string) hrm.EmployeeList {
		t.Helper()
		l, err := f.hrm.Employees(hr, hrm.EmployeeFilter{Status: status, Sort: "code", Paging: platform.Paging{Page: 1, PageSize: 20}})
		if err != nil {
			t.Fatal(err)
		}
		return l
	}
	if l := list(""); len(l.Items) != 1 || l.Items[0].ID != id || l.Items[0].Status != "active" {
		t.Fatalf("all: %+v", l.Items)
	}
	if l := list("active"); l.Total != 1 {
		t.Fatalf("active: %+v", l.Items)
	}
	if l := list("terminated"); l.Total != 0 {
		t.Fatalf("terminated: %+v", l.Items)
	}
}

// A "%" in the search matches a literal percent sign, not everything.
func TestEmployeeListSearchLiteralPercent(t *testing.T) {
	f := newFixture(t)
	hr := f.ctxFor("hr", "hr@*")
	must(t)(f.hrm.CreateEmployee(hr, employee("P1", f.a)))
	l, err := f.hrm.Employees(hr, hrm.EmployeeFilter{Q: "%", Sort: "code", Paging: platform.Paging{Page: 1, PageSize: 20}})
	if err != nil {
		t.Fatal(err)
	}
	if l.Total != 0 || len(l.Items) != 0 {
		t.Fatalf("q %% matched %d", l.Total)
	}
}
