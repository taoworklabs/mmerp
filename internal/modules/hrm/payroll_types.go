package hrm

import "github.com/taoworklabs/mmerp/internal/platform"

// Period is a payroll period, as listed in payroll_period_closed.
type Period struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type NewPayroll struct {
	RequestID     string `json:"request_id,omitempty" format:"uuid" doc:"Generated once per form: sending it again returns the first document"`
	LegalEntityID int64  `json:"legal_entity_id"`
	Month         string `json:"month" pattern:"^[0-9]{4}-(0[1-9]|1[0-2])$" doc:"The payroll period, YYYY-MM"`
}

// PayrollJob is a payroll and the job computing it.
type PayrollJob struct {
	ID    int64 `json:"id"`
	JobID int64 `json:"job_id" doc:"0 when a retried request returns the payroll created the first time"`
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
	Severance          int64 `json:"severance" doc:"Severance or job-loss allowance on leaving; in gross and exempt"`
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
	// overtime_month_limit, overtime_year_limit, work_on_leave_day.
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
