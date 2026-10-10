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
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/core/approval/internal/store"
	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/notification"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
)

type Service struct{ d Deps }

func NewService(d Deps) *Service { return &Service{d: d} }

// Submit implements record.ApprovalGate: it opens an instance with the rule's
// steps whose condition holds for d, and starts the first one.
func (s *Service) Submit(ctx context.Context, d record.Doc) (int64, bool, error) {
	q := store.New(platform.DBFrom(ctx))
	rule, err := q.GetRule(ctx, d.Type)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	var steps []Step
	if err := json.Unmarshal(rule.Steps, &steps); err != nil {
		return 0, false, err
	}
	var apply []Step
	for _, st := range steps {
		ok, err := s.holds(ctx, d, st.Condition)
		if err != nil {
			return 0, false, err
		}
		if ok {
			apply = append(apply, st)
		}
	}
	if len(apply) == 0 {
		return 0, false, nil
	}
	actor, _ := platform.ActorFrom(ctx)
	id, err := q.InsertInstance(ctx, store.InsertInstanceParams{
		DocType: d.Type, DocID: d.ID, Version: d.Version, SubmittedBy: actor,
		MaxLevels: rule.MaxLevels, FallbackProduct: rule.FallbackProduct, FallbackRole: rule.FallbackRole,
	})
	if err != nil {
		return 0, false, err
	}
	for i, st := range apply {
		raw, err := json.Marshal(st.Approver)
		if err != nil {
			return 0, false, err
		}
		if err := q.InsertStep(ctx, store.InsertStepParams{InstanceID: id, Position: int32(i + 1), Approver: raw}); err != nil {
			return 0, false, err
		}
	}
	inst, err := q.GetInstance(ctx, id)
	if err != nil {
		return 0, false, err
	}
	return id, true, s.startStep(ctx, inst, d, 1)
}

// Withdrawn implements record.ApprovalGate.
func (s *Service) Withdrawn(ctx context.Context, d record.Doc) error {
	return store.New(platform.DBFrom(ctx)).CloseInstance(ctx, store.CloseInstanceParams{ID: *d.ApprovalTicket, Status: "withdrawn"})
}

// startStep fixes who approves a step. The submitter never approves; when nobody
// is left the step goes to the fallback role, and with nobody there either the
// submission fails: approval is never automatic.
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
		users = []int64{*a.UserID}
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
	if len(users) == 0 {
		return ErrNoApprover
	}
	if err := q.StartStep(ctx, store.StartStepParams{InstanceID: inst.ID, Position: pos, Approvers: users, Fallback: fallback}); err != nil {
		return err
	}
	return s.d.Notification.Send(ctx, notification.ApprovalRequested, d.Ref, users)
}

// holds reports whether a step applies to d. A condition that cannot be evaluated
// (missing or malformed value) applies the step: a doubt adds approval, never removes it.
func (s *Service) holds(ctx context.Context, d record.Doc, c *Condition) (bool, error) {
	if c == nil {
		return true, nil
	}
	v, ok := valueOf(d, c.Field)
	if !ok {
		return true, nil
	}
	switch c.Op {
	case OpEq:
		return v == c.Value, nil
	case OpWithin:
		unit, err1 := strconv.ParseInt(v, 10, 64)
		ancestor, err2 := strconv.ParseInt(c.Value, 10, 64)
		if err1 != nil || err2 != nil {
			return true, nil
		}
		return store.New(platform.DBFrom(ctx)).Within(ctx, store.WithinParams{Unit: unit, Ancestor: ancestor})
	}
	x, ok1 := record.ParseNumber(v)
	y, ok2 := record.ParseNumber(c.Value)
	if !ok1 || !ok2 {
		return true, nil
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
// checks the instance is open at step. A stale instance is closed and the close
// committed before ErrApprovalStale is returned.
func (s *Service) act(ctx context.Context, id int64, step int32, fn func(ctx context.Context, inst store.ApprovalInstance, d record.Doc) error) error {
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
		if inst.Status != "open" || inst.CurrentStep != step {
			return ErrClosed
		}
		if t, _ := s.d.Record.TypeOf(ref.Type); platform.ProductGate(ctx, t.Product, platform.ClassWrite) != nil {
			return platform.ProductGate(ctx, t.Product, platform.ClassWrite)
		}
		if d.Status != record.PendingApproval || d.ApprovalTicket == nil || *d.ApprovalTicket != id || d.Version != inst.Version {
			stale = true
			return q.CloseInstance(ctx, store.CloseInstanceParams{ID: id, Status: "stale"})
		}
		return fn(ctx, inst, d)
	})
	if err == nil && stale {
		return record.ErrApprovalStale
	}
	return err
}

