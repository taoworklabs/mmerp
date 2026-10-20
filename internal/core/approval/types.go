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
	OpHas    Op = "has"    // the set holds the id
)

// opsByKind lists the comparisons each kind of field allows.
var opsByKind = map[record.FieldKind][]Op{
	record.Number:  {OpGT, OpGTE},
	record.Choice:  {OpEq},
	record.OrgUnit: {OpWithin},
	record.Set:     {OpHas},
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

// Approver says who approves a step.
type Approver struct {
	Kind    ApproverKind `json:"kind" enum:"role,user,module" doc:"role: holders of a role covering the document's org unit; module: chosen by the document type, e.g. the direct manager"`
	Product string       `json:"product,omitempty"`
	Role    string       `json:"role,omitempty"`
	// Kind user names up to 20 approvers.
	UserIDs []int64 `json:"user_ids,omitempty" maxItems:"20"`
}

type RuleInput struct {
	Flow Flow `json:"flow" minItems:"1" nullable:"false"`
	// How far up module approvers may go when a level has nobody but the submitter.
	MaxLevels       int32  `json:"max_levels" minimum:"1" maximum:"10" default:"3"`
	FallbackProduct string `json:"fallback_product" minLength:"1"`
	FallbackRole    string `json:"fallback_role" minLength:"1"`
}

// Field is an approval field of a document type, for building conditions.
type Field struct {
	Key     string          `json:"key"`
	Kind    string          `json:"kind" enum:"number,choice,org_unit,set"`
	Label   string          `json:"label" doc:"i18n key"`
	Options []record.Option `json:"options,omitempty"`
	// The comparisons a condition on this field may use.
	Ops []Op `json:"ops" nullable:"false"`
	// Values are picked by searching the field's options endpoint.
	Searchable bool `json:"searchable"`
}

// TypeRule is one action of a document type, with its approval fields and its rule, if any.
// The frontend labels Action through i18n (shared.rule.action.<name>, or <product>.action.<name>
// for module actions).
type TypeRule struct {
	DocType string  `json:"doc_type"`
	Action  string  `json:"action"`
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
	Action          string `json:"action"`
	Number          string `json:"number"`
	Date            string `json:"date" format:"date"`
	Step            int32  `json:"step"`
	SubmittedByName string `json:"submitted_by_name"`
	SubmittedAt     string `json:"submitted_at" format:"date-time"`
}

// StepVoter is one who may vote on a step, with their vote once cast.
type StepVoter struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Decision  *string `json:"decision" enum:"approved,rejected"`
	Reason    *string `json:"reason"`
	DecidedAt *string `json:"decided_at" format:"date-time"`
}

type StepView struct {
	Position   int32  `json:"position"`
	Status     string `json:"status" enum:"waiting,open,approved,rejected,cancelled"`
	Approvals  int32  `json:"approvals"`
	Rejections int32  `json:"rejections"`
	// The step went to the fallback role because nobody else could approve it.
	Fallback bool        `json:"fallback"`
	Voters   []StepVoter `json:"voters" nullable:"false"` // empty while waiting
	// approve, reject (the actor may vote), reassign (fallback role holder).
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

// Instance is the latest submission of a document, for the approval panel.
type Instance struct {
	ID int64 `json:"id"`
	// post, cancel, or a module action of the document's type.
	Action          string     `json:"action"`
	Status          string     `json:"status" enum:"open,approved,rejected,withdrawn,stale"`
	SubmittedByName string     `json:"submitted_by_name"`
	SubmittedAt     string     `json:"submitted_at" format:"date-time"`
	Plan            Plan       `json:"plan"`
	Steps           []StepView `json:"steps" nullable:"false"` // by position
	// withdraw, for the submitter of an open request other than post.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

// Delegation lets a delegate vote for a user in approval steps while it lasts.
type Delegation struct {
	ID           int64  `json:"id"`
	UserID       int64  `json:"user_id"`
	UserName     string `json:"user_name"`
	DelegateID   int64  `json:"delegate_id"`
	DelegateName string `json:"delegate_name"`
	StartsOn     string `json:"starts_on" format:"date"`
	EndsOn       string `json:"ends_on" format:"date"`
	Active       bool   `json:"active"` // today falls within it
	// delete, for its user and for core.approval.manage holders.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

type DelegationInput struct {
	UserID     *int64 `json:"user_id,omitempty" doc:"Whose seats; omitted means the actor's own"`
	DelegateID int64  `json:"delegate_id"`
	StartsOn   string `json:"starts_on" format:"date"`
	EndsOn     string `json:"ends_on" format:"date"`
}

var (
	// The instance is no longer open, or its current step is not the one acted on.
	ErrClosed     = &platform.Error{Status: http.StatusConflict, Code: "approval_closed"}
	ErrNoApprover = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "no_approver"}
	// Each voter votes once per step; a voter who voted can no longer be reassigned.
	ErrAlreadyVoted = &platform.Error{Status: http.StatusConflict, Code: "approval_already_voted"}
	// Reassigning to someone already a voter on the step would double them up.
	ErrAlreadyApprover = &platform.Error{Status: http.StatusConflict, Code: "approval_already_approver"}
	// Nobody approves their own submission, not even through reassignment.
	ErrSelfApproval = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "self_approval"}
	// A reassigned approver must be able to view the document, or the step could never move.
	ErrApproverCannotView = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "approver_cannot_view"}
)

// errNoApprover names the step that found too few approvers.
func errNoApprover(step int32) error {
	return &platform.Error{Status: ErrNoApprover.Status, Code: ErrNoApprover.Code, Params: map[string]any{"step": step}}
}

// errInvalidRule names the offending flow element by path ("" for the rule itself).
func errInvalidRule(path, reason string) error {
	return &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_rule", Params: map[string]any{"path": path, "reason": reason}}
}

// ErrInvalidDelegation says why a delegation was refused: self, dates, past or overlap.
func ErrInvalidDelegation(reason string) *platform.Error {
	return &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_delegation", Params: map[string]any{"reason": reason}}
}
