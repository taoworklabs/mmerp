package setting

// LegalEntitySetting is one legal-entity setting with its value (stored or default).
type LegalEntitySetting struct {
	Key    string   `json:"key" doc:"Translated as <key> and <key>.<value>"`
	Value  string   `json:"value"`
	Values []string `json:"values" nullable:"false"`
}
