package stock

import (
	"github.com/taoworklabs/mmerp/internal/platform/data"
	"github.com/taoworklabs/mmerp/internal/shared/stock/internal/store"
)

// adjustmentList shows the adjustments of legal entities within the actor's scope.
var adjustmentList = data.Register(data.List[store.AdjustmentListBaseRow]{
	Name:   "stock.AdjustmentList",
	Base:   store.AdjustmentListBase,
	Scope:  data.ByColumn("org_unit_id"),
	Search: []string{"number", "reason"},
	Sorts: map[string][]data.Order{
		"date":    {data.Asc("date")},
		"-date":   {data.Desc("date")},
		"number":  {data.Asc("number")},
		"-number": {data.Desc("number")},
	},
	Default:  []data.Order{data.Desc("date")},
	Tiebreak: data.Desc("id"),
})

func adjustmentQuery(sc data.Scope, f AdjustmentFilter) data.Query {
	return data.Query{
		Scope: sc, Q: f.Q, Sort: f.Sort, Paging: f.Paging,
		Where: data.And(
			data.When(f.Status != "", data.Eq("status", f.Status)),
			data.When(f.WarehouseID != 0, data.Eq("warehouse_id", f.WarehouseID)),
		),
	}
}
