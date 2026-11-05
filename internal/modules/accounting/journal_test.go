package accounting_test

import (
	"context"
	"errors"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/record/recordtest"
	"github.com/taoworklabs/mmerp/internal/modules/accounting"
	"github.com/taoworklabs/mmerp/internal/platform"
)

const journalType = "accounting.journal_entry"

// fields books 100 from account 334 to 642.
func (l *ledger) journal(date string) accounting.JournalEntryFields {
	return accounting.JournalEntryFields{Date: date, LegalEntityID: l.le, Description: "Điều chỉnh", Lines: []accounting.JournalEntryLine{
		{AccountID: l.acc["642"], Debit: 100},
		{AccountID: l.acc["334"], OrgUnitID: &l.unit, Credit: 100},
	}}
}

func (l *ledger) newJournal(t *testing.T, f accounting.JournalEntryFields) record.Ref {
	t.Helper()
	id, err := l.s.CreateJournalEntry(l.core.Admin, accounting.NewJournalEntry{JournalEntryFields: f})
	if err != nil {
		t.Fatal(err)
	}
	return record.Ref{Type: journalType, ID: id}
}

func (l *ledger) post(ref record.Ref) error {
	e, err := l.s.JournalEntry(l.core.Admin, ref.ID)
	if err != nil {
		return err
	}
	_, err = l.core.Record.Transition(l.core.Admin, ref, e.Version, record.Posted)
	return err
}

func TestJournalEntryRecordContract(t *testing.T) {
	recordtest.Run(t, func(t *testing.T) recordtest.Harness {
		l := newLedger(t)
		approver, err := l.core.IAM.CreateUser(l.core.Admin, "approver", "Approver", "long enough")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := l.core.IAM.GrantRole(l.core.Admin, approver, "accounting", "admin", nil); err != nil {
			t.Fatal(err)
		}
		actor := platform.WithActor(l.core.Ctx, approver)
		return recordtest.Harness{
			Core: l.core, Actor: l.core.Admin, Admin: l.core.Admin, LegalEntity: l.le, Table: "accounting.journal_entries",
			Get: func(ctx context.Context, r record.Ref) (any, error) { return l.s.JournalEntry(ctx, r.ID) },
			Create: func(ctx context.Context, date, requestID string) (record.Ref, error) {
				id, err := l.s.CreateJournalEntry(ctx, accounting.NewJournalEntry{RequestID: requestID, JournalEntryFields: l.journal(date)})
				return record.Ref{Type: journalType, ID: id}, err
			},
			Edit: func(ctx context.Context, r record.Ref, version int32, date string) error {
				return l.s.UpdateJournalEntry(ctx, r.ID, accounting.JournalEntryUpdate{Version: version, JournalEntryFields: l.journal(date)})
			},
			RequireApproval: func(t *testing.T, a record.Action) {
				if err := l.core.Approval.SaveRule(l.core.Admin, journalType, a, approval.RuleInput{
					Flow:      approval.Flow{{Step: &approval.StepDef{Approver: approval.Approver{Kind: "role", Product: "accounting", Role: "admin"}}}},
					MaxLevels: 3, FallbackProduct: "accounting", FallbackRole: "admin",
				}); err != nil {
					t.Fatal(err)
				}
			},
			Approve: func(instance int64) error { return l.core.Approval.Approve(actor, instance, 1) },
			Reject:  func(instance int64) error { return l.core.Approval.Reject(actor, instance, 1, "không") },
			// An inactive account is refused at posting.
			BreakPosting: func(t *testing.T, _ record.Ref) {
				f := acct("642", "expense", nil)
				f.Active = false
				if err := l.s.UpdateAccount(l.core.Admin, l.acc["642"], f); err != nil {
					t.Fatal(err)
				}
			},
			Effects: func(t *testing.T, r record.Ref) int {
				var n int
				if err := l.core.Pool.QueryRow(l.core.Ctx, `SELECT count(*) FROM accounting.entries e WHERE e.doc_type = $1 AND e.doc_id = $2
					AND e.reverses IS NULL AND NOT EXISTS (SELECT 1 FROM accounting.entries r WHERE r.reverses = e.id)`, r.Type, r.ID).Scan(&n); err != nil {
					t.Fatal(err)
				}
				return n
			},
		}
	})
}

