// Package purchase owns suppliers, purchase orders, goods receipts, supplier invoices and payments.
package purchase

import (
	"context"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/purchase/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform/data"
)

const supplierType = "purchase.supplier"

type Service struct{ d Deps }

// NewService registers Purchasing roles and record types with core.
func NewService(d Deps) *Service {
	declareKinds()
	s := &Service{d: d}
	data.RegisterConstraints(data.Constraints{
		"purchase.suppliers_code_key":             ErrSupplierCodeTaken,
		"purchase.suppliers_partner_org_unit_key": ErrSupplierExists,
	})
	work := []string{PermSupplierView, PermSupplierEdit, PermOrderView, PermOrderEdit}
	d.IAM.RegisterRoles("purchase", map[string][]string{
		// The same permissions under two names, so approval rules can route steps to managers.
		"staff":   work,
		"manager": work,
		"viewer":  {PermSupplierView, PermOrderView},
		// The approval rules apply to every org unit.
		"approval_admin": {PermApprovalManage},
	}, "approval_admin")
	d.Numbering.RegisterSeries(supplierType, "purchase", "NC")
	// A supplier's files are as open as the supplier.
	d.Record.Register(record.Type{Code: supplierType, Product: "purchase", Kind: record.Catalog,
		Perms:   record.Perms{View: PermSupplierView, Edit: PermSupplierEdit, ViewFiles: PermSupplierView, Attach: PermSupplierEdit},
		Locate:  locateSupplier,
		HasData: func(ctx context.Context) (bool, error) { return store.AnyCatalog.Run(ctx) }})
	s.registerOrder()
	return s
}

func (s *Service) require(ctx context.Context, perm string, unit int64, more ...int64) error {
	return s.d.IAM.Require(ctx, "purchase", perm, unit, more...)
}
