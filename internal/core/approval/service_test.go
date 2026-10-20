package approval_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/coretest"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

type fixture struct {
	t        *testing.T
	ids      *iam.Service
	rec      *record.Service
	appr     *approval.Service
	core     *coretest.Core
	admin    context.Context
	unit     int64
	users    map[string]int64
	as       func(login string) context.Context
	managers map[int][]int64         // level → what the type's Approvers returns
	denied   map[int64]record.Action // a user the grant refuses one action
	stamps   []stamp                 // each run of test.doc's stamp action
}

type stamp struct {
	actor  int64
	params string
}

// newFixture registers test.doc with fields days and kind, module approvers from f.managers,
// a type other.doc of product other, and users submitter, boss, big_boss, hr (role test.hr
// at the unit), rules (test.approval_admin), outsider and c1, c2, c3 (no role: test.doc's
// grant lets them view, and they stay out of the fallback role).
func newFixture(t *testing.T) *fixture {
	core := coretest.New(t, "test", "other")
	ctx := core.Ctx
	f := &fixture{t: t, ids: core.IAM, rec: core.Record, appr: core.Approval, core: core, admin: core.Admin, users: map[string]int64{}, managers: map[int][]int64{}, denied: map[int64]record.Action{}}
	f.ids.RegisterRoles("test", map[string][]string{"hr": {"test.doc.view"}, "approval_admin": {"test.approval.manage"}}, "approval_admin")
	f.rec.Register(record.Type{
		Code: "test.doc", Product: "test", Kind: record.Document, NumberPrefix: "T",
		// Everyone but the outsider may do anything with it.
		Perms: record.Perms{View: "test.doc.view", Edit: "test.doc.view"},
		Grant: func(ctx context.Context, _ int64, a record.Action) (bool, error) {
			id, _ := platform.ActorFrom(ctx)
			return id != f.users["outsider"] && f.denied[id] != a, nil
		},
		ModuleActions: map[record.Action]record.ModuleAction{"stamp": func(ctx context.Context, _ record.Doc, params json.RawMessage) error {
			actor, _ := platform.ActorFrom(ctx)
			f.stamps = append(f.stamps, stamp{actor, string(params)})
			return nil
		}},
		OnDelete:       func(context.Context, int64) error { return nil },
		SubjectCreates: true,
		Fields: []record.Field{{Key: "days", Kind: record.Number}, {Key: "kind", Kind: record.Choice,
			Options: func(context.Context) ([]record.Option, error) {
				return []record.Option{{Value: "a"}, {Value: "b"}}, nil
			}},
			{Key: "parts", Kind: record.Set,
				Search: func(_ context.Context, q string) ([]record.FieldOption, error) {
					if q == "none" {
						return nil, nil
					}
					return []record.FieldOption{{ID: 17, Code: "P-17", Name: "part " + q}}, nil
				},
				Labels: func(_ context.Context, ids []int64) ([]record.FieldOption, error) {
					out := []record.FieldOption{}
					for _, id := range ids {
						out = append(out, record.FieldOption{ID: id, Code: "P-" + strconv.FormatInt(id, 10)})
					}
					return out, nil
				}}},
		Approvers: func(_ context.Context, _ record.Ref, level int) ([]int64, error) { return f.managers[level], nil },
	})
	// A type of another product, for rules a product's administrator may not touch.
	f.ids.RegisterRoles("other", map[string][]string{"member": {"other.doc.edit"}})
	f.rec.Register(record.Type{Code: "other.doc", Product: "other", Kind: record.Document, NumberPrefix: "O",
		Perms: record.Perms{View: "other.doc.edit", Edit: "other.doc.edit"}, OnDelete: func(context.Context, int64) error { return nil }})
	c, err := f.ids.CreateOrgUnit(f.admin, iam.OrgUnitInput{Kind: "company", Name: "C"})
	f.check(err)
	f.unit, err = f.ids.CreateOrgUnit(f.admin, iam.OrgUnitInput{ParentID: &c, Kind: "department", Name: "A"})
	f.check(err)
	for _, login := range []string{"submitter", "boss", "big_boss", "hr", "rules", "outsider", "c1", "c2", "c3"} {
		f.users[login], err = f.ids.CreateUser(f.admin, login, login, "long enough")
		f.check(err)
	}
	_, err = f.ids.GrantRole(f.admin, f.users["hr"], "test", "hr", &f.unit)
	f.check(err)
	_, err = f.ids.GrantRole(f.admin, f.users["rules"], "test", "approval_admin", nil)
	f.check(err)
	f.as = func(login string) context.Context { return platform.WithActor(ctx, f.users[login]) }
	return f
}

func (f *fixture) check(err error) {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) rule(flow ...approval.Element) {
	f.t.Helper()
	f.check(f.appr.SaveRule(f.admin, "test.doc", record.Post, ruleOf(flow...)))
}

func ruleOf(flow ...approval.Element) approval.RuleInput {
	return approval.RuleInput{Flow: flow, MaxLevels: 3, FallbackProduct: "test", FallbackRole: "hr"}
}

// by is a step for the named users needing approvals approvals and rejections rejections.
func (f *fixture) by(approvals, rejections int32, logins ...string) approval.Element {
	a := approval.Approver{Kind: "user"}
	for _, l := range logins {
		a.UserIDs = append(a.UserIDs, f.users[l])
	}
	return approval.Element{Step: &approval.StepDef{Approver: a, Approvals: approvals, Rejections: rejections}}
}

// user is a one-approval step for the named users.
func (f *fixture) user(logins ...string) approval.Element { return f.by(1, 1, logins...) }

