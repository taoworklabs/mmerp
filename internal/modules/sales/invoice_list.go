package sales

import (
	"context"

	"github.com/taoworklabs/mmerp/internal/modules/sales/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
)

// invoiceList shows the invoices of org units within the actor's scope.
var invoiceList = data.Register(data.List[store.InvoiceListBaseRow]{
	Name:   "sales.InvoiceList",
	Base:   store.InvoiceListBase,
	Scope:  data.ByColumn("org_unit_id"),
	Search: []string{"number", "order_number", "customer_name"},
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

func (s *Service) Invoices(ctx context.Context, f InvoiceFilter) (InvoiceList, error) {
	out := InvoiceList{Items: []InvoiceListItem{}}
	sc, err := s.d.IAM.DataScope(ctx, "sales", PermOrderView)
	if err != nil {
		return out, err
	}
	page, err := invoiceList.Run(ctx, data.Query{
		Scope: sc, Q: f.Q, Sort: f.Sort, Paging: f.Paging,
		Where: data.And(
			data.When(f.Status != "", data.Eq("status", f.Status)),
			data.When(f.OrderID != 0, data.Eq("order_id", f.OrderID)),
			data.When(f.CustomerID != 0, data.Eq("customer_id", f.CustomerID)),
		),
	})
	if err != nil {
		return out, err
	}
	out.Total = page.Total
	for _, r := range page.Items {
		out.Items = append(out.Items, InvoiceListItem{ID: r.ID, Number: r.Number, Status: r.Status, Date: *platform.DatePtr(r.Date),
			OrderID: r.OrderID, OrderNumber: r.OrderNumber, CustomerID: r.CustomerID, CustomerName: r.CustomerName,
			Total: r.Total, OrgUnitName: r.OrgUnitName})
	}
	return out, nil
}
