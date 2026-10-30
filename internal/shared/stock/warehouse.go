package stock

import (
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/platform/data"
	"github.com/taoworklabs/mmerp/internal/shared/stock/internal/store"
)

// warehouseList: tenant-wide, open to anyone who may view stock somewhere.
var warehouseList = data.Register(data.List[store.StockWarehouse]{
	Name:   "stock.WarehouseList",
	Base:   store.WarehouseListBase,
	Scope:  data.Unscoped("tenant-wide warehouses"),
	Search: []string{"code", "name"},
	Sorts: map[string][]data.Order{
		"code":  {data.Asc("code").Lower()},
		"-code": {data.Desc("code").Lower()},
		"name":  {data.Asc("name")},
		"-name": {data.Desc("name")},
	},
	Default:  []data.Order{data.Asc("code").Lower()},
	Tiebreak: data.Asc("id"),
})

func warehouseQuery(f WarehouseFilter) data.Query {
	a := pgtype.Bool{Bool: f.Active == "true", Valid: f.Active != ""}
	return data.Query{
		Q: f.Q, Sort: f.Sort, Paging: f.Paging,
		Where: data.When(a.Valid, data.Eq("active", a.Bool)),
	}
}
