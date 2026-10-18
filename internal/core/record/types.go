package record

import (
	"context"
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type Kind int

const (
	// Catalog is master data without a lifecycle.
	Catalog Kind = iota + 1
	// Document has a number, date, legal entity and lifecycle.
	Document
)

type Action string

const (
	View   Action = "view"
	Edit   Action = "edit"
	Post   Action = "post"
	Cancel Action = "cancel"
	Export Action = "export"
	// ViewFiles lists and downloads attachments; a type answers it to ask for more than view.
	ViewFiles Action = "view_files"
	// Attach adds attachments and removes those of others.
	Attach Action = "attach"
	// Print makes a PDF of the record; only types with a print template answer it.
	Print Action = "print"
	// Create and Import come before a record exists, so they are asked of a type, not an id.
	Create Action = "create"
	Import Action = "import"
)

// Perms names, for each action, the permission of the type's product that grants it.
// Export and Print follow View, Post and Cancel follow Edit, Create follows Edit;
// ViewFiles, Attach and Import left empty mean the type has none.
type Perms struct {
	View, Edit, Post, Cancel, Export, Print, ViewFiles, Attach, Create, Import string
}

// permFields lists every action with the Perms field naming its permission and the
// action it follows when that field is empty.
var permFields = []struct {
	action  Action
	field   func(Perms) string
	follows Action
}{
	{View, func(p Perms) string { return p.View }, ""},
	{Edit, func(p Perms) string { return p.Edit }, ""},
	{Post, func(p Perms) string { return p.Post }, Edit},
	{Cancel, func(p Perms) string { return p.Cancel }, Edit},
	{Create, func(p Perms) string { return p.Create }, Edit},
	{Export, func(p Perms) string { return p.Export }, View},
	{Print, func(p Perms) string { return p.Print }, View},
	{ViewFiles, func(p Perms) string { return p.ViewFiles }, ""},
	{Attach, func(p Perms) string { return p.Attach }, ""},
	{Import, func(p Perms) string { return p.Import }, ""},
}

// of is the permission that grants a, or "" when the type has none.
func (p Perms) of(a Action) string {
	for _, f := range permFields {
		if f.action != a {
			continue
		}
		if own := f.field(p); own != "" || f.follows == "" {
			return own
		}
		return p.of(f.follows)
	}
	return ""
}

type Status string

const (
	Draft           Status = "draft"
	PendingApproval Status = "pending_approval"
	Posted          Status = "posted"
	Cancelled       Status = "cancelled"
)

// Ref names one record.
type Ref struct {
	Type string
	ID   int64
}

// Type is what a module registers for each of its record types.
type Type struct {
	Code    string // <module>.<type>, e.g. hrm.employee
	Product string
	Kind    Kind
	// Perms declares the permission of each action, held at the record's org unit.
	Perms Perms
	// Grant, the relation grant, may still grant a declared action the permission did
	// not, to a user related to the record (e.g. the employee's own requests).
	Grant func(ctx context.Context, id int64, action Action) (bool, error)
	// SubjectCreates lets the subject named in Create's header create without Perms.Create.
	SubjectCreates bool
	// Refine may only narrow an action already granted (e.g. a further sensitive permission).
	Refine func(ctx context.Context, id int64, action Action) (bool, error)
	// Locate returns a catalog record's org unit, or platform.ErrNotFound. Catalogs only.
	Locate func(ctx context.Context, id int64) (int64, error)

	// Documents only.
	NumberPrefix string  // e.g. NP gives NP-2026-00001
	Fields       []Field // approval fields
	// OnTransition runs in the transaction of every status change except into
	// pending_approval; d holds the new status. An error rolls the change back.
	OnTransition func(ctx context.Context, d Doc, from Status) error
	// BeforeSubmit runs when a draft is sent, before approval is asked; an error
	// keeps it a draft (e.g. a payroll whose sources changed).
	BeforeSubmit func(ctx context.Context, d Doc) error
	// OnDelete deletes the module's own rows of a draft being deleted, in the deletion's
	// transaction after the document's row is gone; an error keeps the draft.
	OnDelete func(ctx context.Context, id int64) error
	// Approvers returns the users for an approval step chosen by the module;
	// level 1 is the nearest (e.g. the direct manager), 2 the one above, and so on.
	Approvers func(ctx context.Context, ref Ref, level int) ([]int64, error)
	// Subjects returns the users a document is about (e.g. the employee on leave);
	// like its submitter, they never approve it.
	Subjects func(ctx context.Context, id int64) ([]int64, error)

	// Catalog types only. HasData reports whether the product holds any such record, so a
	// disabled product with master data but no document stays readable.
	HasData func(ctx context.Context) (bool, error)
}

type FieldKind string

const (
	Number  FieldKind = "number"   // decimal string, e.g. "2.5"
	Choice  FieldKind = "choice"   // one of Options
	OrgUnit FieldKind = "org_unit" // org unit id
)

// Field is an approval field: a typed value approval rules may test.
type Field struct {
	Key     string
	Kind    FieldKind
	Label   string // i18n key
	Options func(ctx context.Context) ([]Option, error)
}

// Option is one value of a choice field; Label is shown as is.
type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Header is what a module tells record about a document. Field values are
// strings in the field's format.
type Header struct {
	Date      string // YYYY-MM-DD
	OrgUnitID int64
	Amount    *int64
	Fields    map[string]string
	// RequestID makes Create once per user (a UUID, empty for none); Edit ignores it.
	RequestID string
	// Subject is the account the new document is about (e.g. the employee's); only Create reads it.
	Subject int64
}

// Doc is a document as record keeps it.
type Doc struct {
	Ref
	Number         string
	Status         Status
	Version        int32
	Date           string
	LegalEntityID  int64
	OrgUnitID      int64
	Amount         *int64
	Fields         map[string]string
	ApprovalTicket *int64
	// Who sent it while it is pending_approval; only they withdraw it.
	SubmittedBy *int64
	// Reused is set by Create when it returns the document of an earlier request with the
	// same request id: the module then skips its own insert.
	Reused bool
}

type PeriodLock struct {
	LegalEntityID   int64   `json:"legal_entity_id"`
	LegalEntityName string  `json:"legal_entity_name"`
	LockedUntil     *string `json:"locked_until" format:"date"`
}

type HistoryEntry struct {
	ID        int64          `json:"id"`
	At        string         `json:"at" format:"date-time"`
	ActorName *string        `json:"actor_name"`
	Action    string         `json:"action"`
	Data      map[string]any `json:"data"`
}

var (
	ErrVersionConflict     = &platform.Error{Status: http.StatusConflict, Code: "version_conflict"}
	ErrNotEditable         = &platform.Error{Status: http.StatusConflict, Code: "document_not_editable"}
	ErrInvalidTransition   = &platform.Error{Status: http.StatusConflict, Code: "invalid_transition"}
	ErrLegalEntityChanged  = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "legal_entity_changed"}
	ErrOrgUnitHasDocuments = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "org_unit_has_documents"}
	ErrRequestReused       = &platform.Error{Status: http.StatusConflict, Code: "request_id_reused"}
	// The approval instance no longer matches the document: withdrawn, resubmitted or already decided.
	ErrApprovalStale = &platform.Error{Status: http.StatusConflict, Code: "approval_stale"}
)

func errPeriodLocked(lockedUntil string) error {
	return &platform.Error{Status: http.StatusConflict, Code: "period_locked", Params: map[string]any{"locked_until": lockedUntil}}
}

func errInvalidField(field string) error {
	return &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_field", Params: map[string]any{"field": field}}
}
