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
	ErrStartDateFixed       = &platform.Error{Status: http.StatusConflict, Code: "start_date_fixed"}
	ErrAccountInUse         = &platform.Error{Status: http.StatusConflict, Code: "account_in_use"}
)
