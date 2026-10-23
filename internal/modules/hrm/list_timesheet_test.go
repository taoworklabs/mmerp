package hrm_test

import (
	"cmp"
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

// timesheetListFixture seeds timesheets directly (the list reads only
// record.documents and hrm.timesheets). Unit names sort against their id order.
type timesheetListFixture struct {
	t                *testing.T
	core             *coretest.Core
	hrm              *hrm.Service
	root, uZ, uM, uA int64
	hrScoped, nobody context.Context
	ids              map[string]int64
	number           string // overrides the generated document number
}

func newTimesheetListFixture(t *testing.T) *timesheetListFixture {
	t.Helper()
	core := coretest.New(t, "hrm")
	h := hrm.NewService(hrm.Deps{IAM: core.IAM, Record: core.Record, Audit: core.Audit, Setting: core.Setting,
		DataIO: core.DataIO, Printing: core.Printing, Posting: posting.NewService(posting.Hooks{}), Numbering: core.Numbering})
	f := &timesheetListFixture{t: t, core: core, hrm: h, ids: map[string]int64{}}
	f.root = must(t)(core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{Kind: "company", Name: "Root"}))
	unit := func(name string) int64 {
		return must(t)(core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{ParentID: &f.root, Kind: "department", Name: name}))
	}
	f.uZ, f.uM, f.uA = unit("Zulu"), unit("Mike"), unit("Alpha")
	grant := func(login, role string, u *int64) context.Context {
		id := must(t)(core.IAM.CreateUser(core.Admin, login, login, "long enough"))
		if role != "" {
			must(t)(core.IAM.GrantRole(core.Admin, id, "hrm", role, u))
		}
		return platform.WithActor(core.Ctx, id)
	}
	f.hrScoped, f.nobody = grant("hr_scoped", "hr", &f.uZ), grant("nobody", "", nil)
	// Three units in each of three months, with varied statuses.
	statuses := []string{"draft", "posted", "cancelled"}
	n := 0
	for _, m := range []string{"2026-03", "2026-02", "2026-01"} {
		for _, u := range []int64{f.uZ, f.uM, f.uA} {
			f.seed(m, u, statuses[n%3], true)
			n++
		}
	}
	return f
}

func (f *timesheetListFixture) seed(month string, unit int64, status string, active bool) int64 {
	f.t.Helper()
	start, end, err := platform.MonthPeriod(month)
	if err != nil {
		f.t.Fatal(err)
	}
	var id int64
	if err := f.core.System.QueryRow(f.core.Admin, `
		INSERT INTO record.documents (doc_type, number, status, version, date, legal_entity_id, org_unit_id)
		VALUES ('hrm.timesheet', $1, $2, 1, $3, $4, $5) RETURNING id`,
		cmp.Or(f.number, fmt.Sprintf("BC-SEED-%d-%s-%v-%d", unit, month, active, len(f.ids))), status, end, f.root, unit).Scan(&id); err != nil {
		f.t.Fatal(err)
	}
	if _, err := f.core.System.Exec(f.core.Admin, `INSERT INTO hrm.timesheets (id, org_unit_id, period_start, period_end, active) VALUES ($1, $2, $3, $4, $5)`,
		id, unit, start, end, active); err != nil {
		f.t.Fatal(err)
	}
	f.ids[fmt.Sprintf("%d/%s/%v/%d", unit, month, active, len(f.ids))] = id
	return id
}

func tsIDs(l hrm.TimesheetList) []int64 {
	out := []int64{}
	for _, it := range l.Items {
		out = append(out, it.ID)
	}
	return out
}

// Rows that tie on the sort key order by unit name, then id descending.
func TestTimesheetListOrgUnitTiebreak(t *testing.T) {
	f := newTimesheetListFixture(t)
	f.number = "BC-SAME"
	// Same number and period in every unit, plus an inactive twin of Alpha's sheet.
	alpha := f.seed("2026-04", f.uA, "draft", true)
	f.seed("2026-04", f.uZ, "draft", true)
	f.seed("2026-04", f.uM, "draft", true)
	twin := f.seed("2026-04", f.uA, "draft", false)
	for _, sort := range []string{"-period", "period", "number", "-number"} {
		l, err := f.hrm.Timesheets(f.core.Admin, hrm.TimesheetFilter{Month: "2026-04", Sort: sort, Paging: platform.Paging{Page: 1, PageSize: 20}})
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, it := range l.Items {
			names = append(names, it.OrgUnitName)
		}
		got := tsIDs(l)
		if !slices.Equal(names, []string{"Alpha", "Alpha", "Mike", "Zulu"}) || got[0] != twin || got[1] != alpha {
			t.Fatalf("sort %q: ids %v names %v, want Alpha twin %d then %d, Mike, Zulu", sort, got, names, twin, alpha)
		}
	}
}
