package approval_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/notification"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

// day is today in the tenant time zone plus n days, as YYYY-MM-DD.
func (f *fixture) day(n int) string {
	f.t.Helper()
	today, err := f.core.Setting.Today(f.admin)
	f.check(err)
	return today.Time.AddDate(0, 0, n).Format(time.DateOnly)
}

func (f *fixture) delegation(user, delegate string, from, to int) approval.DelegationInput {
	u := f.users[user]
	return approval.DelegationInput{UserID: &u, DelegateID: f.users[delegate], StartsOn: f.day(from), EndsOn: f.day(to)}
}

func reasonOf(err error) any {
	if e, ok := errors.AsType[*platform.Error](err); ok && e.Code == "invalid_delegation" {
		return e.Params["reason"]
	}
	return nil
}

func TestAnyUserListsDelegateUsers(t *testing.T) {
	f := newFixture(t)
	users, err := f.appr.DelegateUsers(f.as("outsider"))
	f.check(err)
	// Ordinary users get names, not logins, which are sign-in identifiers.
	if !slices.ContainsFunc(users, func(u approval.ApproverUser) bool { return u.ID == f.users["boss"] && u.Name == "boss" }) ||
		slices.ContainsFunc(users, func(u approval.ApproverUser) bool { return u.Login != "" }) {
		t.Fatalf("outsider users %+v", users)
	}
	users, err = f.appr.DelegateUsers(f.admin)
	f.check(err)
	if !slices.ContainsFunc(users, func(u approval.ApproverUser) bool { return u.ID == f.users["boss"] && u.Login == "boss" }) {
		t.Fatalf("admin users %+v", users)
	}
}

func TestCreateOwnDelegation(t *testing.T) {
	f := newFixture(t)
	id, err := f.appr.CreateDelegation(f.as("boss"), approval.DelegationInput{DelegateID: f.users["big_boss"], StartsOn: f.day(0), EndsOn: f.day(7)})
	f.check(err)
	list, err := f.appr.Delegations(f.as("boss"))
	f.check(err)
	if len(list) != 1 {
		t.Fatalf("delegations %+v, want one", list)
	}
	d := list[0]
	if d.ID != id || d.UserID != f.users["boss"] || d.UserName != "boss" || d.DelegateID != f.users["big_boss"] || d.DelegateName != "big_boss" ||
		d.StartsOn != f.day(0) || d.EndsOn != f.day(7) || !d.Active || !slices.Equal(d.AllowedActions, []string{"delete"}) {
		t.Fatalf("delegation %+v", d)
	}
	theirs, err := f.appr.Delegations(f.as("big_boss"))
	f.check(err)
	if len(theirs) != 0 {
		t.Fatalf("the delegate sees %+v, want nothing", theirs)
	}
	all, err := f.appr.Delegations(f.admin)
	f.check(err)
	if len(all) != 1 || !slices.Equal(all[0].AllowedActions, []string{"delete"}) {
		t.Fatalf("admin sees %+v, want boss's", all)
	}
}

