package hrm

import (
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform/data"
)

// timesheetList shows timesheets of the org units in the actor's scope, latest
// period first; sheets that tie on the sort key go by unit name.
var timesheetList = data.Register(data.List[store.TimesheetListBaseRow]{
	Name:  "hrm.TimesheetList",
	Base:  store.TimesheetListBase,
	Scope: data.ByColumn("org_unit_id"),
	Sorts: map[string][]data.Order{
		"period":  {data.Asc("period_start"), data.Asc("org_unit_name")},
		"-period": {data.Desc("period_start"), data.Asc("org_unit_name")},
		"number":  {data.Asc("number"), data.Asc("org_unit_name")},
		"-number": {data.Desc("number"), data.Asc("org_unit_name")},
	},
	Default:  []data.Order{data.Desc("period_start"), data.Asc("org_unit_name")},
	Tiebreak: data.Desc("id"),
})

// timesheetQuery turns the API filter into a query over timesheetList.
func timesheetQuery(sc data.Scope, f TimesheetFilter, start pgtype.Date) data.Query {
	return data.Query{
		Scope: sc, Paging: f.Paging, Sort: f.Sort,
		Where: data.And(
			data.When(f.Status != "", data.Eq("status", f.Status)),
			data.When(f.OrgUnitID != 0, data.Eq("org_unit_id", f.OrgUnitID)),
			data.When(start.Valid, data.Eq("period_start", start)),
		),
	}
}
