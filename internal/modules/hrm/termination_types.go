package hrm

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

const terminationType = "hrm.termination"

// Terminations by the employee's org unit.
const (
	PermTerminationView = "hrm.termination.view"
	PermTerminationEdit = "hrm.termination.edit"
)

var (
	// An employee has at most one termination that is not cancelled; params name it.
	ErrTerminationExists = &platform.Error{Status: http.StatusConflict, Code: "termination_exists"}
	ErrTerminationReason = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_termination_reason"}
	ErrTerminationMonths = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_termination_months"}
	// Only a termination sets or changes an employee's termination date.
	ErrTerminationViaDocument = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "termination_via_document"}
	// The employee moved to another legal entity since the termination was drafted.
	ErrTerminationLegalEntity = &platform.Error{Status: http.StatusConflict, Code: "termination_legal_entity_changed"}
)

// TerminationFields are the editable fields of a termination.
type TerminationFields struct {
	TerminationDate string  `json:"termination_date" format:"date" doc:"Last day employed"`
	Reason          string  `json:"reason" enum:"contract_expired,work_completed,mutual_agreement,imprisonment,foreigner_expelled,death,employer_ceased,dismissal,employee_resignation,employer_unilateral,absent_without_reason,restructuring,work_permit_expired,probation_failed"`
	DecisionDate    *string `json:"decision_date" format:"date"`
	PensionEligible bool    `json:"pension_eligible" doc:"Eligible for a pension: no severance allowance"`
	InsuredMonths   int     `json:"insured_months" minimum:"0" maximum:"1200" doc:"Months in unemployment insurance, from the social insurance book"`
	PaidMonths      int     `json:"paid_months" minimum:"0" maximum:"1200" doc:"Months this employer already paid an allowance for"`
	Handover        *string `json:"handover" maxLength:"4000"`
	Note            *string `json:"note" maxLength:"1000"`
}

type NewTermination struct {
	RequestID  string `json:"request_id,omitempty" format:"uuid" doc:"Generated once per form: sending it again returns the first document"`
	EmployeeID int64  `json:"employee_id"`
	TerminationFields
}

type TerminationUpdate struct {
	Version int32 `json:"version"`
	TerminationFields
}

type Termination struct {
	ID           int64  `json:"id"`
	Number       string `json:"number"`
	Status       string `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Version      int32  `json:"version"`
	EmployeeID   int64  `json:"employee_id"`
	EmployeeCode string `json:"employee_code"`
	EmployeeName string `json:"employee_name"`
	TerminationFields
	// What is still open past the termination date; nothing is refused for it.
	Warnings  []TerminationWarning `json:"warnings" nullable:"false"`
	Allowance TerminationAllowance `json:"allowance"`
	// edit, delete, submit, withdraw, cancel, print.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

type TerminationListItem struct {
	ID              int64  `json:"id"`
	Number          string `json:"number"`
	Status          string `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	EmployeeID      int64  `json:"employee_id"`
	EmployeeCode    string `json:"employee_code"`
	EmployeeName    string `json:"employee_name"`
	OrgUnitName     string `json:"org_unit_name"`
	TerminationDate string `json:"termination_date" format:"date"`
	Reason          string `json:"reason"`
}

type TerminationList struct {
	Items []TerminationListItem `json:"items" nullable:"false"`
	Total int64                 `json:"total"`
	// create, when the actor may create terminations somewhere.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

type TerminationFilter struct {
	Q          string `query:"q" maxLength:"100" doc:"Number, or the employee's code or name"`
	Status     string `query:"status" enum:"draft,pending_approval,posted,cancelled"`
	OrgUnitID  int64  `query:"org_unit_id"`
	EmployeeID int64  `query:"employee_id"`
	Sort       string `query:"sort" enum:"date,-date,number,-number" default:"-date"`
	platform.Paging
}

// TerminationWarning is one thing still open past the termination date: contract_in_force_after,
// leave_after_termination, overtime_after_termination, leave_balance_to_pay.
type TerminationWarning struct {
	Code   string         `json:"code"`
	Params map[string]any `json:"params"`
}

// TerminationAllowance previews what the payroll of the termination month will pay. The
// average and the amount come only with hrm.salary.view.
type TerminationAllowance struct {
	Kind          string `json:"kind" enum:"none,severance,job_loss"`
	MonthsWorked  int    `json:"months_worked"`
	MonthsCounted int    `json:"months_counted" doc:"Worked, less unemployment insurance and months already paid"`
	Years         string `json:"years"`
	Average       *int64 `json:"average,omitempty" doc:"Average monthly salary of the six months before the termination month"`
	Amount        *int64 `json:"amount,omitempty"`
}

// TerminationRef is the termination of an employee that stands, on their profile.
type TerminationRef struct {
	ID     int64  `json:"id"`
	Number string `json:"number"`
}
