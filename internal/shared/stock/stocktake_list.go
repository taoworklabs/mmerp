package stock

import (
	"github.com/taoworklabs/mmerp/internal/platform/data"
	"github.com/taoworklabs/mmerp/internal/shared/stock/internal/store"
)

// stocktakeList shows the stocktakes of legal entities within the actor's scope.
var stocktakeList = data.Register(data.List[store.StocktakeListBaseRow]{
	Name:   "stock.StocktakeList",
	Base:   store.StocktakeListBase,
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

func stocktakeQuery(sc data.Scope, f StocktakeFilter) data.Query {
	return data.Query{
		Scope: sc, Q: f.Q, Sort: f.Sort, Paging: f.Paging,
		Where: data.And(
			data.When(f.Status != "", data.Eq("status", f.Status)),
			data.When(f.WarehouseID != 0, data.Eq("warehouse_id", f.WarehouseID)),
		),
	}
}
