package purchase

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// LineInput is one line as entered; the server computes its amounts.
type LineInput struct {
	ItemID          int64  `json:"item_id"`
	Description     string `json:"description" minLength:"1" maxLength:"500"`
	Quantity        string `json:"quantity" pattern:"^[0-9]{1,12}(\\.[0-9]{1,3})?$" doc:"More than 0, up to 3 decimals"`
	UnitPrice       int64  `json:"unit_price" minimum:"0"`
	DiscountPercent string `json:"discount_percent" pattern:"^[0-9]{1,3}(\\.[0-9]{1,2})?$" doc:"0 to 100, up to 2 decimals"`
	VatRate         string `json:"vat_rate" enum:"none,0,5,8,10"`
}

// OrderFields are the editable fields of a purchase order.
type OrderFields struct {
	Date         string      `json:"date" format:"date"`
	OrgUnitID    int64       `json:"org_unit_id"`
	SupplierID   int64       `json:"supplier_id"`
	ExpectedDate *string     `json:"expected_date" format:"date" doc:"When the goods are expected; not before the order date"`
	PaymentTerms *string     `json:"payment_terms" maxLength:"1000"`
	Note         *string     `json:"note" maxLength:"2000"`
	Lines        []LineInput `json:"lines" minItems:"1" maxItems:"200" nullable:"false"`
}

type NewOrder struct {
	RequestID string `json:"request_id" format:"uuid" doc:"Generated once per form: sending it again returns the first document"`
	OrderFields
}

type OrderUpdate struct {
	Version int32 `json:"version"`
	OrderFields
}

type Line struct {
	LineInput
	ItemCode string `json:"item_code"`
	Unit     string `json:"unit"`
	// Whether the item had stock tracking when the line was saved: goods receipts take only these.
	Stocked  bool  `json:"stocked"`
	Amount   int64 `json:"amount" doc:"Quantity × unit price"`
	Discount int64 `json:"discount"`
	Vat      int64 `json:"vat"`
	// Posted orders' stock lines: what posted goods receipts brought in.
	Received *string `json:"received,omitempty"`
	// Posted orders: what posted invoices billed.
	Invoiced *string `json:"invoiced,omitempty"`
}

// DocRef names another document.
type DocRef struct {
	ID     int64  `json:"id"`
	Number string `json:"number"`
	Status string `json:"status" enum:"draft,pending_approval,posted,cancelled"`
}

type Totals struct {
	Subtotal      int64 `json:"subtotal"`
	DiscountTotal int64 `json:"discount_total"`
	VatTotal      int64 `json:"vat_total"`
	Total         int64 `json:"total"`
}

type Order struct {
	ID           int64        `json:"id"`
	Number       string       `json:"number"`
	Status       string       `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Version      int32        `json:"version"`
	Date         string       `json:"date" format:"date"`
	OrgUnitID    int64        `json:"org_unit_id"`
	OrgUnitName  string       `json:"org_unit_name"`
	Supplier     SupplierCopy `json:"supplier"`
	ExpectedDate *string      `json:"expected_date" format:"date"`
	PaymentTerms *string      `json:"payment_terms"`
	Note         *string      `json:"note"`
	Lines        []Line       `json:"lines" nullable:"false"`
	Totals
	// Posted orders: how far posted goods receipts went over the stock lines, and the receipts not cancelled.
	ReceiptState  string   `json:"receipt_state,omitempty" enum:"none,partial,full,not_applicable"`
	GoodsReceipts []DocRef `json:"goods_receipts,omitempty"`
	// Posted orders: how far posted invoices went over all lines, by quantity, and the invoices not cancelled.
	InvoiceState string   `json:"invoice_state,omitempty" enum:"none,partial,full"`
	Invoices     []DocRef `json:"invoices,omitempty"`
	// edit, delete, submit, withdraw, cancel, print, create_goods_receipt, create_invoice.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

type OrderFilter struct {
	Q          string `query:"q" maxLength:"100" doc:"Number, supplier code or name"`
	Status     string `query:"status" enum:"draft,pending_approval,posted,cancelled"`
	SupplierID int64  `query:"supplier_id"`
	ReceiptState string `query:"receipt_state" enum:"none,partial,full,not_applicable"`
	InvoiceState string `query:"invoice_state" enum:"none,partial,full"`
	Sort       string `query:"sort" enum:"date,-date,number,-number,total,-total" default:"-date"`
	platform.Paging
}

type OrderListItem struct {
	ID           int64  `json:"id"`
	Number       string `json:"number"`
	Status       string `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Date         string `json:"date" format:"date"`
	SupplierID   int64  `json:"supplier_id"`
	SupplierCode string `json:"supplier_code"`
	SupplierName string `json:"supplier_name"`
	OrgUnitName  string `json:"org_unit_name"`
	Total        int64  `json:"total"`
	ReceiptState string `json:"receipt_state" enum:"none,partial,full,not_applicable"`
	InvoiceState string `json:"invoice_state" enum:"none,partial,full"`
}

type OrderList struct {
	Items []OrderListItem `json:"items" nullable:"false"`
	Total int64           `json:"total"`
}

var (
	ErrExpectedDate     = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "purchase_order_expected_before_date"}
	ErrOrderHasReceipt  = &platform.Error{Status: http.StatusConflict, Code: "purchase_order_has_receipt"}
	ErrOrderHasInvoice  = &platform.Error{Status: http.StatusConflict, Code: "purchase_order_has_invoice"}
	ErrOrderNotPosted   = &platform.Error{Status: http.StatusConflict, Code: "purchase_order_not_posted"}
)

// errLine names the line (1-based) a value is wrong on.
func errLine(code string, line int) error {
	return &platform.Error{Status: http.StatusUnprocessableEntity, Code: code, Params: map[string]any{"line": line}}
}
