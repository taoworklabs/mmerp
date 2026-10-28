package hrm

import (
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform/data"
)

// terminationList shows the terminations in the actor's units.
var terminationList = data.Register(data.List[store.TerminationListBaseRow]{
	Name:   "hrm.TerminationList",
	Base:   store.TerminationListBase,
	Scope:  data.ByColumn("org_unit_id"),
	Search: []string{"number", "employee_code", "employee_name"},
	Sorts: map[string][]data.Order{
		"date":    {data.Asc("date")},
		"-date":   {data.Desc("date")},
		"number":  {data.Asc("number")},
		"-number": {data.Desc("number")},
	},
	Default:  []data.Order{data.Desc("date")},
	Tiebreak: data.Desc("id"),
})

func terminationQuery(sc data.Scope, f TerminationFilter) data.Query {
	return data.Query{
		Scope: sc, Q: f.Q, Sort: f.Sort, Paging: f.Paging,
		Where: data.And(
			data.When(f.Status != "", data.Eq("status", f.Status)),
			data.When(f.OrgUnitID != 0, data.Eq("org_unit_id", f.OrgUnitID)),
			data.When(f.EmployeeID != 0, data.Eq("employee_id", f.EmployeeID)),
		),
	}
}
