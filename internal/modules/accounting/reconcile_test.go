package accounting_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/taoworklabs/mmerp/internal/core/coretest"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/accounting"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

const testDoc = "accounting.testdoc"

// ledger: company le with department unit, accounts 642 (expense), 334, 338, 3335
// (liabilities), a rule for every payroll kind and a start date of 2026-01-01.
type ledger struct {
	core     *coretest.Core
	s        *accounting.Service
	pst      *posting.Service
	le, unit int64
	acc      map[string]int64
}

func newLedger(t *testing.T) *ledger {
	t.Helper()
	core, s := newAccounting(t)
	// A bare document type, so the ledger has documents to post lines for.
	core.Record.Register(record.Type{Code: testDoc, Product: "accounting", Kind: record.Document, NumberPrefix: "TT",
		Perms: record.Perms{View: accounting.PermView, Edit: accounting.PermEntryEdit}, OnDelete: func(context.Context, int64) error { return nil }})
	l := &ledger{core: core, s: s, pst: posting.NewService(posting.Hooks{}), acc: map[string]int64{}}
	var err error
	if l.le, err = core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{Kind: "company", Name: "C"}); err != nil {
		t.Fatal(err)
	}
	if l.unit, err = core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{ParentID: &l.le, Kind: "department", Name: "A"}); err != nil {
		t.Fatal(err)
	}
	l.acc["642"] = mustCreate(t, s, core.Admin, acct("642", "expense", nil))
	for _, c := range []string{"334", "338", "3335"} {
		l.acc[c] = mustCreate(t, s, core.Admin, acct(c, "liability", nil))
	}
	for kind, rule := range map[posting.Kind][2]string{
		posting.SalaryExpense: {"642", "debit"}, posting.EmployerInsuranceExpense: {"642", "debit"},
		posting.SalaryPayable: {"334", "credit"}, posting.InsurancePayable: {"338", "credit"}, posting.PitPayable: {"3335", "credit"},
	} {
		if err := s.SaveRule(core.Admin, string(kind), l.acc[rule[0]], rule[1]); err != nil {
			t.Fatal(err)
		}
	}
	start := "2026-01-01"
	if err := s.SetStart(core.Admin, l.le, &start); err != nil {
		t.Fatal(err)
	}
	return l
}

func (l *ledger) doc(t *testing.T, date string) record.Ref {
	t.Helper()
	d, err := l.core.Record.Create(l.core.Admin, testDoc, record.Header{Date: date, OrgUnitID: l.le})
	if err != nil {
		t.Fatal(err)
	}
	return d.Ref
}

func (l *ledger) record(t *testing.T, ref record.Ref, lines ...posting.Line) {
	t.Helper()
	if err := platform.InTx(l.core.Ctx, func(ctx context.Context) error { return l.pst.Record(ctx, ref, lines) }); err != nil {
		t.Fatal(err)
	}
}

func (l *ledger) void(t *testing.T, ref record.Ref) {
	t.Helper()
	if err := platform.InTx(l.core.Ctx, func(ctx context.Context) error { return l.pst.Void(ctx, ref) }); err != nil {
		t.Fatal(err)
	}
}

func (l *ledger) reconcile(t *testing.T, ref record.Ref) {
	t.Helper()
	if err := l.s.Reconcile(l.core.Ctx, ref); err != nil {
		t.Fatal(err)
	}
}

// payroll is the balanced set: 642 debit 120, 334 credit 80, 338 credit 30, 3335 credit 10.
func (l *ledger) payroll() []posting.Line {
	return []posting.Line{
		{Kind: posting.SalaryExpense, OrgUnitID: l.unit, Amount: 100},
		{Kind: posting.EmployerInsuranceExpense, OrgUnitID: l.unit, Amount: 20},
		{Kind: posting.SalaryPayable, OrgUnitID: l.unit, Amount: 80},
		{Kind: posting.InsurancePayable, OrgUnitID: l.le, Amount: 30},
		{Kind: posting.PitPayable, OrgUnitID: l.le, Amount: 10},
	}
}

type entry struct {
	id       int64
	reverses *int64
	// Debit and credit per account code.
	debit, credit map[string]int64
}

