package sales

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

const (
	// Customers by their owning org unit.
	PermCustomerView = "sales.customer.view"
	PermCustomerEdit = "sales.customer.edit"
	// Quotations and orders by their org unit; edit also sends, withdraws, cancels and attaches.
	PermQuoteView = "sales.quote.view"
	PermQuoteEdit = "sales.quote.edit"
	PermOrderView = "sales.order.view"
	PermOrderEdit = "sales.order.edit"
	// The item catalogue is tenant-wide: viewing needs the permission anywhere, managing tenant-wide.
	PermItemView   = "sales.item.view"
	PermItemManage = "sales.item.manage"
	// Approval rules of Sales document types; core's approval checks it by name.
	PermApprovalManage = "sales.approval.manage"
)

// VatRates are the VAT rates a line may carry; none is not subject to VAT.
var VatRates = []string{"none", "0", "5", "8", "10"}

type CustomerFields struct {
	Code         string  `json:"code" minLength:"1" maxLength:"50"`
	Name         string  `json:"name" minLength:"1" maxLength:"300"`
	TaxCode      *string `json:"tax_code" maxLength:"20"`
	Address      *string `json:"address" maxLength:"500"`
	Phone        *string `json:"phone" maxLength:"50"`
	Email        *string `json:"email" maxLength:"200"`
	ContactName  *string `json:"contact_name" maxLength:"200"`
	PaymentTerms *string `json:"payment_terms" maxLength:"1000" doc:"Prefilled on new quotations and orders"`
	OrgUnitID    int64   `json:"org_unit_id" doc:"Owning org unit: who may see the customer"`
	Active       bool    `json:"active"`
}

