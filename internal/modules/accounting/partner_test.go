package accounting_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/accounting"
	"github.com/taoworklabs/mmerp/internal/platform"
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

// partner inserts a partner row; partners are written by shared/partner, not by accounting.
func (l *ledger) partner(t *testing.T, code string) int64 {
	t.Helper()
	var id int64
	if err := l.core.System.QueryRow(l.core.Ctx, `INSERT INTO partner.partners (code, name) VALUES ($1, 'Partner ' || $1) RETURNING id`, code).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// receivables adds 131 (tracked) with test_receivable mapped to it on the debit side.
func (l *ledger) receivables(t *testing.T) {
	t.Helper()
	l.acc["131"] = mustCreate(t, l.s, l.core.Admin, tracked("131", "asset", nil))
	if err := l.s.SaveRule(l.core.Admin, string(testReceivable), l.acc["131"], "debit"); err != nil {
		t.Fatal(err)
	}
}

type partnerLine struct {
	entry, partner, debit, credit int64
	code                          string
}

// partnerLines lists ref's entry lines as (entry, account code, partner or 0, debit, credit).
func (l *ledger) partnerLines(t *testing.T, ref record.Ref) []partnerLine {
	t.Helper()
	rows, err := l.core.Pool.Query(l.core.Ctx, `SELECT e.id, a.code, COALESCE(x.partner_id, 0), x.debit, x.credit
		FROM accounting.entries e JOIN accounting.entry_lines x ON x.entry_id = e.id JOIN accounting.accounts a ON a.id = x.account_id
		WHERE e.doc_type = $1 AND e.doc_id = $2 ORDER BY e.id, a.code, x.partner_id NULLS FIRST`, ref.Type, ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []partnerLine
	for rows.Next() {
		var p partnerLine
		if err := rows.Scan(&p.entry, &p.code, &p.partner, &p.debit, &p.credit); err != nil {
			t.Fatal(err)
		}
		out = append(out, p)
	}
	return out
}

func TestReconcileCarriesPartner(t *testing.T) {
	l := newLedger(t)
	l.receivables(t)
	a, b := l.partner(t, "A"), l.partner(t, "B")
	ref := l.doc(t, "2026-03-31")
	l.record(t, ref, posting.Line{Kind: testReceivable, OrgUnitID: l.unit, PartnerID: &a, Amount: 70},
		posting.Line{Kind: testReceivable, OrgUnitID: l.unit, PartnerID: &b, Amount: 30},
		posting.Line{Kind: posting.SalaryPayable, OrgUnitID: l.unit, Amount: 100})
	l.reconcile(t, ref)
	got := l.partnerLines(t, ref)
	if len(got) != 3 || got[0].code != "131" || got[0].partner != a || got[0].debit != 70 ||
		got[1].code != "131" || got[1].partner != b || got[1].debit != 30 ||
		got[2].code != "334" || got[2].partner != 0 || got[2].credit != 100 {
		t.Fatalf("lines %+v", got)
	}
}

func TestReconcileVoidReversesPartner(t *testing.T) {
	l := newLedger(t)
	l.receivables(t)
	a := l.partner(t, "A")
	ref := l.doc(t, "2026-03-31")
	l.record(t, ref, posting.Line{Kind: testReceivable, OrgUnitID: l.unit, PartnerID: &a, Amount: 70},
		posting.Line{Kind: posting.SalaryPayable, OrgUnitID: l.unit, Amount: 70})
	l.reconcile(t, ref)
	l.void(t, ref)
	l.reconcile(t, ref)
	got := l.partnerLines(t, ref)
	if len(got) != 4 || got[2].code != "131" || got[2].partner != a || got[2].credit != 70 || got[2].entry == got[0].entry {
		t.Fatalf("lines %+v", got)
	}
}

func TestWriteEntryRefusesPartnerMismatch(t *testing.T) {
	l := newLedger(t)
	l.receivables(t)
	a := l.partner(t, "A")
	ref := l.doc(t, "2026-03-31")
	date := "2026-03-31"
	for name, lines := range map[string][]accounting.JournalLine{
		"tracked account without partner": {{AccountID: l.acc["131"], Debit: 5}, {AccountID: l.acc["334"], Credit: 5}},
		"untracked account with partner":  {{AccountID: l.acc["131"], PartnerID: &a, Debit: 5}, {AccountID: l.acc["334"], PartnerID: &a, Credit: 5}},
	} {
		err := platform.InTx(l.core.Ctx, func(ctx context.Context) error {
			_, err := accounting.WriteEntryForTest(ctx, l.le, platform.NullDate(&date), ref, nil, lines)
			return err
		})
		if !errors.Is(err, accounting.ErrEntryPartnerMismatch) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestLedgerShowsPartner(t *testing.T) {
	l := newLedger(t)
	l.receivables(t)
	a := l.partner(t, "A")
	ref := l.doc(t, "2026-03-31")
	l.record(t, ref, posting.Line{Kind: testReceivable, OrgUnitID: l.unit, PartnerID: &a, Amount: 70},
		posting.Line{Kind: posting.SalaryPayable, OrgUnitID: l.unit, Amount: 70})
	l.reconcile(t, ref)
	got, err := l.s.Ledger(l.core.Admin, l.le, l.acc["131"], "2026-01-01", "2026-12-31")
	if err != nil || len(got.Lines) != 1 {
		t.Fatalf("ledger %+v %v", got, err)
	}
	if ln := got.Lines[0]; ln.PartnerID == nil || *ln.PartnerID != a || ln.PartnerCode == nil || *ln.PartnerCode != "A" || ln.PartnerName == nil || *ln.PartnerName != "Partner A" {
		t.Fatalf("line %+v", ln)
	}
}

func lineNo(err error) any {
	e, _ := asErr(err)
	if e == nil {
		return nil
	}
	return e.Params["line_no"]
}

// opening books 50 to 131 against 411 on a manual entry; partner is set on the 131 line when given.
func (l *ledger) opening(t *testing.T, partner *int64) accounting.JournalEntryFields {
	t.Helper()
	if _, ok := l.acc["411"]; !ok {
		l.acc["411"] = mustCreate(t, l.s, l.core.Admin, acct("411", "equity", nil))
	}
	return accounting.JournalEntryFields{Date: "2026-02-01", LegalEntityID: l.le, Description: "Số dư đầu kỳ", Lines: []accounting.JournalEntryLine{
		{AccountID: l.acc["131"], PartnerID: partner, Debit: 50},
		{AccountID: l.acc["411"], Credit: 50},
	}}
}

func TestJournalEntryPartnerChecks(t *testing.T) {
	l := newLedger(t)
	l.receivables(t)
	a := l.partner(t, "A")
	missing := l.newJournal(t, l.opening(t, nil))
	if err := l.post(missing); !isCode(err, "journal_line_partner_required") || lineNo(err) != 1 {
		t.Fatalf("no partner on the tracked line: %v", err)
	}
	extra := l.opening(t, &a)
	extra.Lines[1].PartnerID = &a
	if err := l.post(l.newJournal(t, extra)); !isCode(err, "journal_line_partner_not_allowed") || lineNo(err) != 2 {
		t.Fatalf("partner on an untracked line: %v", err)
	}
	ok := l.newJournal(t, l.opening(t, &a))
	if err := l.post(ok); err != nil {
		t.Fatal(err)
	}
	got := l.partnerLines(t, ok)
	if len(got) != 2 || got[0].code != "131" || got[0].partner != a || got[1].partner != 0 {
		t.Fatalf("lines %+v", got)
	}
	e, err := l.s.JournalEntry(l.core.Admin, ok.ID)
	if err != nil || e.Lines[0].PartnerCode == nil || *e.Lines[0].PartnerCode != "A" || e.Lines[0].PartnerName == nil || e.Lines[1].PartnerCode != nil {
		t.Fatalf("entry %+v %v", e, err)
	}
}

func TestJournalEntryUnknownPartner(t *testing.T) {
	l := newLedger(t)
	l.receivables(t)
	unknown := int64(-1)
	if _, err := l.s.CreateJournalEntry(l.core.Admin, accounting.NewJournalEntry{JournalEntryFields: l.opening(t, &unknown)}); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestAccountingPartnersLookup(t *testing.T) {
	l := newLedger(t)
	l.partner(t, "ANH")
	l.partner(t, "BINH")
	user := func(login, role string) context.Context {
		id, err := l.core.IAM.CreateUser(l.core.Admin, login, login, "long enough")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := l.core.IAM.GrantRole(l.core.Admin, id, "accounting", role, nil); err != nil {
			t.Fatal(err)
		}
		return platform.WithActor(l.core.Ctx, id)
	}
	got, err := l.s.Partners(user("acc", "accountant"), "nh")
	if err != nil || len(got.Items) != 2 {
		t.Fatalf("partners %+v %v", got, err)
	}
	got, err = l.s.Partners(user("acc2", "accountant"), "Partner BI")
	if err != nil || len(got.Items) != 1 || got.Items[0].Code != "BINH" || got.Items[0].Name != "Partner BINH" {
		t.Fatalf("partners %+v %v", got, err)
	}
	if _, err := l.s.Partners(user("view", "viewer"), ""); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("viewer: %v", err)
	}
}

// owed posts a manual entry moving amount on account for partner against 411: positive debits the account.
func (l *ledger) owed(t *testing.T, date string, account, partner, amount int64) {
	t.Helper()
	if _, ok := l.acc["411"]; !ok {
		l.acc["411"] = mustCreate(t, l.s, l.core.Admin, acct("411", "equity", nil))
	}
	a := accounting.JournalEntryLine{AccountID: account, PartnerID: &partner, Debit: amount}
	b := accounting.JournalEntryLine{AccountID: l.acc["411"], Credit: amount}
	if amount < 0 {
		a.Debit, a.Credit, b.Credit, b.Debit = 0, -amount, 0, -amount
	}
	if err := l.post(l.newJournal(t, accounting.JournalEntryFields{Date: date, LegalEntityID: l.le, Description: "x", Lines: []accounting.JournalEntryLine{a, b}})); err != nil {
		t.Fatal(err)
	}
}

func TestPartnerBalancesMatchTrialBalance(t *testing.T) {
	l := newLedger(t)
	l.receivables(t)
	a, b := l.partner(t, "A"), l.partner(t, "B")
	ar := l.acc["131"]
	l.owed(t, "2026-01-10", ar, a, 100)
	l.owed(t, "2026-02-10", ar, a, 40)
	l.owed(t, "2026-02-15", ar, a, -30)
	l.owed(t, "2026-02-20", ar, b, 25)
	l.owed(t, "2026-03-10", ar, b, 999)
	got, err := l.s.PartnerBalances(l.core.Admin, l.le, &ar, "2026-02-01", "2026-02-28")
	if err != nil || len(got.Rows) != 2 {
		t.Fatalf("balances %+v %v", got, err)
	}
	ra, rb := got.Rows[0], got.Rows[1]
	if ra.PartnerCode != "A" || ra.AccountCode != "131" || ra.Opening != 100 || ra.Debit != 40 || ra.Credit != 30 || ra.Closing != 110 ||
		rb.PartnerCode != "B" || rb.PartnerName != "Partner B" || rb.Opening != 0 || rb.Debit != 25 || rb.Closing != 25 {
		t.Fatalf("rows %+v", got.Rows)
	}
	tb, err := l.s.TrialBalance(l.core.Admin, l.le, "2026-02-01", "2026-02-28")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range tb.Rows {
		if r.Code == "131" && (r.Opening != got.TotalOpening || r.Debit != got.TotalDebit || r.Credit != got.TotalCredit || r.Closing != got.TotalClosing) {
			t.Fatalf("trial %+v, partners %+v", r, got)
		}
	}
	untracked := l.acc["642"]
	if _, err := l.s.PartnerBalances(l.core.Admin, l.le, &untracked, "2026-02-01", "2026-02-28"); !errors.Is(err, accounting.ErrAccountNotTracked) {
		t.Fatalf("untracked account: %v", err)
	}
}

func TestPartnerBalancesOneRowPerAccount(t *testing.T) {
	l := newLedger(t)
	l.receivables(t)
	ap := mustCreate(t, l.s, l.core.Admin, tracked("331", "liability", nil))
	a := l.partner(t, "A")
	l.owed(t, "2026-02-10", l.acc["131"], a, 40)
	l.owed(t, "2026-02-11", ap, a, -15)
	got, err := l.s.PartnerBalances(l.core.Admin, l.le, nil, "2026-02-01", "2026-02-28")
	if err != nil || len(got.Rows) != 2 || got.Rows[0].AccountCode != "131" || got.Rows[0].Closing != 40 ||
		got.Rows[1].AccountCode != "331" || got.Rows[1].Closing != -15 || got.TotalClosing != 25 {
		t.Fatalf("balances %+v %v", got, err)
	}
}

func TestPartnerLedgerRunningBalance(t *testing.T) {
	l := newLedger(t)
	l.receivables(t)
	a, b := l.partner(t, "A"), l.partner(t, "B")
	ar := l.acc["131"]
	l.owed(t, "2026-01-10", ar, a, 100)
	l.owed(t, "2026-02-10", ar, a, 40)
	l.owed(t, "2026-02-12", ar, b, 7)
	l.owed(t, "2026-02-15", ar, a, -30)
	got, err := l.s.PartnerLedger(l.core.Admin, l.le, a, nil, "2026-02-01", "2026-02-28")
	if err != nil || got.Opening != 100 || got.Closing != 110 || len(got.Lines) != 2 {
		t.Fatalf("ledger %+v %v", got, err)
	}
	if x, y := got.Lines[0], got.Lines[1]; x.Balance != 140 || x.DocNumber == "" || x.AccountCode != "131" || y.Balance != 110 || y.Credit != 30 {
		t.Fatalf("lines %+v", got.Lines)
	}
}

func TestPartnerReportsStayInLegalEntity(t *testing.T) {
	l := newLedger(t)
	l.receivables(t)
	a := l.partner(t, "A")
	le2, err := l.core.IAM.CreateOrgUnit(l.core.Admin, iam.OrgUnitInput{Kind: "company", Name: "C2"})
	if err != nil {
		t.Fatal(err)
	}
	start := "2026-01-01"
	if err := l.s.SetStart(l.core.Admin, le2, &start); err != nil {
		t.Fatal(err)
	}
	l.owed(t, "2026-02-10", l.acc["131"], a, 40)
	ref := l.newJournal(t, accounting.JournalEntryFields{Date: "2026-02-10", LegalEntityID: le2, Description: "x", Lines: []accounting.JournalEntryLine{
		{AccountID: l.acc["131"], PartnerID: &a, Debit: 5}, {AccountID: l.acc["411"], Credit: 5}}})
	if err := l.post(ref); err != nil {
		t.Fatal(err)
	}
	for le, want := range map[int64]int64{l.le: 40, le2: 5} {
		b, err := l.s.PartnerBalances(l.core.Admin, le, nil, "2026-02-01", "2026-02-28")
		if err != nil || len(b.Rows) != 1 || b.Rows[0].Closing != want {
			t.Fatalf("balances %d %+v %v", le, b, err)
		}
		g, err := l.s.PartnerLedger(l.core.Admin, le, a, nil, "2026-02-01", "2026-02-28")
		if err != nil || len(g.Lines) != 1 || g.Closing != want {
			t.Fatalf("ledger %d %+v %v", le, g, err)
		}
	}
	id, err := l.core.IAM.CreateUser(l.core.Admin, "view2", "View", "long enough")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.core.IAM.GrantRole(l.core.Admin, id, "accounting", "viewer", &le2); err != nil {
		t.Fatal(err)
	}
	other := platform.WithActor(l.core.Ctx, id)
	if _, err := l.s.PartnerBalances(other, l.le, nil, "2026-02-01", "2026-02-28"); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("balances elsewhere: %v", err)
	}
	if _, err := l.s.PartnerLedger(other, l.le, a, nil, "2026-02-01", "2026-02-28"); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("ledger elsewhere: %v", err)
	}
}
