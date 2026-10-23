package partner

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

const (
	// Both held anywhere cover every partner.
	PermView = "partner.view"
	PermEdit = "partner.edit"
)

var (
	ErrCodeTaken = &platform.Error{Status: http.StatusConflict, Code: "partner_code_taken"}
	// Params: partner_id, the partner holding the tax code.
	ErrTaxCodeTaken = &platform.Error{Status: http.StatusConflict, Code: "partner_tax_code_taken"}
)

// PartnerFields are a partner's identity; the code is set once, at creation.
type PartnerFields struct {
	Name        string  `json:"name" minLength:"1" maxLength:"300"`
	TaxCode     *string `json:"tax_code" maxLength:"20"`
	Address     *string `json:"address" maxLength:"500"`
	Phone       *string `json:"phone" maxLength:"50"`
	Email       *string `json:"email" maxLength:"200"`
	ContactName *string `json:"contact_name" maxLength:"200"`
}

type NewPartner struct {
	Code string `json:"code,omitempty" maxLength:"50" doc:"Empty or absent: issued from the partner.partner code series"`
	PartnerFields
}

type ListItem struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	PartnerFields
}

type Partner struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	PartnerFields
	// edit, attach.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

type Filter struct {
	Q       string `query:"q" maxLength:"100"`
	TaxCode string `query:"tax_code" maxLength:"20" doc:"Exact, trimmed and case-insensitive"`
	Sort    string `query:"sort" enum:"code,-code,name,-name" default:"code"`
	platform.Paging
}

type List struct {
	Items []ListItem `json:"items" nullable:"false"`
	Total int64      `json:"total"`
}
