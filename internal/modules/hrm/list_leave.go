package hrm

import (
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
)

// ownAndReports keeps rows whose employee_id is the actor's employee or any employee
// below it on the manager chain. Own and reports form one ARRAY, computed once up
// front: it filters on employee_id alone, so it can use that column's index, and
// unlike a subquery it lets the count run in parallel.
func ownAndReports(v data.Viewer) data.Cond {
	return data.SQL(`b."employee_id" = ANY(ARRAY(WITH RECURSIVE down AS (
    SELECT id FROM hrm.employees WHERE user_id = `).Arg(v.Actor).Text(`::bigint
    UNION
    SELECT e.id FROM hrm.employees e JOIN down ON e.manager_id = down.id
) SELECT down.id FROM down))`).Cond()
}

// leaveList shows requests in the actor's units, their own, and those of every
// employee below them on the manager chain.
var leaveList = data.Register(data.List[store.LeaveListBaseRow]{
	Name:       "hrm.LeaveList",
	Base:       store.LeaveListBase,
	Scope:      data.ByColumn("org_unit_id"),
	Visibility: ownAndReports,
	Search:     []string{"number", "employee_code", "employee_name"},
	Sorts: map[string][]data.Order{
		"start_date":  {data.Asc("start_date")},
		"-start_date": {data.Desc("start_date")},
		"number":      {data.Asc("number")},
		"-number":     {data.Desc("number")},
	},
	Default:  []data.Order{data.Desc("start_date")},
	Tiebreak: data.Desc("id"),
})

// leaveQuery turns the API filter into a query over leaveList.
func leaveQuery(sc data.Scope, f LeaveFilter) data.Query {
	from, to := platform.NullDate(platform.Optional(f.From)), platform.NullDate(platform.Optional(f.To))
	return data.Query{
		Scope: sc, Q: f.Q, Sort: f.Sort, Paging: f.Paging,
		Where: data.And(
			data.When(f.Status != "", data.Eq("status", f.Status)),
			data.When(f.OrgUnitID != 0, data.Eq("org_unit_id", f.OrgUnitID)),
			data.When(f.EmployeeID != 0, data.Eq("employee_id", f.EmployeeID)),
			data.When(from.Valid, data.Gte("end_date", from)),
			data.When(to.Valid, data.Lte("start_date", to)),
		),
	}
}
