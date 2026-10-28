package hrm

import "github.com/taoworklabs/mmerp/internal/platform"

type LeaveTypeInput struct {
	Name           string `json:"name" minLength:"1" maxLength:"100"`
	DeductsBalance bool   `json:"deducts_balance" doc:"Approved requests of this kind deduct the leave balance"`
	Paid           bool   `json:"paid" doc:"The company pays these days: they count as paid days in payroll"`
	Active         bool   `json:"active"`
	// Article 115 kinds are granted per event, so the cap is per request.
	MaxDays            *string `json:"max_days,omitempty" pattern:"^[0-9]{1,3}(\\.[05])?$" doc:"Most days one request of this kind may take; absent for no cap"`
	RequiresAttachment bool    `json:"requires_attachment,omitempty" doc:"A request of this kind needs a file before it is sent"`
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
	Days        string  `json:"days,omitempty" pattern:"^[0-9]{1,3}(\\.[05])?$" doc:"Computed from the work calendar; when sent, the count the client expects, refused if it differs"`
	Reason      *string `json:"reason" maxLength:"1000"`
	// Half a day on the first or the last working date of the range.
	StartHalf bool `json:"start_half,omitempty" doc:"Only the afternoon of the first working date"`
	EndHalf   bool `json:"end_half,omitempty" doc:"Only the morning of the last working date"`
}

type NewLeave struct {
	RequestID string `json:"request_id,omitempty" format:"uuid" doc:"Generated once per form: sending it again returns the first document"`
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
	Q          string   `query:"q" maxLength:"100" doc:"Number, or the employee's code or name"`
	Status     []string `query:"status,explode" enum:"draft,pending_approval,posted,cancelled"`
	OrgUnitID  int64    `query:"org_unit_id"`
	EmployeeID int64    `query:"employee_id"`
	Reports    bool     `query:"reports" doc:"Only requests of the employees below the actor on the manager chain"`
	From       string   `query:"from" format:"date" doc:"Requests ending on or after"`
	To         string   `query:"to" format:"date" doc:"Requests starting on or before"`
	Sort       string   `query:"sort" enum:"start_date,-start_date,number,-number" default:"-start_date"`
	platform.Paging
}

type LeaveBalance struct {
	Year int32  `json:"year"`
	Days string `json:"days"`
	// Carried over from last year, part of days, until CarriedUntil.
	Carried      string  `json:"carried"`
	CarriedUntil *string `json:"carried_until" format:"date"`
}

type BalanceAdjustment struct {
	Delta  string `json:"delta" pattern:"^-?[0-9]{1,3}(\\.[05])?$" doc:"Days to add; negative to take away"`
	Reason string `json:"reason" minLength:"1" maxLength:"1000"`
}
