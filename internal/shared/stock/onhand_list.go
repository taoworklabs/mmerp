package stock

import (
	"github.com/taoworklabs/mmerp/internal/platform/data"
	"github.com/taoworklabs/mmerp/internal/shared/stock/internal/store"
)

// onHandList: tenant-wide, open to anyone who may view stock somewhere.
var onHandList = data.Register(data.List[store.OnHandListBaseRow]{
	Name:   "stock.OnHandList",
	Base:   store.OnHandListBase,
	Scope:  data.Unscoped("tenant-wide balances"),
	Search: []string{"item_code", "item_name", "lot_code", "serial_code"},
	Sorts:  map[string][]data.Order{},
	Default: []data.Order{data.Asc("warehouse_code").Lower(), data.Asc("item_code").Lower(), data.Asc("lot_expiry").NullsLast(),
		data.Asc("lot_code").Lower().NullsLast(), data.Asc("serial_code").Lower().NullsLast()},
	Tiebreak: data.Asc("key"),
})

func onHandQuery(f OnHandFilter) data.Query {
	return data.Query{
		Q: f.Q, Paging: f.Paging,
		Where: data.And(
			data.When(f.WarehouseID != 0, data.Eq("warehouse_id", f.WarehouseID)),
			data.When(f.ItemID != 0, data.Eq("item_id", f.ItemID)),
			data.When(f.ExpiringBefore != "", data.SQL("lot_expiry < ").Arg(f.ExpiringBefore).Text("::date").Cond()),
		),
	}
}
