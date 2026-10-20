package approval_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/taoworklabs/mmerp/internal/core/approval"
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
