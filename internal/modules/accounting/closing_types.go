package accounting

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

var (
	ErrClosingAccountsUnset  = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "closing_accounts_unset"}
	ErrClosingNothingToClose = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "closing_nothing_to_close"}
	ErrClosingTooLarge       = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "closing_too_large"}
)

// ClosingAccounts are where a closing posts: the result account (911) and retained earnings (4212).
type ClosingAccounts struct {
	ResultAccountID   int64 `json:"result_account_id"`
	RetainedAccountID int64 `json:"retained_account_id"`
}

type ClosingAccountsOut struct {
	ClosingAccounts
	ResultCode   string `json:"result_code"`
	ResultName   string `json:"result_name"`
	RetainedCode string `json:"retained_code"`
	RetainedName string `json:"retained_name"`
}

// ClosingFields are the editable fields of a closing entry; its lines are computed at posting.
type ClosingFields struct {
	Date          string `json:"date" format:"date"`
	LegalEntityID int64  `json:"legal_entity_id"`
	Description   string `json:"description" minLength:"1" maxLength:"500"`
}

type NewClosing struct {
	RequestID string `json:"request_id" format:"uuid" doc:"Generated once per form: sending it again returns the first document"`
	ClosingFields
}

type ClosingUpdate struct {
	Version int32 `json:"version"`
	ClosingFields
}

type ClosingLine struct {
	AccountID   int64  `json:"account_id"`
	AccountCode string `json:"account_code"`
	AccountName string `json:"account_name"`
	OrgUnitID   *int64 `json:"org_unit_id"`
	PartnerID   *int64 `json:"partner_id"`
	Debit       int64  `json:"debit" doc:"Minor unit"`
	Credit      int64  `json:"credit" doc:"Minor unit"`
}

type Closing struct {
	ID            int64  `json:"id"`
	Number        string `json:"number"`
	Status        string `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Version       int32  `json:"version"`
	Date          string `json:"date" format:"date"`
	LegalEntityID int64  `json:"legal_entity_id"`
	Description   string `json:"description"`
	// Lines: while not posted, what posting would write at the date (empty without closing
	// accounts); once posted, the entry written.
	Lines []ClosingLine `json:"lines" nullable:"false"`
	// edit, delete, submit, withdraw, cancel.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

type ClosingListItem struct {
	ID            int64  `json:"id"`
	Number        string `json:"number"`
	Status        string `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Date          string `json:"date" format:"date"`
	LegalEntityID int64  `json:"legal_entity_id"`
	Description   string `json:"description"`
}

type ClosingList struct {
	Items []ClosingListItem `json:"items" nullable:"false"`
	Total int64             `json:"total"`
}

type ClosingFilter struct {
	Q             string `query:"q" maxLength:"100" doc:"Number or description"`
	Status        string `query:"status" enum:"draft,pending_approval,posted,cancelled"`
	LegalEntityID int64  `query:"legal_entity_id"`
	From          string `query:"from" format:"date"`
	To            string `query:"to" format:"date"`
	Sort          string `query:"sort" enum:"date,-date,number,-number" default:"-date"`
	platform.Paging
}

type ClosingActions struct {
	// create.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}
