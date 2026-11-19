package sales

import (
	"context"

	"github.com/taoworklabs/mmerp/internal/modules/sales/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
)

// receiptList shows the receipts of org units within the actor's scope.
var receiptList = data.Register(data.List[store.ReceiptListBaseRow]{
	Name:   "sales.ReceiptList",
	Base:   store.ReceiptListBase,
	Scope:  data.ByColumn("org_unit_id"),
	Search: []string{"number", "customer_name"},
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

func (s *Service) Receipts(ctx context.Context, f ReceiptFilter) (ReceiptList, error) {
	out := ReceiptList{Items: []ReceiptListItem{}}
	sc, err := s.d.IAM.DataScope(ctx, "sales", PermOrderView)
	if err != nil {
		return out, err
	}
	page, err := receiptList.Run(ctx, data.Query{
		Scope: sc, Q: f.Q, Sort: f.Sort, Paging: f.Paging,
		Where: data.And(
			data.When(f.Status != "", data.Eq("status", f.Status)),
			data.When(f.CustomerID != 0, data.Eq("customer_id", f.CustomerID)),
			data.When(f.Method != "", data.Eq("method", f.Method)),
		),
	})
	if err != nil {
		return out, err
	}
	out.Total = page.Total
	for _, r := range page.Items {
		out.Items = append(out.Items, ReceiptListItem{ID: r.ID, Number: r.Number, Status: r.Status, Date: *platform.DatePtr(r.Date),
			CustomerID: r.CustomerID, CustomerName: r.CustomerName, Method: r.Method, Amount: r.Amount, OrgUnitName: r.OrgUnitName})
	}
	return out, nil
}
