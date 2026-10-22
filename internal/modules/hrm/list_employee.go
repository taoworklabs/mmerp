package hrm

import (
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform/data"
)

// employeeList shows employees of the units in the actor's scope.
var employeeList = data.Register(data.List[store.EmployeeListBaseRow]{
	Name:   "hrm.EmployeeList",
	Base:   store.EmployeeListBase,
	Scope:  data.ByColumn("org_unit_id"),
	Search: []string{"code", "full_name"},
	Sorts: map[string][]data.Order{
		"code":       {data.Asc("code").Lower()},
		"-code":      {data.Desc("code").Lower()},
		"full_name":  {data.Asc("full_name")},
		"-full_name": {data.Desc("full_name")},
		"hire_date":  {data.Asc("hire_date")},
		"-hire_date": {data.Desc("hire_date")},
	},
	Default:  []data.Order{data.Asc("code").Lower()},
	Tiebreak: data.Asc("id"),
})

// employeeQuery turns the API filter into a query over employeeList. today is
// the tenant's date: an employee terminated today is still active.
func employeeQuery(sc data.Scope, f EmployeeFilter, today pgtype.Date) data.Query {
	return data.Query{
		Scope: sc, Q: f.Q, Sort: f.Sort, Paging: f.Paging,
		Where: data.And(
			data.When(f.OrgUnitID != 0, data.Eq("org_unit_id", f.OrgUnitID)),
			data.When(f.Status == "active", data.Or(data.IsNull("termination_date"), data.Gte("termination_date", today))),
			data.When(f.Status == "terminated", data.SQL(`b."termination_date" < `).Arg(today).Cond()),
			// manager_of drops that employee and everyone under them, who cannot become their manager.
			data.When(f.ManagerOf != 0, data.SQL(`b."id" NOT IN (WITH RECURSIVE under(id) AS (SELECT `).Arg(f.ManagerOf).
				Text(`::bigint UNION SELECT s.id FROM hrm.employees s JOIN under ON s.manager_id = under.id) SELECT id FROM under)`).Cond()),
		),
	}
}