func TestDelegationRules(t *testing.T) {
	f := newFixture(t)
	missing := int64(1 << 40)
	for name, tc := range map[string]struct {
		in     approval.DelegationInput
		reason string
		code   string
	}{
		"self":             {in: f.delegation("boss", "boss", 0, 1), reason: "self"},
		"unknown delegate": {in: approval.DelegationInput{UserID: new(f.users["boss"]), DelegateID: missing, StartsOn: f.day(0), EndsOn: f.day(1)}, code: "not_found"},
		"dates":            {in: f.delegation("boss", "big_boss", 2, 1), reason: "dates"},
		"past":             {in: f.delegation("boss", "big_boss", -3, -1), reason: "past"},
	} {
		_, err := f.appr.CreateDelegation(f.as("boss"), tc.in)
		if tc.reason != "" && reasonOf(err) != tc.reason || tc.code != "" && code(err) != tc.code {
			t.Errorf("%s: got %v", name, err)
		}
	}
	_, err := f.appr.CreateDelegation(f.admin, approval.DelegationInput{UserID: &missing, DelegateID: f.users["boss"], StartsOn: f.day(0), EndsOn: f.day(1)})
	if code(err) != "not_found" {
		t.Errorf("unknown user: got %v", err)
	}
	if _, err := f.appr.CreateDelegation(f.as("hr"), f.delegation("boss", "big_boss", 0, 1)); code(err) != "forbidden" {
		t.Errorf("hr for boss: got %v", err)
	}
	// Ended yesterday is past; ending today is still allowed.
	if _, err := f.appr.CreateDelegation(f.as("boss"), f.delegation("boss", "big_boss", -2, 0)); err != nil {
		t.Errorf("ending today: %v", err)
	}
	if _, err := f.appr.CreateDelegation(f.admin, f.delegation("hr", "big_boss", 0, 1)); err != nil {
		t.Errorf("admin for hr: %v", err)
	}
}

func TestDelegationsDoNotOverlap(t *testing.T) {
	f := newFixture(t)
	first, err := f.appr.CreateDelegation(f.as("boss"), f.delegation("boss", "big_boss", 0, 5))
	f.check(err)
	_, err = f.appr.CreateDelegation(f.as("boss"), f.delegation("boss", "c1", 5, 9))
	if e, ok := errors.AsType[*platform.Error](err); !ok || e.Params["reason"] != "overlap" || e.Params["id"] != first {
		t.Fatalf("overlap by one day: got %v", err)
	}
	if _, err := f.appr.CreateDelegation(f.as("boss"), f.delegation("boss", "c1", 6, 9)); err != nil {
		t.Fatalf("adjacent: %v", err)
	}
	// Another user's delegation never overlaps boss's.
	if _, err := f.appr.CreateDelegation(f.as("hr"), f.delegation("hr", "big_boss", 0, 5)); err != nil {
		t.Fatalf("hr's own: %v", err)
	}

	// Two creates at once: the second waits on the user's lock, then sees the first.
	var second error
	var wg sync.WaitGroup
	f.check(platform.InTx(f.as("c2"), func(ctx context.Context) error {
		if _, err := f.appr.CreateDelegation(ctx, f.delegation("c2", "c3", 1, 3)); err != nil {
			return err
		}
		wg.Go(func() { _, second = f.appr.CreateDelegation(f.as("c2"), f.delegation("c2", "c1", 2, 4)) })
		pgtest.AwaitLockWait(t, f.core.Pool)
		return nil
	}))
	wg.Wait()
	if reasonOf(second) != "overlap" {
		t.Fatalf("concurrent create: got %v, want overlap", second)
	}
}

func TestDeleteDelegation(t *testing.T) {
	f := newFixture(t)
	id, err := f.appr.CreateDelegation(f.as("boss"), f.delegation("boss", "big_boss", 0, 1))
	f.check(err)
	for _, who := range []string{"big_boss", "hr"} {
		if err := f.appr.DeleteDelegation(f.as(who), id); code(err) != "not_found" {
			t.Fatalf("%s deletes: got %v", who, err)
		}
	}
	f.check(f.appr.DeleteDelegation(f.as("boss"), id))
	list, err := f.appr.Delegations(f.as("boss"))
	f.check(err)
	if len(list) != 0 {
		t.Fatalf("delegations %+v after delete", list)
	}
	other, err := f.appr.CreateDelegation(f.as("boss"), f.delegation("boss", "big_boss", 0, 1))
	f.check(err)
	f.check(f.appr.DeleteDelegation(f.admin, other))
}

func TestDelegationsMeta(t *testing.T) {
	f := newFixture(t)
	for login, want := range map[string]bool{"boss": false, "rules": false} {
		got, err := f.appr.CanManageAllDelegations(f.as(login))
		f.check(err)
		if got != want {
			t.Errorf("%s: manage_all %v", login, got)
		}
	}
	if got, err := f.appr.CanManageAllDelegations(f.admin); err != nil || !got {
		t.Errorf("admin: %v %v", got, err)
	}
}

