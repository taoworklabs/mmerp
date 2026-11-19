package purchase

import (
	"context"

	"github.com/taoworklabs/mmerp/internal/modules/purchase/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
)

// paymentList shows the payments of org units within the actor's scope.
var paymentList = data.Register(data.List[store.PaymentListBaseRow]{
	Name:   "purchase.PaymentList",
	Base:   store.PaymentListBase,
	Scope:  data.ByColumn("org_unit_id"),
	Search: []string{"number", "supplier_name"},
	Sorts: map[string][]data.Order{
		"date":    {data.Asc("date")},
		"-date":   {data.Desc("date")},
		"number":  {data.Asc("number")},
		"-number": {data.Desc("number")},
		"amount":  {data.Asc("amount")},
		"-amount": {data.Desc("amount")},
	},
	Default:  []data.Order{data.Desc("date")},
	Tiebreak: data.Desc("id"),
})

// Payments lists the supplier payments of org units where the actor may view orders.
func (s *Service) Payments(ctx context.Context, f PaymentFilter) (PaymentList, error) {
	out := PaymentList{Items: []PaymentListItem{}}
	sc, err := s.d.IAM.DataScope(ctx, "purchase", PermOrderView)
	if err != nil {
		return out, err
	}
	page, err := paymentList.Run(ctx, data.Query{
		Scope: sc, Q: f.Q, Sort: f.Sort, Paging: f.Paging,
		Where: data.And(
			data.When(f.Status != "", data.Eq("status", f.Status)),
			data.When(f.SupplierID != 0, data.Eq("supplier_id", f.SupplierID)),
			data.When(f.Method != "", data.Eq("method", f.Method)),
		),
	})
	if err != nil {
		return out, err
	}
	out.Total = page.Total
	for _, r := range page.Items {
		out.Items = append(out.Items, PaymentListItem{ID: r.ID, Number: r.Number, Status: r.Status, Date: *platform.DatePtr(r.Date),
			SupplierID: r.SupplierID, SupplierName: r.SupplierName, Method: r.Method, Amount: r.Amount, OrgUnitName: r.OrgUnitName})
	}
	return out, nil
}
