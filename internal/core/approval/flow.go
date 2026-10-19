package approval

import (
	"slices"
	"strconv"

	"github.com/taoworklabs/mmerp/internal/core/record"
)

// Flow is a document type's approval flow: steps and splits in reading order.
type Flow []Element

// Element is one step of a flow. Exactly one of Step and Split is set.
type Element struct {
	Step  *StepDef `json:"step,omitempty"`
	Split *Split   `json:"split,omitempty"`
}

// StepDef is one approval step: who approves it, and how many approvals or
// rejections it takes to move on.
type StepDef struct {
	Approver   Approver `json:"approver"`
	Approvals  int32    `json:"approvals" required:"false" minimum:"1" maximum:"20" default:"1"`
	Rejections int32    `json:"rejections" required:"false" minimum:"1" maximum:"20" default:"1"`
}

// SplitMode says whether a split picks one branch or runs every holding branch.
type SplitMode string

const (
	ChooseOne SplitMode = "one"
	Parallel  SplitMode = "all"
)

// Split branches a flow: choose-one takes the first holding branch, falling
// back to Otherwise; parallel runs every holding branch at once.
type Split struct {
	Mode      SplitMode `json:"mode" enum:"one,all"`
	Branches  []Branch  `json:"branches" minItems:"1" maxItems:"10" nullable:"false"`
	Otherwise Flow      `json:"otherwise,omitempty"` // choose-one only
}

// Branch is one path of a split, taken when its Condition holds.
type Branch struct {
	Condition *Expr `json:"condition,omitempty"` // nil holds
	Body      Flow  `json:"body" nullable:"false"`
}

// Expr is a condition tree. Exactly one of All, Any and the leaf (Field, Op,
// Value) is set.
type Expr struct {
	All   []Expr `json:"all,omitempty"`
	Any   []Expr `json:"any,omitempty"`
	Field string `json:"field,omitempty"`
	Op    Op     `json:"op,omitempty" enum:"gt,gte,eq,within,has"`
	Value string `json:"value,omitempty" minLength:"1" maxLength:"30"`
}

// Plan is a resolved flow: a leaf (Step > 0), a sequence or a parallel group.
type Plan struct {
	Step int32  `json:"step,omitempty"`
	Seq  []Plan `json:"seq,omitempty"`
	Par  []Plan `json:"par,omitempty"`
}

