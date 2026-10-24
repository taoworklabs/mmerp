package hrm

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type leaveTypesOutput struct{ Body []LeaveType }

type leaveTypeInput struct{ Body LeaveTypeInput }

type updateLeaveTypeInput struct {
	ID   int64 `path:"id"`
	Body LeaveTypeInput
}

type leavesOutput struct{ Body LeaveList }

type leaveOutput struct{ Body Leave }

type newLeaveInput struct{ Body NewLeave }

type leaveWorkingDaysInput struct {
	Start      string `query:"start" format:"date" required:"true"`
	End        string `query:"end" format:"date" required:"true"`
	EmployeeID int64  `query:"employee_id" doc:"Whose calendar; omit for the actor's own record"`
}

type leaveWorkingDaysOutput struct {
	Body struct {
		WorkingDays int `json:"working_days"`
	}
}

type updateLeaveInput struct {
	ID   int64 `path:"id"`
	Body LeaveUpdate
}

type balancesOutput struct{ Body []LeaveBalance }

type adjustBalanceInput struct {
	ID   int64 `path:"id"`
	Year int32 `path:"year" minimum:"2000" maximum:"2100"`
	Body BalanceAdjustment
}

func registerLeaveHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-leave-types", Method: http.MethodGet, Path: "/hrm/leave-types"},
		func(ctx context.Context, _ *struct{}) (*leaveTypesOutput, error) {
			ts, err := s.LeaveTypes(ctx)
			return &leaveTypesOutput{Body: ts}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-leave-type", Method: http.MethodPost, Path: "/hrm/leave-types", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *leaveTypeInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.SaveLeaveType(ctx, 0, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "update-leave-type", Method: http.MethodPut, Path: "/hrm/leave-types/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateLeaveTypeInput) (*struct{}, error) {
			_, err := s.SaveLeaveType(ctx, in.ID, in.Body)
			return nil, err
		})
	huma.Register(api, huma.Operation{OperationID: "list-leaves", Method: http.MethodGet, Path: "/hrm/leaves"},
		func(ctx context.Context, in *LeaveFilter) (*leavesOutput, error) {
			l, err := s.Leaves(ctx, *in)
			return &leavesOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-leave-actions", Method: http.MethodGet, Path: "/hrm/leaves/actions",
		Description: "What the actor may do before picking a leave request, and their own employee record."},
		func(ctx context.Context, _ *struct{}) (*selfServiceActionsOutput, error) {
			a, err := s.LeaveActions(ctx)
			return &selfServiceActionsOutput{Body: a}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-leave-working-days", Method: http.MethodGet, Path: "/hrm/leaves/working-days",
		Description: "How many days a request over this range may claim: the days that are neither a weekly rest day nor a public holiday."},
		func(ctx context.Context, in *leaveWorkingDaysInput) (*leaveWorkingDaysOutput, error) {
			var employee *int64
			if in.EmployeeID != 0 {
				employee = &in.EmployeeID
			}
			n, err := s.LeaveWorkingDays(ctx, employee, in.Start, in.End)
			out := &leaveWorkingDaysOutput{}
			out.Body.WorkingDays = n
			return out, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-leave", Method: http.MethodPost, Path: "/hrm/leaves", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *newLeaveInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.CreateLeave(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-leave", Method: http.MethodGet, Path: "/hrm/leaves/{id}"},
		func(ctx context.Context, in *platform.IDInput) (*leaveOutput, error) {
			l, err := s.Leave(ctx, in.ID)
			return &leaveOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "update-leave", Method: http.MethodPut, Path: "/hrm/leaves/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateLeaveInput) (*struct{}, error) {
			return nil, s.UpdateLeave(ctx, in.ID, in.Body)
		})
	huma.Register(api, huma.Operation{OperationID: "list-leave-balances", Method: http.MethodGet, Path: "/hrm/employees/{id}/leave-balances"},
		func(ctx context.Context, in *platform.IDInput) (*balancesOutput, error) {
			b, err := s.LeaveBalances(ctx, in.ID)
			return &balancesOutput{Body: b}, err
		})
	huma.Register(api, huma.Operation{OperationID: "adjust-leave-balance", Method: http.MethodPost, Path: "/hrm/employees/{id}/leave-balances/{year}/adjustments", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *adjustBalanceInput) (*struct{}, error) {
			return nil, s.AdjustLeaveBalance(ctx, in.ID, in.Year, in.Body)
		})
}
