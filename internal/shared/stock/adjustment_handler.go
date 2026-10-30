package stock

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type adjustmentsOutput struct{ Body AdjustmentList }

type adjustmentOutput struct{ Body Adjustment }

type actionsOutput struct{ Body AdjustmentActions }

type newAdjustmentInput struct{ Body NewAdjustment }

type updateAdjustmentInput struct {
	ID   int64 `path:"id"`
	Body AdjustmentUpdate
}

func registerAdjustmentHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-stock-adjustments", Method: http.MethodGet, Path: "/stock/adjustments"},
		func(ctx context.Context, in *AdjustmentFilter) (*adjustmentsOutput, error) {
			l, err := s.Adjustments(ctx, *in)
			return &adjustmentsOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-stock-adjustment-actions", Method: http.MethodGet, Path: "/stock/adjustments/actions"},
		func(ctx context.Context, _ *struct{}) (*actionsOutput, error) {
			a, err := s.AdjustmentActions(ctx)
			return &actionsOutput{Body: a}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-stock-adjustment", Method: http.MethodPost, Path: "/stock/adjustments", DefaultStatus: http.StatusCreated,
		Description: "Sending the same request_id again returns the document created the first time."},
		func(ctx context.Context, in *newAdjustmentInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.CreateAdjustment(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-stock-adjustment", Method: http.MethodGet, Path: "/stock/adjustments/{id}"},
		func(ctx context.Context, in *platform.IDInput) (*adjustmentOutput, error) {
			a, err := s.Adjustment(ctx, in.ID)
			return &adjustmentOutput{Body: a}, err
		})
	huma.Register(api, huma.Operation{OperationID: "update-stock-adjustment", Method: http.MethodPut, Path: "/stock/adjustments/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateAdjustmentInput) (*struct{}, error) {
			return nil, s.UpdateAdjustment(ctx, in.ID, in.Body)
		})
}