func split(mode approval.SplitMode, otherwise approval.Flow, branches ...approval.Branch) approval.Element {
	return approval.Element{Split: &approval.Split{Mode: mode, Branches: branches, Otherwise: otherwise}}
}

func when(c *approval.Expr, body ...approval.Element) approval.Branch {
	return approval.Branch{Condition: c, Body: body}
}

func leaf(field, op, value string) *approval.Expr {
	return &approval.Expr{Field: field, Op: approval.Op(op), Value: value}
}

func names(s approval.StepView) []string {
	out := []string{}
	for _, v := range s.Voters {
		out = append(out, v.Name)
	}
	return out
}

func (f *fixture) status(d record.Doc) record.Status {
	f.t.Helper()
	d, err := f.rec.Get(f.admin, d.Ref)
	f.check(err)
	return d.Status
}

func (f *fixture) inbox(login string) []approval.InboxItem {
	f.t.Helper()
	items, err := f.appr.Inbox(f.as(login))
	f.check(err)
	return items
}

// send creates a draft with days and kind as the submitter and sends it.
func (f *fixture) send(days, kind string) (record.Doc, error) {
	f.t.Helper()
	d, err := f.rec.Create(f.as("submitter"), "test.doc", record.Header{Date: "2026-03-10", OrgUnitID: f.unit, Subject: f.users["submitter"],
		Fields: map[string]string{"days": days, "kind": kind}})
	f.check(err)
	if _, err := f.rec.Transition(f.as("submitter"), d.Ref, 1, record.Posted); err != nil {
		return d, err
	}
	return f.rec.Get(f.admin, d.Ref)
}

func (f *fixture) panel(login string, d record.Doc) *approval.Instance {
	f.t.Helper()
	inst, err := f.appr.DocumentInstance(f.as(login), d.Ref)
	f.check(err)
	return inst
}

var module = approval.Element{Step: &approval.StepDef{Approver: approval.Approver{Kind: "module"}}}

func code(err error) string {
	if e, ok := errors.AsType[*platform.Error](err); ok {
		return e.Code
	}
	return ""
}

func TestModuleApproversSkipTheSubmitter(t *testing.T) {
	f := newFixture(t)
	f.rule(module)
	f.managers[1] = []int64{f.users["submitter"]} // the submitter manages themself at level 1
	f.managers[2] = []int64{f.users["boss"]}
	d, err := f.send("1", "a")
	f.check(err)
	inst := f.panel("boss", d)
	if !slices.Equal(names(inst.Steps[0]), []string{"boss"}) || inst.Steps[0].Fallback {
		t.Fatalf("want boss: %+v", inst.Steps[0])
	}
	if !slices.Equal(inst.Steps[0].AllowedActions, []string{"approve", "reject"}) {
		t.Fatalf("boss actions %v", inst.Steps[0].AllowedActions)
	}
	if acts := f.panel("big_boss", d).Steps[0].AllowedActions; len(acts) != 0 {
		t.Fatalf("big_boss actions %v", acts)
	}
	// A manager change after the step started does not change its approvers.
	f.managers[2] = []int64{f.users["big_boss"]}
	if code(f.appr.Approve(f.as("big_boss"), *d.ApprovalTicket, 1)) != "forbidden" {
		t.Fatal("big_boss approved")
	}
	f.check(f.appr.Approve(f.as("boss"), *d.ApprovalTicket, 1))
	if d, _ := f.rec.Get(f.admin, d.Ref); d.Status != record.Posted {
		t.Fatalf("status %s", d.Status)
	}
}

func TestNobodyLeftGoesToTheFallbackRole(t *testing.T) {
	f := newFixture(t)
	f.rule(module)
	d, err := f.send("1", "a")
	f.check(err)
	inst := f.panel("hr", d)
	if !slices.Equal(names(inst.Steps[0]), []string{"hr"}) || !inst.Steps[0].Fallback {
		t.Fatalf("want hr as fallback: %+v", inst.Steps[0])
	}
	if !slices.Equal(inst.Steps[0].AllowedActions, []string{"approve", "reject", "reassign"}) {
		t.Fatalf("hr actions %v", inst.Steps[0].AllowedActions)
	}
}

func TestNoApproverAtAllRefusesToSend(t *testing.T) {
	f := newFixture(t)
	f.rule(f.user("submitter"))
	f.check(f.ids.RevokeRole(f.admin, f.users["hr"], mustGrant(t, f, "hr")))
	if _, err := f.send("1", "a"); code(err) != "no_approver" {
		t.Fatalf("got %v", err)
	}
}

func mustGrant(t *testing.T, f *fixture, login string) int64 {
	grants, err := f.ids.UserRoles(f.admin, f.users[login])
	f.check(err)
	return grants[0].ID
}

func TestConditionsPickTheSteps(t *testing.T) {
	f := newFixture(t)
	f.rule(f.user("boss"), split(approval.ChooseOne, nil, when(leaf("days", "gt", "3"), f.user("big_boss"))))

	short, err := f.send("3", "a")
	f.check(err)
	if n := len(f.panel("boss", short).Steps); n != 1 {
		t.Fatalf("3 days: %d steps", n)
	}
	d, err := f.send("3.5", "a")
	f.check(err)
	f.check(f.appr.Approve(f.as("boss"), *d.ApprovalTicket, 1))
	if code(f.appr.Approve(f.as("boss"), *d.ApprovalTicket, 1)) != "approval_closed" {
		t.Fatal("step 1 approved twice")
	}
	inst := f.panel("big_boss", d)
	if openSteps(inst) != "2" || !slices.Equal(inst.Steps[1].AllowedActions, []string{"approve", "reject"}) {
		t.Fatalf("want step 2 for big_boss: %+v", inst)
	}
	f.check(f.appr.Approve(f.as("big_boss"), *d.ApprovalTicket, 2))
	if d, _ := f.rec.Get(f.admin, d.Ref); d.Status != record.Posted {
		t.Fatalf("status %s", d.Status)
	}
}

