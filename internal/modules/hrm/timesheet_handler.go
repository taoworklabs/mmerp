package hrm

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type timesheetsOutput struct{ Body TimesheetList }

type timesheetOutput struct{ Body Timesheet }

type newTimesheetInput struct{ Body NewTimesheet }

type updateTimesheetInput struct {
	ID   int64 `path:"id"`
	Body TimesheetUpdate
}

func registerTimesheetHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-timesheets", Method: http.MethodGet, Path: "/hrm/timesheets"},
		func(ctx context.Context, in *TimesheetFilter) (*timesheetsOutput, error) {
			l, err := s.Timesheets(ctx, *in)
			return &timesheetsOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-timesheet-actions", Method: http.MethodGet, Path: "/hrm/timesheets/actions",
		Description: "What the actor may do before picking a timesheet: create."},
		func(ctx context.Context, _ *struct{}) (*platform.ActionsOutput, error) {
			actions, err := s.TimesheetActions(ctx)
			out := &platform.ActionsOutput{}
			out.Body.AllowedActions = actions
			return out, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-timesheet", Method: http.MethodPost, Path: "/hrm/timesheets", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *newTimesheetInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.CreateTimesheet(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-timesheet", Method: http.MethodGet, Path: "/hrm/timesheets/{id}"},
		func(ctx context.Context, in *platform.IDInput) (*timesheetOutput, error) {
			t, err := s.Timesheet(ctx, in.ID)
			return &timesheetOutput{Body: t}, err
		})
	huma.Register(api, huma.Operation{OperationID: "update-timesheet", Method: http.MethodPut, Path: "/hrm/timesheets/{id}",
		Description: "Replaces the period and every line of a draft.", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateTimesheetInput) (*struct{}, error) {
			return nil, s.SaveTimesheet(ctx, in.ID, in.Body)
		})
}
