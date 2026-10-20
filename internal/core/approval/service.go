// Package approval owns approval rules and instances. It is record's approval gate.
package approval

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
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

// Submit implements record.ApprovalGate.
func (s *Service) Submit(ctx context.Context, d record.Doc) (int64, bool, error) {
	return s.open(ctx, d, record.Post, nil)
}

// Hold implements record.ApprovalGate.
func (s *Service) Hold(ctx context.Context, d record.Doc, action record.Action, params json.RawMessage) (bool, error) {
	_, needed, err := s.open(ctx, d, action, params)
	return needed, err
}

// HeldBy implements record.ApprovalGate.
func (s *Service) HeldBy(ctx context.Context, ref record.Ref) (*record.Hold, error) {
	inst, err := store.New(platform.DBFrom(ctx)).HeldBy(ctx, store.HeldByParams{DocType: ref.Type, DocID: ref.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record.Hold{Ticket: inst.ID, Action: record.Action(inst.Action), Params: inst.Params,
		SubmittedBy: inst.SubmittedBy, Version: inst.Version}, nil
}

// HeldInPeriod implements record.ApprovalGate.
func (s *Service) HeldInPeriod(ctx context.Context, legalEntity int64, until pgtype.Date) ([]record.HeldDoc, error) {
	rows, err := store.New(platform.DBFrom(ctx)).HeldInPeriod(ctx, store.HeldInPeriodParams{LegalEntityID: legalEntity, Date: until})
	out := make([]record.HeldDoc, len(rows))
	for i, r := range rows {
		out[i] = record.HeldDoc(r)
	}
	return out, err
}

// open resolves the flow of the rule for (d's type, action) into a plan, opens an
// instance with one step per resolved step, and starts the first ones. No rule, or a
// flow resolving to no step, needs no approval.
func (s *Service) open(ctx context.Context, d record.Doc, action record.Action, params json.RawMessage) (int64, bool, error) {
	q := store.New(platform.DBFrom(ctx))
	rule, err := q.GetRule(ctx, store.GetRuleParams{DocType: d.Type, Action: string(action)})
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
		Action: string(action), Params: params,
	})
	if platform.Violates(err, "instances_one_open_idx") {
		return 0, false, record.ErrAwaitingApproval
	}
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
	return s.requestVotes(ctx, d, excluded, users)
}

// requestVotes tells users, and their active delegates who could vote, that a step waits for them.
func (s *Service) requestVotes(ctx context.Context, d record.Doc, excluded, users []int64) error {
	today, err := s.d.Setting.Today(ctx)
	if err != nil {
		return err
	}
	delegates, err := store.New(platform.DBFrom(ctx)).DelegatesOf(ctx, store.DelegatesOfParams{Users: users, Today: today})
	if err != nil {
		return err
	}
	told := slices.Clone(users)
	for _, id := range delegates {
		if slices.Contains(excluded, id) || slices.Contains(told, id) {
			continue
		}
		if ok, err := s.d.Record.Can(s.d.IAM.AsUser(ctx, id), d.Type, d.ID, record.View); err != nil {
			return err
		} else if ok {
			told = append(told, id)
		}
	}
	return s.d.Notification.Send(ctx, notification.ApprovalRequested, d.Ref, told)
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
	case OpHas:
		ids, ok := record.ParseSet(v)
		id, err := strconv.ParseInt(c.Value, 10, 64)
		if !ok || err != nil {
			return doubt, nil
		}
		return slices.Contains(ids, id), nil
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
// checks the instance and step are open; step 0 acts on the instance alone. A stale
// instance is closed and the close committed before ErrApprovalStale is returned.
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
		var st store.ApprovalInstanceStep
		if step != 0 {
			st, err = q.GetStep(ctx, store.GetStepParams{InstanceID: id, Position: step})
			if errors.Is(err, pgx.ErrNoRows) || err == nil && st.Status != "open" {
				return ErrClosed
			}
			if err != nil {
				return err
			}
		}
		if t, _ := s.d.Record.TypeOf(ref.Type); platform.ProductGate(ctx, t.Product, platform.ClassWrite) != nil {
			return platform.ProductGate(ctx, t.Product, platform.ClassWrite)
		}
		// A post moves the document to pending_approval; any other request holds it at
		// the version it was asked on.
		current := d.Version == inst.Version
		if inst.Action == string(record.Post) {
			current = current && d.Status == record.PendingApproval && d.ApprovalTicket != nil && *d.ApprovalTicket == id
		}
		if !current {
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
// next steps of the plan start; the last one posts the document, or runs the held
// action, in the same transaction. If that fails the vote is not kept and the
// instance stays open.
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
		if inst.Action != string(record.Post) {
			// While the instance is open it is the hold record checks the run against.
			if err := s.d.Record.CompleteAction(ctx, d.Ref, id); err != nil {
				return err
			}
		}
		if err := q.CloseInstance(ctx, store.CloseInstanceParams{ID: id, Status: "approved"}); err != nil {
			return err
		}
		if inst.Action == string(record.Post) {
			if err := s.d.Record.CompleteApproval(ctx, d.Ref, id, inst.Version); err != nil {
				return err
			}
		}
		return s.d.Notification.Send(ctx, notification.ApprovalApproved, d.Ref, []int64{inst.SubmittedBy})
	})
}

