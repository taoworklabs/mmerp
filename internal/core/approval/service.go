// Package approval owns approval rules and instances. It is record's approval gate.
package approval

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/taoworklabs/mmerp/internal/core/approval/internal/store"
	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/notification"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
)

type Service struct{ d Deps }

func NewService(d Deps) *Service { return &Service{d: d} }

// Submit implements record.ApprovalGate: it resolves the rule's flow for d into
// a plan, opens an instance with one step per resolved step, and starts the first ones.
func (s *Service) Submit(ctx context.Context, d record.Doc) (int64, bool, error) {
	q := store.New(platform.DBFrom(ctx))
	rule, err := q.GetRule(ctx, d.Type)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	var flow Flow
	if err := json.Unmarshal(rule.Flow, &flow); err != nil {
		return 0, false, err
	}
	plan, steps, err := resolve(flow.normalize(), func(e Expr, doubt bool) (bool, error) { return s.holds(ctx, d, e, doubt) })
	if err != nil || len(steps) == 0 {
		return 0, false, err
	}
	rawPlan, err := json.Marshal(plan)
	if err != nil {
		return 0, false, err
	}
	actor, _ := platform.ActorFrom(ctx)
	id, err := q.InsertInstance(ctx, store.InsertInstanceParams{
		DocType: d.Type, DocID: d.ID, Version: d.Version, SubmittedBy: actor, Plan: rawPlan,
		MaxLevels: rule.MaxLevels, FallbackProduct: rule.FallbackProduct, FallbackRole: rule.FallbackRole,
	})
	if err != nil {
		return 0, false, err
	}
	for i, st := range steps {
		raw, err := json.Marshal(st.Approver)
		if err != nil {
			return 0, false, err
		}
		if err := q.InsertStep(ctx, store.InsertStepParams{InstanceID: id, Position: int32(i + 1), Approver: raw,
			Approvals: st.Approvals, Rejections: st.Rejections}); err != nil {
			return 0, false, err
		}
	}
	inst, err := q.GetInstance(ctx, id)
	if err != nil {
		return 0, false, err
	}
	start, _ := plan.next(nil, nil)
	return id, true, s.startSteps(ctx, inst, d, start)
}

func (s *Service) startSteps(ctx context.Context, inst store.ApprovalInstance, d record.Doc, positions []int32) error {
	for _, pos := range positions {
		if err := s.startStep(ctx, inst, d, pos); err != nil {
			return err
		}
	}
	return nil
}

// Withdrawn implements record.ApprovalGate.
func (s *Service) Withdrawn(ctx context.Context, d record.Doc) error {
	q := store.New(platform.DBFrom(ctx))
	if err := q.CancelOpenSteps(ctx, *d.ApprovalTicket); err != nil {
		return err
	}
	return q.CloseInstance(ctx, store.CloseInstanceParams{ID: *d.ApprovalTicket, Status: "withdrawn"})
}

// startStep fixes who approves a step. The submitter never approves; when nobody
// is left the step goes to the fallback role, and with nobody there either the
// submission fails: approval is never automatic. Fewer approvers than the step's
// approval count fail it too, since it could never pass.
func (s *Service) startStep(ctx context.Context, inst store.ApprovalInstance, d record.Doc, pos int32) error {
	q := store.New(platform.DBFrom(ctx))
	st, err := q.GetStep(ctx, store.GetStepParams{InstanceID: inst.ID, Position: pos})
	if err != nil {
		return err
	}
	var a Approver
	if err := json.Unmarshal(st.Approver, &a); err != nil {
		return err
	}
	excluded, err := s.excluded(ctx, inst, d.Ref)
	if err != nil {
		return err
	}
	// Approvers must be able to view the document; anyone else could never act on the step.
	// ponytail: one Can per candidate; batch by role scope if roles grow to hundreds of holders.
	var failed error
	others := func(ids []int64) []int64 {
		return slices.DeleteFunc(ids, func(id int64) bool {
			if slices.Contains(excluded, id) {
				return true
			}
			ok, err := s.d.Record.Can(s.d.IAM.AsUser(ctx, id), d.Type, d.ID, record.View)
			if err != nil {
				failed = err
			}
			return !ok
		})
	}
	var users []int64
	switch a.Kind {
	case ByRole:
		if users, err = s.d.IAM.UsersWithRole(ctx, a.Product, a.Role, d.OrgUnitID); err != nil {
			return err
		}
	case ByUser:
		// A repeated id must not count as two voters.
		users = slices.Compact(slices.Sorted(slices.Values(a.UserIDs)))
	case ByModule:
		t, _ := s.d.Record.TypeOf(d.Type)
		for level := 1; level <= int(inst.MaxLevels) && len(users) == 0; level++ {
			ids, err := t.Approvers(ctx, d.Ref, level)
			if err != nil {
				return err
			}
			users = others(ids)
		}
	}
	users, fallback := others(users), false
	if len(users) == 0 {
		ids, err := s.d.IAM.UsersWithRole(ctx, inst.FallbackProduct, inst.FallbackRole, d.OrgUnitID)
		if err != nil {
			return err
		}
		users, fallback = others(ids), true
	}
	if failed != nil {
		return failed
	}
	if len(users) < int(st.Approvals) {
		return errNoApprover(pos)
	}
	if err := q.StartStep(ctx, store.StartStepParams{InstanceID: inst.ID, Position: pos, Approvers: users, Fallback: fallback}); err != nil {
		return err
	}
	return s.d.Notification.Send(ctx, notification.ApprovalRequested, d.Ref, users)
}

