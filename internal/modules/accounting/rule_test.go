package accounting_test

import (
	"errors"
	"testing"

	"github.com/taoworklabs/mmerp/internal/modules/accounting"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

func TestRulesListEveryKind(t *testing.T) {
	core, s := newAccounting(t)
	rs, err := s.Rules(core.Admin)
	if err != nil || len(rs) != len(posting.Kinds()) {
		t.Fatalf("rules %+v %v", rs, err)
	}
	for i, r := range rs {
		if r.Kind != string(posting.Kinds()[i]) || r.AccountID != nil || r.Side != nil {
			t.Fatalf("rule %d %+v", i, r)
		}
	}
}

func TestSaveRuleAndDelete(t *testing.T) {
	core, s := newAccounting(t)
	a := mustCreate(t, s, core.Admin, acct("642", "expense", nil))
	if err := s.SaveRule(core.Admin, string(posting.SalaryExpense), a, "debit"); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveRule(core.Admin, string(posting.SalaryExpense), a, "credit"); err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	rs, _ := s.Rules(core.Admin)
	if rs[0].AccountID == nil || *rs[0].AccountID != a || *rs[0].Side != "credit" {
		t.Fatalf("rules %+v", rs)
	}
	if err := s.DeleteAccount(core.Admin, a); !errors.Is(err, accounting.ErrAccountInUse) {
		t.Fatalf("delete referenced account: %v", err)
	}
	if err := s.DeleteRule(core.Admin, string(posting.SalaryExpense)); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteAccount(core.Admin, a); err != nil {
		t.Fatal(err)
	}
}

func TestSaveRuleRefusesParentAccount(t *testing.T) {
	core, s := newAccounting(t)
	p := mustCreate(t, s, core.Admin, acct("333", "liability", nil))
	mustCreate(t, s, core.Admin, acct("3335", "liability", &p))
	if err := s.SaveRule(core.Admin, string(posting.PitPayable), p, "credit"); !errors.Is(err, accounting.ErrAccountNotPostable) {
		t.Fatalf("got %v", err)
	}
}

func TestSaveRuleRefusesInactiveAccount(t *testing.T) {
	core, s := newAccounting(t)
	f := acct("642", "expense", nil)
	f.Active = false
	a := mustCreate(t, s, core.Admin, f)
	if err := s.SaveRule(core.Admin, string(posting.SalaryExpense), a, "debit"); !errors.Is(err, accounting.ErrAccountNotPostable) {
		t.Fatalf("got %v", err)
	}
}

// A rule for an undeclared kind is listed; posting lines alone add nothing, so the list never scans them.
func TestRuleForUnknownKindIsListed(t *testing.T) {
	l := newLedger(t)
	if err := l.s.SaveRule(l.core.Admin, "legacy_kind", l.acc["642"], "debit"); err != nil {
		t.Fatal(err)
	}
	l.record(t, l.doc(t, "2026-03-31"), posting.Line{Kind: "lines_only_kind", OrgUnitID: l.unit, Amount: 1})
	rs, _ := l.s.Rules(l.core.Admin)
	if len(rs) != len(posting.Kinds())+1 || rs[len(rs)-1].Kind != "legacy_kind" {
		t.Fatalf("rules %+v", rs)
	}
}

func TestSaveRuleRefusesBadSide(t *testing.T) {
	core, s := newAccounting(t)
	a := mustCreate(t, s, core.Admin, acct("642", "expense", nil))
	if err := s.SaveRule(core.Admin, string(posting.SalaryExpense), a, "left"); !errors.Is(err, accounting.ErrRuleSideInvalid) {
		t.Fatalf("got %v", err)
	}
}
