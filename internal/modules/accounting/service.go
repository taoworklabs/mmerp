// Package accounting owns the chart of accounts and the general ledger.
package accounting

import (
	"context"

	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
)

type Service struct{ d Deps }

// NewService registers Accounting roles with core.
func NewService(d Deps) *Service {
	data.RegisterConstraints(data.Constraints{"accounting.accounts_code_key": ErrAccountCodeTaken,
		"accounting.closing_accounts_result_fkey": ErrAccountInUse, "accounting.closing_accounts_retained_fkey": ErrAccountInUse})
	d.IAM.RegisterRoles("accounting", map[string][]string{
		"viewer":     {PermView},
		"accountant": {PermView, PermEntryEdit, PermPost},
		"admin":      {PermView, PermEntryEdit, PermPost, PermManage, PermApprovalManage},
	}, "admin")
	s := &Service{d: d}
	s.registerJournalEntry()
	s.registerClosing()
	return s
}

// requireView lets through anyone who views accounting somewhere: tenant-wide data (closing
// accounts, statement forms) is the same for every legal entity.
func (s *Service) requireView(ctx context.Context) error {
	sc, err := s.d.IAM.Scope(ctx, "accounting", PermView)
	if err != nil {
		return err
	}
	if !sc.Any() {
		return platform.ErrForbidden
	}
	return nil
}
