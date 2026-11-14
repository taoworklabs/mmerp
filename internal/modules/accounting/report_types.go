package accounting

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

var (
	ErrInvalidRange = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_range"}
	// ErrAccountNotTracked: partner reports read accounts tracked by partner only.
	ErrAccountNotTracked = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "account_not_tracked"}
)

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
	PartnerID   *int64  `json:"partner_id"`
	PartnerCode *string `json:"partner_code"`
	PartnerName *string `json:"partner_name"`
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

// PartnerBalanceRow is one partner on one tracked account; balances are signed debit minus credit.
type PartnerBalanceRow struct {
	AccountID   int64  `json:"account_id"`
	AccountCode string `json:"account_code"`
	PartnerID   int64  `json:"partner_id"`
	PartnerCode string `json:"partner_code"`
	PartnerName string `json:"partner_name"`
	Opening     int64  `json:"opening"`
	Debit       int64  `json:"debit"`
	Credit      int64  `json:"credit"`
	Closing     int64  `json:"closing"`
}

// PartnerBalances totals equal the trial balance rows of the accounts covered.
type PartnerBalances struct {
	Rows         []PartnerBalanceRow `json:"rows" nullable:"false"`
	TotalOpening int64               `json:"total_opening"`
	TotalDebit   int64               `json:"total_debit"`
	TotalCredit  int64               `json:"total_credit"`
	TotalClosing int64               `json:"total_closing"`
}