// holds evaluates one condition leaf for d; a leaf that cannot be evaluated
// (missing or malformed value) is doubt, and resolve decides what doubt means.
func (s *Service) holds(ctx context.Context, d record.Doc, c Expr, doubt bool) (bool, error) {
	v, ok := valueOf(d, c.Field)
	if !ok {
		return doubt, nil
	}
	switch c.Op {
	case OpEq:
		return v == c.Value, nil
	case OpWithin:
		unit, err1 := strconv.ParseInt(v, 10, 64)
		ancestor, err2 := strconv.ParseInt(c.Value, 10, 64)
		if err1 != nil || err2 != nil {
			return doubt, nil
		}
		return store.New(platform.DBFrom(ctx)).Within(ctx, store.WithinParams{Unit: unit, Ancestor: ancestor})
	}
	x, ok1 := record.ParseNumber(v)
	y, ok2 := record.ParseNumber(c.Value)
	if !ok1 || !ok2 {
		return doubt, nil
	}
	if c.Op == OpGTE {
		return x.GreaterThanOrEqual(y), nil
	}
	return x.GreaterThan(y), nil
}

// valueOf reads a header or approval field of d as a string; false when it has none.
func valueOf(d record.Doc, field string) (string, bool) {
	switch field {
	case FieldAmount:
		if d.Amount == nil {
			return "", false
		}
		return strconv.FormatInt(*d.Amount, 10), true
	case FieldOrgUnit:
		return strconv.FormatInt(d.OrgUnitID, 10), true
	}
	v, ok := d.Fields[field]
	return v, ok
}

// kindOf is the kind of a header or approval field of t; false for an unknown field.
func kindOf(t record.Type, field string) (record.FieldKind, bool) {
	switch field {
	case FieldAmount:
		return record.Number, true
	case FieldOrgUnit:
		return record.OrgUnit, true
	}
	i := slices.IndexFunc(t.Fields, func(f record.Field) bool { return f.Key == field })
	if i < 0 {
		return "", false
	}
	return t.Fields[i].Kind, true
}

// act locks in the shared order (the document's locks, then the instance) and
// checks the instance and step are open. A stale instance is closed and the close
// committed before ErrApprovalStale is returned.
func (s *Service) act(ctx context.Context, id int64, step int32, fn func(ctx context.Context, inst store.ApprovalInstance, st store.ApprovalInstanceStep, d record.Doc) error) error {
	stale := false
	err := platform.InTx(ctx, func(ctx context.Context) error {
		q := store.New(platform.DBFrom(ctx))
		inst, err := q.GetInstance(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return platform.ErrNotFound
		}
		if err != nil {
			return err
		}
		ref := record.Ref{Type: inst.DocType, ID: inst.DocID}
		d, err := s.d.Record.Lock(ctx, ref)
		if errors.Is(err, platform.ErrNotFound) {
			return ErrClosed
		}
		if err != nil {
			return err
		}
		if inst, err = q.LockInstance(ctx, id); err != nil {
			return err
		}
		if ok, err := s.d.Record.Can(ctx, ref.Type, ref.ID, record.View); err != nil || !ok {
			return platform.OrErr(err, platform.ErrNotFound)
		}
		if inst.Status != "open" {
			return ErrClosed
		}
		st, err := q.GetStep(ctx, store.GetStepParams{InstanceID: id, Position: step})
		if errors.Is(err, pgx.ErrNoRows) || err == nil && st.Status != "open" {
			return ErrClosed
		}
		if err != nil {
			return err
		}
		if t, _ := s.d.Record.TypeOf(ref.Type); platform.ProductGate(ctx, t.Product, platform.ClassWrite) != nil {
			return platform.ProductGate(ctx, t.Product, platform.ClassWrite)
		}
		if d.Status != record.PendingApproval || d.ApprovalTicket == nil || *d.ApprovalTicket != id || d.Version != inst.Version {
			stale = true
			if err := q.CancelOpenSteps(ctx, id); err != nil {
				return err
			}
			return q.CloseInstance(ctx, store.CloseInstanceParams{ID: id, Status: "stale"})
		}
		return fn(ctx, inst, st, d)
	})
	if err == nil && stale {
		return record.ErrApprovalStale
	}
	return err
}