func TestRejectReturnsToDraft(t *testing.T) {
	f := newFixture(t)
	f.rule(f.user("boss"))
	d, err := f.send("1", "a")
	f.check(err)
	f.check(f.appr.Reject(f.as("boss"), *d.ApprovalTicket, 1, "too long"))
	after, _ := f.rec.Get(f.admin, d.Ref)
	if after.Status != record.Draft || after.ApprovalTicket != nil || after.Version != d.Version+1 {
		t.Fatalf("after reject: %+v", after)
	}
	inst := f.panel("submitter", d)
	if inst.Status != "rejected" || inst.Steps[0].Status != "rejected" || *inst.Steps[0].Voters[0].Reason != "too long" {
		t.Fatalf("instance %+v", inst)
	}
}

func TestReassignByTheFallbackRole(t *testing.T) {
	f := newFixture(t)
	f.rule(f.user("boss"))
	d, err := f.send("1", "a")
	f.check(err)
	boss := f.users["boss"]
	if code(f.appr.Reassign(f.as("boss"), *d.ApprovalTicket, 1, boss, "big_boss")) != "forbidden" {
		t.Fatal("boss reassigned without the fallback role")
	}
	if code(f.appr.Reassign(f.as("hr"), *d.ApprovalTicket, 1, boss, "submitter")) != "self_approval" {
		t.Fatal("reassigned to the submitter")
	}
	// One who cannot view the document could never act on the step.
	if code(f.appr.Reassign(f.as("hr"), *d.ApprovalTicket, 1, boss, "outsider")) != "approver_cannot_view" {
		t.Fatal("reassigned to someone who cannot view it")
	}
	f.check(f.appr.Reassign(f.as("hr"), *d.ApprovalTicket, 1, boss, "big_boss"))
	if code(f.appr.Approve(f.as("boss"), *d.ApprovalTicket, 1)) != "forbidden" {
		t.Fatal("old approver still approves")
	}
	items, err := f.appr.Inbox(f.as("big_boss"))
	f.check(err)
	if len(items) != 1 || items[0].Number != d.Number {
		t.Fatalf("big_boss inbox %+v", items)
	}
	f.check(f.appr.Approve(f.as("big_boss"), *d.ApprovalTicket, 1))
}

func TestInboxHidesDocumentsTheActorCannotView(t *testing.T) {
	f := newFixture(t)
	f.rule(f.user("outsider"))
	_, err := f.send("1", "a")
	f.check(err)
	items, err := f.appr.Inbox(f.as("outsider"))
	f.check(err)
	if len(items) != 0 {
		t.Fatalf("outsider sees %+v", items)
	}
}

func TestApproversWhoCannotViewAreSkipped(t *testing.T) {
	f := newFixture(t)
	f.rule(f.user("outsider"))
	d, err := f.send("1", "a")
	f.check(err)
	if s := f.panel("hr", d).Steps[0]; !slices.Equal(names(s), []string{"hr"}) || !s.Fallback {
		t.Fatalf("want the fallback role instead of outsider: %+v", s)
	}
}

func TestRuleChecksConditionsAgainstFields(t *testing.T) {
	f := newFixture(t)
	bad := split(approval.ChooseOne, nil, when(leaf("kind", "gt", "1"), module))
	err := f.appr.SaveRule(f.admin, "test.doc", record.Post, ruleOf(module, bad))
	if e, ok := errors.AsType[*platform.Error](err); !ok || e.Code != "invalid_rule" || e.Params["path"] != "2.branches.1" || e.Params["reason"] != "invalid_condition" {
		t.Fatalf("got %v", err)
	}
	if code(f.appr.SaveRule(f.as("hr"), "test.doc", record.Post, approval.RuleInput{})) != "forbidden" {
		t.Fatal("hr saved a rule")
	}
}

func TestUnknownActionRefused(t *testing.T) {
	f := newFixture(t)
	err := f.appr.SaveRule(f.admin, "test.doc", "create_order", approval.RuleInput{})
	if e, ok := errors.AsType[*platform.Error](err); !ok || e.Code != "invalid_rule" || e.Params["reason"] != "unknown_action" {
		t.Fatalf("got %v", err)
	}
	if code(f.appr.DeleteRule(f.admin, "test.doc", "create_order")) != "invalid_rule" {
		t.Fatal("delete didn't refuse unknown action")
	}
}

// docTypes lists the distinct document types in rules, in the order they first appear.
func docTypes(rules []approval.TypeRule) []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range rules {
		if !seen[r.DocType] {
			seen[r.DocType] = true
			out = append(out, r.DocType)
		}
	}
	return out
}

// ruleOfAction finds the rule entry for docType and action, or nil.
func ruleOfAction(rules []approval.TypeRule, docType string, action record.Action) *approval.TypeRule {
	for i := range rules {
		if rules[i].DocType == docType && rules[i].Action == string(action) {
			return &rules[i]
		}
	}
	return nil
}