// Reject votes against step. Once the step rejects, the other steps are cancelled,
// the instance closes and a submitted document goes back to draft; a held one is
// left as it was.
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
		if inst.Action == string(record.Post) {
			if err := s.d.Record.Reject(ctx, d.Ref, id, inst.Version); err != nil {
				return err
			}
		}
		return s.d.Notification.Send(ctx, notification.ApprovalRejected, d.Ref, []int64{inst.SubmittedBy})
	})
}

// Withdraw takes back an open request for an action other than post; only its
// submitter may. A submission to post is withdrawn through its document instead.
func (s *Service) Withdraw(ctx context.Context, id int64) error {
	return s.act(ctx, id, 0, func(ctx context.Context, inst store.ApprovalInstance, _ store.ApprovalInstanceStep, d record.Doc) error {
		if actor, _ := platform.ActorFrom(ctx); actor != inst.SubmittedBy {
			return platform.ErrForbidden
		}
		if inst.Action == string(record.Post) {
			return record.ErrInvalidTransition
		}
		q := store.New(platform.DBFrom(ctx))
		if err := q.CancelOpenSteps(ctx, id); err != nil {
			return err
		}
		if err := q.CloseInstance(ctx, store.CloseInstanceParams{ID: id, Status: "withdrawn"}); err != nil {
			return err
		}
		return s.d.Audit.RecordFor(ctx, "approval.withdrawn", audit.Ref(d.Ref), map[string]any{"instance": id, "action": inst.Action})
	})
}

