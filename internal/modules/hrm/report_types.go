package hrm

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// ReportParams name a report's legal entity and period: month YYYY-MM, quarter YYYY-Qn or year.
type ReportParams struct {
	LegalEntityID int64  `json:"legal_entity_id"`
	Month         string `json:"month,omitempty"`
	Quarter       string `json:"quarter,omitempty"`
	Year          int    `json:"year,omitempty"`
}

var (
	// A month the report covers has no posted payroll; params: months, as MM/YYYY.
	ErrReportPayrollMissing = &platform.Error{Status: http.StatusConflict, Code: "report_payroll_missing"}
	// The period predates the form version mmerp produces.
	ErrReportFormNotSupported = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "report_form_not_supported"}
)
