package approval_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/notification"
	"github.com/taoworklabs/mmerp/internal/core/numbering"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/setting"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

type fixture struct {
	t        *testing.T
	ids      *iam.Service
	rec      *record.Service
	appr     *approval.Service
	admin    context.Context
	unit     int64
	users    map[string]int64
	as       func(login string) context.Context
	managers map[int][]int64 // level → what the type's Approvers returns
}

// newFixture registers test.doc with fields days and kind, module approvers from f.managers,
// a type other.doc of product other, and users submitter, boss, big_boss, hr (role test.hr
// at the unit), rules (test.approval_admin) and outsider.
func newFixture(t *testing.T) *fixture {
	pool := pgtest.New(t)
	ctx := platform.WithKeyring(platform.WithProducts(platform.WithDB(t.Context(), pool), []string{"test", "other"}), pgtest.Keyring())
	f := &fixture{t: t, users: map[string]int64{}, managers: map[int][]int64{}}
	f.ids = iam.NewService(iam.Deps{Setting: setting.NewService(), Audit: audit.NewService()})
	f.ids.RegisterRoles("test", map[string][]string{"hr": {"test.doc.view"}, "approval_admin": {"test.approval.manage"}}, "approval_admin")
	f.rec = record.NewService(record.Deps{IAM: f.ids, Numbering: numbering.NewService(), Audit: audit.NewService()})
	f.appr = approval.NewService(approval.Deps{IAM: f.ids, Record: f.rec, Audit: audit.NewService(), Notification: notification.NewService(notification.Deps{Record: f.rec, IAM: f.ids, Audit: audit.NewService(), Setting: setting.NewService()})})
	f.rec.SetApprovalGate(f.appr)
	f.rec.Register(record.Type{
		Code: "test.doc", Product: "test", Kind: record.Document, NumberPrefix: "T",
		Can: func(ctx context.Context, _ int64, _ record.Action) (bool, error) {
			id, _ := platform.ActorFrom(ctx)
			return id != f.users["outsider"], nil
		},
		Fields: []record.Field{{Key: "days", Kind: record.Number}, {Key: "kind", Kind: record.Choice,
			Options: func(context.Context) ([]record.Option, error) {
				return []record.Option{{Value: "a"}, {Value: "b"}}, nil
			}}},
		Approvers: func(_ context.Context, _ record.Ref, level int) ([]int64, error) { return f.managers[level], nil },
	})
	// A type of another product, for rules a product's administrator may not touch.
	f.rec.Register(record.Type{Code: "other.doc", Product: "other", Kind: record.Document, NumberPrefix: "O",
		Can: func(context.Context, int64, record.Action) (bool, error) { return true, nil }})
	var err error
	adminID, err := f.ids.CreateAdmin(ctx, "admin", "Admin", "long enough")
	f.check(err)
	f.admin = platform.WithActor(ctx, adminID)
	c, err := f.ids.CreateOrgUnit(f.admin, iam.OrgUnitInput{Kind: "company", Name: "C"})
	f.check(err)
	f.unit, err = f.ids.CreateOrgUnit(f.admin, iam.OrgUnitInput{ParentID: &c, Kind: "department", Name: "A"})
	f.check(err)
	for _, login := range []string{"submitter", "boss", "big_boss", "hr", "rules", "outsider"} {
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

func (f *fixture) rule(steps ...approval.Step) {
	f.t.Helper()
	f.check(f.appr.SaveRule(f.admin, "test.doc", approval.RuleInput{Steps: steps, MaxLevels: 3, FallbackProduct: "test", FallbackRole: "hr"}))
}

// send creates a draft with days and kind as the submitter and sends it.
func (f *fixture) send(days, kind string) (record.Doc, error) {
	f.t.Helper()
	d, err := f.rec.Create(f.as("submitter"), "test.doc", record.Header{Date: "2026-03-10", OrgUnitID: f.unit, Fields: map[string]string{"days": days, "kind": kind}})
	f.check(err)
	if err := f.rec.Transition(f.as("submitter"), d.Ref, 1, record.Posted); err != nil {
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

var module = approval.Step{Approver: approval.Approver{Kind: "module"}}

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
	if !slices.Equal(inst.Steps[0].ApproverNames, []string{"boss"}) || inst.Steps[0].Fallback {
		t.Fatalf("want boss: %+v", inst.Steps[0])
	}
	if !slices.Equal(inst.AllowedActions, []string{"approve", "reject"}) {
		t.Fatalf("boss actions %v", inst.AllowedActions)
	}
	if acts := f.panel("big_boss", d).AllowedActions; len(acts) != 0 {
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
	if !slices.Equal(inst.Steps[0].ApproverNames, []string{"hr"}) || !inst.Steps[0].Fallback {
		t.Fatalf("want hr as fallback: %+v", inst.Steps[0])
	}
	if !slices.Equal(inst.AllowedActions, []string{"approve", "reject", "reassign"}) {
		t.Fatalf("hr actions %v", inst.AllowedActions)
	}
}

func TestNoApproverAtAllRefusesToSend(t *testing.T) {
	f := newFixture(t)
	f.rule(approval.Step{Approver: approval.Approver{Kind: "user", UserID: new(f.users["submitter"])}})
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
	boss := approval.Step{Approver: approval.Approver{Kind: "user", UserID: new(f.users["boss"])}}
	long := approval.Step{Condition: &approval.Condition{Field: "days", Op: "gt", Value: "3"},
		Approver: approval.Approver{Kind: "user", UserID: new(f.users["big_boss"])}}
	f.rule(boss, long)

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
	if inst.CurrentStep != 2 || !slices.Equal(inst.AllowedActions, []string{"approve", "reject"}) {
		t.Fatalf("want step 2 for big_boss: %+v", inst)
	}
	f.check(f.appr.Approve(f.as("big_boss"), *d.ApprovalTicket, 2))
	if d, _ := f.rec.Get(f.admin, d.Ref); d.Status != record.Posted {
		t.Fatalf("status %s", d.Status)
	}
}

func TestRejectReturnsToDraft(t *testing.T) {
	f := newFixture(t)
	f.rule(approval.Step{Approver: approval.Approver{Kind: "user", UserID: new(f.users["boss"])}})
	d, err := f.send("1", "a")
	f.check(err)
	f.check(f.appr.Reject(f.as("boss"), *d.ApprovalTicket, 1, "too long"))
	after, _ := f.rec.Get(f.admin, d.Ref)
	if after.Status != record.Draft || after.ApprovalTicket != nil || after.Version != d.Version+1 {
		t.Fatalf("after reject: %+v", after)
	}
	inst := f.panel("submitter", d)
	if inst.Status != "rejected" || *inst.Steps[0].Reason != "too long" {
		t.Fatalf("instance %+v", inst)
	}
}

func TestReassignByTheFallbackRole(t *testing.T) {
	f := newFixture(t)
	f.rule(approval.Step{Approver: approval.Approver{Kind: "user", UserID: new(f.users["boss"])}})
	d, err := f.send("1", "a")
	f.check(err)
	if code(f.appr.Reassign(f.as("boss"), *d.ApprovalTicket, 1, "big_boss")) != "forbidden" {
		t.Fatal("boss reassigned without the fallback role")
	}
	if code(f.appr.Reassign(f.as("hr"), *d.ApprovalTicket, 1, "submitter")) != "self_approval" {
		t.Fatal("reassigned to the submitter")
	}
	f.check(f.appr.Reassign(f.as("hr"), *d.ApprovalTicket, 1, "big_boss"))
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
	f.rule(approval.Step{Approver: approval.Approver{Kind: "user", UserID: new(f.users["outsider"])}})
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
	f.rule(approval.Step{Approver: approval.Approver{Kind: "user", UserID: new(f.users["outsider"])}})
	d, err := f.send("1", "a")
	f.check(err)
	if s := f.panel("hr", d).Steps[0]; !slices.Equal(s.ApproverNames, []string{"hr"}) || !s.Fallback {
		t.Fatalf("want the fallback role instead of outsider: %+v", s)
	}
}

func TestRuleChecksConditionsAgainstFields(t *testing.T) {
	f := newFixture(t)
	bad := approval.Step{Condition: &approval.Condition{Field: "kind", Op: "gt", Value: "1"}, Approver: approval.Approver{Kind: "module"}}
	err := f.appr.SaveRule(f.admin, "test.doc", approval.RuleInput{Steps: []approval.Step{bad}, MaxLevels: 3, FallbackProduct: "test", FallbackRole: "hr"})
	if code(err) != "invalid_rule" {
		t.Fatalf("got %v", err)
	}
	if code(f.appr.SaveRule(f.as("hr"), "test.doc", approval.RuleInput{})) != "forbidden" {
		t.Fatal("hr saved a rule")
	}
}

func docTypes(rules []approval.TypeRule) []string {
	var out []string
	for _, r := range rules {
		out = append(out, r.DocType)
	}
	return out
}

func TestRulesByProduct(t *testing.T) {
	f := newFixture(t)
	in := approval.RuleInput{Steps: []approval.Step{module}, MaxLevels: 3, FallbackProduct: "test", FallbackRole: "hr"}

	// A product's administrator manages only that product's types.
	rules, err := f.appr.Rules(f.as("rules"), "")
	f.check(err)
	if got := docTypes(rules); !slices.Equal(got, []string{"test.doc"}) {
		t.Fatalf("product admin lists %v", got)
	}
	if !slices.Equal(rules[0].AllowedActions, []string{"save", "delete"}) {
		t.Fatalf("product admin actions %v", rules[0].AllowedActions)
	}
	f.check(f.appr.SaveRule(f.as("rules"), "test.doc", in))
	f.check(f.appr.DeleteRule(f.as("rules"), "test.doc"))
	if _, err := f.appr.Users(f.as("rules")); err != nil {
		t.Fatalf("product admin lists users: %v", err)
	}
	if code(f.appr.SaveRule(f.as("rules"), "other.doc", in)) != "forbidden" {
		t.Fatal("product admin saved another product's rule")
	}
	if code(f.appr.DeleteRule(f.as("rules"), "other.doc")) != "forbidden" {
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
	byBoss := approval.Step{Approver: approval.Approver{Kind: "user", UserID: new(f.users["boss"])}}
	f.check(f.appr.SaveRule(f.admin, "other.doc", approval.RuleInput{Steps: []approval.Step{byBoss}, MaxLevels: 3, FallbackProduct: "test", FallbackRole: "hr"}))

	// Without any rule permission, everything is refused.
	if _, err := f.appr.Rules(f.as("outsider"), ""); code(err) != "forbidden" {
		t.Fatalf("outsider lists rules: %v", err)
	}
	if _, err := f.appr.Users(f.as("outsider")); code(err) != "forbidden" {
		t.Fatalf("outsider lists users: %v", err)
	}
	if code(f.appr.SaveRule(f.as("outsider"), "test.doc", in)) != "forbidden" || code(f.appr.DeleteRule(f.as("outsider"), "test.doc")) != "forbidden" {
		t.Fatal("outsider changed a rule")
	}

	// A disabled product stays readable, without actions, and refuses writes.
	off := platform.WithProducts(f.as("rules"), []string{"other"})
	rules, err = f.appr.Rules(off, "")
	f.check(err)
	if len(rules) != 1 || len(rules[0].AllowedActions) != 0 {
		t.Fatalf("disabled product rules %+v", rules)
	}
	if code(f.appr.SaveRule(off, "test.doc", in)) != "product_not_enabled" || code(f.appr.DeleteRule(off, "test.doc")) != "product_not_enabled" {
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

	f.check(f.appr.SaveRule(f.as("rules"), "test.doc", approval.RuleInput{
		Steps: []approval.Step{{Approver: approval.Approver{Kind: "user", UserID: new(f.users["big_boss"])}}}, MaxLevels: 3, FallbackProduct: "test", FallbackRole: "hr",
	}))
	if got := f.panel("boss", d).Steps; len(got) != len(steps) || !slices.Equal(got[0].ApproverNames, steps[0].ApproverNames) {
		t.Fatalf("after save: %+v, want %+v", got, steps)
	}
	f.check(f.appr.DeleteRule(f.as("rules"), "test.doc"))
	if got := f.panel("boss", d).Steps; len(got) != len(steps) || !slices.Equal(got[0].ApproverNames, steps[0].ApproverNames) {
		t.Fatalf("after delete: %+v, want %+v", got, steps)
	}
	f.check(f.appr.Approve(f.as("boss"), *d.ApprovalTicket, 1))
}
