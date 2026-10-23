package hrm

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type overtimesOutput struct{ Body OvertimeList }

type overtimeOutput struct{ Body Overtime }

type newOvertimeInput struct{ Body NewOvertime }

type updateOvertimeInput struct {
	ID   int64 `path:"id"`
	Body OvertimeUpdate
}

func registerOvertimeHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-overtimes", Method: http.MethodGet, Path: "/hrm/overtimes"},
		func(ctx context.Context, in *OvertimeFilter) (*overtimesOutput, error) {
			l, err := s.Overtimes(ctx, *in)
			return &overtimesOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-overtime-actions", Method: http.MethodGet, Path: "/hrm/overtimes/actions",
		Description: "What the actor may do before picking an overtime request, and their own employee record."},
		func(ctx context.Context, _ *struct{}) (*selfServiceActionsOutput, error) {
			a, err := s.OvertimeActions(ctx)
			return &selfServiceActionsOutput{Body: a}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-overtime", Method: http.MethodPost, Path: "/hrm/overtimes", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *newOvertimeInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.CreateOvertime(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-overtime", Method: http.MethodGet, Path: "/hrm/overtimes/{id}"},
		func(ctx context.Context, in *platform.IDInput) (*overtimeOutput, error) {
			o, err := s.Overtime(ctx, in.ID)
			return &overtimeOutput{Body: o}, err
		})
	huma.Register(api, huma.Operation{OperationID: "update-overtime", Method: http.MethodPut, Path: "/hrm/overtimes/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateOvertimeInput) (*struct{}, error) {
			return nil, s.UpdateOvertime(ctx, in.ID, in.Body)
		})
}