func (l *ledger) entries(t *testing.T, ref record.Ref) []entry {
	t.Helper()
	rows, err := l.core.Pool.Query(l.core.Ctx, `SELECT e.id, e.reverses, a.code, sum(x.debit), sum(x.credit)
		FROM accounting.entries e JOIN accounting.entry_lines x ON x.entry_id = e.id JOIN accounting.accounts a ON a.id = x.account_id
		WHERE e.doc_type = $1 AND e.doc_id = $2 GROUP BY e.id, a.code ORDER BY e.id, a.code`, ref.Type, ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []entry
	for rows.Next() {
		var id, d, c int64
		var rev *int64
		var code string
		if err := rows.Scan(&id, &rev, &code, &d, &c); err != nil {
			t.Fatal(err)
		}
		if len(out) == 0 || out[len(out)-1].id != id {
			out = append(out, entry{id: id, reverses: rev, debit: map[string]int64{}, credit: map[string]int64{}})
		}
		e := &out[len(out)-1]
		if d != 0 {
			e.debit[code] = d
		}
		if c != 0 {
			e.credit[code] = c
		}
	}
	return out
}

type source struct {
	exists  bool
	posted  *int64
	problem *string
	missing []string
}

func (l *ledger) source(t *testing.T, ref record.Ref) source {
	t.Helper()
	var s source
	err := l.core.Pool.QueryRow(l.core.Ctx, `SELECT posted_recording_id, problem, missing_kinds FROM accounting.sources WHERE doc_type = $1 AND doc_id = $2`,
		ref.Type, ref.ID).Scan(&s.posted, &s.problem, &s.missing)
	if err == nil {
		s.exists = true
	} else if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal(err)
	}
	return s
}

func wantPayrollEntry(t *testing.T, e entry) {
	t.Helper()
	if e.reverses != nil || len(e.debit) != 1 || e.debit["642"] != 120 || len(e.credit) != 3 ||
		e.credit["334"] != 80 || e.credit["338"] != 30 || e.credit["3335"] != 10 {
		t.Fatalf("entry %+v", e)
	}
}

func TestReconcilePostsBalancedEntry(t *testing.T) {
	l := newLedger(t)
	ref := l.doc(t, "2026-03-31")
	l.record(t, ref, l.payroll()...)
	l.reconcile(t, ref)
	es := l.entries(t, ref)
	if len(es) != 1 {
		t.Fatalf("entries %+v", es)
	}
	wantPayrollEntry(t, es[0])
	var units []int64
	rows, _ := l.core.Pool.Query(l.core.Ctx, `SELECT org_unit_id FROM accounting.entry_lines WHERE entry_id = $1 ORDER BY line_no`, es[0].id)
	for rows.Next() {
		var u int64
		_ = rows.Scan(&u)
		units = append(units, u)
	}
	rows.Close()
	if !slices.Equal(units, []int64{l.unit, l.unit, l.unit, l.le, l.le}) {
		t.Fatalf("org units %v", units)
	}
	var date string
	if err := l.core.Pool.QueryRow(l.core.Ctx, `SELECT date::text FROM accounting.entries WHERE id = $1`, es[0].id).Scan(&date); err != nil || date != "2026-03-31" {
		t.Fatalf("date %s %v", date, err)
	}
	if s := l.source(t, ref); s.posted == nil || s.problem != nil {
		t.Fatalf("source %+v", s)
	}
}

func TestReconcileIdempotent(t *testing.T) {
	l := newLedger(t)
	ref := l.doc(t, "2026-03-31")
	l.record(t, ref, l.payroll()...)
	l.reconcile(t, ref)
	l.reconcile(t, ref)
	if es := l.entries(t, ref); len(es) != 1 {
		t.Fatalf("entries %+v", es)
	}
}