// delegate records user → delegate from today+from to today+to, as user.
func (f *fixture) delegate(user, delegate string, from, to int) {
	f.t.Helper()
	_, err := f.appr.CreateDelegation(f.as(user), f.delegation(user, delegate, from, to))
	f.check(err)
}

func TestDelegateApproves(t *testing.T) {
	f := newFixture(t)
	f.rule(f.user("boss"))
	f.delegate("boss", "big_boss", 0, 3)
	d, err := f.send("1", "a")
	f.check(err)
	items := f.inbox("big_boss")
	if len(items) != 1 || items[0].OnBehalfOfName == nil || *items[0].OnBehalfOfName != "boss" {
		t.Fatalf("big_boss inbox %+v", items)
	}
	if acts := f.panel("big_boss", d).Steps[0].AllowedActions; !slices.Equal(acts, []string{"approve", "reject"}) {
		t.Fatalf("big_boss actions %v", acts)
	}
	f.check(f.appr.Approve(f.as("big_boss"), *d.ApprovalTicket, 1))
	if st := f.status(d); st != record.Posted {
		t.Fatalf("status %s", st)
	}
	v := f.panel("submitter", d).Steps[0].Voters
	if len(v) != 1 || v[0].Name != "boss" || v[0].Decision == nil || *v[0].Decision != "approved" || v[0].CastByName == nil || *v[0].CastByName != "big_boss" {
		t.Fatalf("voters %+v", v)
	}
}

func TestDelegationDeclaredWhileWaiting(t *testing.T) {
	f := newFixture(t)
	f.rule(f.user("boss"))
	d, err := f.send("1", "a")
	f.check(err)
	f.delegate("boss", "big_boss", 0, 0)
	f.check(f.appr.Approve(f.as("big_boss"), *d.ApprovalTicket, 1))
	if st := f.status(d); st != record.Posted {
		t.Fatalf("status %s", st)
	}
}

func TestDelegateVotesOncePerStep(t *testing.T) {
	f := newFixture(t)
	f.rule(f.by(2, 1, "c1", "c2"))
	f.delegate("c2", "c1", 0, 1)
	d, err := f.send("1", "a")
	f.check(err)
	if items := f.inbox("c1"); len(items) != 1 || items[0].OnBehalfOfName != nil {
		t.Fatalf("c1 inbox %+v, want their own seat", items)
	}
	f.check(f.appr.Approve(f.as("c1"), *d.ApprovalTicket, 1))
	if code(f.appr.Approve(f.as("c1"), *d.ApprovalTicket, 1)) != "approval_already_voted" {
		t.Fatal("c1 voted twice")
	}
	if s := f.panel("c1", d).Steps[0]; s.Status != "open" || len(s.AllowedActions) != 0 {
		t.Fatalf("step %+v", s)
	}
	if items := f.inbox("c1"); len(items) != 0 {
		t.Fatalf("c1 inbox after voting %+v", items)
	}
	f.check(f.appr.Approve(f.as("c2"), *d.ApprovalTicket, 1))
	if st := f.status(d); st != record.Posted {
		t.Fatalf("status %s", st)
	}
}

func TestDelegateWhoSubmittedCannotVote(t *testing.T) {
	f := newFixture(t)
	f.rule(f.user("boss"))
	f.delegate("boss", "submitter", 0, 1)
	d, err := f.send("1", "a")
	f.check(err)
	if acts := f.panel("submitter", d).Steps[0].AllowedActions; len(acts) != 0 {
		t.Fatalf("submitter actions %v", acts)
	}
	if items := f.inbox("submitter"); len(items) != 0 {
		t.Fatalf("submitter inbox %+v", items)
	}
	if code(f.appr.Approve(f.as("submitter"), *d.ApprovalTicket, 1)) != "self_approval" {
		t.Fatal("the submitter approved as a delegate")
	}
	f.check(f.appr.Approve(f.as("boss"), *d.ApprovalTicket, 1))
}

