package sales

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// InvoiceLineInput bills a quantity of one order line.
type InvoiceLineInput struct {
	OrderPosition int    `json:"order_position" minimum:"1"`
	Quantity      string `json:"quantity" pattern:"^[0-9]{1,12}(\\.[0-9]{1,3})?$" doc:"More than 0, up to 3 decimals"`
}

type InvoiceFields struct {
	Date  string             `json:"date" format:"date"`
	Note  *string            `json:"note" maxLength:"2000"`
	Lines []InvoiceLineInput `json:"lines" minItems:"1" maxItems:"500" nullable:"false"`
}

type NewInvoice struct {
	RequestID string `json:"request_id" format:"uuid" doc:"Generated once per form: sending it again returns the first invoice"`
	OrderID   int64  `json:"order_id"`
	InvoiceFields
}

type InvoiceUpdate struct {
	Version int32 `json:"version"`
	InvoiceFields
}

// InvoiceLine is a line as saved: item, texts, price, discount and VAT rate are the order line's.
type InvoiceLine struct {
	InvoiceLineInput
	ItemID          int64  `json:"item_id"`
	ItemCode        string `json:"item_code"`
	Description     string `json:"description"`
	Unit            string `json:"unit"`
	UnitPrice       int64  `json:"unit_price"`
	DiscountPercent string `json:"discount_percent"`
	VatRate         string `json:"vat_rate" enum:"none,0,5,8,10"`
	Amount          int64  `json:"amount"`
	Discount        int64  `json:"discount"`
	Vat             int64  `json:"vat"`
}

// Invoice is a sales invoice; the customer is the posted order's copy, already final.
type Invoice struct {
	ID          int64         `json:"id"`
	OrderID     int64         `json:"order_id"`
	OrderNumber string        `json:"order_number"`
	Number      string        `json:"number"`
	Status      string        `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Version     int32         `json:"version"`
	Date        string        `json:"date" format:"date"`
	OrgUnitID   int64         `json:"org_unit_id"`
	OrgUnitName string        `json:"org_unit_name"`
	Customer    CustomerCopy  `json:"customer"`
	Note        *string       `json:"note"`
	Lines       []InvoiceLine `json:"lines" nullable:"false"`
	Totals      Totals        `json:"totals"`
	// Paid counts posted receipts; remaining is the total less paid.
	Paid         int64        `json:"paid"`
	Remaining    int64        `json:"remaining"`
	PaymentState string       `json:"payment_state" enum:"unpaid,partial,paid"`
	Receipts     []ReceiptRef `json:"receipts" nullable:"false" doc:"Receipts not cancelled, drafts included, that the actor may view"`
	// edit, delete, submit, withdraw, cancel, print, create_receipt.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

// InvoiceOptionLine is an order line with what non-cancelled invoices leave of it.
// ReceiptRef names a receipt settling an invoice and the amount it settles.
type ReceiptRef struct {
	ID     int64  `json:"id"`
	Number string `json:"number"`
	Status string `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Amount int64  `json:"amount"`
}

type InvoiceOptionLine struct {
	OrderPosition   int    `json:"order_position"`
	ItemID          int64  `json:"item_id"`
	ItemCode        string `json:"item_code"`
	Description     string `json:"description"`
	Unit            string `json:"unit"`
	UnitPrice       int64  `json:"unit_price"`
	DiscountPercent string `json:"discount_percent"`
	VatRate         string `json:"vat_rate" enum:"none,0,5,8,10"`
	Ordered         string `json:"ordered"`
	Remaining       string `json:"remaining"`
}

type InvoiceOptions struct {
	Lines []InvoiceOptionLine `json:"lines" nullable:"false"`
}

// Payment states of an invoice or order, and receipt methods.
const (
	paymentUnpaid  = "unpaid"
	paymentPartial = "partial"
	paymentPaid    = "paid"
	methodBank     = "bank"
)

var (
	ErrOrderHasInvoice = &platform.Error{Status: http.StatusConflict, Code: "sales_order_has_invoice"}
	// A line takes more than ordered less other non-cancelled invoices (params line_no, order_position, remaining).
	ErrInvoiceOverOrdered = &platform.Error{Status: http.StatusConflict, Code: "sales_invoice_over_ordered"}
	// The order has no line at the given order_position (param line_no).
	ErrInvoicePosition    = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "sales_invoice_position"}
	ErrInvoiceBeforeOrder = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "sales_invoice_before_order"}
	ErrInvoiceHasReceipt  = &platform.Error{Status: http.StatusConflict, Code: "sales_invoice_has_receipt"}
)

type InvoiceFilter struct {
	Q            string `query:"q" maxLength:"100" doc:"Number, order number or customer name"`
	Status       string `query:"status" enum:"draft,pending_approval,posted,cancelled"`
	OrderID      int64  `query:"order_id"`
	CustomerID   int64  `query:"customer_id"`
	PaymentState string `query:"payment_state" enum:"unpaid,partial,paid"`
	Sort         string `query:"sort" enum:"date,-date,number,-number,total,-total" default:"-date"`
	platform.Paging
}

type InvoiceListItem struct {
	ID           int64  `json:"id"`
	Number       string `json:"number"`
	Status       string `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Date         string `json:"date" format:"date"`
	OrderID      int64  `json:"order_id"`
	OrderNumber  string `json:"order_number"`
	CustomerID   int64  `json:"customer_id"`
	CustomerName string `json:"customer_name"`
	Total        int64  `json:"total"`
	OrgUnitName  string `json:"org_unit_name"`
	// Posted invoices only.
	PaymentState string `json:"payment_state,omitempty" enum:"unpaid,partial,paid"`
}

type InvoiceList struct {
	Items []InvoiceListItem `json:"items" nullable:"false"`
	Total int64             `json:"total"`
}