func TestReconcileVoidWritesReversal(t *testing.T) {
	l := newLedger(t)
	ref := l.doc(t, "2026-03-31")
	l.record(t, ref, l.payroll()...)
	l.reconcile(t, ref)
	l.void(t, ref)
	l.reconcile(t, ref)
	l.reconcile(t, ref)
	es := l.entries(t, ref)
	if len(es) != 2 || es[1].reverses == nil || *es[1].reverses != es[0].id {
		t.Fatalf("entries %+v", es)
	}
	if r := es[1]; r.credit["642"] != 120 || r.debit["334"] != 80 || r.debit["338"] != 30 || r.debit["3335"] != 10 || len(r.credit) != 1 {
		t.Fatalf("reversal %+v", r)
	}
	var nonzero int
	if err := l.core.Pool.QueryRow(l.core.Ctx, `SELECT count(*) FROM (SELECT account_id FROM accounting.entry_lines GROUP BY account_id
		HAVING sum(debit) <> sum(credit)) x`).Scan(&nonzero); err != nil || nonzero != 0 {
		t.Fatalf("accounts not netted: %d %v", nonzero, err)
	}
	if s := l.source(t, ref); s.posted != nil || s.problem != nil {
		t.Fatalf("source %+v", s)
	}
}

func TestReconcileOrderIndependent(t *testing.T) {
	l := newLedger(t)
	ref := l.doc(t, "2026-03-31")
	l.record(t, ref, l.payroll()...)
	l.void(t, ref)
	l.reconcile(t, ref)
	l.reconcile(t, ref)
	if es := l.entries(t, ref); len(es) != 0 {
		t.Fatalf("entries %+v", es)
	}
}

func TestReconcileUnmapped(t *testing.T) {
	l := newLedger(t)
	ref := l.doc(t, "2026-03-31")
	l.record(t, ref, l.payroll()...)
	if err := l.s.DeleteRule(l.core.Admin, string(posting.PitPayable)); err != nil {
		t.Fatal(err)
	}
	l.reconcile(t, ref)
	if es := l.entries(t, ref); len(es) != 0 {
		t.Fatalf("entries %+v", es)
	}
	if s := l.source(t, ref); s.posted != nil || s.problem == nil || *s.problem != "unmapped" || !slices.Equal(s.missing, []string{"pit_payable"}) {
		t.Fatalf("source %+v", s)
	}
	if err := l.s.SaveRule(l.core.Admin, string(posting.PitPayable), l.acc["3335"], "credit"); err != nil {
		t.Fatal(err)
	}
	l.reconcile(t, ref)
	if es := l.entries(t, ref); len(es) != 1 {
		t.Fatalf("entries %+v", es)
	}
	if s := l.source(t, ref); s.posted == nil || s.problem != nil || len(s.missing) != 0 {
		t.Fatalf("source %+v", s)
	}
}

func TestReconcileInactiveAccountIsUnmapped(t *testing.T) {
	l := newLedger(t)
	ref := l.doc(t, "2026-03-31")
	l.record(t, ref, l.payroll()...)
	off := acct("3335", "liability", nil)
	off.Active = false
	if err := l.s.UpdateAccount(l.core.Admin, l.acc["3335"], off); err != nil {
		t.Fatal(err)
	}
	l.reconcile(t, ref)
	if s := l.source(t, ref); s.problem == nil || *s.problem != "unmapped" || !slices.Equal(s.missing, []string{"pit_payable"}) {
		t.Fatalf("source %+v", s)
	}
	if es := l.entries(t, ref); len(es) != 0 {
		t.Fatalf("entries %+v", es)
	}
}

func TestReconcileNegativeAmountFlipsSide(t *testing.T) {
	l := newLedger(t)
	ref := l.doc(t, "2026-03-31")
	l.record(t, ref, posting.Line{Kind: posting.SalaryExpense, OrgUnitID: l.unit, Amount: -50},
		posting.Line{Kind: posting.SalaryPayable, OrgUnitID: l.unit, Amount: -50})
	l.reconcile(t, ref)
	es := l.entries(t, ref)
	if len(es) != 1 || es[0].credit["642"] != 50 || es[0].debit["334"] != 50 || len(es[0].debit) != 1 || len(es[0].credit) != 1 {
		t.Fatalf("entries %+v", es)
	}
}

