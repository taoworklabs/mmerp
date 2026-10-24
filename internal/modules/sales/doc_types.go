package sales

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

// errLine names the line (1-based) a value is wrong on.
func errLine(code string, line int) error {
	return &platform.Error{Status: http.StatusUnprocessableEntity, Code: code, Params: map[string]any{"line": line}}
}
