package hrm

import "github.com/taoworklabs/mmerp/internal/platform"

// OvertimeFields are the editable fields of an overtime request: the hours of one day.
type OvertimeFields struct {
	Date       string  `json:"date" format:"date"`
	DayKind    string  `json:"day_kind" enum:"weekday,weekly_off,holiday" doc:"Entered by hand until there is a work calendar"`
	DayHours   string  `json:"day_hours" pattern:"^[0-9]{1,2}(\\.[05])?$" doc:"Day shift hours, half-hour steps"`
	NightHours string  `json:"night_hours" pattern:"^[0-9]{1,2}(\\.[05])?$" doc:"Night shift hours; day and night together are more than 0 and at most 24"`
	Reason     *string `json:"reason" maxLength:"1000"`
}

type NewOvertime struct {
	RequestID string `json:"request_id,omitempty" format:"uuid" doc:"Generated once per form: sending it again returns the first document"`
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
	Q          string `query:"q" maxLength:"100" doc:"Number, or the employee's code or name"`
	Status     string `query:"status" enum:"draft,pending_approval,posted,cancelled"`
	OrgUnitID  int64  `query:"org_unit_id"`
	EmployeeID int64  `query:"employee_id"`
	Reports    bool   `query:"reports" doc:"Only requests of the employees below the actor on the manager chain"`
	From       string `query:"from" format:"date"`
	To         string `query:"to" format:"date"`
	Sort       string `query:"sort" enum:"date,-date,number,-number" default:"-date"`
	platform.Paging
}
