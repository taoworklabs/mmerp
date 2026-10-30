package sales

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type deliveryOutput struct{ Body Delivery }

type newDeliveryInput struct{ Body NewDelivery }

type updateDeliveryInput struct {
	ID   int64 `path:"id"`
	Body DeliveryUpdate
}

func registerDeliveryHandlers(api huma.API, s *Service) {
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