type Customer struct {
	ID int64 `json:"id"`
	CustomerFields
	OrgUnitName string `json:"org_unit_name"`
	// edit, attach.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

type CustomerFilter struct {
	Q      string `query:"q" maxLength:"100"`
	Active string `query:"active" enum:"true,false"`
	Sort   string `query:"sort" enum:"code,-code,name,-name" default:"code"`
	platform.Paging
}

type CustomerListItem struct {
	ID          int64   `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	TaxCode     *string `json:"tax_code"`
	Phone       *string `json:"phone"`
	OrgUnitName string  `json:"org_unit_name"`
	Active      bool    `json:"active"`
	// Prefilled on a new quotation or order of the customer.
	PaymentTerms *string `json:"payment_terms"`
}

type CustomerList struct {
	Items []CustomerListItem `json:"items" nullable:"false"`
	Total int64              `json:"total"`
}

// Actions is what the actor may do before picking a record: create.
type Actions struct {
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

type ItemFields struct {
	Code    string `json:"code" minLength:"1" maxLength:"50"`
	Name    string `json:"name" minLength:"1" maxLength:"300"`
	Unit    string `json:"unit" minLength:"1" maxLength:"30"`
	Price   int64  `json:"price" minimum:"0" doc:"Default unit price in đồng"`
	VatRate string `json:"vat_rate" enum:"none,0,5,8,10"`
	Active  bool   `json:"active"`
}

type Item struct {
	ID int64 `json:"id"`
	ItemFields
}

type ItemFilter struct {
	Q      string `query:"q" maxLength:"100"`
	Active string `query:"active" enum:"true,false"`
	platform.Paging
}

type ItemList struct {
	Items []Item `json:"items" nullable:"false"`
	Total int64  `json:"total"`
}

// LineInput is one line as entered; the server computes its amounts.
type LineInput struct {
	ItemID          int64  `json:"item_id"`
	Description     string `json:"description" minLength:"1" maxLength:"500"`
	Quantity        string `json:"quantity" pattern:"^[0-9]{1,12}(\\.[0-9]{1,3})?$" doc:"More than 0, up to 3 decimals"`
	UnitPrice       int64  `json:"unit_price" minimum:"0"`
	DiscountPercent string `json:"discount_percent" pattern:"^[0-9]{1,3}(\\.[0-9]{1,2})?$" doc:"0 to 100, up to 2 decimals"`
	VatRate         string `json:"vat_rate" enum:"none,0,5,8,10"`
}

// DocFields are the editable fields of a quotation or order. ValidUntil is for quotations
// (required), DeliveryDate for orders (optional).
type DocFields struct {
	Date          string      `json:"date" format:"date"`
	OrgUnitID     int64       `json:"org_unit_id"`
	CustomerID    int64       `json:"customer_id"`
	ValidUntil    *string     `json:"valid_until" format:"date"`
	DeliveryDate  *string     `json:"delivery_date" format:"date"`
	PaymentTerms  *string     `json:"payment_terms" maxLength:"1000"`
	DeliveryTerms *string     `json:"delivery_terms" maxLength:"1000"`
	Note          *string     `json:"note" maxLength:"2000"`
	Lines         []LineInput `json:"lines" minItems:"1" maxItems:"200" nullable:"false"`
}

type NewDoc struct {
	RequestID string `json:"request_id" format:"uuid" doc:"Generated once per form: sending it again returns the first document"`
	DocFields
}

type DocUpdate struct {
	Version int32 `json:"version"`
	DocFields
}

// CustomerCopy is the customer as the document copied it on its last draft save.
type CustomerCopy struct {
	ID          int64   `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	TaxCode     *string `json:"tax_code"`
	Address     *string `json:"address"`
	Phone       *string `json:"phone"`
	Email       *string `json:"email"`
	ContactName *string `json:"contact_name"`
}

type Line struct {
	LineInput
	ItemCode string `json:"item_code"`
	Unit     string `json:"unit"`
	Amount   int64  `json:"amount" doc:"Quantity × unit price"`
	Discount int64  `json:"discount"`
	Vat      int64  `json:"vat"`
}

// DocRef names another document: the quotation of an order, the order of a quotation.
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

// Doc is a quotation or a sales order.
type Doc struct {
	ID            int64        `json:"id"`
	Kind          string       `json:"kind" enum:"quote,order"`
	Number        string       `json:"number"`
	Status        string       `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Version       int32        `json:"version"`
	Date          string       `json:"date" format:"date"`
	OrgUnitID     int64        `json:"org_unit_id"`
	OrgUnitName   string       `json:"org_unit_name"`
	Customer      CustomerCopy `json:"customer"`
	ValidUntil    *string      `json:"valid_until" format:"date"`
	DeliveryDate  *string      `json:"delivery_date" format:"date"`
	PaymentTerms  *string      `json:"payment_terms"`
	DeliveryTerms *string      `json:"delivery_terms"`
	Note          *string      `json:"note"`
	Lines         []Line       `json:"lines" nullable:"false"`
	Totals
	// The approval field: the largest line discount against the catalogue price, in percent.
	MaxDiscount string `json:"max_discount"`
	// Quotations: past their validity date; an order not cancelled comes from it, and that
	// order when the actor may view it.
	Expired bool    `json:"expired"`
	Ordered bool    `json:"ordered"`
	Order   *DocRef `json:"order"`
	// Orders: the quotation they came from, when the actor may view it.
	Quote *DocRef `json:"quote"`
	// edit, delete, submit, withdraw, cancel, print; quotations also create_order.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

type DocFilter struct {
	Q          string `query:"q" maxLength:"100" doc:"Number, customer code or name"`
	Status     string `query:"status" enum:"draft,pending_approval,posted,cancelled"`
	CustomerID int64  `query:"customer_id"`
	Sort       string `query:"sort" enum:"date,-date,number,-number,total,-total" default:"-date"`
	platform.Paging
}

type DocListItem struct {
	ID           int64   `json:"id"`
	Number       string  `json:"number"`
	Status       string  `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Date         string  `json:"date" format:"date"`
	CustomerID   int64   `json:"customer_id"`
	CustomerCode string  `json:"customer_code"`
	CustomerName string  `json:"customer_name"`
	OrgUnitName  string  `json:"org_unit_name"`
	Total        int64   `json:"total"`
	ValidUntil   *string `json:"valid_until" format:"date"`
	Expired      bool    `json:"expired"`
	// Quotations: an order not cancelled comes from it. Orders: made from a quotation.
	Ordered   bool `json:"ordered"`
	FromQuote bool `json:"from_quote"`
}

type DocList struct {
	Items []DocListItem `json:"items" nullable:"false"`
	Total int64         `json:"total"`
}

var (
	ErrCustomerCodeTaken = &platform.Error{Status: http.StatusConflict, Code: "customer_code_taken"}
	ErrItemCodeTaken     = &platform.Error{Status: http.StatusConflict, Code: "item_code_taken"}
	ErrCustomerInactive  = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "customer_inactive"}
	ErrValidUntil        = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "quote_valid_until_before_date"}
	ErrDeliveryDate      = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "order_delivery_before_date"}
	ErrQuoteExpired      = &platform.Error{Status: http.StatusConflict, Code: "quote_expired"}
	ErrQuoteNotPosted    = &platform.Error{Status: http.StatusConflict, Code: "quote_not_posted"}
	ErrQuoteHasOrder     = &platform.Error{Status: http.StatusConflict, Code: "quote_has_order"}
	// The request id was already used for another kind of document, or one the actor cannot see.
	ErrRequestReused = &platform.Error{Status: http.StatusConflict, Code: "request_id_reused"}
)

// errLine names the line (1-based) a value is wrong on.
func errLine(code string, line int) error {
	return &platform.Error{Status: http.StatusUnprocessableEntity, Code: code, Params: map[string]any{"line": line}}
}
