package accounting_test

import (
	"context"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/record/recordtest"
	"github.com/taoworklabs/mmerp/internal/modules/accounting"
	"github.com/taoworklabs/mmerp/internal/platform"
)

const closingType = "accounting.closing"

// closingLedger adds revenue 511, cash 111, result 911 and retained earnings 4212, and sets
// the closing accounts.
func closingLedger(t *testing.T) *ledger {
	t.Helper()
	l := newLedger(t)
	l.acc["511"] = mustCreate(t, l.s, l.core.Admin, acct("511", "revenue", nil))
	l.acc["111"] = mustCreate(t, l.s, l.core.Admin, acct("111", "asset", nil))
	l.acc["911"] = mustCreate(t, l.s, l.core.Admin, acct("911", "equity", nil))
	l.acc["4212"] = mustCreate(t, l.s, l.core.Admin, acct("4212", "equity", nil))
	if err := l.s.SetClosingAccounts(l.core.Admin, accounting.ClosingAccounts{ResultAccountID: l.acc["911"], RetainedAccountID: l.acc["4212"]}); err != nil {
		t.Fatal(err)
	}
	return l
}

// book posts a manual entry: amount from credit to debit, debit on unit when it is set.
func (l *ledger) book(t *testing.T, date, debit, credit string, amount int64, unit *int64) {
	t.Helper()
	ref := l.newJournal(t, accounting.JournalEntryFields{Date: date, LegalEntityID: l.le, Description: "Bút toán", Lines: []accounting.JournalEntryLine{
		{AccountID: l.acc[debit], OrgUnitID: unit, Debit: amount},
		{AccountID: l.acc[credit], Credit: amount},
	}})
	if err := l.post(ref); err != nil {
		t.Fatal(err)
	}
}

func (l *ledger) newClosing(ctx context.Context, date string) (record.Ref, error) {
	id, err := l.s.CreateClosing(ctx, accounting.NewClosing{ClosingFields: accounting.ClosingFields{Date: date, LegalEntityID: l.le, Description: "Kết chuyển"}})
	return record.Ref{Type: closingType, ID: id}, err
}

func (l *ledger) postClosing(ref record.Ref) error {
	c, err := l.s.Closing(l.core.Admin, ref.ID)
	if err != nil {
		return err
	}
	_, err = l.core.Record.Transition(l.core.Admin, ref, c.Version, record.Posted)
	return err
}

// balances are the closing balances, debit minus credit, by account code, up to date.
func (l *ledger) balances(t *testing.T, date string) map[string]int64 {
	t.Helper()
	tb, err := l.s.TrialBalance(l.core.Admin, l.le, "2026-01-01", date)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]int64{}
	for _, r := range tb.Rows {
		out[r.Code] = r.Closing
	}
	return out
}

