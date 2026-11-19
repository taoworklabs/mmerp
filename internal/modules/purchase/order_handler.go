package purchase

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type ordersOutput struct{ Body OrderList }

type orderOutput struct{ Body Order }

type newOrderInput struct{ Body NewOrder }

type updateOrderInput struct {
	ID   int64 `path:"id"`
	Body OrderUpdate
}

type lineItemsInput struct {
	Q string `query:"q" maxLength:"100"`
}

type lineItemsOutput struct{ Body LineItemList }

func registerOrderHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-purchase-orders", Method: http.MethodGet, Path: "/purchase/orders"},
		func(ctx context.Context, in *OrderFilter) (*ordersOutput, error) {
			l, err := s.Orders(ctx, *in)
			return &ordersOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-purchase-order-actions", Method: http.MethodGet, Path: "/purchase/orders/actions"},
		func(ctx context.Context, _ *struct{}) (*actionsOutput, error) {
			a, err := s.OrderActions(ctx)
			return &actionsOutput{Body: a}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-purchase-order", Method: http.MethodPost, Path: "/purchase/orders", DefaultStatus: http.StatusCreated,
		Description: "Sending the same request_id again returns the document created the first time."},
		func(ctx context.Context, in *newOrderInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.CreateOrder(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-purchase-order", Method: http.MethodGet, Path: "/purchase/orders/{id}"},
		func(ctx context.Context, in *platform.IDInput) (*orderOutput, error) {
			o, err := s.Order(ctx, in.ID)
			return &orderOutput{Body: o}, err
		})
	huma.Register(api, huma.Operation{OperationID: "update-purchase-order", Method: http.MethodPut, Path: "/purchase/orders/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateOrderInput) (*struct{}, error) {
			return nil, s.UpdateOrder(ctx, in.ID, in.Body)
		})
	huma.Register(api, huma.Operation{OperationID: "list-purchase-line-items", Method: http.MethodGet, Path: "/purchase/line-items"},
		func(ctx context.Context, in *lineItemsInput) (*lineItemsOutput, error) {
			l, err := s.LineItems(ctx, in.Q)
			return &lineItemsOutput{Body: l}, err
		})
}
