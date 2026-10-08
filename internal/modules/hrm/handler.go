package hrm

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type employeesOutput struct{ Body EmployeeList }

type idInput struct {
	ID int64 `path:"id"`
}

type employeeOutput struct{ Body Employee }

type actionsOutput struct {
	Body struct {
		AllowedActions []string `json:"allowed_actions" nullable:"false"`
	}
}

type employeeInput struct{ Body EmployeeInput }

type updateEmployeeInput struct {
	ID   int64 `path:"id"`
	Body EmployeeInput
}

type createdOutput struct {
	Body struct {
		ID int64 `json:"id"`
	}
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

func created(id int64, err error) (*createdOutput, error) {
	if err != nil {
		return nil, err
	}
	out := &createdOutput{}
	out.Body.ID = id
	return out, nil
}

type leaveTypesOutput struct{ Body []LeaveType }

type leaveTypeInput struct{ Body LeaveTypeInput }

type updateLeaveTypeInput struct {
	ID   int64 `path:"id"`
	Body LeaveTypeInput
}

type leavesOutput struct{ Body LeaveList }

type selfServiceActionsOutput struct{ Body SelfServiceActions }

type leaveOutput struct{ Body Leave }

type newLeaveInput struct{ Body NewLeave }

type updateLeaveInput struct {
	ID   int64 `path:"id"`
	Body LeaveUpdate
}

type deleteDocInput struct {
	ID      int64 `path:"id"`
	Version int32 `query:"version" required:"true"`
}

type balancesOutput struct{ Body []LeaveBalance }

type adjustBalanceInput struct {
	ID   int64 `path:"id"`
	Year int32 `path:"year" minimum:"2000" maximum:"2100"`
	Body BalanceAdjustment
}

type contractTypesOutput struct{ Body []ContractType }

type contractTypeInput struct{ Body ContractTypeInput }

type updateContractTypeInput struct {
	ID   int64 `path:"id"`
	Body ContractTypeInput
}

type contractsOutput struct{ Body []ContractListItem }

type contractOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         Contract
}

type newContractInput struct{ Body NewContract }

type updateContractInput struct {
	ID   int64 `path:"id"`
	Body ContractUpdate
}

type overtimesOutput struct{ Body OvertimeList }

type contractListOutput struct{ Body ContractList }

type timesheetsOutput struct{ Body TimesheetList }

type timesheetOutput struct{ Body Timesheet }

type newTimesheetInput struct{ Body NewTimesheet }

type updateTimesheetInput struct {
	ID   int64 `path:"id"`
	Body TimesheetUpdate
}

type overtimeOutput struct{ Body Overtime }

type newOvertimeInput struct{ Body NewOvertime }

type updateOvertimeInput struct {
	ID   int64 `path:"id"`
	Body OvertimeUpdate
}

type calendarInput struct {
	LegalEntity int64 `path:"legal_entity"`
	Year        int   `query:"year" minimum:"2000" maximum:"2100" required:"true"`
}

type calendarOutput struct{ Body WorkCalendar }

type workWeekInput struct {
	LegalEntity   int64  `path:"legal_entity"`
	EffectiveFrom string `path:"effective_from" format:"date"`
	Body          struct {
		OffDays []int `json:"off_days" nullable:"false" maxItems:"7"`
	}
}

type workWeekRefInput struct {
	LegalEntity   int64  `path:"legal_entity"`
	EffectiveFrom string `path:"effective_from" format:"date"`
}

type holidayInput struct {
	LegalEntity int64  `path:"legal_entity"`
	Date        string `path:"date" format:"date"`
	Body        struct {
		Name string `json:"name" minLength:"1" maxLength:"200"`
	}
}

type holidayRefInput struct {
	LegalEntity int64  `path:"legal_entity"`
	Date        string `path:"date" format:"date"`
}

type legalParamsOutput struct{ Body LegalParams }

type legalParamInput struct {
	Key           string `path:"key"`
	EffectiveFrom string `path:"effective_from" format:"date"`
	Body          struct {
		Value string `json:"value" minLength:"1" maxLength:"500"`
	}
}

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