func TestRulesPerAction(t *testing.T) {
	f := newFixture(t)
	f.rule(f.user("boss")) // the post rule
	f.check(f.appr.SaveRule(f.admin, "test.doc", record.Cancel, ruleOf(f.user("big_boss"))))

	rules, err := f.appr.Rules(f.admin, "test")
	f.check(err)
	post, cancel := ruleOfAction(rules, "test.doc", record.Post), ruleOfAction(rules, "test.doc", record.Cancel)
	if post == nil || post.Rule == nil || cancel == nil || cancel.Rule == nil {
		t.Fatalf("want both post and cancel rules: %+v", rules)
	}

	f.check(f.appr.DeleteRule(f.admin, "test.doc", record.Cancel))
	rules, err = f.appr.Rules(f.admin, "test")
	f.check(err)
	post, cancel = ruleOfAction(rules, "test.doc", record.Post), ruleOfAction(rules, "test.doc", record.Cancel)
	if post == nil || post.Rule == nil {
		t.Fatalf("post rule should stay: %+v", rules)
	}
	if cancel == nil || cancel.Rule != nil {
		t.Fatalf("cancel rule should be gone: %+v", cancel)
	}
}

func TestRulesOfferTheOrgUnit(t *testing.T) {
	f := newFixture(t)
	rules, err := f.appr.Rules(f.admin, "test")
	f.check(err)
	want := approval.Field{Key: "org_unit", Kind: "org_unit", Label: "shared.rule.field.org_unit", Ops: []approval.Op{approval.OpWithin}}
	if !reflect.DeepEqual(rules[0].Fields[0], want) {
		t.Fatalf("org unit field %+v", rules[0].Fields[0])
	}
	var keys []string
	for _, f := range rules[0].Fields[1:] {
		keys = append(keys, f.Key)
	}
	if !slices.Equal(keys, []string{"days", "kind", "parts"}) || !rules[0].Fields[3].Searchable || rules[0].Fields[1].Searchable {
		t.Fatalf("fields after org unit: %v", keys)
	}
}

func TestRulesByProduct(t *testing.T) {
	f := newFixture(t)
	in := ruleOf(module)

	// A product's administrator manages only that product's types.
	rules, err := f.appr.Rules(f.as("rules"), "")
	f.check(err)
	if got := docTypes(rules); !slices.Equal(got, []string{"test.doc"}) {
		t.Fatalf("product admin lists %v", got)
	}
	if !slices.Equal(rules[0].AllowedActions, []string{"save", "delete"}) {
		t.Fatalf("product admin actions %v", rules[0].AllowedActions)
	}
	f.check(f.appr.SaveRule(f.as("rules"), "test.doc", record.Post, in))
	f.check(f.appr.DeleteRule(f.as("rules"), "test.doc", record.Post))
	if _, err := f.appr.Users(f.as("rules")); err != nil {
		t.Fatalf("product admin lists users: %v", err)
	}
	if code(f.appr.SaveRule(f.as("rules"), "other.doc", record.Post, in)) != "forbidden" {
		t.Fatal("product admin saved another product's rule")
	}
	if code(f.appr.DeleteRule(f.as("rules"), "other.doc", record.Post)) != "forbidden" {
		t.Fatal("product admin deleted another product's rule")
	}

	// core.approval.manage covers every product; the filter narrows the list.
	all, err := f.appr.Rules(f.admin, "")
	f.check(err)
	if got := docTypes(all); !slices.Equal(got, []string{"other.doc", "test.doc"}) {
		t.Fatalf("admin lists %v", got)
	}
	other, err := f.appr.Rules(f.admin, "other")
	f.check(err)
	if got := docTypes(other); !slices.Equal(got, []string{"other.doc"}) {
		t.Fatalf("admin filtered by product lists %v", got)
	}
	f.check(f.appr.SaveRule(f.admin, "other.doc", record.Post, ruleOf(f.user("boss"))))

	// Without any rule permission, everything is refused.
	if _, err := f.appr.Rules(f.as("outsider"), ""); code(err) != "forbidden" {
		t.Fatalf("outsider lists rules: %v", err)
	}
	if _, err := f.appr.Users(f.as("outsider")); code(err) != "forbidden" {
		t.Fatalf("outsider lists users: %v", err)
	}
	if code(f.appr.SaveRule(f.as("outsider"), "test.doc", record.Post, in)) != "forbidden" ||
		code(f.appr.DeleteRule(f.as("outsider"), "test.doc", record.Post)) != "forbidden" {
		t.Fatal("outsider changed a rule")
	}

	// A disabled product stays readable, without actions, and refuses writes.
	off := platform.WithProducts(f.as("rules"), []string{"other"})
	rules, err = f.appr.Rules(off, "")
	f.check(err)
	if got := docTypes(rules); !slices.Equal(got, []string{"test.doc"}) || len(rules[0].AllowedActions) != 0 {
		t.Fatalf("disabled product rules %+v", rules)
	}
	if code(f.appr.SaveRule(off, "test.doc", record.Post, in)) != "product_not_enabled" ||
		code(f.appr.DeleteRule(off, "test.doc", record.Post)) != "product_not_enabled" {
		t.Fatal("rule changed on a disabled product")
	}
}

func TestRuleChangesKeepOpenInstances(t *testing.T) {
	f := newFixture(t)
	f.rule(module)
	f.managers[1] = []int64{f.users["boss"]}
	d, err := f.send("1", "a")
	f.check(err)
	steps := f.panel("boss", d).Steps

	f.check(f.appr.SaveRule(f.as("rules"), "test.doc", record.Post, ruleOf(f.user("big_boss"))))
	if got := f.panel("boss", d).Steps; len(got) != len(steps) || !slices.Equal(names(got[0]), names(steps[0])) {
		t.Fatalf("after save: %+v, want %+v", got, steps)
	}
	f.check(f.appr.DeleteRule(f.as("rules"), "test.doc", record.Post))
	if got := f.panel("boss", d).Steps; len(got) != len(steps) || !slices.Equal(names(got[0]), names(steps[0])) {
		t.Fatalf("after delete: %+v, want %+v", got, steps)
	}
	f.check(f.appr.Approve(f.as("boss"), *d.ApprovalTicket, 1))
}

