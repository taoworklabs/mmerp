package accounting

// AccountingPartner is a partner as a journal entry line editor picks it.
type AccountingPartner struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type AccountingPartnerList struct {
	Items []AccountingPartner `json:"items" nullable:"false"`
}
