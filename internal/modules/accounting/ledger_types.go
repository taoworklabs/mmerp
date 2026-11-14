package accounting

// MaxLineAmount and MaxLines bound a journal entry so its sums fit int64.
const (
	MaxLineAmount = 1_000_000_000_000_000
	MaxLines      = 500
)

// JournalLine is one line of a journal entry: exactly one of Debit and Credit is positive.
type JournalLine struct {
	AccountID   int64
	OrgUnitID   *int64
	PartnerID   *int64
	Kind        string
	Description string
	Debit       int64
	Credit      int64
}
