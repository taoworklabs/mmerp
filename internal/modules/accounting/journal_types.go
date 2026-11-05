package accounting

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

var (
	ErrLineSideInvalid          = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "line_side_invalid"}
	ErrJournalEntryUnbalanced   = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "journal_entry_unbalanced"}
	ErrEntryDescriptionRequired = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "journal_entry_description_required"}
	ErrLineAmountInvalid        = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "line_amount_invalid"}
	ErrLineOrgUnitInvalid       = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "line_org_unit_invalid"}
	ErrTooManyLines             = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "journal_entry_too_many_lines"}
	ErrAccountingNotStarted     = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "accounting_not_started"}
)

// JournalEntryLine is one line of a manual journal entry: exactly one of Debit and Credit is positive.
type JournalEntryLine struct {
	AccountID   int64   `json:"account_id"`
	OrgUnitID   *int64  `json:"org_unit_id,omitempty"`
	Description *string `json:"description,omitempty" minLength:"1" maxLength:"500"`
	Debit       int64   `json:"debit" minimum:"0" maximum:"1000000000000000" doc:"Minor unit"`
	Credit      int64   `json:"credit" minimum:"0" maximum:"1000000000000000" doc:"Minor unit"`
}

// JournalEntryFields are the editable fields of a manual journal entry.
type JournalEntryFields struct {
	Date          string             `json:"date" format:"date"`
	LegalEntityID int64              `json:"legal_entity_id"`
	Description   string             `json:"description" minLength:"1" maxLength:"500"`
	Lines         []JournalEntryLine `json:"lines" minItems:"1" maxItems:"500" nullable:"false"`
}

type NewJournalEntry struct {
	RequestID string `json:"request_id" format:"uuid" doc:"Generated once per form: sending it again returns the first document"`
	JournalEntryFields
}

type JournalEntryUpdate struct {
	Version int32 `json:"version"`
	JournalEntryFields
}

type JournalEntryLineOut struct {
	JournalEntryLine
	AccountCode string `json:"account_code"`
	AccountName string `json:"account_name"`
}

type JournalEntry struct {
	ID            int64                 `json:"id"`
	Number        string                `json:"number"`
	Status        string                `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Version       int32                 `json:"version"`
	Date          string                `json:"date" format:"date"`
	LegalEntityID int64                 `json:"legal_entity_id"`
	Description   string                `json:"description"`
	Lines         []JournalEntryLineOut `json:"lines" nullable:"false"`
	// edit, delete, submit, withdraw, cancel.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

type JournalEntryFilter struct {
	Q             string `query:"q" maxLength:"100" doc:"Number or description"`
	Status        string `query:"status" enum:"draft,pending_approval,posted,cancelled"`
	LegalEntityID int64  `query:"legal_entity_id"`
	From          string `query:"from" format:"date"`
	To            string `query:"to" format:"date"`
	Sort          string `query:"sort" enum:"date,-date,number,-number" default:"-date"`
	platform.Paging
}

type JournalEntryListItem struct {
	ID            int64  `json:"id"`
	Number        string `json:"number"`
	Status        string `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Date          string `json:"date" format:"date"`
	LegalEntityID int64  `json:"legal_entity_id"`
	Description   string `json:"description"`
	Total         int64  `json:"total" doc:"Sum of debits, minor unit"`
}

type JournalEntryList struct {
	Items []JournalEntryListItem `json:"items" nullable:"false"`
	Total int64                  `json:"total"`
}

// JournalEntryActions are what the actor may do before picking an entry.
type JournalEntryActions struct {
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}
