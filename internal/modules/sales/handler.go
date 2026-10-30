package sales

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type actionsOutput struct{ Body Actions }

type lineItemsInput struct {
	Q string `query:"q" maxLength:"100"`
}

type lineItemsOutput struct{ Body LineItemList }

func registerHandlers(api huma.API, s *Service) {
	registerCustomerHandlers(api, s)
	registerDocHandlers(api, s)
	registerDeliveryHandlers(api, s)
	huma.Register(api, huma.Operation{OperationID: "list-line-items", Method: http.MethodGet, Path: "/sales/line-items"},
		func(ctx context.Context, in *lineItemsInput) (*lineItemsOutput, error) {
			l, err := s.LineItems(ctx, in.Q)
			return &lineItemsOutput{Body: l}, err
		})
}
