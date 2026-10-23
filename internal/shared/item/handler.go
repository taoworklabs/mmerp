package item

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type itemsOutput struct{ Body ItemList }

type itemInput struct{ Body NewItem }

type updateItemInput struct {
	ID   int64 `path:"id"`
	Body ItemFields
}

func registerHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-items", Method: http.MethodGet, Path: "/items"},
		func(ctx context.Context, in *ItemFilter) (*itemsOutput, error) {
			l, err := s.Items(ctx, *in)
			return &itemsOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-item", Method: http.MethodPost, Path: "/items", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *itemInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.CreateItem(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "update-item", Method: http.MethodPut, Path: "/items/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateItemInput) (*struct{}, error) {
			return nil, s.UpdateItem(ctx, in.ID, in.Body)
		})
}
