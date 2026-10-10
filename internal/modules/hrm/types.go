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

// EmployeeFields are the plain fields of an employee record. Nullable fields
// must be sent as null to clear them.
type EmployeeFields struct {
	Code            string  `json:"code" minLength:"1" maxLength:"50"`
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

type Employee struct {
	ID int64 `json:"id"`
	EmployeeFields
	OrgUnitName string            `json:"org_unit_name"`
	ManagerCode *string           `json:"manager_code"`
	ManagerName *string           `json:"manager_name"`
	Status      string            `json:"status" enum:"active,terminated"`
	Sensitive   SensitivePresence `json:"sensitive"`
	// view, edit, view_sensitive, adjust_leave_balance, view_contracts, create_contract.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
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

type DependentInput struct {
	FullName      string  `json:"full_name" minLength:"1" maxLength:"200"`
	Relationship  string  `json:"relationship" enum:"child,spouse,parent,other"`
	DateOfBirth   *string `json:"date_of_birth" format:"date"`
	IDNumber      *string `json:"id_number" maxLength:"50" doc:"National ID or personal tax code"`
	DeductionFrom *string `json:"deduction_from" pattern:"^[0-9]{4}-[0-9]{2}$" doc:"First month of the deduction, YYYY-MM"`
	DeductionTo   *string `json:"deduction_to" pattern:"^[0-9]{4}-[0-9]{2}$"`
}

type Dependent struct {
	ID int64 `json:"id"`
	DependentInput
}

type LeaveTypeInput struct {
	Name           string `json:"name" minLength:"1" maxLength:"100"`
	DeductsBalance bool   `json:"deducts_balance" doc:"Approved requests of this kind deduct the leave balance"`
	Paid           bool   `json:"paid" doc:"The company pays these days: they count as paid days in payroll"`
	Active         bool   `json:"active"`
}

type LeaveType struct {
	ID int64 `json:"id"`
	LeaveTypeInput
}

// LeaveFields are the editable fields of a leave request.
type LeaveFields struct {
	LeaveTypeID int64   `json:"leave_type_id"`
	StartDate   string  `json:"start_date" format:"date"`
	EndDate     string  `json:"end_date" format:"date"`
	Days        string  `json:"days" pattern:"^[0-9]{1,3}(\\.[05])?$" doc:"Half-day steps, e.g. 2.5; at most the calendar days of the range"`
	Reason      *string `json:"reason" maxLength:"1000"`
}

type NewLeave struct {
	// Defaults to the actor's own employee record.
	EmployeeID *int64 `json:"employee_id,omitempty"`
	LeaveFields
}

type LeaveUpdate struct {
	Version int32 `json:"version"`
	LeaveFields
}

type Leave struct {
	ID            int64  `json:"id"`
	Number        string `json:"number"`
	Status        string `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Version       int32  `json:"version"`
	EmployeeID    int64  `json:"employee_id"`
	EmployeeCode  string `json:"employee_code"`
	EmployeeName  string `json:"employee_name"`
	LeaveTypeName string `json:"leave_type_name"`
	LeaveFields
	// Remaining days of the employee in the start year; null when the actor may not see it.
	Balance *string `json:"balance"`
	// edit, delete, submit, withdraw, cancel.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

type LeaveListItem struct {
	ID            int64  `json:"id"`
	Number        string `json:"number"`
	Status        string `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	EmployeeID    int64  `json:"employee_id"`
	EmployeeCode  string `json:"employee_code"`
	EmployeeName  string `json:"employee_name"`
	LeaveTypeName string `json:"leave_type_name"`
	StartDate     string `json:"start_date" format:"date"`
	EndDate       string `json:"end_date" format:"date"`
	Days          string `json:"days"`
}

type LeaveList struct {
	Items []LeaveListItem `json:"items" nullable:"false"`
	Total int64           `json:"total"`
}

type LeaveFilter struct {
	Status     string `query:"status" enum:"draft,pending_approval,posted,cancelled"`
	EmployeeID int64  `query:"employee_id"`
	From       string `query:"from" format:"date" doc:"Requests ending on or after"`
	To         string `query:"to" format:"date" doc:"Requests starting on or before"`
	Sort       string `query:"sort" enum:"start_date,-start_date,number,-number" default:"-start_date"`
	platform.Paging
}

// SelfServiceActions is what the actor may do before picking a leave or overtime request.
type SelfServiceActions struct {
	// create: for oneself (linked to an employee) or as HR; create_for_others: as HR, for another employee.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
	// The actor's own employee record, if linked.
	SelfEmployeeID   *int64  `json:"self_employee_id"`
	SelfEmployeeName *string `json:"self_employee_name"`
}

type LeaveBalance struct {
	Year int32  `json:"year"`
	Days string `json:"days"`
}

type BalanceAdjustment struct {
	Delta  string `json:"delta" pattern:"^-?[0-9]{1,3}(\\.[05])?$" doc:"Days to add; negative to take away"`
	Reason string `json:"reason" minLength:"1" maxLength:"1000"`
}

// Period is a payroll period, as listed in payroll_period_closed.
type Period struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

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
	EmployeeID int64 `json:"employee_id"`
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
	Status         string `query:"status" enum:"draft,pending_approval,posted,cancelled"`
	ContractTypeID int64  `query:"contract_type_id"`
	OrgUnitID      int64  `query:"org_unit_id"`
	EmployeeID     int64  `query:"employee_id"`
	Expiring       bool   `query:"expiring" doc:"Posted originals ending within 30 days from today"`
	Sort           string `query:"sort" enum:"start_date,-start_date,end_date,-end_date,number,-number" default:"-start_date"`
	platform.Paging
}

// OvertimeFields are the editable fields of an overtime request: the hours of one day.
type OvertimeFields struct {
	Date       string  `json:"date" format:"date"`
	DayKind    string  `json:"day_kind" enum:"weekday,weekly_off,holiday" doc:"Entered by hand until there is a work calendar"`
	DayHours   string  `json:"day_hours" pattern:"^[0-9]{1,2}(\\.[05])?$" doc:"Day shift hours, half-hour steps"`
	NightHours string  `json:"night_hours" pattern:"^[0-9]{1,2}(\\.[05])?$" doc:"Night shift hours; day and night together are more than 0 and at most 24"`
	Reason     *string `json:"reason" maxLength:"1000"`
}

type NewOvertime struct {
	// Defaults to the actor's own employee record.
	EmployeeID *int64 `json:"employee_id,omitempty"`
	OvertimeFields
}

type OvertimeUpdate struct {
	Version int32 `json:"version"`
	OvertimeFields
}

type Overtime struct {
	ID           int64  `json:"id"`
	Number       string `json:"number"`
	Status       string `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Version      int32  `json:"version"`
	EmployeeID   int64  `json:"employee_id"`
	EmployeeCode string `json:"employee_code"`
	EmployeeName string `json:"employee_name"`
	OvertimeFields
	// edit, delete, submit, withdraw, cancel.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

type OvertimeListItem struct {
	ID           int64  `json:"id"`
	Number       string `json:"number"`
	Status       string `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	EmployeeID   int64  `json:"employee_id"`
	EmployeeCode string `json:"employee_code"`
	EmployeeName string `json:"employee_name"`
	Date         string `json:"date" format:"date"`
	DayKind      string `json:"day_kind" enum:"weekday,weekly_off,holiday"`
	DayHours     string `json:"day_hours"`
	NightHours   string `json:"night_hours"`
}

type OvertimeList struct {
	Items []OvertimeListItem `json:"items" nullable:"false"`
	Total int64              `json:"total"`
}

type OvertimeFilter struct {
	Status     string `query:"status" enum:"draft,pending_approval,posted,cancelled"`
	EmployeeID int64  `query:"employee_id"`
	From       string `query:"from" format:"date"`
	To         string `query:"to" format:"date"`
	Sort       string `query:"sort" enum:"date,-date,number,-number" default:"-date"`
	platform.Paging
}

type NewTimesheet struct {
	OrgUnitID int64  `json:"org_unit_id"`
	Month     string `json:"month" pattern:"^[0-9]{4}-(0[1-9]|1[0-2])$" doc:"The payroll period, YYYY-MM"`
}

// TimesheetLine is the days one employee worked on one day; a blank cell has no line.
type TimesheetLine struct {
	EmployeeID int64  `json:"employee_id"`
	Date       string `json:"date" format:"date"`
	Days       string `json:"days" enum:"0.5,1"`
}

// TimesheetUpdate replaces a draft's period and every line.
type TimesheetUpdate struct {
	Version int32           `json:"version"`
	Month   string          `json:"month" pattern:"^[0-9]{4}-(0[1-9]|1[0-2])$"`
	Lines   []TimesheetLine `json:"lines" nullable:"false" maxItems:"100000"`
}

// TimesheetEmployee is a row of the grid: an employee of the unit employed during
// the period, or one who already has lines.
type TimesheetEmployee struct {
	ID              int64   `json:"id"`
	Code            string  `json:"code"`
	FullName        string  `json:"full_name"`
	HireDate        string  `json:"hire_date" format:"date"`
	TerminationDate *string `json:"termination_date" format:"date"`
}

type Timesheet struct {
	ID          int64               `json:"id"`
	Number      string              `json:"number"`
	Status      string              `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Version     int32               `json:"version"`
	OrgUnitID   int64               `json:"org_unit_id"`
	OrgUnitName string              `json:"org_unit_name"`
	PeriodStart string              `json:"period_start" format:"date"`
	PeriodEnd   string              `json:"period_end" format:"date"`
	Employees   []TimesheetEmployee `json:"employees" nullable:"false"`
	Lines       []TimesheetLine     `json:"lines" nullable:"false"`
	// Weekly days off and holidays of the period, from the work calendar.
	OffDays []string `json:"off_days" nullable:"false"`
	// edit, delete, submit, withdraw, cancel, export, import.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

type TimesheetListItem struct {
	ID          int64  `json:"id"`
	Number      string `json:"number"`
	Status      string `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	OrgUnitID   int64  `json:"org_unit_id"`
	OrgUnitName string `json:"org_unit_name"`
	PeriodStart string `json:"period_start" format:"date"`
	PeriodEnd   string `json:"period_end" format:"date"`
}

type TimesheetList struct {
	Items []TimesheetListItem `json:"items" nullable:"false"`
	Total int64               `json:"total"`
}

type TimesheetFilter struct {
	Month     string `query:"month" pattern:"^[0-9]{4}-(0[1-9]|1[0-2])$"`
	OrgUnitID int64  `query:"org_unit_id"`
	Status    string `query:"status" enum:"draft,pending_approval,posted,cancelled"`
	platform.Paging
}

// WorkWeek is a version of a legal entity's weekly days off, 0 (Sunday) to 6 (Saturday).
type WorkWeek struct {
	EffectiveFrom string `json:"effective_from" format:"date"`
	OffDays       []int  `json:"off_days" nullable:"false" maxItems:"7"`
}

// Holiday is a public holiday or a day off in lieu.
type Holiday struct {
	Date string `json:"date" format:"date"`
	Name string `json:"name" minLength:"1" maxLength:"200"`
}

// WorkCalendar is a legal entity's weekly days off by version and its holidays of one year.
// Without any version the days off are Saturday and Sunday.
type WorkCalendar struct {
	WorkWeeks []WorkWeek `json:"work_weeks" nullable:"false"`
	Holidays  []Holiday  `json:"holidays" nullable:"false"`
	// manage.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

// LegalParam is one version of a legal parameter. The value is a decimal, or for
// pit_brackets JSON [[upper bound or null, rate], …].
type LegalParam struct {
	Key           string `json:"key"`
	EffectiveFrom string `json:"effective_from" format:"date"`
	Value         string `json:"value"`
}

type LegalParams struct {
	Items []LegalParam `json:"items" nullable:"false"`
	// manage.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

type NewPayroll struct {
	LegalEntityID int64  `json:"legal_entity_id"`
	Month         string `json:"month" pattern:"^[0-9]{4}-(0[1-9]|1[0-2])$" doc:"The payroll period, YYYY-MM"`
}

// PayrollJob is a payroll and the job computing it.
type PayrollJob struct {
	ID    int64 `json:"id"`
	JobID int64 `json:"job_id"`
}

// PayrollAmounts are the money columns of a payroll, in đồng, for one employee or summed.
type PayrollAmounts struct {
	Earned             int64 `json:"earned" doc:"Pay for the paid days"`
	Overtime           int64 `json:"overtime"`
	UnusedLeave        int64 `json:"unused_leave" doc:"Unused leave paid on leaving"`
	Adjustment         int64 `json:"adjustment" doc:"Back pay (+) or recovery (-)"`
	Gross              int64 `json:"gross"`
	Exempt             int64 `json:"exempt" doc:"Tax-free income"`
	InsuranceBase      int64 `json:"insurance_base"`
	SocialInsurance    int64 `json:"social_insurance"`
	HealthInsurance    int64 `json:"health_insurance"`
	UnemploymentIns    int64 `json:"unemployment_insurance"`
	PersonalDeduction  int64 `json:"personal_deduction"`
	DependentDeduction int64 `json:"dependent_deduction"`
	Taxable            int64 `json:"taxable" doc:"Income the tax brackets apply to"`
	IncomeTax          int64 `json:"income_tax"`
	Net                int64 `json:"net"`
	EmployerSocial     int64 `json:"employer_social_insurance"`
	EmployerHealth     int64 `json:"employer_health_insurance"`
	EmployerUnemploy   int64 `json:"employer_unemployment_insurance"`
	EmployerUnion      int64 `json:"employer_union_fee"`
	Cost               int64 `json:"cost" doc:"Gross plus the employer's contributions"`
}

// PayrollTotal sums the amounts of a department's employees.
type PayrollTotal struct {
	OrgUnitID   int64  `json:"org_unit_id"`
	OrgUnitName string `json:"org_unit_name"`
	Employees   int    `json:"employees"`
	PayrollAmounts
}

// PayrollLine is one employee's pay. Days and hours are decimal strings.
type PayrollLine struct {
	EmployeeID    int64  `json:"employee_id"`
	EmployeeCode  string `json:"employee_code"`
	EmployeeName  string `json:"employee_name"`
	OrgUnitID     int64  `json:"org_unit_id"`
	StandardDays  int    `json:"standard_days"`
	PaidDays      string `json:"paid_days"`
	OvertimeHours string `json:"overtime_hours"`
	// overtime_month_limit, overtime_year_limit.
	Warnings []string `json:"warnings" nullable:"false"`
	PayrollAmounts
}

// PayrollAdjustmentInput is back pay (positive) or a recovery (negative) of a closed period.
type PayrollAdjustmentInput struct {
	EmployeeID   int64  `json:"employee_id"`
	Amount       int64  `json:"amount" doc:"Đồng; positive pays back, negative recovers; not 0"`
	SourcePeriod string `json:"source_period" pattern:"^[0-9]{4}-(0[1-9]|1[0-2])$" doc:"The closed period it corrects, YYYY-MM"`
	Reason       string `json:"reason" minLength:"1" maxLength:"1000"`
}

type PayrollAdjustment struct {
	EmployeeCode string `json:"employee_code"`
	EmployeeName string `json:"employee_name"`
	PayrollAdjustmentInput
}

// PayrollAdjustments replaces every adjustment of a draft.
type PayrollAdjustments struct {
	Version int32                    `json:"version"`
	Items   []PayrollAdjustmentInput `json:"items" nullable:"false" maxItems:"1000"`
}

type Payroll struct {
	ID              int64  `json:"id"`
	Number          string `json:"number"`
	Status          string `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Version         int32  `json:"version"`
	LegalEntityID   int64  `json:"legal_entity_id"`
	LegalEntityName string `json:"legal_entity_name"`
	PeriodStart     string `json:"period_start" format:"date"`
	PeriodEnd       string `json:"period_end" format:"date"`
	// Null until the first computation is done.
	ComputedAt *string `json:"computed_at" format:"date-time"`
	// The sources changed since the computation: compute again before sending.
	SourcesChanged bool           `json:"sources_changed"`
	Totals         []PayrollTotal `json:"totals" nullable:"false"`
	// Each employee's pay and the adjustments; null without hrm.salary.view, each read audited.
	Lines       []PayrollLine       `json:"lines"`
	Adjustments []PayrollAdjustment `json:"adjustments"`
	// edit, delete, submit, withdraw, cancel, compute, adjust, export.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

type PayrollListItem struct {
	ID              int64   `json:"id"`
	Number          string  `json:"number"`
	Status          string  `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	LegalEntityID   int64   `json:"legal_entity_id"`
	LegalEntityName string  `json:"legal_entity_name"`
	PeriodStart     string  `json:"period_start" format:"date"`
	PeriodEnd       string  `json:"period_end" format:"date"`
	ComputedAt      *string `json:"computed_at" format:"date-time"`
}

type PayrollList struct {
	Items []PayrollListItem `json:"items" nullable:"false"`
	Total int64             `json:"total"`
}

type PayrollFilter struct {
	LegalEntityID int64  `query:"legal_entity_id"`
	Month         string `query:"month" pattern:"^[0-9]{4}-(0[1-9]|1[0-2])$"`
	Status        string `query:"status" enum:"draft,pending_approval,posted,cancelled"`
	platform.Paging
}

// PayrollParams names the payroll of a payroll export.
type PayrollParams struct {
	PayrollID int64 `json:"payroll_id"`
}

// TimesheetParams names the timesheet of a timesheet import or export.
type TimesheetParams struct {
	TimesheetID int64 `json:"timesheet_id"`
}

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
	ErrNotAnEmployee         = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "not_an_employee"}
	ErrLeaveDates            = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "leave_end_before_start"}
	ErrLeaveSpansYears       = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "leave_spans_years"}
	// Days must be positive half-day steps no larger than the calendar days of the range.
	ErrLeaveDays            = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_leave_days"}
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

// errTimesheetLine refuses a line: timesheet_date_outside_period, timesheet_employee_not_in_unit,
// timesheet_date_outside_employment or invalid_timesheet_days.
func errTimesheetLine(code string, l TimesheetLine) error {
	return &platform.Error{Status: http.StatusUnprocessableEntity, Code: code, Params: map[string]any{"employee_id": l.EmployeeID, "date": l.Date}}
}
