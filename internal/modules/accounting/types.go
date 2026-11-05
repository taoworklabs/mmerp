package accounting

const (
	PermView      = "accounting.view"
	PermEntryEdit = "accounting.entry.edit"
	PermPost      = "accounting.post"
	// Chart of accounts and accounting settings.
	PermManage = "accounting.manage"
	// Approval rules of accounting document types; core's approval checks it by name.
	PermApprovalManage = "accounting.approval.manage"
)