func TestReconcileUnbalanced(t *testing.T) {
	l := newLedger(t)
	ref := l.doc(t, "2026-03-31")
	l.record(t, ref, posting.Line{Kind: posting.SalaryExpense, OrgUnitID: l.unit, Amount: 100},
		posting.Line{Kind: posting.SalaryPayable, OrgUnitID: l.unit, Amount: 90})
	l.reconcile(t, ref)
	if es := l.entries(t, ref); len(es) != 0 {
		t.Fatalf("entries %+v", es)
	}
	if s := l.source(t, ref); s.posted != nil || s.problem == nil || *s.problem != "unbalanced" {
		t.Fatalf("source %+v", s)
	}
}

func TestReconcileBeforeStartDate(t *testing.T) {
	l := newLedger(t)
	ref := l.doc(t, "2025-12-31")
	l.record(t, ref, l.payroll()...)
	l.reconcile(t, ref)
	if s := l.source(t, ref); s.exists {
		t.Fatalf("source %+v", s)
	}
	if es := l.entries(t, ref); len(es) != 0 {
		t.Fatalf("entries %+v", es)
	}
}

func TestReconcileRuleChangeDoesNotRemap(t *testing.T) {
	l := newLedger(t)
	ref := l.doc(t, "2026-03-31")
	l.record(t, ref, l.payroll()...)
	l.reconcile(t, ref)
	other := mustCreate(t, l.s, l.core.Admin, acct("6422", "expense", nil))
	if err := l.s.SaveRule(l.core.Admin, string(posting.SalaryExpense), other, "debit"); err != nil {
		t.Fatal(err)
	}
	l.void(t, ref)
	l.reconcile(t, ref)
	es := l.entries(t, ref)
	if len(es) != 2 || es[1].credit["642"] != 120 || es[1].credit["6422"] != 0 || es[1].debit["6422"] != 0 {
		t.Fatalf("entries %+v", es)
	}
}

func TestReconcileFirstPostingIntoLockedPeriod(t *testing.T) {
	l := newLedger(t)
	ref := l.doc(t, "2026-03-31")
	l.record(t, ref, l.payroll()...)
	until := "2026-12-31"
	if err := l.core.Record.SetPeriodLock(l.core.Admin, l.le, &until); err != nil {
		t.Fatal(err)
	}
	l.reconcile(t, ref)
	if es := l.entries(t, ref); len(es) != 1 {
		t.Fatalf("entries %+v", es)
	}
}

