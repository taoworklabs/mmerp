package posting

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// ErrAlreadyRecorded: the document already has lines that are not voided.
var ErrAlreadyRecorded = &platform.Error{Status: http.StatusConflict, Code: "posting_already_recorded"}

// Kind is the type of a posting line; accounting maps kinds to accounts. The column has no
// CHECK: each product adds kinds, and only this service writes them.
type Kind string

const (
	SalaryExpense            Kind = "salary_expense"
	EmployerInsuranceExpense Kind = "employer_insurance_expense"
	SalaryPayable            Kind = "salary_payable"
	InsurancePayable         Kind = "insurance_payable"
	PitPayable               Kind = "pit_payable"
)

// Line is one posting line: a signed amount in the minor unit, by org unit, never per person.
type Line struct {
	Kind      Kind
	OrgUnitID int64
	Amount    int64
}
