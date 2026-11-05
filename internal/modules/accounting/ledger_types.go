package accounting

// JournalLine is one line of a journal entry: exactly one of Debit and Credit is positive.
type JournalLine struct {
	AccountID   int64
	OrgUnitID   *int64
	Kind        string
	Description string
	Debit       int64
	Credit      int64
}
