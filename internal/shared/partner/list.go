package partner

import (
	"github.com/taoworklabs/mmerp/internal/platform/data"
	"github.com/taoworklabs/mmerp/internal/shared/partner/internal/store"
)

// partnerList: partners are tenant-wide, open to anyone who may view them somewhere.
var partnerList = data.Register(data.List[store.PartnerPartner]{
	Name:   "partner.PartnerList",
	Base:   store.PartnerListBase,
	Scope:  data.Unscoped("tenant-wide master data"),
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

// partnerQuery turns the API filter into a query over partnerList.
func partnerQuery(f PartnerFilter) data.Query {
	return data.Query{
		Q: f.Q, Sort: f.Sort, Paging: f.Paging,
		Where: data.When(f.TaxCode != "", data.SQL("lower(btrim(tax_code)) = lower(btrim(").Arg(f.TaxCode).Text("))").Cond()),
	}
}
