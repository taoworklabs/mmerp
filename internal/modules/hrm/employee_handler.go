package hrm

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type employeesOutput struct{ Body EmployeeList }

type employeeOutput struct{ Body Employee }

type employeeInput struct{ Body NewEmployee }

type updateEmployeeInput struct {
	ID   int64 `path:"id"`
	Body EmployeeInput
}

type sensitiveInput struct {
	ID    int64  `path:"id"`
	Field string `path:"field" enum:"national_id,social_insurance_no,tax_code,bank_account"`
}

type sensitiveOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         struct {
		Value *string `json:"value"`
	}
}

type dependentsOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         []Dependent
}

type createDependentInput struct {
	ID   int64 `path:"id"`
	Body DependentInput
}

type dependentInput struct {
	ID          int64 `path:"id"`
	DependentID int64 `path:"dependent"`
	Body        DependentInput
}

type dependentRefInput struct {
	ID          int64 `path:"id"`
	DependentID int64 `path:"dependent"`
}

func registerEmployeeHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-employees", Method: http.MethodGet, Path: "/hrm/employees"},
		func(ctx context.Context, in *EmployeeFilter) (*employeesOutput, error) {
			list, err := s.Employees(ctx, *in)
			return &employeesOutput{Body: list}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-employee-actions", Method: http.MethodGet, Path: "/hrm/employees/actions",
		Description: "What the actor may do before picking an employee: create, view_sensitive, import_leave_balances, import_employees."},
		func(ctx context.Context, _ *struct{}) (*platform.ActionsOutput, error) {
			actions, err := s.EmployeeActions(ctx)
			out := &platform.ActionsOutput{}
			out.Body.AllowedActions = actions
			return out, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-employee", Method: http.MethodPost, Path: "/hrm/employees", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *employeeInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.CreateEmployee(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-employee", Method: http.MethodGet, Path: "/hrm/employees/{id}"},
		func(ctx context.Context, in *platform.IDInput) (*employeeOutput, error) {
			e, err := s.Employee(ctx, in.ID)
			return &employeeOutput{Body: e}, err
		})
	huma.Register(api, huma.Operation{OperationID: "update-employee", Method: http.MethodPut, Path: "/hrm/employees/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateEmployeeInput) (*struct{}, error) {
			return nil, s.UpdateEmployee(ctx, in.ID, in.Body)
		})
	huma.Register(api, huma.Operation{OperationID: "reveal-employee-sensitive", Method: http.MethodGet, Path: "/hrm/employees/{id}/sensitive/{field}",
		Description: "Each call is audited."},
		func(ctx context.Context, in *sensitiveInput) (*sensitiveOutput, error) {
			v, err := s.RevealSensitive(ctx, in.ID, in.Field)
			out := &sensitiveOutput{CacheControl: "no-store"}
			out.Body.Value = v
			return out, err
		})
	huma.Register(api, huma.Operation{OperationID: "list-dependents", Method: http.MethodGet, Path: "/hrm/employees/{id}/dependents",
		Description: "Each call is audited."},
		func(ctx context.Context, in *platform.IDInput) (*dependentsOutput, error) {
			ds, err := s.Dependents(ctx, in.ID)
			return &dependentsOutput{CacheControl: "no-store", Body: ds}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-dependent", Method: http.MethodPost, Path: "/hrm/employees/{id}/dependents", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *createDependentInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.SaveDependent(ctx, in.ID, 0, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "update-dependent", Method: http.MethodPut, Path: "/hrm/employees/{id}/dependents/{dependent}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *dependentInput) (*struct{}, error) {
			_, err := s.SaveDependent(ctx, in.ID, in.DependentID, in.Body)
			return nil, err
		})
	huma.Register(api, huma.Operation{OperationID: "delete-dependent", Method: http.MethodDelete, Path: "/hrm/employees/{id}/dependents/{dependent}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *dependentRefInput) (*struct{}, error) {
			return nil, s.DeleteDependent(ctx, in.ID, in.DependentID)
		})
}
