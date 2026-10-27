package hrm

// MyProfile is the actor's own employee record: ordinary fields only, nothing sensitive.
type MyProfile struct {
	EmployeeID      int64   `json:"employee_id"`
	Code            string  `json:"code"`
	FullName        string  `json:"full_name"`
	DateOfBirth     *string `json:"date_of_birth" format:"date"`
	Gender          *string `json:"gender" enum:"male,female,other"`
	Phone           *string `json:"phone"`
	Email           *string `json:"email"`
	Address         *string `json:"address"`
	OrgUnitName     string  `json:"org_unit_name"`
	ManagerName     *string `json:"manager_name"`
	HireDate        string  `json:"hire_date" format:"date"`
	TerminationDate *string `json:"termination_date" format:"date"`
	// The posted contract in force today, without its terms.
	Contract *MyContract `json:"contract"`
}

type MyContract struct {
	TypeName  string  `json:"type_name"`
	StartDate string  `json:"start_date" format:"date"`
	EndDate   *string `json:"end_date" format:"date"`
}

// MyDays is the actor's month, day by day.
type MyDays struct {
	Month        string  `json:"month"`
	StandardDays int     `json:"standard_days"`
	Days         []MyDay `json:"days" nullable:"false"`
}

type MyDay struct {
	Date string `json:"date" format:"date"`
	Kind string `json:"kind" enum:"work,rest,holiday"`
	// Days worked on the month's timesheet, "1" or "0.5"; null when none.
	Worked *string `json:"worked"`
	// The timesheet holding Worked is posted.
	WorkedClosed bool `json:"worked_closed"`
	// Posted leave placed on the day, as the payroll places it.
	Leave *MyDayLeave `json:"leave"`
	// Posted overtime hours, day and night shift together.
	OvertimeHours *string `json:"overtime_hours"`
}

type MyDayLeave struct {
	TypeName string `json:"type_name"`
	Days     string `json:"days"`
	Paid     bool   `json:"paid"`
}

type MyPayslipSummary struct {
	PayrollID   int64  `json:"payroll_id"`
	PeriodStart string `json:"period_start" format:"date"`
	Net         int64  `json:"net"`
}

// MyPayslip is the actor's part of a posted payroll, as its print shows it.
type MyPayslip struct {
	PayrollID     int64                    `json:"payroll_id"`
	Number        string                   `json:"number"`
	PeriodStart   string                   `json:"period_start" format:"date"`
	StandardDays  int                      `json:"standard_days"`
	PaidDays      string                   `json:"paid_days"`
	OvertimeHours string                   `json:"overtime_hours"`
	Amounts       PayrollAmounts           `json:"amounts"`
	Adjustments   []PayrollAdjustmentInput `json:"adjustments" nullable:"false"`
}
