package accounting_test

import (
	"errors"
	"os"
	"testing"

	"github.com/taoworklabs/mmerp/internal/modules/accounting"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

// testReceivable is a kind whose lines name a partner; no product declares one yet.
const testReceivable posting.Kind = "test_receivable"

func TestMain(m *testing.M) {
	posting.DeclareKind(testReceivable, true)
	os.Exit(m.Run())
}

func tracked(code, typ string, parent *int64) accounting.AccountFields {
	f := acct(code, typ, parent)
	f.ByPartner = true
	return f
}

func TestRuleMustMatchPartnerTracking(t *testing.T) {
	core, s := newAccounting(t)
	ar := mustCreate(t, s, core.Admin, tracked("131", "asset", nil))
	exp := mustCreate(t, s, core.Admin, acct("642", "expense", nil))
	if err := s.SaveRule(core.Admin, string(posting.SalaryExpense), ar, "debit"); !errors.Is(err, accounting.ErrRulePartnerMismatch) {
		t.Fatalf("payroll kind to tracked account: %v", err)
	}
	if err := s.SaveRule(core.Admin, string(testReceivable), exp, "debit"); !errors.Is(err, accounting.ErrRulePartnerMismatch) {
		t.Fatalf("partner kind to untracked account: %v", err)
	}
	if err := s.SaveRule(core.Admin, string(testReceivable), ar, "debit"); err != nil {
		t.Fatal(err)
	}
}

func TestAccountFlagLockedByLinesAndRules(t *testing.T) {
	l := newLedger(t)
	s, ctx := l.s, l.core.Admin
	// 334 gets lines from a payroll.
	ref := l.doc(t, "2026-03-31")
	l.record(t, ref, l.payroll()...)
	l.reconcile(t, ref)
	if err := s.UpdateAccount(ctx, l.acc["334"], tracked("334", "liability", nil)); !errors.Is(err, accounting.ErrAccountInUse) {
		t.Fatalf("flag on an account with lines: %v", err)
	}
	// An unused account that a payroll kind maps to.
	fresh := mustCreate(t, s, ctx, acct("3336", "liability", nil))
	if err := s.SaveRule(ctx, string(posting.PitPayable), fresh, "credit"); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateAccount(ctx, fresh, tracked("3336", "liability", nil)); !errors.Is(err, accounting.ErrAccountRuleMismatch) {
		t.Fatalf("flag on while a payroll kind maps to it: %v", err)
	}
	ar := mustCreate(t, s, ctx, tracked("131", "asset", nil))
	if err := s.SaveRule(ctx, string(testReceivable), ar, "debit"); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateAccount(ctx, ar, acct("131", "asset", nil)); !errors.Is(err, accounting.ErrAccountRuleMismatch) {
		t.Fatalf("flag off while a partner kind maps to it: %v", err)
	}
	unused := mustCreate(t, s, ctx, acct("331", "liability", nil))
	if err := s.UpdateAccount(ctx, unused, tracked("331", "liability", nil)); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Accounts(ctx)
	for _, a := range got {
		if a.ID == unused && !a.ByPartner {
			t.Fatal("flag not saved")
		}
	}
}

func TestTrackedAccountTakesNoChild(t *testing.T) {
	core, s := newAccounting(t)
	ar := mustCreate(t, s, core.Admin, tracked("131", "asset", nil))
	if _, err := s.CreateAccount(core.Admin, acct("1311", "asset", &ar)); !errors.Is(err, accounting.ErrAccountParentInvalid) {
		t.Fatalf("child of a tracked account: %v", err)
	}
	p := mustCreate(t, s, core.Admin, acct("331", "liability", nil))
	mustCreate(t, s, core.Admin, acct("3311", "liability", &p))
	if err := s.UpdateAccount(core.Admin, p, tracked("331", "liability", nil)); !errors.Is(err, accounting.ErrAccountNotPostable) {
		t.Fatalf("flag on a parent: %v", err)
	}
	if _, err := s.CreateAccount(core.Admin, tracked("332", "liability", &p)); err != nil {
		t.Fatalf("tracked leaf under an untracked parent: %v", err)
	}
}

func TestRulesListShowsPartnerKinds(t *testing.T) {
	core, s := newAccounting(t)
	rs, err := s.Rules(core.Admin)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rs {
		if r.ByPartner != (r.Kind == string(testReceivable)) {
			t.Fatalf("rule %+v", r)
		}
	}
}