// Approve votes for step as one of its voters. Once the step has its approvals the
// next steps of the plan start; the last one posts the document in the same
// transaction. If posting fails the vote is not kept and the instance stays open.
func (s *Service) Approve(ctx context.Context, id int64, step int32) error {
	return s.act(ctx, id, step, func(ctx context.Context, inst store.ApprovalInstance, st store.ApprovalInstanceStep, d record.Doc) error {
		status, err := s.vote(ctx, inst, st, "approved", nil)
		if err != nil || status != "approved" {
			return err
		}
		q := store.New(platform.DBFrom(ctx))
		rows, err := q.StepStatuses(ctx, id)
		if err != nil {
			return err
		}
		done, started := map[int32]bool{}, map[int32]bool{}
		for _, r := range rows {
			done[r.Position], started[r.Position] = r.Status == "approved", r.Status != "waiting"
		}
		var plan Plan
		if err := json.Unmarshal(inst.Plan, &plan); err != nil {
			return err
		}
		if start, finished := plan.next(done, started); !finished {
			return s.startSteps(ctx, inst, d, start)
		}
		if err := q.CloseInstance(ctx, store.CloseInstanceParams{ID: id, Status: "approved"}); err != nil {
			return err
		}
		if err := s.d.Record.CompleteApproval(ctx, d.Ref, id, inst.Version); err != nil {
			return err
		}
		return s.d.Notification.Send(ctx, notification.ApprovalApproved, d.Ref, []int64{inst.SubmittedBy})
	})
}

// Reject votes against step. Once the step rejects, the other steps are cancelled,
// the instance closes and the document goes back to draft.
func (s *Service) Reject(ctx context.Context, id int64, step int32, reason string) error {
	return s.act(ctx, id, step, func(ctx context.Context, inst store.ApprovalInstance, st store.ApprovalInstanceStep, d record.Doc) error {
		status, err := s.vote(ctx, inst, st, "rejected", &reason)
		if err != nil || status != "rejected" {
			return err
		}
		q := store.New(platform.DBFrom(ctx))
		if err := q.CancelOpenSteps(ctx, id); err != nil {
			return err
		}
		if err := q.CloseInstance(ctx, store.CloseInstanceParams{ID: id, Status: "rejected"}); err != nil {
			return err
		}
		if err := s.d.Record.Reject(ctx, d.Ref, id, inst.Version); err != nil {
			return err
		}
		return s.d.Notification.Send(ctx, notification.ApprovalRejected, d.Ref, []int64{inst.SubmittedBy})
	})
}

