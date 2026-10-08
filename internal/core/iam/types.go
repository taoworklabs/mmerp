package iam

import (
	"net/http"
	"slices"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// Me is the signed-in user as the frontend sees it.
type Me struct {
	ID     int64  `json:"id"`
	Login  string `json:"login"`
	Name   string `json:"name"`
	Locale string `json:"locale" enum:"vi,en"`
	// Tenant time zone; "today" and day boundaries use it.
	Timezone string `json:"timezone"`
	// Opaque: the frontend only compares it with X-Authz-Version.
	AuthzVersion string   `json:"authz_version"`
	Products     []string `json:"products" nullable:"false"`
	// Products with at least one record, enabled or not: a disabled one stays readable.
	ProductsWithData []string `json:"products_with_data" nullable:"false"`
	// Every permission the user has at any org unit, by product. Only for hiding menus and pages.
	Permissions map[string][]string `json:"permissions"`
}

// User is an account as administrators see it.
type User struct {
	ID    int64   `json:"id"`
	Login string  `json:"login"`
	Name  string  `json:"name"`
	Email *string `json:"email" doc:"Where email notifications go; null for none"`
}

// Role is a product role and the permissions it grants.
type Role struct {
	Product     string   `json:"product"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	// Granted only tenant-wide, never at an org unit.
	TenantWide bool `json:"tenant_wide"`
}

// Grant is one role given to a user at an org unit, or tenant-wide when OrgUnitID is nil.
type Grant struct {
	ID          int64   `json:"id"`
	Product     string  `json:"product"`
	Role        string  `json:"role"`
	OrgUnitID   *int64  `json:"org_unit_id"`
	OrgUnitName *string `json:"org_unit_name"`
}

// OrgUnit is a node of the permission-scope tree. A company node is a legal entity.
type OrgUnit struct {
	ID int64 `json:"id"`
	OrgUnitInput
}

type OrgUnitInput struct {
	ParentID  *int64  `json:"parent_id"`
	Kind      string  `json:"kind" enum:"group,company,branch,department"`
	Name      string  `json:"name" minLength:"1" maxLength:"200"`
	TaxCode   *string `json:"tax_code,omitempty" maxLength:"20"`
	LegalName *string `json:"legal_name,omitempty" maxLength:"300"`
	Address   *string `json:"address,omitempty" maxLength:"500"`
}

// Scope is where a user has a permission: tenant-wide, or these units (subtrees included).
type Scope struct {
	All   bool
	Units []int64
}

func (s Scope) Has(unit int64) bool { return s.All || slices.Contains(s.Units, unit) }

// Any reports whether the permission holds anywhere.
func (s Scope) Any() bool { return s.All || len(s.Units) > 0 }

// Session is what a valid session cookie resolves to.
type Session struct {
	UserID       int64
	AuthzVersion string
}

var (
	ErrInvalidCredentials = &platform.Error{Status: http.StatusUnauthorized, Code: "invalid_credentials"}
	ErrPasswordTooShort   = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "password_too_short", Params: map[string]any{"min": MinPasswordLength}}
	ErrLoginTaken         = &platform.Error{Status: http.StatusConflict, Code: "login_taken"}
	ErrUnknownRole        = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "unknown_role"}
	ErrRoleTenantWide     = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "role_tenant_wide"}
	ErrRoleAlreadyGranted = &platform.Error{Status: http.StatusConflict, Code: "role_already_granted"}
	ErrOrgUnitCycle       = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "org_unit_cycle"}
	// A company or group under a company, or another kind outside every company.
	ErrOrgUnitMisplaced       = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "org_unit_misplaced"}
	ErrLegalFieldsCompanyOnly = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "legal_fields_company_only"}
)

const MinPasswordLength = 8
