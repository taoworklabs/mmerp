package accounting

// PendingDoc is a document whose posting lines have not reached the ledger.
type PendingDoc struct {
	DocType       string   `json:"doc_type"`
	DocID         int64    `json:"doc_id"`
	Number        string   `json:"number"`
	Date          string   `json:"date" format:"date"`
	LegalEntityID int64    `json:"legal_entity_id"`
	Reason        string   `json:"reason" enum:"queued,unmapped,unbalanced"`
	MissingKinds  []string `json:"missing_kinds"`
}

type PendingList struct {
	Items []PendingDoc `json:"items"`
}
