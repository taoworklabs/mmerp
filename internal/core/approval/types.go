package approval

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// Op compares a document's field with a condition's value.
type Op string

const (
	OpGT     Op = "gt"
	OpGTE    Op = "gte"
	OpEq     Op = "eq"
	OpWithin Op = "within" // the unit or below it
)

// opsByKind lists the comparisons each kind of field allows.
var opsByKind = map[record.FieldKind][]Op{
	record.Number:  {OpGT, OpGTE},
	record.Choice:  {OpEq},
	record.OrgUnit: {OpWithin},
}

// Header fields every document has, beside the approval fields of its type.
const (
	FieldAmount  = "amount"
	FieldOrgUnit = "org_unit"
)

// ApproverKind says how a step finds its approvers.
type ApproverKind string

const (
	// ByRole: holders of a role covering the document's org unit.
	ByRole ApproverKind = "role"
	ByUser ApproverKind = "user"
	// ByModule: chosen by the document type, e.g. the direct manager.
	ByModule ApproverKind = "module"
)

// Condition tests one header field (amount, org_unit) or approval field of a document.
type Condition struct {
	Field string `json:"field" minLength:"1"`
	Op    Op     `json:"op" enum:"gt,gte,eq,within"`
	Value string `json:"value" minLength:"1" maxLength:"30"`
}

// Approver says who approves a step.
type Approver struct {
	Kind    ApproverKind `json:"kind" enum:"role,user,module" doc:"role: holders of a role covering the document's org unit; module: chosen by the document type, e.g. the direct manager"`
	Product string       `json:"product,omitempty"`
	Role    string       `json:"role,omitempty"`
	UserID  *int64       `json:"user_id,omitempty"`
}

type Step struct {
	// Without a condition the step always applies.
	Condition *Condition `json:"condition,omitempty"`
	Approver  Approver   `json:"approver"`
}

type RuleInput struct {
	Steps []Step `json:"steps" minItems:"1" maxItems:"10" nullable:"false"`
	// How far up module approvers may go when a level has nobody but the submitter.
	MaxLevels       int32  `json:"max_levels" minimum:"1" maximum:"10" default:"3"`
	FallbackProduct string `json:"fallback_product" minLength:"1"`
	FallbackRole    string `json:"fallback_role" minLength:"1"`
}

// Field is an approval field of a document type, for building conditions.
type Field struct {
	Key     string          `json:"key"`
	Kind    string          `json:"kind" enum:"number,choice,org_unit"`
	Label   string          `json:"label" doc:"i18n key"`
	Options []record.Option `json:"options,omitempty"`
	// The comparisons a condition on this field may use.
	Ops []Op `json:"ops" nullable:"false"`
}

// TypeRule is a document type with its approval fields and its rule, if any.
type TypeRule struct {
	DocType string  `json:"doc_type"`
	Product string  `json:"product"`
	Fields  []Field `json:"fields" nullable:"false"`
	// Whether the type chooses approvers itself (approver kind module).
	ModuleApprovers bool       `json:"module_approvers"`
	Rule            *RuleInput `json:"rule,omitempty"`
	// save and delete, or none while the product is disabled.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

// ApproverUser is an account a rule may name as approver.
type ApproverUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
}

type InboxItem struct {
	InstanceID      int64  `json:"instance_id"`
	DocType         string `json:"doc_type"`
	DocID           int64  `json:"doc_id"`
	Number          string `json:"number"`
	Date            string `json:"date" format:"date"`
	Step            int32  `json:"step"`
	SubmittedByName string `json:"submitted_by_name"`
	SubmittedAt     string `json:"submitted_at" format:"date-time"`
}

type StepView struct {
	Position      int32    `json:"position"`
	ApproverNames []string `json:"approver_names" nullable:"false"`
	// The step went to the fallback role because nobody else could approve it.
	Fallback      bool    `json:"fallback"`
	Decision      *string `json:"decision" enum:"approved,rejected"`
	DecidedByName *string `json:"decided_by_name"`
	Reason        *string `json:"reason"`
	DecidedAt     *string `json:"decided_at" format:"date-time"`
}

// Instance is the latest submission of a document, for the approval panel.
type Instance struct {
	ID              int64      `json:"id"`
	Status          string     `json:"status" enum:"open,approved,rejected,withdrawn,stale"`
	CurrentStep     int32      `json:"current_step"`
	SubmittedByName string     `json:"submitted_by_name"`
	SubmittedAt     string     `json:"submitted_at" format:"date-time"`
	Steps           []StepView `json:"steps" nullable:"false"`
	// approve, reject, reassign: what the actor may do at the current step.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

var (
	// The instance is no longer open, or its current step is not the one acted on.
	ErrClosed     = &platform.Error{Status: http.StatusConflict, Code: "approval_closed"}
	ErrNoApprover = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "no_approver"}
	// Nobody approves their own submission, not even through reassignment.
	ErrSelfApproval = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "self_approval"}
	// A reassigned approver must be able to view the document, or the step could never move.
	ErrApproverCannotView = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "approver_cannot_view"}
)

func errInvalidRule(step int, reason string) error {
	return &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_rule", Params: map[string]any{"step": step, "reason": reason}}
}