// vote records the actor's vote on an open step and returns the step's new status:
// approved, rejected, or "" while it is still open. A step rejects on its rejection
// count, or as soon as the remaining voters can no longer bring its approvals.
func (s *Service) vote(ctx context.Context, inst store.ApprovalInstance, st store.ApprovalInstanceStep, decision string, reason *string) (string, error) {
	q := store.New(platform.DBFrom(ctx))
	actor, _ := platform.ActorFrom(ctx)
	if !slices.Contains(st.Approvers, actor) {
		return "", platform.ErrForbidden
	}
	// Approvers were fixed when the step started; who the document is about may have changed since.
	if excluded, err := s.excluded(ctx, inst, record.Ref{Type: inst.DocType, ID: inst.DocID}); err != nil {
		return "", err
	} else if slices.Contains(excluded, actor) {
		return "", ErrSelfApproval
	}
	err := q.InsertVote(ctx, store.InsertVoteParams{InstanceID: inst.ID, Position: st.Position, UserID: actor,
		Decision: decision, Reason: platform.NullText(reason)})
	if platform.Violates(err, "step_votes_pkey") {
		return "", ErrAlreadyVoted
	}
	if err != nil {
		return "", err
	}
	if err := s.d.Audit.RecordFor(ctx, "approval."+decision, audit.Ref{Type: inst.DocType, ID: inst.DocID},
		map[string]any{"instance": inst.ID, "step": st.Position, "reason": reason}); err != nil {
		return "", err
	}
	c, err := q.StepVotes(ctx, store.StepVotesParams{InstanceID: inst.ID, Position: st.Position})
	if err != nil {
		return "", err
	}
	left := int32(len(st.Approvers)) - c.Approved - c.Rejected
	var status string
	switch {
	case c.Approved >= st.Approvals:
		status = "approved"
	case c.Rejected >= st.Rejections || c.Approved+left < st.Approvals:
		status = "rejected"
	default:
		return "", nil
	}
	return status, q.SetStepStatus(ctx, store.SetStepStatusParams{InstanceID: inst.ID, Position: st.Position, Status: status})
}

// Reassign replaces voter from, who has not voted, with the user login; the step's
// counts stay. Only holders of the rule's fallback role may do it, e.g. when a voter has left.
func (s *Service) Reassign(ctx context.Context, id int64, step int32, from int64, login string) error {
	return s.act(ctx, id, step, func(ctx context.Context, inst store.ApprovalInstance, st store.ApprovalInstanceStep, d record.Doc) error {
		if ok, err := s.isFallback(ctx, inst, d); err != nil || !ok {
			return platform.OrErr(err, platform.ErrForbidden)
		}
		user, err := s.d.IAM.UserIDByLogin(ctx, login)
		if err != nil {
			return err
		}
		if excluded, err := s.excluded(ctx, inst, d.Ref); err != nil {
			return err
		} else if slices.Contains(excluded, user) {
			return ErrSelfApproval
		}
		if ok, err := s.d.Record.Can(s.d.IAM.AsUser(ctx, user), d.Type, d.ID, record.View); err != nil || !ok {
			return platform.OrErr(err, ErrApproverCannotView)
		}
		q := store.New(platform.DBFrom(ctx))
		i := slices.Index(st.Approvers, from)
		if i < 0 {
			return platform.ErrNotFound
		}
		if slices.Contains(st.Approvers, user) {
			return ErrAlreadyApprover
		}
		if voted, err := q.HasVoted(ctx, store.HasVotedParams{InstanceID: id, Position: step, UserID: from}); err != nil || voted {
			return platform.OrErr(err, ErrAlreadyVoted)
		}
		st.Approvers[i] = user
		if err := q.SetApprovers(ctx, store.SetApproversParams{InstanceID: id, Position: step, Approvers: st.Approvers}); err != nil {
			return err
		}
		if err := s.d.Notification.Send(ctx, notification.ApprovalRequested, d.Ref, []int64{user}); err != nil {
			return err
		}
		return s.d.Audit.RecordFor(ctx, "approval.reassigned", audit.Ref{Type: inst.DocType, ID: inst.DocID},
			map[string]any{"instance": id, "step": step, "from": from, "to": user})
	})
}

// excluded lists who may never approve the document: its submitter and the users it is about.
func (s *Service) excluded(ctx context.Context, inst store.ApprovalInstance, ref record.Ref) ([]int64, error) {
	out := []int64{inst.SubmittedBy}
	if t, _ := s.d.Record.TypeOf(ref.Type); t.Subjects != nil {
		ids, err := t.Subjects(ctx, ref.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, ids...)
	}
	return out, nil
}

func (s *Service) isFallback(ctx context.Context, inst store.ApprovalInstance, d record.Doc) (bool, error) {
	ids, err := s.d.IAM.UsersWithRole(ctx, inst.FallbackProduct, inst.FallbackRole, d.OrgUnitID)
	actor, _ := platform.ActorFrom(ctx)
	return slices.Contains(ids, actor), err
}