func TestClosingMovesResultToRetainedEarnings(t *testing.T) {
	l := closingLedger(t)
	l.book(t, "2026-03-10", "111", "511", 1_000, nil)
	l.book(t, "2026-03-15", "642", "111", 300, &l.unit)

	ref, err := l.newClosing(l.core.Admin, "2026-03-31")
	if err != nil {
		t.Fatal(err)
	}
	// The draft previews what posting would write: 511 and 642 cleared through 911, by org unit.
	c, err := l.s.Closing(l.core.Admin, ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Lines) != 8 {
		t.Fatalf("preview %+v", c.Lines)
	}
	if err := l.postClosing(ref); err != nil {
		t.Fatal(err)
	}
	b := l.balances(t, "2026-03-31")
	if b["511"] != 0 || b["642"] != 0 || b["911"] != 0 || b["4212"] != -700 || b["111"] != 700 {
		t.Fatalf("after closing %v", b)
	}

	// A document backdated into the month: a second closing moves only what is left.
	l.book(t, "2026-03-20", "642", "111", 50, &l.unit)
	second, err := l.newClosing(l.core.Admin, "2026-03-31")
	if err != nil {
		t.Fatal(err)
	}
	if err := l.postClosing(second); err != nil {
		t.Fatal(err)
	}
	if b := l.balances(t, "2026-03-31"); b["642"] != 0 || b["911"] != 0 || b["4212"] != -650 {
		t.Fatalf("after second closing %v", b)
	}

	// Cancelling the first closing reverses it; the second still moved its 50.
	c, err = l.s.Closing(l.core.Admin, ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.core.Record.Transition(l.core.Admin, ref, c.Version, record.Cancelled); err != nil {
		t.Fatal(err)
	}
	if b := l.balances(t, "2026-03-31"); b["511"] != -1_000 || b["642"] != 300 || b["911"] != 0 || b["4212"] != 50 {
		t.Fatalf("after cancelling %v", b)
	}
}

func TestClosingRefusals(t *testing.T) {
	l := newLedger(t)
	ref, err := l.newClosing(l.core.Admin, "2026-03-31")
	if err != nil {
		t.Fatal(err)
	}
	if err := l.postClosing(ref); !isCode(err, "closing_accounts_unset") {
		t.Fatalf("no closing accounts: %v", err)
	}

	l = closingLedger(t)
	ref, err = l.newClosing(l.core.Admin, "2026-03-31")
	if err != nil {
		t.Fatal(err)
	}
	if err := l.postClosing(ref); !isCode(err, "closing_nothing_to_close") {
		t.Fatalf("nothing to close: %v", err)
	}
	early, err := l.newClosing(l.core.Admin, "2025-12-31")
	if err != nil {
		t.Fatal(err)
	}
	if err := l.postClosing(early); !isCode(err, "accounting_not_started") {
		t.Fatalf("before the start date: %v", err)
	}
	// Closing accounts must be postable.
	parent := mustCreate(t, l.s, l.core.Admin, acct("42", "equity", nil))
	mustCreate(t, l.s, l.core.Admin, acct("421", "equity", &parent))
	if err := l.s.SetClosingAccounts(l.core.Admin, accounting.ClosingAccounts{ResultAccountID: l.acc["911"], RetainedAccountID: parent}); !isCode(err, "account_not_postable") {
		t.Fatalf("parent as retained earnings: %v", err)
	}
}

func TestClosingRecordContract(t *testing.T) {
	recordtest.Run(t, func(t *testing.T) recordtest.Harness {
		l := closingLedger(t)
		l.book(t, "2026-01-05", "111", "511", 1_000, nil)
		approver, err := l.core.IAM.CreateUser(l.core.Admin, "approver", "Approver", "long enough")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := l.core.IAM.GrantRole(l.core.Admin, approver, "accounting", "admin", nil); err != nil {
			t.Fatal(err)
		}
		actor := platform.WithActor(l.core.Ctx, approver)
		return recordtest.Harness{
			RequireApproval: func(t *testing.T, a record.Action) {
				if err := l.core.Approval.SaveRule(l.core.Admin, closingType, a, approval.RuleInput{
					Flow:      approval.Flow{{Step: &approval.StepDef{Approver: approval.Approver{Kind: "role", Product: "accounting", Role: "admin"}}}},
					MaxLevels: 3, FallbackProduct: "accounting", FallbackRole: "admin",
				}); err != nil {
					t.Fatal(err)
				}
			},
			Approve: func(instance int64) error { return l.core.Approval.Approve(actor, instance, 1) },
			Reject:  func(instance int64) error { return l.core.Approval.Reject(actor, instance, 1, "không") },
			Core:    l.core, Actor: l.core.Admin, Admin: l.core.Admin, LegalEntity: l.le, Table: "accounting.closings",
			Get: func(ctx context.Context, r record.Ref) (any, error) { return l.s.Closing(ctx, r.ID) },
			Create: func(ctx context.Context, date, requestID string) (record.Ref, error) {
				id, err := l.s.CreateClosing(ctx, accounting.NewClosing{RequestID: requestID,
					ClosingFields: accounting.ClosingFields{Date: date, LegalEntityID: l.le, Description: "Kết chuyển"}})
				return record.Ref{Type: closingType, ID: id}, err
			},
			Edit: func(ctx context.Context, r record.Ref, version int32, date string) error {
				return l.s.UpdateClosing(ctx, r.ID, accounting.ClosingUpdate{Version: version,
					ClosingFields: accounting.ClosingFields{Date: date, LegalEntityID: l.le, Description: "Kết chuyển"}})
			},
			// An inactive revenue account still holding a balance is refused at posting.
			BreakPosting: func(t *testing.T, _ record.Ref) {
				f := acct("511", "revenue", nil)
				f.Active = false
				if err := l.s.UpdateAccount(l.core.Admin, l.acc["511"], f); err != nil {
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
