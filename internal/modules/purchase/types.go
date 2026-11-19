package purchase

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