func TestPostingChangedSkipsWhenProductOff(t *testing.T) {
	l := newLedger(t)
	ref := l.doc(t, "2026-03-31")
	ctx := platform.WithProducts(l.core.Ctx, []string{"hrm"})
	if err := platform.InTx(ctx, func(ctx context.Context) error { return l.s.PostingChanged(ctx, ref) }); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := l.core.Pool.QueryRow(l.core.Ctx, `SELECT count(*) FROM river_job WHERE kind = 'accounting.reconcile'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("jobs %d %v", n, err)
	}
}

func TestAccountInUse(t *testing.T) {
	l := newLedger(t)
	ref := l.doc(t, "2026-03-31")
	l.record(t, ref, l.payroll()...)
	l.reconcile(t, ref)
	if err := l.s.DeleteRule(l.core.Admin, string(posting.SalaryExpense)); err != nil {
		t.Fatal(err)
	}
	if err := l.s.DeleteRule(l.core.Admin, string(posting.EmployerInsuranceExpense)); err != nil {
		t.Fatal(err)
	}
	id := l.acc["642"]
	if err := l.s.DeleteAccount(l.core.Admin, id); !errors.Is(err, accounting.ErrAccountInUse) {
		t.Fatalf("delete: %v", err)
	}
	if err := l.s.UpdateAccount(l.core.Admin, id, acct("642", "asset", nil)); !errors.Is(err, accounting.ErrAccountInUse) {
		t.Fatalf("type change: %v", err)
	}
	if err := l.s.UpdateAccount(l.core.Admin, id, acct("6420", "expense", nil)); !errors.Is(err, accounting.ErrAccountInUse) {
		t.Fatalf("code change: %v", err)
	}
	if _, err := l.s.CreateAccount(l.core.Admin, acct("6421", "expense", &id)); !errors.Is(err, accounting.ErrAccountParentInvalid) {
		t.Fatalf("child: %v", err)
	}
	if l, err := l.s.Accounts(l.core.Admin); err != nil || !l[slices.IndexFunc(l, func(a accounting.Account) bool { return a.ID == id })].HasLines {
		t.Fatalf("has lines %+v %v", l, err)
	}
}

func TestStartDateFixedAfterEntry(t *testing.T) {
	l := newLedger(t)
	ref := l.doc(t, "2026-03-31")
	l.record(t, ref, l.payroll()...)
	l.reconcile(t, ref)
	d := "2026-02-01"
	if err := l.s.SetStart(l.core.Admin, l.le, &d); !errors.Is(err, accounting.ErrStartDateFixed) {
		t.Fatalf("set: %v", err)
	}
	if err := l.s.SetStart(l.core.Admin, l.le, nil); !errors.Is(err, accounting.ErrStartDateFixed) {
		t.Fatalf("clear: %v", err)
	}
	st, err := l.s.Starts(l.core.Admin)
	if err != nil || len(st) != 1 || !st[0].Fixed {
		t.Fatalf("starts %+v %v", st, err)
	}
}

// A period locked while accounting was off can hold a cancellation the ledger has not seen:
// the reversal catches up into it.
func TestReconcileReversalIntoLockedPeriodAfterProductOff(t *testing.T) {
	l := newLedger(t)
	ref := l.doc(t, "2026-03-31")
	l.record(t, ref, l.payroll()...)
	l.reconcile(t, ref)
	l.void(t, ref)
	until := "2026-03-31"
	if err := l.core.Record.SetPeriodLock(l.core.Admin, l.le, &until); err != nil {
		t.Fatal(err)
	}
	l.reconcile(t, ref)
	if es := l.entries(t, ref); len(es) != 2 || es[1].reverses == nil || *es[1].reverses != es[0].id {
		t.Fatalf("entries %+v", es)
	}
	if s := l.source(t, ref); s.posted != nil || s.problem != nil {
		t.Fatalf("source %+v", s)
	}
}

func TestReconcileWithoutStartDate(t *testing.T) {
	l := newLedger(t)
	if err := l.s.SetStart(l.core.Admin, l.le, nil); err != nil {
		t.Fatal(err)
	}
	ref := l.doc(t, "2026-03-31")
	l.record(t, ref, l.payroll()...)
	l.reconcile(t, ref)
	if s := l.source(t, ref); s.exists {
		t.Fatalf("source %+v", s)
	}
	if es := l.entries(t, ref); len(es) != 0 {
		t.Fatalf("entries %+v", es)
	}
}

func TestReconcileRuleAccountWithChildIsUnmapped(t *testing.T) {
	l := newLedger(t)
	ref := l.doc(t, "2026-03-31")
	l.record(t, ref, l.payroll()...)
	parent := l.acc["3335"]
	mustCreate(t, l.s, l.core.Admin, acct("33351", "liability", &parent))
	l.reconcile(t, ref)
	if s := l.source(t, ref); s.problem == nil || *s.problem != "unmapped" || !slices.Equal(s.missing, []string{"pit_payable"}) {
		t.Fatalf("source %+v", s)
	}
	if es := l.entries(t, ref); len(es) != 0 {
		t.Fatalf("entries %+v", es)
	}
}

func TestPostingChangedQueuesOneJob(t *testing.T) {
	l := newLedger(t)
	ref := l.doc(t, "2026-03-31")
	jobs, err := river.NewClient(riverpgxv5.New(l.core.Pool), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	err = platform.InTx(platform.WithJobs(l.core.Ctx, jobs), func(ctx context.Context) error {
		if err := l.s.PostingChanged(ctx, ref); err != nil {
			return err
		}
		return l.s.PostingChanged(ctx, ref)
	})
	if err != nil {
		t.Fatal(err)
	}
	var n int
	if err := l.core.Pool.QueryRow(l.core.Ctx, `SELECT count(*) FROM river_job WHERE kind = 'accounting.reconcile'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("jobs %d %v", n, err)
	}
}