// Approve approves step of an instance as one of its approvers. Approving the
// last step posts the document in the same transaction; if posting fails the
// decision is not kept and the instance stays open.
func (s *Service) Approve(ctx context.Context, id int64, step int32) error {
	return s.act(ctx, id, step, func(ctx context.Context, inst store.ApprovalInstance, d record.Doc) error {
		if err := s.decide(ctx, inst, step, "approved", nil); err != nil {
			return err
		}
		q := store.New(platform.DBFrom(ctx))
		n, err := q.StepCount(ctx, id)
		if err != nil {
			return err
		}
		if int64(step) < n {
			if err := q.SetCurrentStep(ctx, store.SetCurrentStepParams{ID: id, CurrentStep: step + 1}); err != nil {
				return err
			}
			return s.startStep(ctx, inst, d, step+1)
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

// Reject sends the document back to draft and closes the instance.
func (s *Service) Reject(ctx context.Context, id int64, step int32, reason string) error {
	return s.act(ctx, id, step, func(ctx context.Context, inst store.ApprovalInstance, d record.Doc) error {
		if err := s.decide(ctx, inst, step, "rejected", &reason); err != nil {
			return err
		}
		if err := store.New(platform.DBFrom(ctx)).CloseInstance(ctx, store.CloseInstanceParams{ID: id, Status: "rejected"}); err != nil {
			return err
		}
		if err := s.d.Record.Reject(ctx, d.Ref, id, inst.Version); err != nil {
			return err
		}
		return s.d.Notification.Send(ctx, notification.ApprovalRejected, d.Ref, []int64{inst.SubmittedBy})
	})
}

func (s *Service) decide(ctx context.Context, inst store.ApprovalInstance, step int32, decision string, reason *string) error {
	q := store.New(platform.DBFrom(ctx))
	st, err := q.GetStep(ctx, store.GetStepParams{InstanceID: inst.ID, Position: step})
	if err != nil {
		return err
	}
	actor, _ := platform.ActorFrom(ctx)
	if !slices.Contains(st.Approvers, actor) {
		return platform.ErrForbidden
	}
	// Approvers were fixed when the step started; who the document is about may have changed since.
	if excluded, err := s.excluded(ctx, inst, record.Ref{Type: inst.DocType, ID: inst.DocID}); err != nil {
		return err
	} else if slices.Contains(excluded, actor) {
		return ErrSelfApproval
	}
	if err := q.Decide(ctx, store.DecideParams{
		InstanceID: inst.ID, Position: step, DecidedBy: pgtype.Int8{Int64: actor, Valid: true},
		Decision: pgtype.Text{String: decision, Valid: true}, Reason: platform.NullText(reason),
	}); err != nil {
		return err
	}
	return s.d.Audit.RecordFor(ctx, "approval."+decision, audit.Ref{Type: inst.DocType, ID: inst.DocID},
		map[string]any{"instance": inst.ID, "step": step, "reason": reason})
}

// Reassign replaces the approvers of the current step with one user. Only holders
// of the rule's fallback role may do it, e.g. when the approver has left.
func (s *Service) Reassign(ctx context.Context, id int64, step int32, login string) error {
	return s.act(ctx, id, step, func(ctx context.Context, inst store.ApprovalInstance, d record.Doc) error {
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
		old, err := q.GetStep(ctx, store.GetStepParams{InstanceID: id, Position: step})
		if err != nil {
			return err
		}
		if err := q.StartStep(ctx, store.StartStepParams{InstanceID: id, Position: step, Approvers: []int64{user}}); err != nil {
			return err
		}
		if err := s.d.Notification.Send(ctx, notification.ApprovalRequested, d.Ref, []int64{user}); err != nil {
			return err
		}
		return s.d.Audit.RecordFor(ctx, "approval.reassigned", audit.Ref{Type: inst.DocType, ID: inst.DocID},
			map[string]any{"instance": id, "step": step, "from": old.Approvers, "to": user})
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
			Step: r.CurrentStep, SubmittedByName: r.SubmittedByName, SubmittedAt: r.SubmittedAt.Time.Format(time.RFC3339),
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
	out := &Instance{ID: inst.ID, Status: inst.Status, CurrentStep: inst.CurrentStep, SubmittedByName: inst.SubmittedByName,
		SubmittedAt: inst.SubmittedAt.Time.Format(time.RFC3339), Steps: []StepView{}, AllowedActions: []string{}}
	actor, _ := platform.ActorFrom(ctx)
	for _, r := range rows {
		v := StepView{Position: r.Position, ApproverNames: r.ApproverNames, Fallback: r.Fallback,
			Decision: platform.TextPtr(r.Decision), DecidedByName: platform.TextPtr(r.DecidedByName), Reason: platform.TextPtr(r.Reason)}
		if r.DecidedAt.Valid {
			at := r.DecidedAt.Time.Format(time.RFC3339)
			v.DecidedAt = &at
		}
		out.Steps = append(out.Steps, v)
		if inst.Status == "open" && r.Position == inst.CurrentStep && slices.Contains(r.Approvers, actor) {
			out.AllowedActions = append(out.AllowedActions, "approve", "reject")
		}
	}
	if inst.Status == "open" {
		d, err := s.d.Record.Get(ctx, ref)
		if err != nil {
			return nil, err
		}
		if ok, err := s.isFallback(ctx, store.ApprovalInstance{FallbackProduct: inst.FallbackProduct, FallbackRole: inst.FallbackRole}, d); err != nil {
			return nil, err
		} else if ok {
			out.AllowedActions = append(out.AllowedActions, "reassign")
		}
	}
	if t, _ := s.d.Record.TypeOf(ref.Type); platform.ProductGate(ctx, t.Product, platform.ClassWrite) != nil {
		out.AllowedActions = []string{}
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
			if err := json.Unmarshal(r.Steps, &tr.Rule.Steps); err != nil {
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

// SaveRule replaces the rule of a document type; open instances keep the steps they started with.
func (s *Service) SaveRule(ctx context.Context, docType string, in RuleInput) error {
	t, err := s.requireType(ctx, docType)
	if err != nil {
		return err
	}
	if !s.roleExists(in.FallbackProduct, in.FallbackRole) {
		return errInvalidRule(0, "unknown_role")
	}
	for i, st := range in.Steps {
		if reason := s.checkStep(t, st); reason != "" {
			return errInvalidRule(i+1, reason)
		}
	}
	steps, err := json.Marshal(in.Steps)
	if err != nil {
		return err
	}
	return platform.InTx(ctx, func(ctx context.Context) error {
		if err := store.New(platform.DBFrom(ctx)).SaveRule(ctx, store.SaveRuleParams{
			DocType: docType, Steps: steps, MaxLevels: in.MaxLevels, FallbackProduct: in.FallbackProduct, FallbackRole: in.FallbackRole,
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

// checkStep returns why a step does not fit the type, or "".
func (s *Service) checkStep(t record.Type, st Step) string {
	switch a := st.Approver; a.Kind {
	case ByRole:
		if !s.roleExists(a.Product, a.Role) {
			return "unknown_role"
		}
	case ByUser:
		if a.UserID == nil {
			return "missing_user"
		}
	case ByModule:
		if t.Approvers == nil {
			return "no_module_approvers"
		}
	}
	c := st.Condition
	if c == nil {
		return ""
	}
	kind, ok := kindOf(t, c.Field)
	if !ok || !slices.Contains(opsByKind[kind], c.Op) {
		return "invalid_condition"
	}
	if _, ok := record.ParseNumber(c.Value); kind != record.Choice && !ok {
		return "invalid_condition"
	}
	return ""
}

func (s *Service) roleExists(product, role string) bool {
	return slices.ContainsFunc(s.d.IAM.Roles(), func(r iam.Role) bool { return r.Product == product && r.Role == role })
}
