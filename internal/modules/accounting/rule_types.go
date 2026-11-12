package accounting

// Rule books the lines of one posting kind to an account, on a side; unset until configured.
type Rule struct {
	Kind      string  `json:"kind"`
	AccountID *int64  `json:"account_id" nullable:"true"`
	Side      *string `json:"side" nullable:"true" enum:"debit,credit"`
	// ByPartner: the kind's lines name a partner, so it maps only to an account tracked by partner.
	ByPartner bool `json:"by_partner"`
}

type RuleList struct {
	Items []Rule `json:"items" nullable:"false"`
}

type RuleFields struct {
	AccountID int64  `json:"account_id"`
	Side      string `json:"side" enum:"debit,credit"`
}

// Start is the day a legal entity starts booking; fixed once it has entries.
type Start struct {
	LegalEntityID int64   `json:"legal_entity_id"`
	StartDate     *string `json:"start_date" nullable:"true"`
	Fixed         bool    `json:"fixed"`
}

type StartList struct {
	Items []Start `json:"items" nullable:"false"`
}