func TestDelegateWhoCannotViewCannotVote(t *testing.T) {
	f := newFixture(t)
	f.rule(f.user("boss"))
	f.delegate("boss", "outsider", 0, 1)
	d, err := f.send("1", "a")
	f.check(err)
	if _, err := f.appr.DocumentInstance(f.as("outsider"), d.Ref); code(err) != "not_found" {
		t.Fatalf("outsider panel: %v", err)
	}
	if items := f.inbox("outsider"); len(items) != 0 {
		t.Fatalf("outsider inbox %+v", items)
	}
	if code(f.appr.Approve(f.as("outsider"), *d.ApprovalTicket, 1)) != "not_found" {
		t.Fatal("outsider approved")
	}
}

func TestDelegationOnlyWhileActive(t *testing.T) {
	f := newFixture(t)
	f.rule(f.user("boss"))
	// Ended yesterday: past ones can't be created, so written directly.
	_, err := f.core.System.Exec(f.admin, `INSERT INTO approval.delegations (user_id, delegate_id, starts_on, ends_on, created_by)
		VALUES ($1, $2, $3, $4, $1)`, f.users["boss"], f.users["big_boss"], f.day(-3), f.day(-1))
	f.check(err)
	f.delegate("boss", "big_boss", 1, 3)
	d, err := f.send("1", "a")
	f.check(err)
	if items := f.inbox("big_boss"); len(items) != 0 {
		t.Fatalf("big_boss inbox %+v", items)
	}
	if acts := f.panel("big_boss", d).Steps[0].AllowedActions; len(acts) != 0 {
		t.Fatalf("big_boss actions %v", acts)
	}
	if code(f.appr.Approve(f.as("big_boss"), *d.ApprovalTicket, 1)) != "forbidden" {
		t.Fatal("an inactive delegation voted")
	}
}

func TestDelegationIsOneHop(t *testing.T) {
	f := newFixture(t)
	f.rule(f.user("boss"))
	f.delegate("boss", "big_boss", 0, 1)
	f.delegate("big_boss", "hr", 0, 1)
	d, err := f.send("1", "a")
	f.check(err)
	if items := f.inbox("hr"); len(items) != 0 {
		t.Fatalf("hr inbox %+v", items)
	}
	if code(f.appr.Approve(f.as("hr"), *d.ApprovalTicket, 1)) != "forbidden" {
		t.Fatal("hr voted through two delegations")
	}
}

func TestDelegateIsNotified(t *testing.T) {
	f := newFixture(t)
	f.rule(f.user("boss"))
	f.delegate("boss", "big_boss", 0, 1)
	f.delegate("big_boss", "hr", 0, 1) // not followed
	d, err := f.send("1", "a")
	f.check(err)
	requested := func(login string) bool {
		page, err := f.core.Notification.List(f.as(login), nil)
		f.check(err)
		return slices.ContainsFunc(page.Items, func(n notification.Notification) bool {
			return n.Kind == "approval_requested" && n.RecordID != nil && *n.RecordID == d.ID
		})
	}
	if !requested("big_boss") {
		t.Fatal("the delegate was not told")
	}
	if requested("hr") {
		t.Fatal("the delegate's delegate was told")
	}
}

func TestReassignToOneWhoVotedAsDelegate(t *testing.T) {
	f := newFixture(t)
	f.rule(f.by(2, 1, "c1", "c2"))
	f.delegate("c1", "big_boss", 0, 1)
	d, err := f.send("1", "a")
	f.check(err)
	f.check(f.appr.Approve(f.as("big_boss"), *d.ApprovalTicket, 1))
	if code(f.appr.Reassign(f.as("hr"), *d.ApprovalTicket, 1, f.users["c2"], "big_boss")) != "approval_already_voted" {
		t.Fatal("reassigned a seat to one who already voted in the step")
	}
}

