package item

import (
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/platform/data"
	"github.com/taoworklabs/mmerp/internal/shared/item/internal/store"
)

// itemList is the catalogue: tenant-wide, open to anyone who may view it somewhere.
var itemList = data.Register(data.List[store.ItemItem]{
	Name:   "item.ItemList",
	Base:   store.ItemListBase,
	Scope:  data.Unscoped("tenant-wide catalogue"),
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

// itemQuery turns the API filter into a query over itemList.
func itemQuery(f ItemFilter) data.Query {
	a := pgtype.Bool{Bool: f.Active == "true", Valid: f.Active != ""}
	return data.Query{
		Q: f.Q, Sort: f.Sort, Paging: f.Paging,
		Where: data.When(a.Valid, data.Eq("active", a.Bool)),
	}
}
