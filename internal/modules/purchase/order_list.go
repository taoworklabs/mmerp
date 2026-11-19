package purchase

import (
	"github.com/taoworklabs/mmerp/internal/modules/purchase/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform/data"
)

// orderList shows the purchase orders of org units within the actor's scope.
var orderList = data.Register(data.List[store.OrderListBaseRow]{
	Name:   "purchase.OrderList",
	Base:   store.OrderListBase,
	Scope:  data.ByColumn("org_unit_id"),
	Search: []string{"number", "supplier_code", "supplier_name"},
	Sorts: map[string][]data.Order{
		"date":    {data.Asc("date")},
		"-date":   {data.Desc("date")},
		"number":  {data.Asc("number")},
		"-number": {data.Desc("number")},
		"total":   {data.Asc("total")},
		"-total":  {data.Desc("total")},
	},
	Default:  []data.Order{data.Desc("date")},
	Tiebreak: data.Desc("id"),
})

// orderQuery turns the API filter into a query over orderList.
func orderQuery(sc data.Scope, f OrderFilter) data.Query {
	return data.Query{
		Scope: sc, Q: f.Q, Sort: f.Sort, Paging: f.Paging,
		Where: data.And(
			data.When(f.Status != "", data.Eq("status", f.Status)),
			data.When(f.SupplierID != 0, data.Eq("supplier_id", f.SupplierID)),
			data.When(f.ReceiptState != "", data.Eq("receipt_state", f.ReceiptState)),
		),
	}
}
