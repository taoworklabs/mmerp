package accounting

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

const (
	PermView      = "accounting.view"
	PermEntryEdit = "accounting.entry.edit"
	PermPost      = "accounting.post"
	// Chart of accounts and accounting settings.
	PermManage = "accounting.manage"
	// Approval rules of accounting document types; core's approval checks it by name.
	PermApprovalManage = "accounting.approval.manage"
)

var (
	ErrAccountCodeTaken     = &platform.Error{Status: http.StatusConflict, Code: "account_code_taken"}
	ErrAccountParentInvalid = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "account_parent_invalid"}
	ErrAccountNameRequired  = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "account_name_required"}
	ErrAccountNotPostable   = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "account_not_postable"}
	ErrRuleSideInvalid      = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "rule_side_invalid"}
	ErrStartDateFixed       = &platform.Error{Status: http.StatusConflict, Code: "start_date_fixed"}
	// Params: locked_until.
	ErrStartDateInLockedPeriod = &platform.Error{Status: http.StatusConflict, Code: "start_date_in_locked_period"}
	ErrAccountInUse            = &platform.Error{Status: http.StatusConflict, Code: "account_in_use"}
	// ErrRulePartnerMismatch: a kind naming a partner maps only to a tracked account, any other kind only to an untracked one.
	ErrRulePartnerMismatch = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "rule_partner_mismatch"}
	// ErrAccountRuleMismatch: changing the flag would leave a rule pointing at the account unmatched.
	ErrAccountRuleMismatch = &platform.Error{Status: http.StatusConflict, Code: "account_partner_rule_mismatch"}
	// Fewer than two lines, or debits differ from credits.
	ErrEntryUnbalanced = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "entry_unbalanced"}
	// ErrEntryPartnerMismatch: a line names a partner on an untracked account, or none on a tracked one.
	ErrEntryPartnerMismatch = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "entry_partner_mismatch"}
)
