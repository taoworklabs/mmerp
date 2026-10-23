package sales

import (
	"github.com/taoworklabs/mmerp/internal/modules/sales/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform/data"
)

// customerList shows the customers owned by org units within the actor's scope.
var customerList = data.Register(data.List[store.CustomerListBaseRow]{
	Name:   "sales.CustomerList",
	Base:   store.CustomerListBase,
	Scope:  data.ByColumn("org_unit_id"),
	Search: []string{"code", "name", "tax_code"},
	Sorts: map[string][]data.Order{
		"code":  {data.Asc("code").Lower()},
		"-code": {data.Desc("code").Lower()},
		"name":  {data.Asc("name")},
		"-name": {data.Desc("name")},
	},
	Default:  []data.Order{data.Asc("code").Lower()},
	Tiebreak: data.Asc("id"),
})

// customerQuery turns the API filter into a query over customerList.
func customerQuery(sc data.Scope, f CustomerFilter) data.Query {
	a := boolFilter(f.Active)
	return data.Query{
		Scope: sc, Q: f.Q, Sort: f.Sort, Paging: f.Paging,
		Where: data.When(a.Valid, data.Eq("active", a.Bool)),
	}
}