func TestReassignTellsTheDelegate(t *testing.T) {
	f := newFixture(t)
	f.rule(f.user("boss"))
	d, err := f.send("1", "a")
	f.check(err)
	f.delegate("c1", "c2", 0, 1)
	f.check(f.appr.Reassign(f.as("hr"), *d.ApprovalTicket, 1, f.users["boss"], "c1"))
	page, err := f.core.Notification.List(f.as("c2"), nil)
	f.check(err)
	if !slices.ContainsFunc(page.Items, func(n notification.Notification) bool {
		return n.Kind == "approval_requested" && n.RecordID != nil && *n.RecordID == d.ID
	}) {
		t.Fatal("the new voter's delegate was not told")
	}
}

func TestManagerCannotDelegateToThemself(t *testing.T) {
	f := newFixture(t)
	admin, err := f.ids.UserIDByLogin(f.admin, "admin")
	f.check(err)
	in := f.delegation("boss", "boss", 0, 1)
	in.DelegateID = admin
	if _, err := f.appr.CreateDelegation(f.admin, in); reasonOf(err) != "creator" {
		t.Fatalf("admin as boss's delegate: got %v", err)
	}
	if _, err := f.appr.CreateDelegation(f.admin, f.delegation("boss", "big_boss", 0, 1)); err != nil {
		t.Fatalf("admin for boss: %v", err)
	}
}

func TestDelegatedSeatWithoutViewIsSkipped(t *testing.T) {
	f := newFixture(t)
	f.rule(f.user("boss"))
	f.delegate("boss", "big_boss", 0, 1)
	d, err := f.send("1", "a")
	f.check(err)
	f.denied[f.users["boss"]] = record.View // boss loses view after the step has started
	if acts := f.panel("big_boss", d).Steps[0].AllowedActions; len(acts) != 0 {
		t.Fatalf("big_boss actions %v", acts)
	}
	if code(f.appr.Approve(f.as("big_boss"), *d.ApprovalTicket, 1)) != "forbidden" {
		t.Fatal("voted a seat whose holder could no longer view the document")
	}
}

func TestDelegateSkipsAnExcludedSeat(t *testing.T) {
	f := newFixture(t)
	f.rule(f.by(2, 1, "c1", "c2"))
	f.delegate("c1", "big_boss", 0, 1)
	f.delegate("c2", "big_boss", 0, 1)
	d, err := f.send("1", "a")
	f.check(err)
	f.subjects = []int64{f.users["c1"]} // c1's seat is now excluded; c2's still isn't
	f.check(f.appr.Approve(f.as("big_boss"), *d.ApprovalTicket, 1))
	v := f.panel("submitter", d).Steps[0].Voters
	i := slices.IndexFunc(v, func(x approval.StepVoter) bool { return x.Name == "c2" })
	if i < 0 || v[i].Decision == nil || *v[i].Decision != "approved" || v[i].CastByName == nil || *v[i].CastByName != "big_boss" {
		t.Fatalf("voters %+v, want the vote on c2's seat", v)
	}
}

func TestExcludedSeatCannotBeDelegated(t *testing.T) {
	f := newFixture(t)
	f.rule(f.user("boss"))
	f.delegate("boss", "big_boss", 0, 1)
	d, err := f.send("1", "a")
	f.check(err)
	f.subjects = []int64{f.users["boss"]} // the document is now about boss
	if acts := f.panel("big_boss", d).Steps[0].AllowedActions; len(acts) != 0 {
		t.Fatalf("big_boss actions %v", acts)
	}
	if items := f.inbox("big_boss"); len(items) != 0 {
		t.Fatalf("big_boss inbox %+v", items)
	}
	if code(f.appr.Approve(f.as("big_boss"), *d.ApprovalTicket, 1)) != "self_approval" {
		t.Fatal("voted the seat of the one the document is about")
	}
}
