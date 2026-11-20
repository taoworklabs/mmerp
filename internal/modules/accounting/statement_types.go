package accounting

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

var (
	ErrStatementFormulaInvalid = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "statement_formula_invalid"}
	ErrStatementFormulaCycle   = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "statement_formula_cycle"}
	ErrStatementRowUnknown     = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "statement_row_unknown"}
	ErrStatementRowDuplicate   = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "statement_row_duplicate"}
)

// Statement forms: the balance sheet (B01) and the income statement (B02).
const (
	FormBalanceSheet    = "balance_sheet"
	FormIncomeStatement = "income_statement"
)

// StatementRow is one row of a statement form; see formula.go for the formula language.
type StatementRow struct {
	Code    string `json:"code" minLength:"1" maxLength:"10" pattern:"^[0-9A-Za-z.]+$" doc:"The row's code on the official form (mã số)"`
	Label   string `json:"label" minLength:"1" maxLength:"300"`
	Formula string `json:"formula" maxLength:"1000"`
	Bold    bool   `json:"bold"`
}

type StatementForm struct {
	Rows []StatementRow `json:"rows" nullable:"false" maxItems:"200"`
}

type StatementLine struct {
	StatementRow
	Current  int64 `json:"current" doc:"Minor unit"`
	Previous int64 `json:"previous" doc:"Comparative column, minor unit"`
}

// Statement is a computed form. NotClosed: a revenue or expense account still has a balance
// at the end date, so rows read from closing entries are incomplete.
type Statement struct {
	Lines     []StatementLine `json:"lines" nullable:"false"`
	NotClosed bool            `json:"not_closed"`
}
