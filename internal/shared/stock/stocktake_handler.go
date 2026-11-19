package stock

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type stocktakesOutput struct{ Body StocktakeList }

type onHandLinesOutput struct{ Body OnHandLines }

type stocktakeOutput struct{ Body Stocktake }

type newStocktakeInput struct{ Body NewStocktake }

type updateStocktakeInput struct {
	ID   int64 `path:"id"`
	Body StocktakeUpdate
}

func registerStocktakeHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-stock-stocktakes", Method: http.MethodGet, Path: "/stock/stocktakes"},
		func(ctx context.Context, in *StocktakeFilter) (*stocktakesOutput, error) {
			l, err := s.Stocktakes(ctx, *in)
			return &stocktakesOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-stock-stocktake-actions", Method: http.MethodGet, Path: "/stock/stocktakes/actions"},
		func(ctx context.Context, _ *struct{}) (*actionsOutput, error) {
			a, err := s.StocktakeActions(ctx)
			return &actionsOutput{Body: a}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-stock-stocktake", Method: http.MethodPost, Path: "/stock/stocktakes", DefaultStatus: http.StatusCreated,
		Description: "Sending the same request_id again returns the document created the first time."},
		func(ctx context.Context, in *newStocktakeInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.CreateStocktake(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-stock-stocktake", Method: http.MethodGet, Path: "/stock/stocktakes/{id}"},
		func(ctx context.Context, in *platform.IDInput) (*stocktakeOutput, error) {
			t, err := s.Stocktake(ctx, in.ID)
			return &stocktakeOutput{Body: t}, err
		})
	huma.Register(api, huma.Operation{OperationID: "update-stock-stocktake", Method: http.MethodPut, Path: "/stock/stocktakes/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateStocktakeInput) (*struct{}, error) {
			return nil, s.UpdateStocktake(ctx, in.ID, in.Body)
		})
	huma.Register(api, huma.Operation{OperationID: "list-stock-on-hand-lines", Method: http.MethodGet, Path: "/stock/warehouses/{id}/on-hand-lines",
		Description: "A warehouse's balances as stocktake lines counting what is on hand."},
		func(ctx context.Context, in *OnHandLinesFilter) (*onHandLinesOutput, error) {
			l, err := s.OnHandLines(ctx, in.ID, in.ItemID)
			return &onHandLinesOutput{Body: l}, err
		})
}