// openSteps lists the positions of the open steps, e.g. "2" or "1,2".
func openSteps(inst *approval.Instance) string {
	var out []string
	for _, s := range inst.Steps {
		if s.Status == "open" {
			out = append(out, strconv.Itoa(int(s.Position)))
		}
	}
	return strings.Join(out, ",")
}

func TestGroupNeedsItsApprovalCount(t *testing.T) {
	f := newFixture(t)
	f.rule(f.by(2, 1, "c1", "c2", "c3"))
	d, err := f.send("1", "a")
	f.check(err)
	f.check(f.appr.Approve(f.as("c1"), *d.ApprovalTicket, 1))
	if st := f.status(d); st != record.PendingApproval || openSteps(f.panel("c1", d)) != "1" {
		t.Fatalf("after one approval: %s %+v", st, f.panel("c1", d).Steps)
	}
	// One who voted has nothing left to do at the step.
	if acts := f.panel("c1", d).Steps[0].AllowedActions; len(acts) != 0 || len(f.inbox("c1")) != 0 {
		t.Fatalf("c1 still acts: %v", acts)
	}
	f.check(f.appr.Approve(f.as("c2"), *d.ApprovalTicket, 1))
	if st := f.status(d); st != record.Posted {
		t.Fatalf("status %s", st)
	}
}

func TestRejectionCountRejects(t *testing.T) {
	f := newFixture(t)
	f.rule(f.by(1, 2, "c1", "c2", "c3"))
	d, err := f.send("1", "a")
	f.check(err)
	f.check(f.appr.Reject(f.as("c1"), *d.ApprovalTicket, 1, "no"))
	if st := f.status(d); st != record.PendingApproval {
		t.Fatalf("after one rejection: %s", st)
	}
	f.check(f.appr.Reject(f.as("c2"), *d.ApprovalTicket, 1, "no"))
	if st, inst := f.status(d), f.panel("c1", d); st != record.Draft || inst.Status != "rejected" {
		t.Fatalf("after two rejections: %s %s", st, inst.Status)
	}
}

func TestStepRejectsWhenItCanNoLongerPass(t *testing.T) {
	f := newFixture(t)
	f.rule(f.by(3, 2, "c1", "c2", "c3"))
	d, err := f.send("1", "a")
	f.check(err)
	f.check(f.appr.Reject(f.as("c1"), *d.ApprovalTicket, 1, "no"))
	if st := f.status(d); st != record.Draft {
		t.Fatalf("status %s", st)
	}
}

func TestOneVotePerStep(t *testing.T) {
	f := newFixture(t)
	f.rule(f.by(2, 1, "c1", "c2", "c3"))
	d, err := f.send("1", "a")
	f.check(err)
	f.check(f.appr.Approve(f.as("c1"), *d.ApprovalTicket, 1))
	if code(f.appr.Approve(f.as("c1"), *d.ApprovalTicket, 1)) != "approval_already_voted" {
		t.Fatal("c1 voted twice")
	}

	// The same voter in two parallel steps votes in each.
	f.rule(f.user("c2"), split(approval.Parallel, nil, when(nil, f.user("c1")), when(nil, f.user("c1"))))
	d, err = f.send("1", "a")
	f.check(err)
	f.check(f.appr.Approve(f.as("c2"), *d.ApprovalTicket, 1))
	if items := f.inbox("c1"); len(items) != 1 || items[0].Step != 2 {
		t.Fatalf("c1 inbox while both parallel steps are open: %+v", items)
	}
	f.check(f.appr.Approve(f.as("c1"), *d.ApprovalTicket, 2))
	f.check(f.appr.Approve(f.as("c1"), *d.ApprovalTicket, 3))
	if st := f.status(d); st != record.Posted {
		t.Fatalf("status %s", st)
	}
}

func (f *fixture) parallelRule() {
	f.rule(split(approval.Parallel, nil, when(nil, f.user("boss")), when(leaf("kind", "eq", "a"), f.user("big_boss"))))
}

func TestParallelStepsOpenTogether(t *testing.T) {
	f := newFixture(t)
	f.parallelRule()
	d, err := f.send("1", "a")
	f.check(err)
	if got := openSteps(f.panel("boss", d)); got != "1,2" {
		t.Fatalf("open steps %q", got)
	}
	if b, bb := f.inbox("boss"), f.inbox("big_boss"); len(b) != 1 || b[0].Step != 1 || len(bb) != 1 || bb[0].Step != 2 {
		t.Fatalf("inboxes %+v %+v", b, bb)
	}
	f.check(f.appr.Approve(f.as("boss"), *d.ApprovalTicket, 1))
	if st := f.status(d); st != record.PendingApproval {
		t.Fatalf("after boss: %s", st)
	}
	f.check(f.appr.Approve(f.as("big_boss"), *d.ApprovalTicket, 2))
	if st := f.status(d); st != record.Posted {
		t.Fatalf("status %s", st)
	}
}

