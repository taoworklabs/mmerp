package purchase

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// InvoiceLineInput bills a quantity of one order line at the price, discount and VAT rate the
// supplier billed; the form pre-fills them from the order line.
type InvoiceLineInput struct {
	OrderPosition   int    `json:"order_position" minimum:"1"`
	Quantity        string `json:"quantity" pattern:"^[0-9]{1,12}(\\.[0-9]{1,3})?$" doc:"More than 0, up to 3 decimals"`
	UnitPrice       int64  `json:"unit_price" minimum:"0"`
	DiscountPercent string `json:"discount_percent" pattern:"^[0-9]{1,3}(\\.[0-9]{1,2})?$" doc:"0 to 100, up to 2 decimals"`
	VatRate         string `json:"vat_rate" enum:"none,0,5,8,10"`
}

// InvoiceFields: the supplier's invoice symbol, number and date may be empty on a draft and are
// required to send it.
type InvoiceFields struct {
	Date                  string             `json:"date" format:"date"`
	SupplierInvoiceSymbol *string            `json:"supplier_invoice_symbol" maxLength:"20" doc:"Ký hiệu hoá đơn, e.g. 1C26TAA"`
	SupplierInvoiceNo     *string            `json:"supplier_invoice_no" maxLength:"20"`
	SupplierInvoiceDate   *string            `json:"supplier_invoice_date" format:"date"`
	Note                  *string            `json:"note" maxLength:"2000"`
	Lines                 []InvoiceLineInput `json:"lines" minItems:"1" maxItems:"500" nullable:"false"`
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

// InvoiceLine is a line as saved: item, texts and stocked are the order line's.
type InvoiceLine struct {
	InvoiceLineInput
	ItemID      int64  `json:"item_id"`
	ItemCode    string `json:"item_code"`
	Description string `json:"description"`
	Unit        string `json:"unit"`
	// Booked as goods when true, as an expense otherwise.
	Stocked  bool  `json:"stocked"`
	Amount   int64 `json:"amount"`
	Discount int64 `json:"discount"`
	Vat      int64 `json:"vat"`
}

// Invoice is a supplier invoice; the supplier is the posted order's copy, already final.
type Invoice struct {
	ID                    int64         `json:"id"`
	OrderID               int64         `json:"order_id"`
	OrderNumber           string        `json:"order_number"`
	Number                string        `json:"number"`
	Status                string        `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Version               int32         `json:"version"`
	Date                  string        `json:"date" format:"date"`
	OrgUnitID             int64         `json:"org_unit_id"`
	OrgUnitName           string        `json:"org_unit_name"`
	Supplier              SupplierCopy  `json:"supplier"`
	SupplierInvoiceSymbol *string       `json:"supplier_invoice_symbol"`
	SupplierInvoiceNo     *string       `json:"supplier_invoice_no"`
	SupplierInvoiceDate   *string       `json:"supplier_invoice_date" format:"date"`
	Note                  *string       `json:"note"`
	Lines                 []InvoiceLine `json:"lines" nullable:"false"`
	Totals                Totals        `json:"totals"`
	// edit, delete, submit, withdraw, cancel, print.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

// InvoiceOptionLine is an order line with what non-cancelled invoices leave of it.
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

type InvoiceFilter struct {
	Q            string `query:"q" maxLength:"100" doc:"Number, order number, supplier name or the supplier's invoice symbol and number"`
	Status       string `query:"status" enum:"draft,pending_approval,posted,cancelled"`
	OrderID      int64  `query:"order_id"`
	SupplierID   int64  `query:"supplier_id"`
	PaymentState string `query:"payment_state" enum:"unpaid,partial,paid"`
	Sort         string `query:"sort" enum:"date,-date,number,-number,total,-total" default:"-date"`
	platform.Paging
}

type InvoiceListItem struct {
	ID                    int64   `json:"id"`
	Number                string  `json:"number"`
	Status                string  `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Date                  string  `json:"date" format:"date"`
	OrderID               int64   `json:"order_id"`
	OrderNumber           string  `json:"order_number"`
	SupplierID            int64   `json:"supplier_id"`
	SupplierName          string  `json:"supplier_name"`
	SupplierInvoiceSymbol *string `json:"supplier_invoice_symbol"`
	SupplierInvoiceNo     *string `json:"supplier_invoice_no"`
	Total                 int64   `json:"total"`
	OrgUnitName           string  `json:"org_unit_name"`
	// Posted invoices only.
	PaymentState string `json:"payment_state,omitempty" enum:"unpaid,partial,paid"`
}

type InvoiceList struct {
	Items []InvoiceListItem `json:"items" nullable:"false"`
	Total int64             `json:"total"`
}

var (
	// A line takes more than ordered less other non-cancelled invoices (params line_no, order_position, remaining).
	ErrInvoiceOverOrder = &platform.Error{Status: http.StatusConflict, Code: "purchase_invoice_over_order"}
	// The order has no line at the given order_position (param line_no).
	ErrInvoicePosition    = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "purchase_invoice_position"}
	ErrInvoiceBeforeOrder = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "purchase_invoice_before_order"}
	// The supplier's invoice symbol, number or date is missing on sending.
	ErrInvoiceSupplierRef = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "purchase_invoice_supplier_ref_required"}
	ErrInvoiceHasPayment  = &platform.Error{Status: http.StatusConflict, Code: "purchase_invoice_has_payment"}
)
