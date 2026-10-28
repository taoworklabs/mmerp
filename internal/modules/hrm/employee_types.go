package hrm

import "github.com/taoworklabs/mmerp/internal/platform"

// EmployeeFields are the plain fields of an employee record an edit may change; the code
// is set once, at creation. Nullable fields must be sent as null to clear them.
type EmployeeFields struct {
	FullName        string  `json:"full_name" minLength:"1" maxLength:"200"`
	DateOfBirth     *string `json:"date_of_birth" format:"date"`
	Gender          *string `json:"gender" enum:"male,female,other"`
	Phone           *string `json:"phone" maxLength:"50"`
	Email           *string `json:"email" maxLength:"200"`
	Address         *string `json:"address" maxLength:"500"`
	OrgUnitID       int64   `json:"org_unit_id"`
	ManagerID       *int64  `json:"manager_id"`
	UserLogin       *string `json:"user_login" maxLength:"200" doc:"Login of the linked account"`
	HireDate        string  `json:"hire_date" format:"date"`
	TerminationDate *string `json:"termination_date" format:"date"`
}

// SensitiveValues sets sensitive fields: an omitted field is unchanged, "" clears it.
type SensitiveValues struct {
	NationalID        *string `json:"national_id,omitempty" maxLength:"50"`
	SocialInsuranceNo *string `json:"social_insurance_no,omitempty" maxLength:"50"`
	TaxCode           *string `json:"tax_code,omitempty" maxLength:"50"`
	BankAccount       *string `json:"bank_account,omitempty" maxLength:"200"`
}

type EmployeeInput struct {
	EmployeeFields
	// Needs hrm.employee.sensitive at the employee's org unit.
	Sensitive *SensitiveValues `json:"sensitive,omitempty"`
}

// SensitivePresence tells which sensitive fields hold a value; values are read one at a time.
type SensitivePresence struct {
	NationalID        bool `json:"national_id"`
	SocialInsuranceNo bool `json:"social_insurance_no"`
	TaxCode           bool `json:"tax_code"`
	BankAccount       bool `json:"bank_account"`
}

// SensitiveTails holds the last four digits of each sensitive value with at least eight, for an
// actor who may read the value; null otherwise. Reading them is not an audited read.
type SensitiveTails struct {
	NationalID        *string `json:"national_id"`
	SocialInsuranceNo *string `json:"social_insurance_no"`
	TaxCode           *string `json:"tax_code"`
	BankAccount       *string `json:"bank_account"`
}

type NewEmployee struct {
	Code string `json:"code,omitempty" maxLength:"50" doc:"Empty or absent: issued from the hrm.employee code series"`
	EmployeeInput
}

type Employee struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	EmployeeFields
	OrgUnitName   string            `json:"org_unit_name"`
	ManagerCode   *string           `json:"manager_code"`
	ManagerName   *string           `json:"manager_name"`
	Status        string            `json:"status" enum:"active,terminated"`
	Sensitive     SensitivePresence `json:"sensitive"`
	SensitiveTail SensitiveTails    `json:"sensitive_tail"`
	// The posted contract in force today; null without one or without hrm.contract.view.
	CurrentContract *CurrentContract `json:"current_contract"`
	// view, edit, view_sensitive, adjust_leave_balance, view_contracts, create_contract.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

type CurrentContract struct {
	ID           int64   `json:"id"`
	Number       string  `json:"number"`
	ContractType string  `json:"contract_type"`
	StartDate    string  `json:"start_date"`
	EndDate      *string `json:"end_date"`
}

type EmployeeListItem struct {
	ID              int64   `json:"id"`
	Code            string  `json:"code"`
	FullName        string  `json:"full_name"`
	OrgUnitID       int64   `json:"org_unit_id"`
	OrgUnitName     string  `json:"org_unit_name"`
	Email           *string `json:"email"`
	Phone           *string `json:"phone"`
	HireDate        string  `json:"hire_date" format:"date"`
	TerminationDate *string `json:"termination_date" format:"date"`
	Status          string  `json:"status" enum:"active,terminated"`
}

type EmployeeList struct {
	Items []EmployeeListItem `json:"items" nullable:"false"`
	Total int64              `json:"total"`
}

type EmployeeFilter struct {
	Q         string `query:"q" maxLength:"200"`
	OrgUnitID int64  `query:"org_unit_id"`
	Status    string `query:"status" enum:"active,terminated"`
	ManagerOf int64  `query:"manager_of" doc:"Only who may become this employee's direct manager: neither the employee nor anyone under them"`
	Sort      string `query:"sort" enum:"code,-code,full_name,-full_name,hire_date,-hire_date" default:"code"`
	platform.Paging
}

// EmployeeExportParams are the list filters an employee export keeps.
type EmployeeExportParams struct {
	OrgUnitID int64  `json:"org_unit_id"`
	Status    string `json:"status"`
	Q         string `json:"q"`
}
