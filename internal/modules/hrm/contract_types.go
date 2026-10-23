package hrm

import "github.com/taoworklabs/mmerp/internal/platform"

type ContractTypeInput struct {
	Name      string `json:"name" minLength:"1" maxLength:"100"`
	FixedTerm bool   `json:"fixed_term" doc:"Contracts of this kind need an end date; the others may not have one"`
	Active    bool   `json:"active"`
}

type ContractType struct {
	ID int64 `json:"id"`
	ContractTypeInput
}

// ContractLine is an allowance (insured, in the overtime rate) or a support (neither,
// taxable or not) of a contract. Amounts are in đồng.
type ContractLine struct {
	Kind    string `json:"kind" enum:"allowance,support"`
	Name    string `json:"name" minLength:"1" maxLength:"200"`
	Amount  int64  `json:"amount" minimum:"0"`
	Taxable bool   `json:"taxable"`
}

// ContractTerms are the money terms of a contract or appendix, stored encrypted.
type ContractTerms struct {
	Salary int64          `json:"salary" minimum:"0"`
	Lines  []ContractLine `json:"lines" nullable:"false" maxItems:"50"`
}

// ContractFields are the editable fields of a contract. An appendix takes its
// original's kind and end date, so it ignores contract_type_id and end_date.
type ContractFields struct {
	ContractTypeID int64         `json:"contract_type_id"`
	StartDate      string        `json:"start_date" format:"date" doc:"Effective date"`
	EndDate        *string       `json:"end_date" format:"date" doc:"Null: no end date"`
	Terms          ContractTerms `json:"terms"`
}

type NewContract struct {
	RequestID  string `json:"request_id,omitempty" format:"uuid" doc:"Generated once per form: sending it again returns the first document"`
	EmployeeID int64  `json:"employee_id"`
	// The original contract, for an appendix.
	ParentID *int64 `json:"parent_id,omitempty"`
	ContractFields
}

type ContractUpdate struct {
	Version int32 `json:"version"`
	ContractFields
}

type Contract struct {
	ID               int64   `json:"id"`
	Number           string  `json:"number"`
	Status           string  `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Version          int32   `json:"version"`
	EmployeeID       int64   `json:"employee_id"`
	EmployeeCode     string  `json:"employee_code"`
	EmployeeName     string  `json:"employee_name"`
	ParentID         *int64  `json:"parent_id"`
	ParentNumber     *string `json:"parent_number"`
	ContractTypeID   int64   `json:"contract_type_id"`
	ContractTypeName string  `json:"contract_type_name"`
	StartDate        string  `json:"start_date" format:"date"`
	EndDate          *string `json:"end_date" format:"date"`
	// Null without hrm.salary.view; each read is audited.
	Terms *ContractTerms `json:"terms"`
	// edit, delete, submit, withdraw, cancel.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

type ContractListItem struct {
	ID               int64   `json:"id"`
	Number           string  `json:"number"`
	Status           string  `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	EmployeeID       int64   `json:"employee_id"`
	EmployeeCode     string  `json:"employee_code"`
	EmployeeName     string  `json:"employee_name"`
	ParentID         *int64  `json:"parent_id"`
	ParentNumber     *string `json:"parent_number"`
	ContractTypeName string  `json:"contract_type_name"`
	StartDate        string  `json:"start_date" format:"date"`
	EndDate          *string `json:"end_date" format:"date"`
	// add_appendix.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

type ContractList struct {
	Items []ContractListItem `json:"items" nullable:"false"`
	Total int64              `json:"total"`
}

type ContractFilter struct {
	Q              string `query:"q" maxLength:"100" doc:"Number, or the employee's code or name"`
	Status         string `query:"status" enum:"draft,pending_approval,posted,cancelled"`
	ContractTypeID int64  `query:"contract_type_id"`
	OrgUnitID      int64  `query:"org_unit_id"`
	EmployeeID     int64  `query:"employee_id"`
	Expiring       bool   `query:"expiring" doc:"Posted originals ending within 30 days from today"`
	Sort           string `query:"sort" enum:"start_date,-start_date,end_date,-end_date,number,-number" default:"-start_date"`
	platform.Paging
}
