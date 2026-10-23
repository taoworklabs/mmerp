package hrm

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type NewTimesheet struct {
	RequestID string `json:"request_id,omitempty" format:"uuid" doc:"Generated once per form: sending it again returns the first document"`
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
	// Cells the payroll treats differently from what they say.
	Warnings []TimesheetWarning `json:"warnings" nullable:"false"`
	// edit, delete, submit, withdraw, cancel, export, import.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

// TimesheetWarning flags one cell: work_on_day_off is work on a weekly day off or a holiday,
// which earns nothing without an overtime request; work_on_leave_day is work on a day an
// approved leave request also covers.
type TimesheetWarning struct {
	EmployeeID int64  `json:"employee_id"`
	Date       string `json:"date" format:"date"`
	Code       string `json:"code" enum:"work_on_day_off,work_on_leave_day"`
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
	Sort      string `query:"sort" enum:"period,-period,number,-number" default:"-period"`
	platform.Paging
}

// TimesheetParams names the timesheet of a timesheet import or export.
type TimesheetParams struct {
	TimesheetID int64 `json:"timesheet_id"`
}

// errTimesheetLine refuses a line: timesheet_date_outside_period, timesheet_employee_not_in_unit,
// timesheet_date_outside_employment or invalid_timesheet_days.
func errTimesheetLine(code string, l TimesheetLine) error {
	return &platform.Error{Status: http.StatusUnprocessableEntity, Code: code, Params: map[string]any{"employee_id": l.EmployeeID, "date": l.Date}}
}