func TestJournalEntryUnbalancedRefused(t *testing.T) {
	l := newLedger(t)
	f := l.journal("2026-03-31")
	f.Lines[1].Credit = 90
	ref := l.newJournal(t, f)
	if err := l.post(ref); !isCode(err, "journal_entry_unbalanced") {
		t.Fatalf("unbalanced: %v", err)
	}
	f.Lines = f.Lines[:1]
	ref = l.newJournal(t, f)
	if err := l.post(ref); !isCode(err, "journal_entry_unbalanced") {
		t.Fatalf("one line: %v", err)
	}
}

func TestJournalEntryLineSideInvalid(t *testing.T) {
	l := newLedger(t)
	for name, mut := range map[string]func(*accounting.JournalEntryLine){
		"both":     func(x *accounting.JournalEntryLine) { x.Debit, x.Credit = 5, 5 },
		"neither":  func(x *accounting.JournalEntryLine) { x.Debit, x.Credit = 0, 0 },
		"negative": func(x *accounting.JournalEntryLine) { x.Debit, x.Credit = -5, 0 },
	} {
		f := l.journal("2026-03-31")
		mut(&f.Lines[1])
		_, err := l.s.CreateJournalEntry(l.core.Admin, accounting.NewJournalEntry{JournalEntryFields: f})
		e, ok := asErr(err)
		if !ok || e.Code != "line_side_invalid" || e.Params["line_no"] != 2 {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestJournalEntryBeforeStartRefused(t *testing.T) {
	l := newLedger(t)
	ref := l.newJournal(t, l.journal("2025-12-31"))
	if err := l.post(ref); !isCode(err, "accounting_not_started") {
		t.Fatalf("before start: %v", err)
	}
	start := "2026-01-01"
	if err := l.s.SetStart(l.core.Admin, l.le, nil); err != nil {
		t.Fatal(err)
	}
	if err := l.post(l.newJournal(t, l.journal("2026-03-31"))); !isCode(err, "accounting_not_started") {
		t.Fatalf("unset: %v", err)
	}
	if err := l.s.SetStart(l.core.Admin, l.le, &start); err != nil {
		t.Fatal(err)
	}
}

func TestJournalEntryNotPostableNamesLine(t *testing.T) {
	l := newLedger(t)
	f := l.journal("2026-03-31")
	parent := mustCreate(t, l.s, l.core.Admin, acct("111", "asset", nil))
	mustCreate(t, l.s, l.core.Admin, acct("1111", "asset", &parent))
	f.Lines[1].AccountID = parent
	err := l.post(l.newJournal(t, f))
	if e, ok := asErr(err); !ok || e.Code != "account_not_postable" || e.Params["line_no"] != 2 {
		t.Fatalf("parent account: %v", err)
	}
}

func TestJournalEntryCancelReverses(t *testing.T) {
	l := newLedger(t)
	ref := l.newJournal(t, l.journal("2026-03-31"))
	if err := l.post(ref); err != nil {
		t.Fatal(err)
	}
	var debit, credit int64
	sum := func() {
		if err := l.core.Pool.QueryRow(l.core.Ctx, `SELECT COALESCE(sum(l.debit), 0), COALESCE(sum(l.credit), 0) FROM accounting.entries e
			JOIN accounting.entry_lines l ON l.entry_id = e.id WHERE e.doc_type = $1 AND e.doc_id = $2 AND e.reverses IS NULL`, ref.Type, ref.ID).Scan(&debit, &credit); err != nil {
			t.Fatal(err)
		}
	}
	if sum(); debit != 100 || credit != 100 {
		t.Fatalf("posted %d %d", debit, credit)
	}
	e, err := l.s.JournalEntry(l.core.Admin, ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.core.Record.Transition(l.core.Admin, ref, e.Version, record.Cancelled); err != nil {
		t.Fatal(err)
	}
	var reversals int
	if err := l.core.Pool.QueryRow(l.core.Ctx, `SELECT count(*) FROM accounting.entries WHERE doc_id = $1 AND reverses IS NOT NULL`, ref.ID).Scan(&reversals); err != nil || reversals != 1 {
		t.Fatalf("reversals %d %v", reversals, err)
	}
}

func asErr(err error) (*platform.Error, bool) {
	var e *platform.Error
	return e, errors.As(err, &e)
}

func isCode(err error, code string) bool {
	e, ok := asErr(err)
	return ok && e.Code == code
}
