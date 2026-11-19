package purchase

import (
	"context"

	"github.com/taoworklabs/mmerp/internal/modules/purchase/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
)

// goodsReceiptList shows the goods receipts of org units within the actor's scope.
var goodsReceiptList = data.Register(data.List[store.GoodsReceiptListBaseRow]{
	Name:   "purchase.GoodsReceiptList",
	Base:   store.GoodsReceiptListBase,
	Scope:  data.ByColumn("org_unit_id"),
	Search: []string{"number", "order_number", "supplier_name"},
	Sorts: map[string][]data.Order{
		"date":    {data.Asc("date")},
		"-date":   {data.Desc("date")},
		"number":  {data.Asc("number")},
		"-number": {data.Desc("number")},
	},
	Default:  []data.Order{data.Desc("date")},
	Tiebreak: data.Desc("id"),
})

// GoodsReceipts lists the goods receipts of org units where the actor may view orders.
func (s *Service) GoodsReceipts(ctx context.Context, f GoodsReceiptFilter) (GoodsReceiptList, error) {
	out := GoodsReceiptList{Items: []GoodsReceiptListItem{}}
	sc, err := s.d.IAM.DataScope(ctx, "purchase", PermOrderView)
	if err != nil {
		return out, err
	}
	page, err := goodsReceiptList.Run(ctx, data.Query{
		Scope: sc, Q: f.Q, Sort: f.Sort, Paging: f.Paging,
		Where: data.And(
			data.When(f.Status != "", data.Eq("status", f.Status)),
			data.When(f.OrderID != 0, data.Eq("order_id", f.OrderID)),
			data.When(f.WarehouseID != 0, data.Eq("warehouse_id", f.WarehouseID)),
		),
	})
	if err != nil {
		return out, err
	}
	out.Total = page.Total
	for _, r := range page.Items {
		out.Items = append(out.Items, GoodsReceiptListItem{ID: r.ID, Number: r.Number, Status: r.Status, Date: *platform.DatePtr(r.Date),
			OrderID: r.OrderID, OrderNumber: r.OrderNumber, SupplierID: r.SupplierID, SupplierName: r.SupplierName,
			WarehouseID: r.WarehouseID, WarehouseName: r.WarehouseName, OrgUnitName: r.OrgUnitName})
	}
	return out, nil
}
