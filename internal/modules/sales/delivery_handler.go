package sales

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type deliveryOutput struct{ Body Delivery }

type newDeliveryInput struct{ Body NewDelivery }

type deliveriesOutput struct{ Body DeliveryList }

type availabilityInput struct {
	OrgUnitID int64   `query:"org_unit_id" required:"true"`
	ItemIDs   []int64 `query:"item_id,explode" required:"true" minItems:"1" maxItems:"200"`
}

type availabilityOutput struct {
	Body struct {
		Items []ItemAvailability `json:"items" nullable:"false"`
	}
}

type updateDeliveryInput struct {
	ID   int64 `path:"id"`
	Body DeliveryUpdate
}

type deliveryOptionsInput struct {
	ID          int64 `path:"id"`
	WarehouseID int64 `query:"warehouse_id" doc:"Default warehouse of the order's legal entity when absent"`
}

type deliveryOptionsOutput struct{ Body DeliveryOptions }

func registerDeliveryHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "get-delivery-options", Method: http.MethodGet, Path: "/sales/orders/{id}/delivery-options"},
		func(ctx context.Context, in *deliveryOptionsInput) (*deliveryOptionsOutput, error) {
			o, err := s.DeliveryOptions(ctx, in.ID, in.WarehouseID)
			return &deliveryOptionsOutput{Body: o}, err
		})
	huma.Register(api, huma.Operation{OperationID: "list-deliveries", Method: http.MethodGet, Path: "/sales/deliveries"},
		func(ctx context.Context, in *DeliveryFilter) (*deliveriesOutput, error) {
			l, err := s.Deliveries(ctx, *in)
			return &deliveriesOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-availability", Method: http.MethodGet, Path: "/sales/availability",
		Description: "Per stock item, in the legal entity of the org unit; items without stock are left out."},
		func(ctx context.Context, in *availabilityInput) (*availabilityOutput, error) {
			items, err := s.Availability(ctx, in.OrgUnitID, in.ItemIDs)
			out := &availabilityOutput{}
			out.Body.Items = items
			return out, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-delivery", Method: http.MethodPost, Path: "/sales/deliveries", DefaultStatus: http.StatusCreated,
		Description: "Sending the same request_id again returns the delivery created the first time."},
		func(ctx context.Context, in *newDeliveryInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.CreateDelivery(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-delivery", Method: http.MethodGet, Path: "/sales/deliveries/{id}"},
		func(ctx context.Context, in *platform.IDInput) (*deliveryOutput, error) {
			d, err := s.Delivery(ctx, in.ID)
			return &deliveryOutput{Body: d}, err
		})
	huma.Register(api, huma.Operation{OperationID: "update-delivery", Method: http.MethodPut, Path: "/sales/deliveries/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateDeliveryInput) (*struct{}, error) {
			return nil, s.UpdateDelivery(ctx, in.ID, in.Body)
		})
}
