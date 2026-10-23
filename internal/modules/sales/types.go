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

// Actions is what the actor may do before picking a record: create.
type Actions struct {
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
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
)
