package stock

import (
	"github.com/taoworklabs/mmerp/internal/platform/data"
	"github.com/taoworklabs/mmerp/internal/shared/stock/internal/store"
)

// transferList shows the transfers of legal entities within the actor's scope.
var transferList = data.Register(data.List[store.TransferListBaseRow]{
	Name:   "stock.TransferList",
	Base:   store.TransferListBase,
	Scope:  data.ByColumn("org_unit_id"),
	Search: []string{"number", "note"},
	Sorts: map[string][]data.Order{
		"date":    {data.Asc("date")},
		"-date":   {data.Desc("date")},
		"number":  {data.Asc("number")},
		"-number": {data.Desc("number")},
	},
	Default:  []data.Order{data.Desc("date")},
	Tiebreak: data.Desc("id"),
})

func transferQuery(sc data.Scope, f TransferFilter) data.Query {
	return data.Query{
		Scope: sc, Q: f.Q, Sort: f.Sort, Paging: f.Paging,
		Where: data.And(
			data.When(f.Status != "", data.Eq("status", f.Status)),
			data.When(f.WarehouseID != 0, data.Or(data.Eq("from_warehouse_id", f.WarehouseID), data.Eq("to_warehouse_id", f.WarehouseID))),
		),
	}
}
