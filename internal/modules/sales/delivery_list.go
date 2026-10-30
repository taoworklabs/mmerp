package sales

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/modules/sales/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
)

// deliveryList shows the deliveries of org units within the actor's scope.
var deliveryList = data.Register(data.List[store.DeliveryListBaseRow]{
	Name:   "sales.DeliveryList",
	Base:   store.DeliveryListBase,
	Scope:  data.ByColumn("org_unit_id"),
	Search: []string{"number", "order_number", "customer_name"},
	Sorts: map[string][]data.Order{
		"date":    {data.Asc("date")},
		"-date":   {data.Desc("date")},
		"number":  {data.Asc("number")},
		"-number": {data.Desc("number")},
	},
	Default:  []data.Order{data.Desc("date")},
	Tiebreak: data.Desc("id"),
})

func (s *Service) Deliveries(ctx context.Context, f DeliveryFilter) (DeliveryList, error) {
	out := DeliveryList{Items: []DeliveryListItem{}}
	sc, err := s.d.IAM.DataScope(ctx, "sales", PermOrderView)
	if err != nil {
		return out, err
	}
	page, err := deliveryList.Run(ctx, data.Query{
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
		out.Items = append(out.Items, DeliveryListItem{ID: r.ID, Number: r.Number, Status: r.Status, Date: *platform.DatePtr(r.Date),
			OrderID: r.OrderID, OrderNumber: r.OrderNumber, CustomerID: r.CustomerID, CustomerName: r.CustomerName,
			WarehouseID: r.WarehouseID, WarehouseName: r.WarehouseName, OrgUnitName: r.OrgUnitName})
	}
	return out, nil
}

// Availability is, per stock item, what the legal entity of the org unit has on hand, what
// its posted orders still have to deliver, and the difference; it needs order view there.
// Items without stock are left out.
func (s *Service) Availability(ctx context.Context, orgUnit int64, items []int64) ([]ItemAvailability, error) {
	if err := s.require(ctx, PermOrderView, orgUnit); err != nil {
		return nil, err
	}
	le, err := s.d.IAM.LegalEntityOf(ctx, orgUnit)
	if err != nil {
		return nil, err
	}
	return s.availability(ctx, le, items)
}

func (s *Service) availability(ctx context.Context, legalEntity int64, items []int64) ([]ItemAvailability, error) {
	rows, err := store.Availability.Run(ctx, store.AvailabilityParams{LegalEntityID: legalEntity, ItemIds: items})
	if err != nil {
		return nil, err
	}
	out := make([]ItemAvailability, len(rows))
	for i, r := range rows {
		on, err := decimal.NewFromString(r.OnHand)
		if err != nil {
			return nil, err
		}
		committed, err := decimal.NewFromString(r.Committed)
		if err != nil {
			return nil, err
		}
		out[i] = ItemAvailability{ItemID: r.ItemID, OnHand: r.OnHand, Committed: r.Committed, Available: on.Sub(committed).StringFixed(3)}
	}
	return out, nil
}
