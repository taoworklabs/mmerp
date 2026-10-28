package hrm

import (
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
)

// overtimeList shows requests in the actor's units, their own, and those of every
// employee below them on the manager chain.
var overtimeList = data.Register(data.List[store.OvertimeListBaseRow]{
	Name:       "hrm.OvertimeList",
	Base:       store.OvertimeListBase,
	Scope:      data.ByColumn("org_unit_id"),
	Visibility: ownAndReports,
	Search:     []string{"number", "employee_code", "employee_name"},
	Sorts: map[string][]data.Order{
		"date":    {data.Asc("date")},
		"-date":   {data.Desc("date")},
		"number":  {data.Asc("number")},
		"-number": {data.Desc("number")},
	},
	Default:  []data.Order{data.Desc("date")},
	Tiebreak: data.Desc("id"),
})

// overtimeQuery turns the API filter into a query over overtimeList.
func overtimeQuery(sc data.Scope, actor int64, f OvertimeFilter) data.Query {
	from, to := platform.NullDate(platform.Optional(f.From)), platform.NullDate(platform.Optional(f.To))
	return data.Query{
		Scope: sc, Q: f.Q, Sort: f.Sort, Paging: f.Paging,
		Where: data.And(
			data.When(f.Status != "", data.Eq("status", f.Status)),
			data.When(f.OrgUnitID != 0, data.Eq("org_unit_id", f.OrgUnitID)),
			data.When(f.EmployeeID != 0, data.Eq("employee_id", f.EmployeeID)),
			data.When(f.Reports, reportsOnly(actor)),
			data.When(from.Valid, data.Gte("date", from)),
			data.When(to.Valid, data.Lte("date", to)),
		),
	}
}