// Inbox lists the instances waiting for the actor, among documents they may view.
func (s *Service) Inbox(ctx context.Context) ([]InboxItem, error) {
	actor, _ := platform.ActorFrom(ctx)
	rows, err := store.New(platform.DBFrom(ctx)).Inbox(ctx, actor)
	if err != nil {
		return nil, err
	}
	out := []InboxItem{}
	for _, r := range rows {
		if ok, err := s.d.Record.Can(ctx, r.DocType, r.DocID, record.View); err != nil {
			return nil, err
		} else if !ok {
			continue
		}
		out = append(out, InboxItem{
			InstanceID: r.ID, DocType: r.DocType, DocID: r.DocID, Number: r.Number, Date: r.Date.Time.Format(time.DateOnly),
			Step: r.Step, SubmittedByName: r.SubmittedByName, SubmittedAt: r.SubmittedAt.Time.Format(time.RFC3339),
		})
	}
	return out, nil
}

// DocumentInstance returns the latest submission of a document the actor may view, or nil.
func (s *Service) DocumentInstance(ctx context.Context, ref record.Ref) (*Instance, error) {
	if ok, err := s.d.Record.Can(ctx, ref.Type, ref.ID, record.View); err != nil || !ok {
		return nil, platform.OrErr(err, platform.ErrNotFound)
	}
	q := store.New(platform.DBFrom(ctx))
	inst, err := q.LatestInstance(ctx, store.LatestInstanceParams{DocType: ref.Type, DocID: ref.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := q.Steps(ctx, inst.ID)
	if err != nil {
		return nil, err
	}
	out := &Instance{ID: inst.ID, Status: inst.Status, SubmittedByName: inst.SubmittedByName,
		SubmittedAt: inst.SubmittedAt.Time.Format(time.RFC3339), Steps: []StepView{}}
	if err := json.Unmarshal(inst.Plan, &out.Plan); err != nil {
		return nil, err
	}
	actor, _ := platform.ActorFrom(ctx)
	t, _ := s.d.Record.TypeOf(ref.Type)
	writable := inst.Status == "open" && platform.ProductGate(ctx, t.Product, platform.ClassWrite) == nil
	fallback := false
	if writable {
		d, err := s.d.Record.Get(ctx, ref)
		if err != nil {
			return nil, err
		}
		if fallback, err = s.isFallback(ctx, store.ApprovalInstance{FallbackProduct: inst.FallbackProduct, FallbackRole: inst.FallbackRole}, d); err != nil {
			return nil, err
		}
	}
	for _, r := range rows {
		var voters []struct {
			ID        int64      `json:"id"`
			Name      string     `json:"name"`
			Decision  *string    `json:"decision"`
			Reason    *string    `json:"reason"`
			DecidedAt *time.Time `json:"decided_at"`
		}
		if err := json.Unmarshal(r.Voters, &voters); err != nil {
			return nil, err
		}
		v := StepView{Position: r.Position, Status: r.Status, Approvals: r.Approvals, Rejections: r.Rejections, Fallback: r.Fallback,
			Voters: []StepVoter{}, AllowedActions: []string{}}
		canVote := false
		for _, x := range voters {
			sv := StepVoter{ID: x.ID, Name: x.Name, Decision: x.Decision, Reason: x.Reason}
			if x.DecidedAt != nil {
				at := x.DecidedAt.Format(time.RFC3339)
				sv.DecidedAt = &at
			}
			canVote = canVote || x.ID == actor && x.Decision == nil
			v.Voters = append(v.Voters, sv)
		}
		if writable && r.Status == "open" {
			if canVote {
				v.AllowedActions = append(v.AllowedActions, "approve", "reject")
			}
			if fallback {
				v.AllowedActions = append(v.AllowedActions, "reassign")
			}
		}
		out.Steps = append(out.Steps, v)
	}
	return out, nil
}

// managesProduct reports whether the actor may configure the rules of a product's document types:
// with core.approval.manage, or the product's own approval.manage, either one tenant-wide.
func (s *Service) managesProduct(ctx context.Context, product string) (bool, error) {
	for _, p := range [][2]string{{"core", iam.PermManageApproval}, {product, product + ".approval.manage"}} {
		sc, err := s.d.IAM.Scope(ctx, p[0], p[1])
		if err != nil || sc.All {
			return sc.All, err
		}
	}
	return false, nil
}

// manageableTypes lists the document types whose rules the actor may configure; none is forbidden.
func (s *Service) manageableTypes(ctx context.Context) ([]record.Type, error) {
	var out []record.Type
	for _, t := range s.d.Record.DocumentTypes() {
		ok, err := s.managesProduct(ctx, t.Product)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return nil, platform.ErrForbidden
	}
	return out, nil
}

// requireType resolves a document type the actor may configure rules of, for a write.
func (s *Service) requireType(ctx context.Context, docType string) (record.Type, error) {
	t, ok := s.d.Record.TypeOf(docType)
	if !ok || t.Kind != record.Document {
		return t, platform.ErrNotFound
	}
	if ok, err := s.managesProduct(ctx, t.Product); err != nil {
		return t, err
	} else if !ok {
		return t, platform.ErrForbidden
	}
	return t, platform.ProductGate(ctx, t.Product, platform.ClassWrite)
}

// Rules lists the document types the actor may configure, of one product when product is set,
// with their approval fields and rule.
func (s *Service) Rules(ctx context.Context, product string) ([]TypeRule, error) {
	types, err := s.manageableTypes(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := store.New(platform.DBFrom(ctx)).ListRules(ctx)
	if err != nil {
		return nil, err
	}
	out := []TypeRule{}
	for _, t := range types {
		if product != "" && t.Product != product {
			continue
		}
		tr := TypeRule{DocType: t.Code, Product: t.Product, Fields: []Field{}, ModuleApprovers: t.Approvers != nil, AllowedActions: []string{}}
		if platform.ProductGate(ctx, t.Product, platform.ClassWrite) == nil {
			tr.AllowedActions = []string{"save", "delete"}
		}
		for _, f := range t.Fields {
			field := Field{Key: f.Key, Kind: string(f.Kind), Label: f.Label, Ops: opsByKind[f.Kind]}
			if f.Options != nil {
				if field.Options, err = f.Options(ctx); err != nil {
					return nil, err
				}
			}
			tr.Fields = append(tr.Fields, field)
		}
		if i := slices.IndexFunc(rows, func(r store.ApprovalRule) bool { return r.DocType == t.Code }); i >= 0 {
			r := rows[i]
			tr.Rule = &RuleInput{MaxLevels: r.MaxLevels, FallbackProduct: r.FallbackProduct, FallbackRole: r.FallbackRole}
			if err := json.Unmarshal(r.Flow, &tr.Rule.Flow); err != nil {
				return nil, err
			}
		}
		out = append(out, tr)
	}
	return out, nil
}

// Users lists the accounts a rule may name as approver, for whoever configures rules.
func (s *Service) Users(ctx context.Context) ([]ApproverUser, error) {
	if _, err := s.manageableTypes(ctx); err != nil {
		return nil, err
	}
	rows, err := store.New(platform.DBFrom(ctx)).ListUsers(ctx)
	out := make([]ApproverUser, len(rows))
	for i, r := range rows {
		out[i] = ApproverUser(r)
	}
	return out, err
}

// SaveRule replaces the rule of a document type; open instances keep the plan they started with.
func (s *Service) SaveRule(ctx context.Context, docType string, in RuleInput) error {
	t, err := s.requireType(ctx, docType)
	if err != nil {
		return err
	}
	if !s.roleExists(in.FallbackProduct, in.FallbackRole) {
		return errInvalidRule("", "unknown_role")
	}
	in.Flow = in.Flow.normalize()
	kind := func(field string) (record.FieldKind, bool) { return kindOf(t, field) }
	if path, reason := checkFlow(in.Flow, kind, s.roleExists, t.Approvers != nil); reason != "" {
		return errInvalidRule(path, reason)
	}
	flow, err := json.Marshal(in.Flow)
	if err != nil {
		return err
	}
	return platform.InTx(ctx, func(ctx context.Context) error {
		if err := store.New(platform.DBFrom(ctx)).SaveRule(ctx, store.SaveRuleParams{
			DocType: docType, Flow: flow, MaxLevels: in.MaxLevels, FallbackProduct: in.FallbackProduct, FallbackRole: in.FallbackRole,
		}); err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "approval.rule_saved", map[string]any{"doc_type": docType, "rule": in})
	})
}

// DeleteRule removes a rule, so sending posts at once.
func (s *Service) DeleteRule(ctx context.Context, docType string) error {
	if _, err := s.requireType(ctx, docType); err != nil {
		return err
	}
	return platform.InTx(ctx, func(ctx context.Context) error {
		if err := store.New(platform.DBFrom(ctx)).DeleteRule(ctx, docType); err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "approval.rule_deleted", map[string]any{"doc_type": docType})
	})
}

func (s *Service) roleExists(product, role string) bool {
	return slices.ContainsFunc(s.d.IAM.Roles(), func(r iam.Role) bool { return r.Product == product && r.Role == role })
}