func registerHandlers(api huma.API, s *Service) {
	registerLeaveHandlers(api, s)
	registerPayrollHandlers(api, s)
	registerSetupHandlers(api, s)
	registerOvertimeHandlers(api, s)
	registerContractHandlers(api, s)
	registerTimesheetHandlers(api, s)
	huma.Register(api, huma.Operation{OperationID: "list-employees", Method: http.MethodGet, Path: "/hrm/employees"},
		func(ctx context.Context, in *EmployeeFilter) (*employeesOutput, error) {
			list, err := s.Employees(ctx, *in)
			return &employeesOutput{Body: list}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-employee-actions", Method: http.MethodGet, Path: "/hrm/employees/actions",
		Description: "What the actor may do before picking an employee: create, view_sensitive, import_leave_balances."},
		func(ctx context.Context, _ *struct{}) (*actionsOutput, error) {
			actions, err := s.EmployeeActions(ctx)
			out := &actionsOutput{}
			out.Body.AllowedActions = actions
			return out, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-employee", Method: http.MethodPost, Path: "/hrm/employees", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *employeeInput) (*createdOutput, error) {
			return created(s.CreateEmployee(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-employee", Method: http.MethodGet, Path: "/hrm/employees/{id}"},
		func(ctx context.Context, in *idInput) (*employeeOutput, error) {
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
		func(ctx context.Context, in *idInput) (*dependentsOutput, error) {
			ds, err := s.Dependents(ctx, in.ID)
			return &dependentsOutput{CacheControl: "no-store", Body: ds}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-dependent", Method: http.MethodPost, Path: "/hrm/employees/{id}/dependents", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *createDependentInput) (*createdOutput, error) {
			return created(s.SaveDependent(ctx, in.ID, 0, in.Body))
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

func registerLeaveHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-leave-types", Method: http.MethodGet, Path: "/hrm/leave-types"},
		func(ctx context.Context, _ *struct{}) (*leaveTypesOutput, error) {
			ts, err := s.LeaveTypes(ctx)
			return &leaveTypesOutput{Body: ts}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-leave-type", Method: http.MethodPost, Path: "/hrm/leave-types", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *leaveTypeInput) (*createdOutput, error) {
			return created(s.SaveLeaveType(ctx, 0, in.Body))
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
	huma.Register(api, huma.Operation{OperationID: "create-leave", Method: http.MethodPost, Path: "/hrm/leaves", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *newLeaveInput) (*createdOutput, error) {
			return created(s.CreateLeave(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-leave", Method: http.MethodGet, Path: "/hrm/leaves/{id}"},
		func(ctx context.Context, in *idInput) (*leaveOutput, error) {
			l, err := s.Leave(ctx, in.ID)
			return &leaveOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "update-leave", Method: http.MethodPut, Path: "/hrm/leaves/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateLeaveInput) (*struct{}, error) {
			return nil, s.UpdateLeave(ctx, in.ID, in.Body)
		})
	huma.Register(api, huma.Operation{OperationID: "delete-leave", Method: http.MethodDelete, Path: "/hrm/leaves/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *deleteDocInput) (*struct{}, error) {
			return nil, s.DeleteLeave(ctx, in.ID, in.Version)
		})
	huma.Register(api, huma.Operation{OperationID: "list-leave-balances", Method: http.MethodGet, Path: "/hrm/employees/{id}/leave-balances"},
		func(ctx context.Context, in *idInput) (*balancesOutput, error) {
			b, err := s.LeaveBalances(ctx, in.ID)
			return &balancesOutput{Body: b}, err
		})
	huma.Register(api, huma.Operation{OperationID: "adjust-leave-balance", Method: http.MethodPost, Path: "/hrm/employees/{id}/leave-balances/{year}/adjustments", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *adjustBalanceInput) (*struct{}, error) {
			return nil, s.AdjustLeaveBalance(ctx, in.ID, in.Year, in.Body)
		})
}

func registerContractHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-contract-types", Method: http.MethodGet, Path: "/hrm/contract-types"},
		func(ctx context.Context, _ *struct{}) (*contractTypesOutput, error) {
			ts, err := s.ContractTypes(ctx)
			return &contractTypesOutput{Body: ts}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-contract-type", Method: http.MethodPost, Path: "/hrm/contract-types", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *contractTypeInput) (*createdOutput, error) {
			return created(s.SaveContractType(ctx, 0, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "update-contract-type", Method: http.MethodPut, Path: "/hrm/contract-types/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateContractTypeInput) (*struct{}, error) {
			_, err := s.SaveContractType(ctx, in.ID, in.Body)
			return nil, err
		})
	huma.Register(api, huma.Operation{OperationID: "list-contracts", Method: http.MethodGet, Path: "/hrm/contracts",
		Description: "Every contract and appendix in scope; without terms."},
		func(ctx context.Context, in *ContractFilter) (*contractListOutput, error) {
			l, err := s.Contracts(ctx, *in)
			return &contractListOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "list-employee-contracts", Method: http.MethodGet, Path: "/hrm/employees/{id}/contracts",
		Description: "Originals, each followed by its appendices; without terms."},
		func(ctx context.Context, in *idInput) (*contractsOutput, error) {
			cs, err := s.EmployeeContracts(ctx, in.ID)
			return &contractsOutput{Body: cs}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-contract", Method: http.MethodPost, Path: "/hrm/contracts", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *newContractInput) (*createdOutput, error) {
			return created(s.CreateContract(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-contract", Method: http.MethodGet, Path: "/hrm/contracts/{id}",
		Description: "Terms come with hrm.salary.view; each such read is audited."},
		func(ctx context.Context, in *idInput) (*contractOutput, error) {
			c, err := s.Contract(ctx, in.ID)
			return &contractOutput{CacheControl: "no-store", Body: c}, err
		})
	huma.Register(api, huma.Operation{OperationID: "update-contract", Method: http.MethodPut, Path: "/hrm/contracts/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateContractInput) (*struct{}, error) {
			return nil, s.UpdateContract(ctx, in.ID, in.Body)
		})
	huma.Register(api, huma.Operation{OperationID: "delete-contract", Method: http.MethodDelete, Path: "/hrm/contracts/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *deleteDocInput) (*struct{}, error) {
			return nil, s.DeleteContract(ctx, in.ID, in.Version)
		})
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
		func(ctx context.Context, in *newOvertimeInput) (*createdOutput, error) {
			return created(s.CreateOvertime(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-overtime", Method: http.MethodGet, Path: "/hrm/overtimes/{id}"},
		func(ctx context.Context, in *idInput) (*overtimeOutput, error) {
			o, err := s.Overtime(ctx, in.ID)
			return &overtimeOutput{Body: o}, err
		})
	huma.Register(api, huma.Operation{OperationID: "update-overtime", Method: http.MethodPut, Path: "/hrm/overtimes/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateOvertimeInput) (*struct{}, error) {
			return nil, s.UpdateOvertime(ctx, in.ID, in.Body)
		})
	huma.Register(api, huma.Operation{OperationID: "delete-overtime", Method: http.MethodDelete, Path: "/hrm/overtimes/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *deleteDocInput) (*struct{}, error) {
			return nil, s.DeleteOvertime(ctx, in.ID, in.Version)
		})
}

func registerTimesheetHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-timesheets", Method: http.MethodGet, Path: "/hrm/timesheets"},
		func(ctx context.Context, in *TimesheetFilter) (*timesheetsOutput, error) {
			l, err := s.Timesheets(ctx, *in)
			return &timesheetsOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-timesheet-actions", Method: http.MethodGet, Path: "/hrm/timesheets/actions",
		Description: "What the actor may do before picking a timesheet: create."},
		func(ctx context.Context, _ *struct{}) (*actionsOutput, error) {
			actions, err := s.TimesheetActions(ctx)
			out := &actionsOutput{}
			out.Body.AllowedActions = actions
			return out, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-timesheet", Method: http.MethodPost, Path: "/hrm/timesheets", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *newTimesheetInput) (*createdOutput, error) {
			return created(s.CreateTimesheet(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-timesheet", Method: http.MethodGet, Path: "/hrm/timesheets/{id}"},
		func(ctx context.Context, in *idInput) (*timesheetOutput, error) {
			t, err := s.Timesheet(ctx, in.ID)
			return &timesheetOutput{Body: t}, err
		})
	huma.Register(api, huma.Operation{OperationID: "update-timesheet", Method: http.MethodPut, Path: "/hrm/timesheets/{id}",
		Description: "Replaces the period and every line of a draft.", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateTimesheetInput) (*struct{}, error) {
			return nil, s.SaveTimesheet(ctx, in.ID, in.Body)
		})
	huma.Register(api, huma.Operation{OperationID: "delete-timesheet", Method: http.MethodDelete, Path: "/hrm/timesheets/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *deleteDocInput) (*struct{}, error) {
			return nil, s.DeleteTimesheet(ctx, in.ID, in.Version)
		})
}

func registerSetupHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "get-work-calendar", Method: http.MethodGet, Path: "/hrm/calendars/{legal_entity}",
		Description: "Weekly days off by version and the holidays of a year; allowed_actions has manage."},
		func(ctx context.Context, in *calendarInput) (*calendarOutput, error) {
			c, err := s.WorkCalendar(ctx, in.LegalEntity, in.Year)
			return &calendarOutput{Body: c}, err
		})
	huma.Register(api, huma.Operation{OperationID: "save-work-week", Method: http.MethodPut, Path: "/hrm/calendars/{legal_entity}/weeks/{effective_from}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *workWeekInput) (*struct{}, error) {
			return nil, s.SaveWorkWeek(ctx, in.LegalEntity, WorkWeek{EffectiveFrom: in.EffectiveFrom, OffDays: in.Body.OffDays})
		})
	huma.Register(api, huma.Operation{OperationID: "delete-work-week", Method: http.MethodDelete, Path: "/hrm/calendars/{legal_entity}/weeks/{effective_from}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *workWeekRefInput) (*struct{}, error) {
			return nil, s.DeleteWorkWeek(ctx, in.LegalEntity, in.EffectiveFrom)
		})
	huma.Register(api, huma.Operation{OperationID: "save-holiday", Method: http.MethodPut, Path: "/hrm/calendars/{legal_entity}/holidays/{date}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *holidayInput) (*struct{}, error) {
			return nil, s.SaveHoliday(ctx, in.LegalEntity, Holiday{Date: in.Date, Name: in.Body.Name})
		})
	huma.Register(api, huma.Operation{OperationID: "delete-holiday", Method: http.MethodDelete, Path: "/hrm/calendars/{legal_entity}/holidays/{date}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *holidayRefInput) (*struct{}, error) {
			return nil, s.DeleteHoliday(ctx, in.LegalEntity, in.Date)
		})
	huma.Register(api, huma.Operation{OperationID: "list-legal-params", Method: http.MethodGet, Path: "/hrm/legal-params",
		Description: "Every version of every legal parameter; allowed_actions has manage."},
		func(ctx context.Context, _ *struct{}) (*legalParamsOutput, error) {
			l, err := s.LegalParams(ctx)
			return &legalParamsOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "save-legal-param", Method: http.MethodPut, Path: "/hrm/legal-params/{key}/{effective_from}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *legalParamInput) (*struct{}, error) {
			return nil, s.SaveLegalParam(ctx, LegalParam{Key: in.Key, EffectiveFrom: in.EffectiveFrom, Value: in.Body.Value})
		})
}

func registerPayrollHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-payrolls", Method: http.MethodGet, Path: "/hrm/payrolls"},
		func(ctx context.Context, in *PayrollFilter) (*payrollsOutput, error) {
			l, err := s.Payrolls(ctx, *in)
			return &payrollsOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-payroll-actions", Method: http.MethodGet, Path: "/hrm/payrolls/actions",
		Description: "What the actor may do before picking a payroll: create."},
		func(ctx context.Context, _ *struct{}) (*actionsOutput, error) {
			actions, err := s.PayrollActions(ctx)
			out := &actionsOutput{}
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
		func(ctx context.Context, in *idInput) (*payrollOutput, error) {
			p, err := s.Payroll(ctx, in.ID)
			return &payrollOutput{CacheControl: "no-store", Body: p}, err
		})
	huma.Register(api, huma.Operation{OperationID: "delete-payroll", Method: http.MethodDelete, Path: "/hrm/payrolls/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *deleteDocInput) (*struct{}, error) {
			return nil, s.DeletePayroll(ctx, in.ID, in.Version)
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