// eval reports whether e holds: all requires every child to hold, any
// requires some child, and a leaf is handed to leaf.
func (e Expr) eval(leaf func(Expr) (bool, error)) (bool, error) {
	if len(e.All) > 0 {
		for _, c := range e.All {
			ok, err := c.eval(leaf)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}
		return true, nil
	}
	if len(e.Any) > 0 {
		for _, c := range e.Any {
			ok, err := c.eval(leaf)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil
	}
	return leaf(e)
}

// normalize sets a zero Approvals or Rejections to 1, recursively: Go callers
// may leave them unset.
func (f Flow) normalize() Flow {
	for _, e := range f {
		if e.Step != nil {
			if e.Step.Approvals == 0 {
				e.Step.Approvals = 1
			}
			if e.Step.Rejections == 0 {
				e.Step.Rejections = 1
			}
		}
		if e.Split != nil {
			for i := range e.Split.Branches {
				e.Split.Branches[i].Body = e.Split.Branches[i].Body.normalize()
			}
			e.Split.Otherwise = e.Split.Otherwise.normalize()
		}
	}
	return f
}

// Limits keep a flow and its conditions reviewable and cheap to resolve.
const (
	maxFlowSteps  = 50
	maxFlowLeaves = 50
	maxFlowDepth  = 5
)

// checkFlow validates a flow against its document type's fields, roles and
// module-approver support. path names the offending element, 1-based and
// dot-joined (e.g. "2.branches.1.body.1"); both are "" when f is valid.
func checkFlow(f Flow, fieldKind func(string) (record.FieldKind, bool), roleExists func(product, role string) bool, moduleApprovers bool) (path string, reason string) {
	c := &flowChecker{fieldKind: fieldKind, roleExists: roleExists, moduleApprovers: moduleApprovers}
	return c.check(f, "", 0)
}

type flowChecker struct {
	fieldKind       func(string) (record.FieldKind, bool)
	roleExists      func(product, role string) bool
	moduleApprovers bool
	steps           int
	leaves          int
}

// check validates one flow (the top-level flow, a branch body or an
// otherwise list) at the given nesting depth, counting steps and leaves as it goes.
func (c *flowChecker) check(f Flow, prefix string, depth int) (string, string) {
	for i, e := range f {
		p := strconv.Itoa(i + 1)
		if prefix != "" {
			p = prefix + "." + p
		}
		switch {
		case e.Step == nil && e.Split == nil:
			return p, "empty_element"
		case e.Step != nil && e.Split != nil:
			return p, "two_kinds"
		case e.Step != nil:
			c.steps++
			if c.steps > maxFlowSteps {
				return p, "too_large"
			}
			if reason := c.checkStep(*e.Step); reason != "" {
				return p, reason
			}
		default:
			if depth >= maxFlowDepth {
				return p, "too_large"
			}
			if path, reason := c.checkSplit(*e.Split, p, depth+1); reason != "" {
				return path, reason
			}
		}
	}
	return "", ""
}

func (c *flowChecker) checkSplit(sp Split, splitPath string, depth int) (string, string) {
	if sp.Mode == Parallel && len(sp.Otherwise) > 0 {
		return splitPath, "otherwise_on_parallel"
	}
	for i, b := range sp.Branches {
		branchPath := splitPath + ".branches." + strconv.Itoa(i+1)
		if b.Condition != nil {
			if reason := c.checkExpr(*b.Condition); reason != "" {
				return branchPath, reason
			}
		}
		if path, reason := c.check(b.Body, branchPath+".body", depth); reason != "" {
			return path, reason
		}
	}
	if len(sp.Otherwise) > 0 {
		if path, reason := c.check(sp.Otherwise, splitPath+".otherwise", depth); reason != "" {
			return path, reason
		}
	}
	return "", ""
}

func (c *flowChecker) checkStep(st StepDef) string {
	switch st.Approver.Kind {
	case ByRole:
		if !c.roleExists(st.Approver.Product, st.Approver.Role) {
			return "unknown_role"
		}
	case ByUser:
		if len(st.Approver.UserIDs) == 0 {
			return "missing_user"
		}
		approvals := st.Approvals
		if approvals == 0 {
			approvals = 1
		}
		// startStep deduplicates user_ids before counting voters, so the check must too.
		if int(approvals) > len(slices.Compact(slices.Sorted(slices.Values(st.Approver.UserIDs)))) {
			return "approvals_exceed_users"
		}
	case ByModule:
		if !c.moduleApprovers {
			return "no_module_approvers"
		}
	}
	return ""
}

// checkExpr validates a condition tree; it does not report a path, since
// checkSplit reports every condition problem at its branch's path.
func (c *flowChecker) checkExpr(e Expr) string {
	kinds := 0
	if len(e.All) > 0 {
		kinds++
	}
	if len(e.Any) > 0 {
		kinds++
	}
	if e.Field != "" {
		kinds++
	}
	switch {
	case kinds == 0:
		return "empty_element"
	case kinds > 1:
		return "two_kinds"
	case len(e.All) > 0:
		for _, sub := range e.All {
			if reason := c.checkExpr(sub); reason != "" {
				return reason
			}
		}
		return ""
	case len(e.Any) > 0:
		for _, sub := range e.Any {
			if reason := c.checkExpr(sub); reason != "" {
				return reason
			}
		}
		return ""
	default:
		c.leaves++
		if c.leaves > maxFlowLeaves {
			return "too_large"
		}
		if e.Value == "" {
			return "invalid_condition"
		}
		kind, ok := c.fieldKind(e.Field)
		if !ok {
			return "invalid_condition"
		}
		ops, ok := opsByKind[kind]
		if !ok || !slices.Contains(ops, e.Op) {
			return "invalid_condition"
		}
		switch kind {
		case record.Choice:
		case record.Set:
			if ids, ok := record.ParseSet(e.Value); !ok || len(ids) != 1 {
				return "invalid_condition"
			}
		default:
			if _, ok := record.ParseNumber(e.Value); !ok {
				return "invalid_condition"
			}
		}
		return ""
	}
}

// resolve evaluates f's conditions, numbering the applying steps 1..n in
// reading order. The result is always a sequence (even of one step), or
// Plan{} when nothing applies. holds evaluates a leaf; doubt is what a leaf it
// cannot read evaluates to.
func resolve(f Flow, holds func(e Expr, doubt bool) (bool, error)) (Plan, []StepDef, error) {
	r := &resolver{holds: holds}
	cs, err := r.candidates(f)
	if err != nil {
		return Plan{}, nil, err
	}
	if len(cs) == 0 {
		return Plan{}, r.steps, nil
	}
	return Plan{Seq: cs}, r.steps, nil
}

type resolver struct {
	holds func(e Expr, doubt bool) (bool, error)
	steps []StepDef
}

// candidates resolves each element of f in order. A choose-one split is
// transparent: the chosen branch's own elements splice straight into the
// list, since two steps run one after another whether or not a split chose
// the path to get there. When doubt chose several bodies they run in parallel. A parallel split is opaque: it contributes at most
// one node (its holding branches, collapsed into a group). Anything empty
// drops out.
func (r *resolver) candidates(f Flow) ([]Plan, error) {
	var out []Plan
	for _, e := range f {
		switch {
		case e.Step != nil:
			r.steps = append(r.steps, *e.Step)
			out = append(out, Plan{Step: int32(len(r.steps))})
		case e.Split == nil:
			// neither Step nor Split: an invalid flow (checkFlow rejects it), nothing to resolve.
		case e.Split.Mode == ChooseOne:
			bodies, err := r.chosenBodies(*e.Split)
			if err != nil {
				return nil, err
			}
			if len(bodies) == 1 {
				ps, err := r.candidates(bodies[0])
				if err != nil {
					return nil, err
				}
				out = append(out, ps...)
				continue
			}
			p, err := r.group(bodies)
			if err != nil {
				return nil, err
			}
			if p.Seq != nil {
				out = append(out, p.Seq...)
			} else if !isEmptyPlan(p) {
				out = append(out, p)
			}
		default: // Parallel
			p, err := r.parallel(*e.Split)
			if err != nil {
				return nil, err
			}
			if !isEmptyPlan(p) {
				out = append(out, p)
			}
		}
	}
	return out, nil
}

// chosenBodies lists the bodies that apply: up to and including the first branch
// that surely holds, or else the otherwise too. A branch that holds only through
// doubt runs and the choice goes on, so doubt adds approval, never skips a
// stricter branch.
func (r *resolver) chosenBodies(sp Split) ([]Flow, error) {
	var out []Flow
	for _, b := range sp.Branches {
		maybe, surely, err := r.holdsBranch(b)
		if err != nil {
			return nil, err
		}
		if !maybe {
			continue
		}
		out = append(out, b.Body)
		if surely {
			return out, nil
		}
	}
	return append(out, sp.Otherwise), nil
}

// parallel resolves every branch that may hold into one node for the parent
// sequence: empty bodies drop out, and a single holding branch collapses into it.
func (r *resolver) parallel(sp Split) (Plan, error) {
	var bodies []Flow
	for _, b := range sp.Branches {
		maybe, _, err := r.holdsBranch(b)
		if err != nil {
			return Plan{}, err
		}
		if maybe {
			bodies = append(bodies, b.Body)
		}
	}
	return r.group(bodies)
}

// group resolves bodies that run at the same time into one plan.
func (r *resolver) group(bodies []Flow) (Plan, error) {
	var par []Plan
	for _, body := range bodies {
		ps, err := r.candidates(body)
		if err != nil {
			return Plan{}, err
		}
		if p := collapse(ps); !isEmptyPlan(p) {
			par = append(par, p)
		}
	}
	return collapsePar(par), nil
}

// holdsBranch evaluates b's condition with unreadable leaves taken as true
// (maybe), then as false (surely); they differ only when the outcome rests on doubt.
func (r *resolver) holdsBranch(b Branch) (maybe, surely bool, err error) {
	if b.Condition == nil {
		return true, true, nil
	}
	if maybe, err = b.Condition.eval(func(e Expr) (bool, error) { return r.holds(e, true) }); err != nil || !maybe {
		return maybe, false, err
	}
	surely, err = b.Condition.eval(func(e Expr) (bool, error) { return r.holds(e, false) })
	return maybe, surely, err
}

// collapse turns a list of plans into an empty plan, the one plan it holds, or a sequence.
func collapse(ps []Plan) Plan {
	switch len(ps) {
	case 0:
		return Plan{}
	case 1:
		return ps[0]
	default:
		return Plan{Seq: ps}
	}
}

// collapsePar is collapse for a parallel group: more than one plan wraps as Par, not Seq.
func collapsePar(ps []Plan) Plan {
	if len(ps) > 1 {
		return Plan{Par: ps}
	}
	return collapse(ps)
}

func isEmptyPlan(p Plan) bool {
	return p.Step == 0 && p.Seq == nil && p.Par == nil
}

// next lists the steps to start now: in a Seq, the first child not fully
// done; in a Par, every child not fully done; a leaf not done and not
// started is returned. finished is true once every leaf is done.
func (p Plan) next(done map[int32]bool, started map[int32]bool) (start []int32, finished bool) {
	switch {
	case p.Step != 0:
		if done[p.Step] {
			return nil, true
		}
		if started[p.Step] {
			return nil, false
		}
		return []int32{p.Step}, false
	case len(p.Seq) > 0:
		for _, c := range p.Seq {
			s, fin := c.next(done, started)
			if !fin {
				return s, false
			}
		}
		return nil, true
	case len(p.Par) > 0:
		finished = true
		for _, c := range p.Par {
			s, fin := c.next(done, started)
			if !fin {
				finished = false
			}
			start = append(start, s...)
		}
		return start, finished
	}
	return nil, true
}