// vote records the actor's vote on an open step and returns the step's new status:
// approved, rejected, or "" while it is still open. A step rejects on its rejection
// count, or as soon as the remaining voters can no longer bring its approvals.
func (s *Service) vote(ctx context.Context, inst store.ApprovalInstance, st store.ApprovalInstanceStep, decision string, reason *string) (string, error) {
	q := store.New(platform.DBFrom(ctx))
	actor, _ := platform.ActorFrom(ctx)
	seat, onBehalfOf, err := s.seatFor(ctx, st, actor)
	if err != nil {
		return "", err
	}
	// Approvers were fixed when the step started; who the document is about may have changed
	// since. A delegate is checked as themself, not as the absent approver.
	if excluded, err := s.excluded(ctx, inst, record.Ref{Type: inst.DocType, ID: inst.DocID}); err != nil {
		return "", err
	} else if slices.Contains(excluded, actor) || onBehalfOf != nil && slices.Contains(excluded, seat) {
		// Nor may an excluded seat holder hand their vote to someone else.
		return "", ErrSelfApproval
	}
	var castBy *int64
	params := map[string]any{"instance": inst.ID, "step": st.Position, "reason": reason}
	if onBehalfOf != nil {
		castBy, params["on_behalf_of"] = &actor, seat
	}
	err = q.InsertVote(ctx, store.InsertVoteParams{InstanceID: inst.ID, Position: st.Position, UserID: seat,
		CastBy: platform.NullInt8(castBy), Decision: decision, Reason: platform.NullText(reason)})
	if platform.Violates(err, "step_votes_pkey") || platform.Violates(err, "step_votes_one_per_person_idx") {
		return "", ErrAlreadyVoted
	}
	if err != nil {
		return "", err
	}
	if err := s.d.Audit.RecordFor(ctx, "approval."+decision, audit.Ref{Type: inst.DocType, ID: inst.DocID}, params); err != nil {
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

// seatFor picks the seat the actor votes in step: their own while pending, else the first
// pending seat, in approver order, of a voter whose active delegation names the actor.
// Delegations are read at vote time and never chained, so a delegate's own delegate gets nothing.
func (s *Service) seatFor(ctx context.Context, st store.ApprovalInstanceStep, actor int64) (int64, *int64, error) {
	q := store.New(platform.DBFrom(ctx))
	votes, err := q.CastVotes(ctx, store.CastVotesParams{InstanceID: st.InstanceID, Position: st.Position})
	if err != nil {
		return 0, nil, err
	}
	voted := map[int64]bool{}
	for _, v := range votes {
		if v.CastBy.Valid && v.CastBy.Int64 == actor || !v.CastBy.Valid && v.UserID == actor {
			return 0, nil, ErrAlreadyVoted
		}
		voted[v.UserID] = true
	}
	member := slices.Contains(st.Approvers, actor)
	if member && !voted[actor] {
		return actor, nil, nil
	}
	today, err := s.d.Setting.Today(ctx)
	if err != nil {
		return 0, nil, err
	}
	from, err := q.DelegatorsOf(ctx, store.DelegatorsOfParams{Delegate: actor, Users: st.Approvers, Today: today})
	if err != nil {
		return 0, nil, err
	}
	for _, a := range st.Approvers {
		if !voted[a] && slices.Contains(from, a) {
			return a, &a, nil
		}
	}
	if member { // their seat was voted by their own delegate
		return 0, nil, ErrAlreadyVoted
	}
	return 0, nil, platform.ErrForbidden
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
		excluded, err := s.excluded(ctx, inst, d.Ref)
		if err != nil {
			return err
		}
		if slices.Contains(excluded, user) {
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
		// Someone who already voted here as a delegate could never use the seat.
		votes, err := q.CastVotes(ctx, store.CastVotesParams{InstanceID: id, Position: step})
		if err != nil {
			return err
		}
		if slices.ContainsFunc(votes, func(v store.CastVotesRow) bool {
			return v.CastBy.Valid && v.CastBy.Int64 == user || !v.CastBy.Valid && v.UserID == user
		}) {
			return ErrAlreadyVoted
		}
		if voted, err := q.HasVoted(ctx, store.HasVotedParams{InstanceID: id, Position: step, UserID: from}); err != nil || voted {
			return platform.OrErr(err, ErrAlreadyVoted)
		}
		st.Approvers[i] = user
		if err := q.SetApprovers(ctx, store.SetApproversParams{InstanceID: id, Position: step, Approvers: st.Approvers}); err != nil {
			return err
		}
		if err := s.requestVotes(ctx, d, excluded, []int64{user}); err != nil {
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
	today, err := s.d.Setting.Today(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := store.New(platform.DBFrom(ctx)).Inbox(ctx, store.InboxParams{Actor: actor, Today: today})
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
		// Own seats left out the excluded when the step started; a delegate is checked here.
		if r.OnBehalfOf.Valid {
			excluded, err := s.excluded(ctx, store.ApprovalInstance{SubmittedBy: r.SubmittedBy}, record.Ref{Type: r.DocType, ID: r.DocID})
			if err != nil {
				return nil, err
			}
			if slices.Contains(excluded, actor) || slices.Contains(excluded, r.OnBehalfOf.Int64) {
				continue
			}
		}
		out = append(out, InboxItem{
			InstanceID: r.ID, DocType: r.DocType, DocID: r.DocID, Action: r.Action, Number: r.Number, Date: r.Date.Time.Format(time.DateOnly),
			Step: r.Step, SubmittedByName: r.SubmittedByName, SubmittedAt: r.SubmittedAt.Time.Format(time.RFC3339),
			OnBehalfOfName: platform.TextPtr(r.OnBehalfOfName),
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
	out := &Instance{ID: inst.ID, Action: inst.Action, Status: inst.Status, SubmittedByName: inst.SubmittedByName,
		SubmittedAt: inst.SubmittedAt.Time.Format(time.RFC3339), Steps: []StepView{}, AllowedActions: []string{}}
	if err := json.Unmarshal(inst.Plan, &out.Plan); err != nil {
		return nil, err
	}
	actor, _ := platform.ActorFrom(ctx)
	t, _ := s.d.Record.TypeOf(ref.Type)
	writable := inst.Status == "open" && platform.ProductGate(ctx, t.Product, platform.ClassWrite) == nil
	if writable && actor == inst.SubmittedBy && inst.Action != string(record.Post) {
		out.AllowedActions = append(out.AllowedActions, "withdraw")
	}
	fallback, excluded := false, []int64(nil)
	if writable {
		d, err := s.d.Record.Get(ctx, ref)
		if err != nil {
			return nil, err
		}
		if fallback, err = s.isFallback(ctx, store.ApprovalInstance{FallbackProduct: inst.FallbackProduct, FallbackRole: inst.FallbackRole}, d); err != nil {
			return nil, err
		}
		if excluded, err = s.excluded(ctx, store.ApprovalInstance{SubmittedBy: inst.SubmittedBy}, ref); err != nil {
			return nil, err
		}
	}
	for _, r := range rows {
		var voters []struct {
			ID         int64      `json:"id"`
			Name       string     `json:"name"`
			Decision   *string    `json:"decision"`
			Reason     *string    `json:"reason"`
			DecidedAt  *time.Time `json:"decided_at"`
			CastByName *string    `json:"cast_by_name"`
		}
		if err := json.Unmarshal(r.Voters, &voters); err != nil {
			return nil, err
		}
		v := StepView{Position: r.Position, Status: r.Status, Approvals: r.Approvals, Rejections: r.Rejections, Fallback: r.Fallback,
			Voters: []StepVoter{}, AllowedActions: []string{}}
		for _, x := range voters {
			sv := StepVoter{ID: x.ID, Name: x.Name, Decision: x.Decision, Reason: x.Reason, CastByName: x.CastByName}
			if x.DecidedAt != nil {
				at := x.DecidedAt.Format(time.RFC3339)
				sv.DecidedAt = &at
			}
			v.Voters = append(v.Voters, sv)
		}
		if writable && r.Status == "open" {
			seat, onBehalfOf, err := s.seatFor(ctx, store.ApprovalInstanceStep{InstanceID: inst.ID, Position: r.Position, Approvers: r.Approvers}, actor)
			if err != nil && !errors.Is(err, platform.ErrForbidden) && !errors.Is(err, ErrAlreadyVoted) {
				return nil, err
			}
			if err == nil && !slices.Contains(excluded, actor) && (onBehalfOf == nil || !slices.Contains(excluded, seat)) {
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

// actionsOf lists the actions a type offers a rule for: post and cancel, then its
// module actions sorted.
func actionsOf(t record.Type) []record.Action {
	return append([]record.Action{record.Post, record.Cancel}, slices.Sorted(maps.Keys(t.ModuleActions))...)
}

// requireRuleAction resolves docType for a write, same as requireType, and checks it offers action.
func (s *Service) requireRuleAction(ctx context.Context, docType string, action record.Action) (record.Type, error) {
	t, err := s.requireType(ctx, docType)
	if err != nil {
		return t, err
	}
	if !slices.Contains(actionsOf(t), action) {
		return t, errInvalidRule("", "unknown_action")
	}
	return t, nil
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
		// Every document has an org unit, before its type's own approval fields; shared
		// across every action of the type, since they don't change what the document is about.
		fields := []Field{{Key: FieldOrgUnit, Kind: string(record.OrgUnit), Label: "shared.rule.field.org_unit", Ops: []Op{OpWithin}}}
		for _, f := range t.Fields {
			field := Field{Key: f.Key, Kind: string(f.Kind), Label: f.Label, Ops: opsByKind[f.Kind], Searchable: f.Search != nil}
			if f.Options != nil {
				if field.Options, err = f.Options(ctx); err != nil {
					return nil, err
				}
			}
			fields = append(fields, field)
		}
		allowed := []string{}
		if platform.ProductGate(ctx, t.Product, platform.ClassWrite) == nil {
			allowed = []string{"save", "delete"}
		}
		for _, a := range actionsOf(t) {
			tr := TypeRule{DocType: t.Code, Action: string(a), Product: t.Product, Fields: fields,
				ModuleApprovers: t.Approvers != nil, AllowedActions: allowed}
			if i := slices.IndexFunc(rows, func(r store.ApprovalRule) bool { return r.DocType == t.Code && r.Action == string(a) }); i >= 0 {
				r := rows[i]
				tr.Rule = &RuleInput{MaxLevels: r.MaxLevels, FallbackProduct: r.FallbackProduct, FallbackRole: r.FallbackRole}
				if err := json.Unmarshal(r.Flow, &tr.Rule.Flow); err != nil {
					return nil, err
				}
			}
			out = append(out, tr)
		}
	}
	return out, nil
}

// FieldOptions finds the values a set field of docType may hold, for whoever configures
// its rules: the entries naming ids when ids is set, else up to 20 matches of q.
func (s *Service) FieldOptions(ctx context.Context, docType, key, q string, ids []int64) ([]record.FieldOption, error) {
	t, ok := s.d.Record.TypeOf(docType)
	if !ok || t.Kind != record.Document {
		return nil, platform.ErrNotFound
	}
	if ok, err := s.managesProduct(ctx, t.Product); err != nil {
		return nil, err
	} else if !ok {
		return nil, platform.ErrForbidden
	}
	i := slices.IndexFunc(t.Fields, func(f record.Field) bool { return f.Key == key && f.Search != nil })
	if i < 0 {
		return nil, platform.ErrNotFound
	}
	var out []record.FieldOption
	var err error
	if len(ids) > 0 {
		out, err = t.Fields[i].Labels(ctx, ids)
	} else {
		out, err = t.Fields[i].Search(ctx, q)
	}
	if out == nil && err == nil {
		out = []record.FieldOption{} // a list, never null
	}
	return out, err
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

// SaveRule replaces the rule of a document type's action; open instances keep the plan they started with.
func (s *Service) SaveRule(ctx context.Context, docType string, action record.Action, in RuleInput) error {
	t, err := s.requireRuleAction(ctx, docType, action)
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
			DocType: docType, Action: string(action), Flow: flow, MaxLevels: in.MaxLevels,
			FallbackProduct: in.FallbackProduct, FallbackRole: in.FallbackRole,
		}); err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "approval.rule_saved", map[string]any{"doc_type": docType, "action": action, "rule": in})
	})
}

// DeleteRule removes a document type's rule for action, so sending posts at once.
func (s *Service) DeleteRule(ctx context.Context, docType string, action record.Action) error {
	if _, err := s.requireRuleAction(ctx, docType, action); err != nil {
		return err
	}
	return platform.InTx(ctx, func(ctx context.Context) error {
		if err := store.New(platform.DBFrom(ctx)).DeleteRule(ctx, store.DeleteRuleParams{DocType: docType, Action: string(action)}); err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "approval.rule_deleted", map[string]any{"doc_type": docType, "action": action})
	})
}

func (s *Service) roleExists(product, role string) bool {
	return slices.ContainsFunc(s.d.IAM.Roles(), func(r iam.Role) bool { return r.Product == product && r.Role == role })
}
