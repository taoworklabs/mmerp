package sales

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// AllocationInput settles an amount of one posted invoice.
type AllocationInput struct {
	InvoiceID int64 `json:"invoice_id"`
	Amount    int64 `json:"amount" minimum:"1" maximum:"1000000000000000"`
}

type ReceiptFields struct {
	Date        string            `json:"date" format:"date"`
	OrgUnitID   int64             `json:"org_unit_id"`
	CustomerID  int64             `json:"customer_id"`
	Method      string            `json:"method" enum:"cash,bank"`
	Note        *string           `json:"note" maxLength:"2000"`
	Allocations []AllocationInput `json:"allocations" minItems:"1" maxItems:"200" nullable:"false"`
}

type NewReceipt struct {
	RequestID string `json:"request_id" format:"uuid" doc:"Generated once per form: sending it again returns the first receipt"`
	ReceiptFields
}

type ReceiptUpdate struct {
	Version int32 `json:"version"`
	ReceiptFields
}

// Allocation is a saved allocation; remaining is what other non-cancelled receipts leave of
// the invoice, drafts included.
type Allocation struct {
	AllocationInput
	InvoiceNumber string `json:"invoice_number"`
	InvoiceDate   string `json:"invoice_date" format:"date"`
	InvoiceTotal  int64  `json:"invoice_total"`
	Remaining     int64  `json:"remaining"`
}

type Receipt struct {
	ID           int64        `json:"id"`
	Number       string       `json:"number"`
	Status       string       `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Version      int32        `json:"version"`
	Date         string       `json:"date" format:"date"`
	OrgUnitID    int64        `json:"org_unit_id"`
	OrgUnitName  string       `json:"org_unit_name"`
	CustomerID   int64        `json:"customer_id"`
	CustomerCode string       `json:"customer_code"`
	CustomerName string       `json:"customer_name"`
	Method       string       `json:"method" enum:"cash,bank"`
	Amount       int64        `json:"amount"`
	Note         *string      `json:"note"`
	Allocations  []Allocation `json:"allocations" nullable:"false"`
	// edit, delete, submit, withdraw, cancel, print.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

var (
	// An allocation exceeds what other non-cancelled receipts leave (params line_no, invoice_id, remaining).
	ErrReceiptOverInvoice = &platform.Error{Status: http.StatusConflict, Code: "sales_receipt_over_invoice"}
	// The invoice is not posted or is of another customer (param line_no).
	ErrReceiptInvoice = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "sales_receipt_invoice"}
	// The invoice is of another org unit than the receipt (param line_no).
	ErrReceiptInvoiceOrgUnit = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "sales_receipt_invoice_org_unit"}
	// The receipt is dated before an invoice it settles (param line_no).
	ErrReceiptBeforeInvoice    = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "sales_receipt_before_invoice"}
	ErrReceiptDuplicateInvoice = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "sales_receipt_duplicate_invoice"}
)

type ReceiptFilter struct {
	Q          string `query:"q" maxLength:"100" doc:"Number or customer name"`
	Status     string `query:"status" enum:"draft,pending_approval,posted,cancelled"`
	CustomerID int64  `query:"customer_id"`
	Method     string `query:"method" enum:"cash,bank"`
	Sort       string `query:"sort" enum:"date,-date,number,-number,amount,-amount" default:"-date"`
	platform.Paging
}

type ReceiptListItem struct {
	ID           int64  `json:"id"`
	Number       string `json:"number"`
	Status       string `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Date         string `json:"date" format:"date"`
	CustomerID   int64  `json:"customer_id"`
	CustomerName string `json:"customer_name"`
	Method       string `json:"method" enum:"cash,bank"`
	Amount       int64  `json:"amount"`
	OrgUnitName  string `json:"org_unit_name"`
}

type ReceiptList struct {
	Items []ReceiptListItem `json:"items" nullable:"false"`
	Total int64             `json:"total"`
}
