package accounting

// AccountFields are what an edit sets; a parent has the same type as its children.
type AccountFields struct {
	Code     string `json:"code" pattern:"^[0-9A-Za-z.]{1,20}$"`
	Name     string `json:"name" minLength:"1" maxLength:"200"`
	Type     string `json:"type" enum:"asset,liability,equity,revenue,expense"`
	ParentID *int64 `json:"parent_id" required:"false" nullable:"true"`
	Active   bool   `json:"active"`
	// ByPartner: every journal line on the account names a partner. Postable accounts only;
	// fixed once the account has lines.
	ByPartner bool `json:"by_partner" required:"false"`
}

// Account is a chart line. Postable: active entries may be booked to it (it has no child).
type Account struct {
	ID int64 `json:"id"`
	AccountFields
	Postable bool `json:"postable"`
	HasLines bool `json:"has_lines"`
}

type AccountList struct {
	Items []Account `json:"items" nullable:"false"`
}
