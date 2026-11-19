package purchase

import (
	"context"

	"github.com/taoworklabs/mmerp/internal/modules/purchase/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
)

// invoiceList shows the supplier invoices of org units within the actor's scope.
var invoiceList = data.Register(data.List[store.InvoiceListBaseRow]{
	Name:   "purchase.InvoiceList",
	Base:   store.InvoiceListBase,
	Scope:  data.ByColumn("org_unit_id"),
	Search: []string{"number", "order_number", "supplier_name", "supplier_invoice_symbol", "supplier_invoice_no"},
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

// Invoices lists the supplier invoices of org units where the actor may view orders.
func (s *Service) Invoices(ctx context.Context, f InvoiceFilter) (InvoiceList, error) {
	out := InvoiceList{Items: []InvoiceListItem{}}
	sc, err := s.d.IAM.DataScope(ctx, "purchase", PermOrderView)
	if err != nil {
		return out, err
	}
	page, err := invoiceList.Run(ctx, data.Query{
		Scope: sc, Q: f.Q, Sort: f.Sort, Paging: f.Paging,
		Where: data.And(
			data.When(f.Status != "", data.Eq("status", f.Status)),
			data.When(f.OrderID != 0, data.Eq("order_id", f.OrderID)),
			data.When(f.SupplierID != 0, data.Eq("supplier_id", f.SupplierID)),
			data.When(f.PaymentState != "", data.Eq("payment_state", f.PaymentState)),
		),
	})
	if err != nil {
		return out, err
	}
	out.Total = page.Total
	for _, r := range page.Items {
		out.Items = append(out.Items, InvoiceListItem{ID: r.ID, Number: r.Number, Status: r.Status, Date: *platform.DatePtr(r.Date),
			OrderID: r.OrderID, OrderNumber: r.OrderNumber, SupplierID: r.SupplierID, SupplierName: r.SupplierName,
			SupplierInvoiceSymbol: platform.TextPtr(r.SupplierInvoiceSymbol), SupplierInvoiceNo: platform.TextPtr(r.SupplierInvoiceNo),
			Total: r.Total, OrgUnitName: r.OrgUnitName, PaymentState: r.PaymentState})
	}
	return out, nil
}
