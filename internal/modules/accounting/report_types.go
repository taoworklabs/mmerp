package accounting

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

var ErrInvalidRange = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_range"}

// LedgerLine is one entry line; Balance is the running signed balance (debit minus credit) after it.
type LedgerLine struct {
	Date        string  `json:"date" format:"date"`
	EntryID     int64   `json:"entry_id"`
	DocType     string  `json:"doc_type"`
	DocID       int64   `json:"doc_id"`
	DocNumber   string  `json:"doc_number"`
	Kind        *string `json:"kind"`
	Description *string `json:"description"`
	OrgUnitID   *int64  `json:"org_unit_id"`
	AccountCode string  `json:"account_code"`
	Debit       int64   `json:"debit"`
	Credit      int64   `json:"credit"`
	Balance     int64   `json:"balance"`
}

// Ledger balances are signed debit minus credit: positive is a debit balance, negative a credit balance.
type Ledger struct {
	Opening int64        `json:"opening"`
	Lines   []LedgerLine `json:"lines"`
	Closing int64        `json:"closing"`
}

type TrialRow struct {
	AccountID int64  `json:"account_id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	ParentID  *int64 `json:"parent_id"`
	Opening   int64  `json:"opening"`
	Debit     int64  `json:"debit"`
	Credit    int64  `json:"credit"`
	Closing   int64  `json:"closing"`
}

// TrialBalance totals count postable accounts only, so a parent is not added on top of its children.
type TrialBalance struct {
	Rows        []TrialRow `json:"rows"`
	TotalDebit  int64      `json:"total_debit"`
	TotalCredit int64      `json:"total_credit"`
}
