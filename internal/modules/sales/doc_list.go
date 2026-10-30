package sales

import (
	"github.com/taoworklabs/mmerp/internal/modules/sales/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform/data"
)

// docList shows the quotations or orders of org units within the actor's scope.
var docList = data.Register(data.List[store.DocListBaseRow]{
	Name:   "sales.DocList",
	Base:   store.DocListBase,
	Scope:  data.ByColumn("org_unit_id"),
	Search: []string{"number", "customer_code", "customer_name"},
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

// docQuery turns the API filter into a query over docList for one kind.
func docQuery(sc data.Scope, k kind, f DocFilter) data.Query {
	return data.Query{
		Scope: sc, Q: f.Q, Sort: f.Sort, Paging: f.Paging,
		Where: data.And(
			data.Eq("kind", k.name),
			data.When(f.Status != "", data.Eq("status", f.Status)),
			data.When(f.CustomerID != 0, data.Eq("customer_id", f.CustomerID)),
			data.When(f.DeliveryState != "", data.Eq("delivery_state", f.DeliveryState)),
		),
	}
}
