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
)

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
	// Can answers for the owning module; the product gate is applied on top.
	Can func(ctx context.Context, id int64, action Action) (bool, error)

	// Documents only.
	NumberPrefix string  // e.g. NP gives NP-2026-00001
	Fields       []Field // approval fields
	// OnTransition runs in the transaction of every status change except into
	// pending_approval; d holds the new status. An error rolls the change back.
	OnTransition func(ctx context.Context, d Doc, from Status) error
	// BeforeSubmit runs when a draft is sent, before approval is asked; an error
	// keeps it a draft (e.g. a payroll whose sources changed).
	BeforeSubmit func(ctx context.Context, d Doc) error
	// Approvers returns the users for an approval step chosen by the module;
	// level 1 is the nearest (e.g. the direct manager), 2 the one above, and so on.
	Approvers func(ctx context.Context, ref Ref, level int) ([]int64, error)
	// Subjects returns the users a document is about (e.g. the employee on leave);
	// like its submitter, they never approve it.
	Subjects func(ctx context.Context, id int64) ([]int64, error)
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
	// The approval instance no longer matches the document: withdrawn, resubmitted or already decided.
	ErrApprovalStale = &platform.Error{Status: http.StatusConflict, Code: "approval_stale"}
)

func errPeriodLocked(lockedUntil string) error {
	return &platform.Error{Status: http.StatusConflict, Code: "period_locked", Params: map[string]any{"locked_until": lockedUntil}}
}

func errInvalidField(field string) error {
	return &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_field", Params: map[string]any{"field": field}}
}
