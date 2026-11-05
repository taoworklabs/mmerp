// Package accounting owns the chart of accounts and the general ledger.
package accounting

import "github.com/taoworklabs/mmerp/internal/platform/data"

type Service struct{ d Deps }

// NewService registers Accounting roles with core.
func NewService(d Deps) *Service {
	data.RegisterConstraints(data.Constraints{"accounting.accounts_code_key": ErrAccountCodeTaken})
	d.IAM.RegisterRoles("accounting", map[string][]string{
		"viewer":     {PermView},
		"accountant": {PermView, PermEntryEdit, PermPost},
		"admin":      {PermView, PermEntryEdit, PermPost, PermManage, PermApprovalManage},
	}, "admin")
	return &Service{d: d}
}
