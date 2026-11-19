// Package purchase owns suppliers, purchase orders, goods receipts, supplier invoices and payments.
package purchase

import (
	"context"
)

type Service struct{ d Deps }

// NewService registers Purchasing roles and record types with core.
func NewService(d Deps) *Service {
	declareKinds()
	s := &Service{d: d}
	work := []string{PermSupplierView, PermSupplierEdit, PermOrderView, PermOrderEdit}
	d.IAM.RegisterRoles("purchase", map[string][]string{
		// The same permissions under two names, so approval rules can route steps to managers.
		"staff":   work,
		"manager": work,
		"viewer":  {PermSupplierView, PermOrderView},
		// The approval rules apply to every org unit.
		"approval_admin": {PermApprovalManage},
	}, "approval_admin")
	return s
}

func (s *Service) require(ctx context.Context, perm string, unit int64, more ...int64) error {
	return s.d.IAM.Require(ctx, "purchase", perm, unit, more...)
}
