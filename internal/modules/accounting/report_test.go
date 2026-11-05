package accounting_test

import (
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/accounting"
)

// entry books amount from account cr to account dr (debit dr, credit cr) and posts it.
func (l *ledger) entry(t *testing.T, date string, dr, cr int64, amount int64) record.Ref {
	t.Helper()
	ref := l.newJournal(t, accounting.JournalEntryFields{Date: date, LegalEntityID: l.le, Description: "x", Lines: []accounting.JournalEntryLine{
		{AccountID: dr, Debit: amount}, {AccountID: cr, Credit: amount}}})
	if err := l.post(ref); err != nil {
		t.Fatal(err)
	}
	return ref
}

func TestLedgerOpeningAndRunningBalance(t *testing.T) {
	l := newLedger(t)
	l.entry(t, "2026-01-15", l.acc["642"], l.acc["334"], 100)
	l.entry(t, "2026-02-10", l.acc["642"], l.acc["334"], 30)
	l.entry(t, "2026-02-20", l.acc["334"], l.acc["642"], 10)
	l.entry(t, "2026-03-05", l.acc["642"], l.acc["334"], 999)
	got, err := l.s.Ledger(l.core.Admin, l.le, l.acc["642"], "2026-02-01", "2026-02-28")
	if err != nil || got.Opening != 100 || got.Closing != 120 || len(got.Lines) != 2 {
		t.Fatalf("ledger %+v %v", got, err)
	}
	if a, b := got.Lines[0], got.Lines[1]; a.Balance != 130 || a.Debit != 30 || a.DocNumber == "" || a.AccountCode != "642" || b.Balance != 120 || b.Credit != 10 {
		t.Fatalf("lines %+v", got.Lines)
	}
	if _, err := l.s.Ledger(l.core.Admin, l.le, l.acc["642"], "2026-03-01", "2026-02-01"); !isCode(err, "invalid_range") {
		t.Fatalf("range %v", err)
	}
}

// parents makes 64 > 6421, 6422 (expenses).
func (l *ledger) parents(t *testing.T) (p, c1, c2 int64) {
	t.Helper()
	p = mustCreate(t, l.s, l.core.Admin, acct("64", "expense", nil))
	c1 = mustCreate(t, l.s, l.core.Admin, acct("6421", "expense", &p))
	c2 = mustCreate(t, l.s, l.core.Admin, acct("6422", "expense", &p))
	return
}

func TestLedgerParentIncludesChildren(t *testing.T) {
	l := newLedger(t)
	p, c1, c2 := l.parents(t)
	l.entry(t, "2026-02-01", c1, l.acc["334"], 40)
	l.entry(t, "2026-02-02", c2, l.acc["334"], 60)
	got, err := l.s.Ledger(l.core.Admin, l.le, p, "2026-02-01", "2026-02-28")
	if err != nil || len(got.Lines) != 2 || got.Closing != 100 || got.Lines[0].AccountCode != "6421" || got.Lines[1].AccountCode != "6422" {
		t.Fatalf("ledger %+v %v", got, err)
	}
}

func TestTrialBalanceTotalsEqual(t *testing.T) {
	l := newLedger(t)
	l.entry(t, "2026-01-10", l.acc["642"], l.acc["334"], 100)
	ref := l.entry(t, "2026-02-10", l.acc["642"], l.acc["338"], 50)
	l.entry(t, "2026-02-11", l.acc["338"], l.acc["334"], 7)
	e, err := l.s.JournalEntry(l.core.Admin, ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.core.Record.Transition(l.core.Admin, ref, e.Version, record.Cancelled); err != nil {
		t.Fatal(err)
	}
	tb, err := l.s.TrialBalance(l.core.Admin, l.le, "2026-02-01", "2026-02-28")
	if err != nil || tb.TotalDebit != tb.TotalCredit || tb.TotalDebit != 50+7+50 {
		t.Fatalf("trial %+v %v", tb, err)
	}
	var open int64
	for _, r := range tb.Rows {
		open += r.Opening
		if r.Closing != r.Opening+r.Debit-r.Credit {
			t.Fatalf("closing %+v", r)
		}
	}
	if open != 0 {
		t.Fatalf("opening sums to %d", open)
	}
}

func TestTrialBalanceRollsUpParents(t *testing.T) {
	l := newLedger(t)
	p, c1, c2 := l.parents(t)
	l.entry(t, "2026-01-10", c1, l.acc["334"], 40)
	l.entry(t, "2026-02-10", c2, l.acc["334"], 60)
	tb, err := l.s.TrialBalance(l.core.Admin, l.le, "2026-02-01", "2026-02-28")
	if err != nil {
		t.Fatal(err)
	}
	by := map[int64]accounting.TrialRow{}
	for _, r := range tb.Rows {
		by[r.AccountID] = r
	}
	if r := by[p]; r.Opening != 40 || r.Debit != 60 || r.Closing != 100 || r.ParentID != nil {
		t.Fatalf("parent %+v", r)
	}
	if r := by[c1]; r.Opening != 40 || r.Debit != 0 || r.ParentID == nil || *r.ParentID != p {
		t.Fatalf("child %+v", r)
	}
	if tb.TotalDebit != 60 || tb.TotalCredit != 60 {
		t.Fatalf("totals %d %d", tb.TotalDebit, tb.TotalCredit)
	}
}

func TestReportsStayInLegalEntity(t *testing.T) {
	l := newLedger(t)
	le2, err := l.core.IAM.CreateOrgUnit(l.core.Admin, iam.OrgUnitInput{Kind: "company", Name: "C2"})
	if err != nil {
		t.Fatal(err)
	}
	start := "2026-01-01"
	if err := l.s.SetStart(l.core.Admin, le2, &start); err != nil {
		t.Fatal(err)
	}
	l.entry(t, "2026-02-10", l.acc["642"], l.acc["334"], 100)
	ref := l.newJournal(t, accounting.JournalEntryFields{Date: "2026-02-10", LegalEntityID: le2, Description: "x", Lines: []accounting.JournalEntryLine{
		{AccountID: l.acc["642"], Debit: 5}, {AccountID: l.acc["334"], Credit: 5}}})
	if err := l.post(ref); err != nil {
		t.Fatal(err)
	}
	for le, want := range map[int64]int64{l.le: 100, le2: 5} {
		g, err := l.s.Ledger(l.core.Admin, le, l.acc["642"], "2026-02-01", "2026-02-28")
		if err != nil || len(g.Lines) != 1 || g.Closing != want {
			t.Fatalf("ledger %d %+v %v", le, g, err)
		}
		tb, err := l.s.TrialBalance(l.core.Admin, le, "2026-02-01", "2026-02-28")
		if err != nil || tb.TotalDebit != want {
			t.Fatalf("trial %d %+v %v", le, tb, err)
		}
	}
	// Pending: a payroll document of le2 is not listed for le.
	doc, err := l.core.Record.Create(l.core.Admin, testDoc, record.Header{Date: "2026-02-10", OrgUnitID: le2})
	if err != nil {
		t.Fatal(err)
	}
	l.record(t, doc.Ref, l.payroll()...)
	if got, err := l.s.Pending(l.core.Admin, l.le); err != nil || len(got) != 0 {
		t.Fatalf("pending of le %+v %v", got, err)
	}
	if got, err := l.s.Pending(l.core.Admin, le2); err != nil || len(got) != 1 {
		t.Fatalf("pending of le2 %+v %v", got, err)
	}
}
