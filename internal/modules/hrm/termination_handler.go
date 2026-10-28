package hrm

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type terminationsOutput struct{ Body TerminationList }

type terminationOutput struct{ Body Termination }

type newTerminationInput struct{ Body NewTermination }

type updateTerminationInput struct {
	ID   int64 `path:"id"`
	Body TerminationUpdate
}

func registerTerminationHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-terminations", Method: http.MethodGet, Path: "/hrm/terminations"},
		func(ctx context.Context, in *TerminationFilter) (*terminationsOutput, error) {
			l, err := s.Terminations(ctx, *in)
			return &terminationsOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-termination", Method: http.MethodPost, Path: "/hrm/terminations", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *newTerminationInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.CreateTermination(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-termination", Method: http.MethodGet, Path: "/hrm/terminations/{id}"},
		func(ctx context.Context, in *platform.IDInput) (*terminationOutput, error) {
			t, err := s.Termination(ctx, in.ID)
			return &terminationOutput{Body: t}, err
		})
	huma.Register(api, huma.Operation{OperationID: "update-termination", Method: http.MethodPut, Path: "/hrm/terminations/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateTerminationInput) (*struct{}, error) {
			return nil, s.UpdateTermination(ctx, in.ID, in.Body)
		})
}
