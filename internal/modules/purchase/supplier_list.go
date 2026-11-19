package purchase

import (
	"github.com/taoworklabs/mmerp/internal/modules/purchase/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform/data"
)

// supplierList shows the suppliers owned by org units within the actor's scope.
var supplierList = data.Register(data.List[store.SupplierListBaseRow]{
	Name:   "purchase.SupplierList",
	Base:   store.SupplierListBase,
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

// supplierQuery turns the API filter into a query over supplierList.
func supplierQuery(sc data.Scope, f SupplierFilter) data.Query {
	a := boolFilter(f.Active)
	return data.Query{
		Scope: sc, Q: f.Q, Sort: f.Sort, Paging: f.Paging,
		Where: data.And(
			data.When(a.Valid, data.Eq("active", a.Bool)),
			data.When(f.PartnerID != 0, data.Eq("partner_id", f.PartnerID)),
		),
	}
}
