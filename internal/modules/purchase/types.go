package purchase

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

const (
	// Suppliers by their owning org unit.
	PermSupplierView = "purchase.supplier.view"
	PermSupplierEdit = "purchase.supplier.edit"
	// Purchase orders by their org unit; goods receipts, invoices and payments reuse them.
	PermOrderView = "purchase.order.view"
	PermOrderEdit = "purchase.order.edit"
	// Approval rules of Purchasing document types; core's approval checks it by name.
	PermApprovalManage = "purchase.approval.manage"
)

// Actions is what the actor may do before picking a record: create.
type Actions struct {
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

var (
	ErrSupplierCodeTaken = &platform.Error{Status: http.StatusConflict, Code: "supplier_code_taken"}
	ErrSupplierExists    = &platform.Error{Status: http.StatusConflict, Code: "supplier_exists_for_partner"}
	ErrPartnerRequired   = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "supplier_partner_required"}
	ErrSupplierInactive  = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "supplier_inactive"}
)
