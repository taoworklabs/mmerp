package posting

import (
	"net/http"
	"slices"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// ErrAlreadyRecorded: the document already has lines that are not voided.
var ErrAlreadyRecorded = &platform.Error{Status: http.StatusConflict, Code: "posting_already_recorded"}

// ErrPartnerMismatch: a line names a partner its kind does not declare, or lacks one it does.
// A caller's bug, not a user's.
var ErrPartnerMismatch = &platform.Error{Status: http.StatusInternalServerError, Code: "posting_partner_mismatch"}

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
// PartnerID is set exactly when the kind's lines name a partner; an employee is never one.
type Line struct {
	Kind      Kind
	OrgUnitID int64
	PartnerID *int64
	Amount    int64
}

// KindDecl declares a kind; ByPartner: its lines name the partner owed by or to.
type KindDecl struct {
	Kind      Kind
	ByPartner bool
}

var kinds = []KindDecl{{Kind: SalaryExpense}, {Kind: EmployerInsuranceExpense}, {Kind: SalaryPayable}, {Kind: InsurancePayable}, {Kind: PitPayable}}

// Kinds lists the declared kinds, in declaration order.
func Kinds() []KindDecl {
	return slices.Clone(kinds)
}

// Declared reports k's declaration; an undeclared kind names no partner.
func Declared(k Kind) (KindDecl, bool) {
	i := slices.IndexFunc(kinds, func(d KindDecl) bool { return d.Kind == k })
	if i < 0 {
		return KindDecl{Kind: k}, false
	}
	return kinds[i], true
}

// DeclareKind adds a kind. It is not safe for concurrent use: call it before serving.
func DeclareKind(k Kind, byPartner bool) {
	if _, ok := Declared(k); ok {
		panic("posting: kind declared twice: " + string(k))
	}
	kinds = append(kinds, KindDecl{Kind: k, ByPartner: byPartner})
}
