package hrm

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

const (
	PermView      = "hrm.employee.view"
	PermEdit      = "hrm.employee.edit"
	PermSensitive = "hrm.employee.sensitive"
	// Leave requests of others, by the request's org unit; one's own needs no role.
	PermLeaveView       = "hrm.leave.view"
	PermLeaveEdit       = "hrm.leave.edit"
	PermBalanceAdjust   = "hrm.leave_balance.adjust"
	PermLeaveTypeManage = "hrm.leave_type.manage"
	// Contracts by the contract's org unit; writing one also needs PermSalaryView.
	PermContractView       = "hrm.contract.view"
	PermContractEdit       = "hrm.contract.edit"
	PermContractTypeManage = "hrm.contract_type.manage"
	// Overtime requests of others, as for leave.
	PermOvertimeView = "hrm.overtime.view"
	PermOvertimeEdit = "hrm.overtime.edit"
	// Every money amount of a person: contract terms, later payroll.
	PermSalaryView = "hrm.salary.view"
	// Timesheets by the timesheet's org unit.
	PermTimesheetView = "hrm.timesheet.view"
	PermTimesheetEdit = "hrm.timesheet.edit"
	// Payrolls by their legal entity; amounts of a person also need PermSalaryView.
	PermPayrollView = "hrm.payroll.view"
	PermPayrollEdit = "hrm.payroll.edit"
	// The work calendar of a legal entity.
	PermCalendarManage = "hrm.calendar.manage"
	// Legal parameters are tenant-wide.
	PermLegalParamManage = "hrm.legal_param.manage"
	// Approval rules of HRM document types; core's approval checks it by name.
	PermApprovalManage = "hrm.approval.manage"
)

// SettingWageRegion is a legal entity's minimum-wage region, 1 to 4: it sets the unemployment insurance cap.
const SettingWageRegion = "hrm.wage_region"

var (
	ErrCodeTaken             = &platform.Error{Status: http.StatusConflict, Code: "employee_code_taken"}
	ErrUserNotFound          = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "user_not_found"}
	ErrUserAlreadyLinked     = &platform.Error{Status: http.StatusConflict, Code: "user_already_linked"}
	ErrManagerNotFound       = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "manager_not_found"}
	ErrManagerCycle          = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "manager_cycle"}
	ErrManagerSelf           = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "manager_self"}
	ErrTerminationBeforeHire = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "termination_before_hire"}
	ErrLeaveTypeTaken        = &platform.Error{Status: http.StatusConflict, Code: "leave_type_taken"}
	ErrLeaveTypeInactive     = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "leave_type_inactive"}
	ErrLeaveAttachment       = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "leave_attachment_required"}
	ErrNotAnEmployee         = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "not_an_employee"}
	ErrLeaveDates            = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "leave_end_before_start"}
	ErrLeaveSpansYears       = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "leave_spans_years"}
	// Days must be positive half-day steps.
	ErrLeaveDays = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_leave_days"}
	// No day of the range is a working day, so the request would take and pay nothing.
	ErrLeaveNoWorkingDays   = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "leave_no_working_days"}
	ErrInsufficientBalance  = &platform.Error{Status: http.StatusConflict, Code: "insufficient_leave_balance"}
	ErrNegativeBalance      = &platform.Error{Status: http.StatusConflict, Code: "leave_balance_negative"}
	ErrContractTypeTaken    = &platform.Error{Status: http.StatusConflict, Code: "contract_type_taken"}
	ErrContractTypeInactive = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "contract_type_inactive"}
	ErrContractDates        = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "contract_end_before_start"}
	ErrContractEndRequired  = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "contract_end_required"}
	ErrContractEndForbidden = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "contract_end_not_allowed"}
	// An appendix points to an original of the same employee, and starts within its term.
	ErrContractParent        = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_contract_parent"}
	ErrAppendixDate          = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "appendix_outside_contract"}
	ErrContractTerms         = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_contract_terms"}
	ErrContractOverlaps      = &platform.Error{Status: http.StatusConflict, Code: "contract_overlaps"}
	ErrContractParentDraft   = &platform.Error{Status: http.StatusConflict, Code: "contract_parent_not_posted"}
	ErrOvertimeHours         = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_overtime_hours"}
	ErrContractHasAppendices = &platform.Error{Status: http.StatusConflict, Code: "contract_has_appendices"}
	ErrReasonRequired        = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "reason_required"}
	// One timesheet per org unit and period, unless cancelled.
	ErrTimesheetExists = &platform.Error{Status: http.StatusConflict, Code: "timesheet_exists"}
	ErrNotLegalEntity  = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "not_a_legal_entity"}
	ErrLegalParam      = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_legal_param"}
	// Sending needs a computation of the current sources.
	ErrPayrollNotComputed   = &platform.Error{Status: http.StatusConflict, Code: "payroll_not_computed"}
	ErrPayrollSourceChanged = &platform.Error{Status: http.StatusConflict, Code: "payroll_sources_changed"}
	// Another payroll of the period is posted.
	ErrPayrollAlreadyPosted = &platform.Error{Status: http.StatusConflict, Code: "payroll_already_posted"}
	// Cancelling a posted payroll that does not hold its period.
	ErrPayrollNotHolding = &platform.Error{Status: http.StatusConflict, Code: "payroll_not_holding_period"}
	ErrAdjustment        = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_payroll_adjustment"}
)