func TestRejectCancelsParallelSteps(t *testing.T) {
	f := newFixture(t)
	f.parallelRule()
	d, err := f.send("1", "a")
	f.check(err)
	f.check(f.appr.Reject(f.as("boss"), *d.ApprovalTicket, 1, "no"))
	inst := f.panel("boss", d)
	if inst.Status != "rejected" || inst.Steps[1].Status != "cancelled" {
		t.Fatalf("instance %+v", inst)
	}
	if items := f.inbox("big_boss"); len(items) != 0 {
		t.Fatalf("big_boss inbox %+v", items)
	}
	if code(f.appr.Approve(f.as("big_boss"), *d.ApprovalTicket, 2)) != "approval_closed" {
		t.Fatal("cancelled step approved")
	}
}

func TestChooseOneByAmountTiers(t *testing.T) {
	f := newFixture(t)
	f.rule(split(approval.ChooseOne, approval.Flow{},
		when(leaf("days", "gt", "5"), f.user("big_boss")), when(leaf("days", "gt", "2"), f.user("boss"))))
	for days, want := range map[string]string{"6": "big_boss", "3": "boss"} {
		d, err := f.send(days, "a")
		f.check(err)
		if s := f.panel("hr", d).Steps; len(s) != 1 || !slices.Equal(names(s[0]), []string{want}) {
			t.Fatalf("%s days: %+v", days, s)
		}
	}
	d, err := f.send("1", "a")
	f.check(err)
	if d.Status != record.Posted || f.panel("hr", d) != nil {
		t.Fatalf("1 day: %+v", d)
	}
}

func TestAllAnyConditions(t *testing.T) {
	f := newFixture(t)
	c := &approval.Expr{Any: []approval.Expr{{All: []approval.Expr{*leaf("days", "gt", "2"), *leaf("kind", "eq", "a")}}, *leaf("days", "gt", "5")}}
	f.rule(split(approval.ChooseOne, nil, when(c, f.user("boss"))))
	for _, tc := range []struct {
		days, kind string
		steps      int
	}{{"3", "a", 1}, {"3", "b", 0}, {"6", "b", 1}} {
		d, err := f.send(tc.days, tc.kind)
		f.check(err)
		n := 0
		if inst := f.panel("hr", d); inst != nil {
			n = len(inst.Steps)
		}
		if n != tc.steps || (n == 0) != (d.Status == record.Posted) {
			t.Fatalf("%s days kind %s: %d steps, %s", tc.days, tc.kind, n, d.Status)
		}
	}
}

func TestTooFewEligibleRefuses(t *testing.T) {
	f := newFixture(t)
	f.rule(f.by(3, 1, "c1", "c2", "submitter"))
	d, err := f.send("1", "a")
	if e, ok := errors.AsType[*platform.Error](err); !ok || e.Code != "no_approver" || e.Params["step"] != int32(1) {
		t.Fatalf("got %v", err)
	}
	if st := f.status(d); st != record.Draft {
		t.Fatalf("status %s", st)
	}
}

func TestReassignReplacesOneVoter(t *testing.T) {
	f := newFixture(t)
	f.rule(f.by(2, 1, "c1", "c2"))
	d, err := f.send("1", "a")
	f.check(err)
	id := *d.ApprovalTicket
	if code(f.appr.Reassign(f.as("hr"), id, 1, f.users["c2"], "c1")) != "approval_already_approver" {
		t.Fatal("reassigned to a voter of the step")
	}
	if code(f.appr.Reassign(f.as("hr"), id, 1, f.users["c3"], "big_boss")) != "not_found" {
		t.Fatal("reassigned someone who is not a voter")
	}
	f.check(f.appr.Reassign(f.as("hr"), id, 1, f.users["c2"], "big_boss"))
	if code(f.appr.Approve(f.as("c2"), id, 1)) != "forbidden" {
		t.Fatal("replaced voter approved")
	}
	f.check(f.appr.Approve(f.as("c1"), id, 1))
	if code(f.appr.Reassign(f.as("hr"), id, 1, f.users["c1"], "c3")) != "approval_already_voted" {
		t.Fatal("reassigned a voter who voted")
	}
	f.check(f.appr.Approve(f.as("big_boss"), id, 1))
	if st := f.status(d); st != record.Posted {
		t.Fatalf("status %s", st)
	}
}

// statuses lists the instance's step statuses by position.
func statuses(inst *approval.Instance) []string {
	var out []string
	for _, s := range inst.Steps {
		out = append(out, s.Status)
	}
	return out
}

func TestClosedInstancesCancelTheirSteps(t *testing.T) {
	f := newFixture(t)
	f.rule(f.user("boss"), f.user("big_boss"))

	d, err := f.send("1", "a")
	f.check(err)
	_, err = f.rec.Transition(f.as("submitter"), d.Ref, d.Version, record.Draft)
	f.check(err)
	if inst := f.panel("boss", d); inst.Status != "withdrawn" || !slices.Equal(statuses(inst), []string{"cancelled", "cancelled"}) {
		t.Fatalf("after withdrawal: %s %v", inst.Status, statuses(inst))
	}

	d, err = f.send("1", "a")
	f.check(err)
	// The document moved on behind the instance's back.
	if _, err := platform.DBFrom(f.admin).Exec(f.admin, `UPDATE record.documents SET version = version + 1 WHERE id = $1`, d.ID); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(f.appr.Approve(f.as("boss"), *d.ApprovalTicket, 1), record.ErrApprovalStale) {
		t.Fatal("want stale")
	}
	if inst := f.panel("boss", d); inst.Status != "stale" || !slices.Equal(statuses(inst), []string{"cancelled", "cancelled"}) {
		t.Fatalf("after stale: %s %v", inst.Status, statuses(inst))
	}
}

