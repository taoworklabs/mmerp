package hrm

import "github.com/taoworklabs/mmerp/internal/platform"

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
