package hrm

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type payrollsOutput struct{ Body PayrollList }

type payrollOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         Payroll
}

type newPayrollInput struct{ Body NewPayroll }

type payrollJobOutput struct{ Body PayrollJob }

type jobOutput struct {
	Body struct {
		JobID int64 `json:"job_id"`
	}
}

type computeInput struct {
	ID   int64 `path:"id"`
	Body struct {
		Version int32 `json:"version"`
	}
}

type adjustmentsInput struct {
	ID   int64 `path:"id"`
	Body PayrollAdjustments
}

func queued(id int64, err error) (*jobOutput, error) {
	out := &jobOutput{}
	out.Body.JobID = id
	return out, err
}

func registerPayrollHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-payrolls", Method: http.MethodGet, Path: "/hrm/payrolls"},
		func(ctx context.Context, in *PayrollFilter) (*payrollsOutput, error) {
			l, err := s.Payrolls(ctx, *in)
			return &payrollsOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-payroll-actions", Method: http.MethodGet, Path: "/hrm/payrolls/actions",
		Description: "What the actor may do before picking a payroll: create."},
		func(ctx context.Context, _ *struct{}) (*platform.ActionsOutput, error) {
			actions, err := s.PayrollActions(ctx)
			out := &platform.ActionsOutput{}
			out.Body.AllowedActions = actions
			return out, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-payroll", Method: http.MethodPost, Path: "/hrm/payrolls", DefaultStatus: http.StatusCreated,
		Description: "Creates a draft and queues its computation; follow job_id."},
		func(ctx context.Context, in *newPayrollInput) (*payrollJobOutput, error) {
			p, err := s.CreatePayroll(ctx, in.Body)
			return &payrollJobOutput{Body: p}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-payroll", Method: http.MethodGet, Path: "/hrm/payrolls/{id}",
		Description: "Each employee's pay comes with hrm.salary.view; each such read is audited."},
		func(ctx context.Context, in *platform.IDInput) (*payrollOutput, error) {
			p, err := s.Payroll(ctx, in.ID)
			return &payrollOutput{CacheControl: "no-store", Body: p}, err
		})
	huma.Register(api, huma.Operation{OperationID: "compute-payroll", Method: http.MethodPost, Path: "/hrm/payrolls/{id}/compute", DefaultStatus: http.StatusAccepted,
		Description: "Queues a computation of a draft from the current sources."},
		func(ctx context.Context, in *computeInput) (*jobOutput, error) {
			return queued(s.RecomputePayroll(ctx, in.ID, in.Body.Version))
		})
	huma.Register(api, huma.Operation{OperationID: "save-payroll-adjustments", Method: http.MethodPut, Path: "/hrm/payrolls/{id}/adjustments", DefaultStatus: http.StatusAccepted,
		Description: "Replaces every adjustment of a draft and queues a computation."},
		func(ctx context.Context, in *adjustmentsInput) (*jobOutput, error) {
			return queued(s.SavePayrollAdjustments(ctx, in.ID, in.Body))
		})
}