// posted is a posted test.doc of the submitter, with a cancel rule for boss.
func (f *fixture) posted() record.Doc {
	f.t.Helper()
	d, err := f.send("1", "a") // no post rule: posts at once
	f.check(err)
	f.check(f.appr.SaveRule(f.admin, "test.doc", record.Cancel, ruleOf(f.user("boss"))))
	return d
}

func (f *fixture) cancel(login string, d record.Doc) (bool, error) {
	return f.rec.Transition(f.as(login), d.Ref, d.Version, record.Cancelled)
}

func (f *fixture) actions(login string, d record.Doc) []string {
	f.t.Helper()
	d, err := f.rec.Get(f.admin, d.Ref)
	f.check(err)
	acts, err := f.rec.Actions(f.as(login), d, "module_action")
	f.check(err)
	return acts
}

// unchanged fails unless d is still posted at the same version.
func (f *fixture) unchanged(d record.Doc) {
	f.t.Helper()
	got, err := f.rec.Get(f.admin, d.Ref)
	f.check(err)
	if got.Status != record.Posted || got.Version != d.Version {
		f.t.Fatalf("document moved: %s v%d, was v%d", got.Status, got.Version, d.Version)
	}
}

func TestCancelWaitsForApproval(t *testing.T) {
	f := newFixture(t)
	d := f.posted()
	held, err := f.cancel("submitter", d)
	f.check(err)
	if !held {
		t.Fatal("cancel did not wait")
	}
	f.unchanged(d)
	if acts := f.actions("submitter", d); len(acts) != 0 {
		t.Fatalf("held document offers %v", acts)
	}
	if _, err := f.rec.Edit(f.as("submitter"), d.Ref, d.Version, record.Header{Date: d.Date, OrgUnitID: f.unit}); code(err) != "awaiting_approval" {
		t.Fatalf("edit: %v", err)
	}
	if _, err := f.cancel("submitter", d); code(err) != "awaiting_approval" {
		t.Fatalf("second cancel: %v", err)
	}
	inst := f.panel("submitter", d)
	if inst.Action != "cancel" || !slices.Equal(inst.AllowedActions, []string{"withdraw"}) {
		t.Fatalf("panel %s %v", inst.Action, inst.AllowedActions)
	}
	f.check(f.appr.Approve(f.as("boss"), inst.ID, 1))
	if s := f.status(d); s != record.Cancelled {
		t.Fatalf("status %s", s)
	}
	if inst := f.panel("boss", d); inst.Status != "approved" {
		t.Fatalf("instance %s", inst.Status)
	}
}

func TestRejectedCancelKeepsTheDocument(t *testing.T) {
	f := newFixture(t)
	d := f.posted()
	_, err := f.cancel("submitter", d)
	f.check(err)
	f.check(f.appr.Reject(f.as("boss"), f.panel("boss", d).ID, 1, "no"))
	f.unchanged(d)
	if acts := f.actions("submitter", d); !slices.Contains(acts, "cancel") {
		t.Fatalf("after reject: %v", acts)
	}
	if inst := f.panel("boss", d); inst.Status != "rejected" || len(inst.AllowedActions) != 0 {
		t.Fatalf("instance %s %v", inst.Status, inst.AllowedActions)
	}

	_, err = f.cancel("submitter", d)
	f.check(err)
	id := f.panel("boss", d).ID
	if code(f.appr.Withdraw(f.as("c1"), id)) != "forbidden" {
		t.Fatal("c1 withdrew")
	}
	if len(f.panel("c1", d).AllowedActions) != 0 {
		t.Fatal("c1 is offered withdraw")
	}
	f.check(f.appr.Withdraw(f.as("submitter"), id))
	f.unchanged(d)
	if acts := f.actions("submitter", d); !slices.Contains(acts, "cancel") {
		t.Fatalf("after withdraw: %v", acts)
	}
	if inst := f.panel("boss", d); inst.Status != "withdrawn" || !slices.Equal(statuses(inst), []string{"cancelled"}) {
		t.Fatalf("instance %s %v", inst.Status, statuses(inst))
	}
}

func TestOneOpenRequestPerDocument(t *testing.T) {
	f := newFixture(t)
	d := f.posted()
	_, err := f.cancel("submitter", d)
	f.check(err)
	if _, err := f.cancel("c1", d); code(err) != "awaiting_approval" {
		t.Fatalf("second cancel: %v", err)
	}

	d, err = f.send("1", "a")
	f.check(err)
	var second error
	var wg sync.WaitGroup
	f.check(platform.InTx(f.as("submitter"), func(ctx context.Context) error {
		if held, err := f.rec.Transition(ctx, d.Ref, d.Version, record.Cancelled); err != nil || !held {
			return platform.OrErr(err, errors.New("not held"))
		}
		wg.Go(func() { _, second = f.cancel("c1", d) })
		pgtest.AwaitLockWait(t, platform.DBFrom(f.admin))
		return nil
	}))
	wg.Wait()
	if code(second) != "awaiting_approval" {
		t.Fatalf("racing cancel: %v", second)
	}
}

func TestCompletionRechecksAsSubmitter(t *testing.T) {
	f := newFixture(t)
	d := f.posted()
	_, err := f.cancel("submitter", d)
	f.check(err)
	id := f.panel("boss", d).ID
	f.denied[f.users["submitter"]] = record.Cancel
	if code(f.appr.Approve(f.as("boss"), id, 1)) != "submitter_forbidden" {
		t.Fatal("approved without the submitter's permission")
	}
	f.unchanged(d)
	inst := f.panel("boss", d)
	if inst.Status != "open" || !slices.Equal(inst.Steps[0].AllowedActions, []string{"approve", "reject"}) {
		t.Fatalf("vote kept: %s %v", inst.Status, inst.Steps[0].AllowedActions)
	}
	delete(f.denied, f.users["submitter"])
	f.check(f.appr.Approve(f.as("boss"), id, 1))
	if s := f.status(d); s != record.Cancelled {
		t.Fatalf("status %s", s)
	}
}

