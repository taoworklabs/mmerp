package purchase

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// AllocationInput settles an amount of one posted invoice.
type AllocationInput struct {
	InvoiceID int64 `json:"invoice_id"`
	Amount    int64 `json:"amount" minimum:"1" maximum:"1000000000000000"`
}

type PaymentFields struct {
	Date        string            `json:"date" format:"date"`
	OrgUnitID   int64             `json:"org_unit_id"`
	SupplierID  int64             `json:"supplier_id"`
	Method      string            `json:"method" enum:"cash,bank"`
	Note        *string           `json:"note" maxLength:"2000"`
	Allocations []AllocationInput `json:"allocations" minItems:"1" maxItems:"200" nullable:"false"`
}

type NewPayment struct {
	RequestID string `json:"request_id" format:"uuid" doc:"Generated once per form: sending it again returns the first payment"`
	PaymentFields
}

type PaymentUpdate struct {
	Version int32 `json:"version"`
	PaymentFields
}

// Allocation is a saved allocation; remaining is what other non-cancelled payments leave of
// the invoice, drafts included.
type Allocation struct {
	AllocationInput
	InvoiceNumber string `json:"invoice_number"`
	InvoiceDate   string `json:"invoice_date" format:"date"`
	InvoiceTotal  int64  `json:"invoice_total"`
	Remaining     int64  `json:"remaining"`
}

type Payment struct {
	ID           int64        `json:"id"`
	Number       string       `json:"number"`
	Status       string       `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Version      int32        `json:"version"`
	Date         string       `json:"date" format:"date"`
	OrgUnitID    int64        `json:"org_unit_id"`
	OrgUnitName  string       `json:"org_unit_name"`
	SupplierID   int64        `json:"supplier_id"`
	SupplierCode string       `json:"supplier_code"`
	SupplierName string       `json:"supplier_name"`
	Method       string       `json:"method" enum:"cash,bank"`
	Amount       int64        `json:"amount"`
	Note         *string      `json:"note"`
	Allocations  []Allocation `json:"allocations" nullable:"false"`
	// edit, delete, submit, withdraw, cancel, print.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

var (
	// An allocation exceeds what other non-cancelled payments leave (params line_no, invoice_id, remaining).
	ErrPaymentOverInvoice = &platform.Error{Status: http.StatusConflict, Code: "purchase_payment_over_invoice"}
	// The invoice is not posted or is of another supplier (param line_no).
	ErrPaymentInvoice = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "purchase_payment_invoice"}
	// The invoice is of another org unit than the payment (param line_no).
	ErrPaymentInvoiceOrgUnit = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "purchase_payment_invoice_org_unit"}
	// The payment is dated before an invoice it settles (param line_no).
	ErrPaymentBeforeInvoice    = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "purchase_payment_before_invoice"}
	ErrPaymentDuplicateInvoice = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "purchase_payment_duplicate_invoice"}
)

type PaymentFilter struct {
	Q          string `query:"q" maxLength:"100" doc:"Number or supplier name"`
	Status     string `query:"status" enum:"draft,pending_approval,posted,cancelled"`
	SupplierID int64  `query:"supplier_id"`
	Method     string `query:"method" enum:"cash,bank"`
	Sort       string `query:"sort" enum:"date,-date,number,-number,amount,-amount" default:"-date"`
	platform.Paging
}

type PaymentListItem struct {
	ID           int64  `json:"id"`
	Number       string `json:"number"`
	Status       string `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Date         string `json:"date" format:"date"`
	SupplierID   int64  `json:"supplier_id"`
	SupplierName string `json:"supplier_name"`
	Method       string `json:"method" enum:"cash,bank"`
	Amount       int64  `json:"amount"`
	OrgUnitName  string `json:"org_unit_name"`
}

type PaymentList struct {
	Items []PaymentListItem `json:"items" nullable:"false"`
	Total int64             `json:"total"`
}

// PaymentRef names a payment settling an invoice and the amount it settles.
type PaymentRef struct {
	ID     int64  `json:"id"`
	Number string `json:"number"`
	Status string `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Amount int64  `json:"amount"`
}

// Payment states of an invoice or order, and payment methods.
const (
	paymentUnpaid  = "unpaid"
	paymentPartial = "partial"
	paymentPaid    = "paid"
	methodBank     = "bank"
)

// paymentState: what was paid against what is owed; owing nothing is paid.
func paymentState(paid, total int64) string {
	switch {
	case paid >= total:
		return paymentPaid
	case paid == 0:
		return paymentUnpaid
	}
	return paymentPartial
}
