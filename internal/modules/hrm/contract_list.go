package hrm

import (
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform/data"
)

// contractList shows every contract and appendix in the actor's units. Open-ended
// contracts come last when the end date is sorted descending.
var contractList = data.Register(data.List[store.ContractListBaseRow]{
	Name:   "hrm.ContractList",
	Base:   store.ContractListBase,
	Scope:  data.ByColumn("org_unit_id"),
	Search: []string{"number", "employee_code", "employee_name"},
	Sorts: map[string][]data.Order{
		"start_date":  {data.Asc("start_date")},
		"-start_date": {data.Desc("start_date")},
		"end_date":    {data.Asc("end_date")},
		"-end_date":   {data.Desc("end_date").NullsLast()},
		"number":      {data.Asc("number")},
		"-number":     {data.Desc("number")},
	},
	Default:  []data.Order{data.Desc("start_date")},
	Tiebreak: data.Desc("id"),
})

// contractQuery turns the API filter into a query over contractList. Expiring:
// posted originals ending from today to today + 30 days.
func contractQuery(sc data.Scope, f ContractFilter, today pgtype.Date) data.Query {
	return data.Query{
		Scope: sc, Q: f.Q, Sort: f.Sort, Paging: f.Paging,
		Where: data.And(
			data.When(f.Status != "", data.Eq("status", f.Status)),
			data.When(f.ContractTypeID != 0, data.Eq("contract_type_id", f.ContractTypeID)),
			data.When(f.OrgUnitID != 0, data.Eq("org_unit_id", f.OrgUnitID)),
			data.When(f.EmployeeID != 0, data.Eq("employee_id", f.EmployeeID)),
			data.When(f.Expiring, data.SQL(`(b."parent_id" IS NULL AND b."status" = 'posted' AND b."end_date" BETWEEN `).
				Arg(today).Text(`::date AND `).Arg(today).Text(`::date + 30)`).Cond()),
		),
	}
}