func TestPeriodLockRefusesHeldDocuments(t *testing.T) {
	f := newFixture(t)
	d := f.posted() // dated 2026-03-10
	_, err := f.cancel("submitter", d)
	f.check(err)
	err = f.rec.SetPeriodLock(f.admin, d.LegalEntityID, new("2026-03-31"))
	e, ok := errors.AsType[*platform.Error](err)
	if !ok || e.Code != "period_has_pending_documents" {
		t.Fatalf("lock: %v", err)
	}
	docs, _ := e.Params["documents"].([]map[string]any)
	if len(docs) != 1 || docs[0]["id"] != d.ID || docs[0]["number"] != d.Number {
		t.Fatalf("listed %v", e.Params["documents"])
	}
}

func TestModuleActionRunsOnce(t *testing.T) {
	f := newFixture(t)
	d := f.posted()
	f.check(f.appr.SaveRule(f.admin, "test.doc", "stamp", ruleOf(f.user("boss"))))
	held, err := f.rec.Request(f.as("submitter"), d.Ref, "stamp", json.RawMessage(`{"n": 1}`))
	f.check(err)
	if !held || len(f.stamps) != 0 {
		t.Fatalf("held %v, stamps %v", held, f.stamps)
	}
	if _, err := f.cancel("submitter", d); code(err) != "awaiting_approval" {
		t.Fatalf("cancel while stamping: %v", err)
	}
	f.check(f.appr.Approve(f.as("boss"), f.panel("boss", d).ID, 1))
	if len(f.stamps) != 1 || f.stamps[0].actor != f.users["submitter"] || f.stamps[0].params != `{"n": 1}` {
		t.Fatalf("stamps %+v", f.stamps)
	}
	f.unchanged(d)
	if acts := f.actions("submitter", d); !slices.Contains(acts, "cancel") {
		t.Fatalf("after stamping: %v", acts)
	}
}

// sendFields creates a draft with the given fields as the submitter and sends it.
func (f *fixture) sendFields(fields map[string]string) record.Doc {
	f.t.Helper()
	d, err := f.rec.Create(f.as("submitter"), "test.doc", record.Header{Date: "2026-03-10", OrgUnitID: f.unit, Subject: f.users["submitter"], Fields: fields})
	f.check(err)
	_, err = f.rec.Transition(f.as("submitter"), d.Ref, 1, record.Posted)
	f.check(err)
	d, err = f.rec.Get(f.admin, d.Ref)
	f.check(err)
	return d
}

func TestHasRoutesBySet(t *testing.T) {
	f := newFixture(t)
	f.rule(split(approval.ChooseOne, approval.Flow{}, when(leaf("parts", "has", "17"), f.user("boss"))))
	d := f.sendFields(map[string]string{"parts": "3,17"})
	if s := f.panel("hr", d).Steps; len(s) != 1 || !slices.Equal(names(s[0]), []string{"boss"}) {
		t.Fatalf("3,17: %+v", s)
	}
	if d := f.sendFields(map[string]string{"parts": "3"}); d.Status != record.Posted {
		t.Fatalf("3: %s", d.Status)
	}
}

// A document saved before its type declared the set field still meets the condition.
func TestSetFieldMissingHolds(t *testing.T) {
	f := newFixture(t)
	f.rule(split(approval.ChooseOne, approval.Flow{}, when(leaf("parts", "has", "17"), f.user("boss"))))
	d := f.sendFields(nil)
	if s := f.panel("hr", d).Steps; len(s) != 1 || !slices.Equal(names(s[0]), []string{"boss"}) {
		t.Fatalf("no parts: %+v", s)
	}
}

func TestFieldOptionsNeedRulePermission(t *testing.T) {
	f := newFixture(t)
	got, err := f.appr.FieldOptions(f.as("rules"), "test.doc", "parts", "bolt", nil)
	f.check(err)
	if !reflect.DeepEqual(got, []record.FieldOption{{ID: 17, Code: "P-17", Name: "part bolt"}}) {
		t.Fatalf("search %+v", got)
	}
	got, err = f.appr.FieldOptions(f.as("rules"), "test.doc", "parts", "", []int64{3, 17})
	f.check(err)
	if !reflect.DeepEqual(got, []record.FieldOption{{ID: 3, Code: "P-3"}, {ID: 17, Code: "P-17"}}) {
		t.Fatalf("labels %+v", got)
	}
	if got, err := f.appr.FieldOptions(f.as("rules"), "test.doc", "parts", "none", nil); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("no match: %#v %v", got, err)
	}
	if _, err := f.appr.FieldOptions(platform.WithProducts(f.as("rules"), nil), "test.doc", "parts", "bolt", nil); err != nil {
		t.Fatalf("disabled product: %v", err)
	}
	if _, err := f.appr.FieldOptions(f.as("outsider"), "test.doc", "parts", "bolt", nil); code(err) != "forbidden" {
		t.Fatalf("outsider: %v", err)
	}
	for _, c := range [][2]string{{"test.doc", "days"}, {"test.doc", "nope"}, {"test.nope", "parts"}} {
		if _, err := f.appr.FieldOptions(f.as("rules"), c[0], c[1], "", nil); code(err) != "not_found" {
			t.Errorf("%v: %v", c, err)
		}
	}
}
