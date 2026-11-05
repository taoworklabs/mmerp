package accounting_test

import (
	"context"
	"errors"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/coretest"
	"github.com/taoworklabs/mmerp/internal/modules/accounting"
	"github.com/taoworklabs/mmerp/internal/platform"
)

func newAccounting(t *testing.T) (*coretest.Core, *accounting.Service) {
	t.Helper()
	core := coretest.New(t, "accounting")
	return core, accounting.NewService(accounting.Deps{IAM: core.IAM, Audit: core.Audit, Record: core.Record})
}

func acct(code, typ string, parent *int64) accounting.AccountFields {
	return accounting.AccountFields{Code: code, Name: "Account " + code, Type: typ, ParentID: parent, Active: true}
}

func mustCreate(t *testing.T, s *accounting.Service, ctx context.Context, f accounting.AccountFields) int64 {
	t.Helper()
	id, err := s.CreateAccount(ctx, f)
	if err != nil {
		t.Fatalf("create %s: %v", f.Code, err)
	}
	return id
}

func TestAccountCreateAndList(t *testing.T) {
	core, s := newAccounting(t)
	mustCreate(t, s, core.Admin, acct("642", "expense", nil))
	mustCreate(t, s, core.Admin, acct("334", "liability", nil))
	l, err := s.Accounts(core.Admin)
	if err != nil || len(l) != 2 || l[0].Code != "334" || l[1].Code != "642" || !l[0].Postable || !l[1].Postable {
		t.Fatalf("list %+v %v", l, err)
	}
}

func TestAccountParentMakesParentNotPostable(t *testing.T) {
	core, s := newAccounting(t)
	p := mustCreate(t, s, core.Admin, acct("333", "liability", nil))
	c := mustCreate(t, s, core.Admin, acct("3335", "liability", &p))
	l, _ := s.Accounts(core.Admin)
	if l[0].Postable || !l[1].Postable {
		t.Fatalf("postable %+v", l)
	}
	if err := s.DeleteAccount(core.Admin, p); !errors.Is(err, accounting.ErrAccountInUse) {
		t.Fatalf("delete parent: %v", err)
	}
	if err := s.DeleteAccount(core.Admin, c); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteAccount(core.Admin, p); err != nil {
		t.Fatal(err)
	}
}

func TestAccountParentRules(t *testing.T) {
	core, s := newAccounting(t)
	p := mustCreate(t, s, core.Admin, acct("333", "liability", nil))
	c := mustCreate(t, s, core.Admin, acct("3335", "liability", &p))
	e := mustCreate(t, s, core.Admin, acct("642", "expense", nil))
	missing := int64(9999)
	for name, err := range map[string]error{
		"other type": func() error { _, err := s.CreateAccount(core.Admin, acct("111", "asset", &p)); return err }(),
		"missing":    func() error { _, err := s.CreateAccount(core.Admin, acct("112", "liability", &missing)); return err }(),
		"cycle":      s.UpdateAccount(core.Admin, p, acct("333", "liability", &c)),
		"itself":     s.UpdateAccount(core.Admin, e, acct("642", "expense", &e)),
	} {
		if !errors.Is(err, accounting.ErrAccountParentInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	// A type change under children would orphan their type rule.
	if err := s.UpdateAccount(core.Admin, p, acct("333", "asset", nil)); !errors.Is(err, accounting.ErrAccountInUse) {
		t.Errorf("type change with children: %v", err)
	}
	ok, err := accounting.PostableActiveForTest(core.Admin, []int64{p, c, e, 9999})
	if err != nil || ok[p] || !ok[c] || !ok[e] || ok[9999] {
		t.Fatalf("postable %v %v", ok, err)
	}
}

func TestAccountCodeTaken(t *testing.T) {
	core, s := newAccounting(t)
	mustCreate(t, s, core.Admin, acct("642", "expense", nil))
	if _, err := s.CreateAccount(core.Admin, acct("642", "expense", nil)); !errors.Is(err, accounting.ErrAccountCodeTaken) {
		t.Fatalf("got %v", err)
	}
}

func TestAccountNeedsManage(t *testing.T) {
	core, s := newAccounting(t)
	id, err := core.IAM.CreateUser(core.Admin, "acc", "Acc", "long enough")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := core.IAM.GrantRole(core.Admin, id, "accounting", "accountant", nil); err != nil {
		t.Fatal(err)
	}
	ctx := platform.WithActor(core.Ctx, id)
	if _, err := s.CreateAccount(ctx, acct("642", "expense", nil)); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.Accounts(ctx); err != nil {
		t.Fatalf("accountant lists: %v", err)
	}
}

func TestBlankAccountNameRefused(t *testing.T) {
	core, s := newAccounting(t)
	f := acct("642", "expense", nil)
	f.Name = "   "
	if _, err := s.CreateAccount(core.Admin, f); !errors.Is(err, accounting.ErrAccountNameRequired) {
		t.Fatalf("create: %v", err)
	}
	id := mustCreate(t, s, core.Admin, acct("642", "expense", nil))
	if err := s.UpdateAccount(core.Admin, id, f); !errors.Is(err, accounting.ErrAccountNameRequired) {
		t.Fatalf("update: %v", err)
	}
}
