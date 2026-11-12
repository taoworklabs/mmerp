package accounting

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/accounting/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// PostingChanged is the posting hook: it queues a reconcile of ref in the caller's
// transaction. With accounting off it does nothing, so other products still post.
func (s *Service) PostingChanged(ctx context.Context, ref record.Ref) error {
	if platform.ProductGate(ctx, "accounting", platform.ClassWrite) != nil {
		return nil
	}
	_, err := platform.Enqueue(ctx, reconcileArgs{DocType: ref.Type, DocID: ref.ID})
	return err
}

// Reconcile brings the ledger in line with ref's current posting lines. It is a system
// entry point: no permission is checked.
func (s *Service) Reconcile(ctx context.Context, ref record.Ref) error {
	return platform.InTx(ctx, func(ctx context.Context) error { return s.reconcile(ctx, ref) })
}

func (s *Service) reconcile(ctx context.Context, ref record.Ref) error {
	d, err := s.d.Record.Lock(ctx, ref)
	if err != nil {
		return err
	}
	start, err := store.ShareStart.Run(ctx, d.LegalEntityID)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && d.Date < start.Time.Format(time.DateOnly) {
		return nil
	}
	if err != nil {
		return err
	}
	live, err := store.LiveRecording.Run(ctx, store.LiveRecordingParams{DocType: ref.Type, DocID: ref.ID})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	liveID := pgtype.Int8{Int64: live, Valid: err == nil}
	date := platform.NullDate(&d.Date)
	posted, err := store.LockSource.Run(ctx, store.LockSourceParams{DocType: ref.Type, DocID: ref.ID, LegalEntityID: d.LegalEntityID, Date: date})
	if err != nil {
		return err
	}
	src := store.SaveSourceParams{DocType: ref.Type, DocID: ref.ID, PostedRecordingID: posted, MissingKinds: []string{}}
	save := func() error { return store.SaveSource.Run(ctx, src) }
	if posted == liveID {
		return save()
	}
	// What the ledger lacks for a document (its first posting, or a reversal) may land in a
	// period locked while accounting was off; SetStart keeps any other way out.
	if posted.Valid {
		entry, err := store.EntryOfRecording.Run(ctx, posted)
		if err == nil {
			_, err = reverseEntry(ctx, entry)
		}
		// No entry: the recording had nothing to book.
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		src.PostedRecordingID = pgtype.Int8{}
	}
	if !liveID.Valid {
		return save()
	}
	lines, missing, err := journal(ctx, ref)
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		src.Problem, src.MissingKinds = pgtype.Text{String: "unmapped", Valid: true}, missing
		return save()
	}
	// Checked first: within the bounds the sums below cannot overflow.
	if len(lines) > MaxLines || slices.ContainsFunc(lines, func(l JournalLine) bool { return l.Debit > MaxLineAmount || l.Credit > MaxLineAmount }) {
		src.Problem = pgtype.Text{String: "too_large", Valid: true}
		return save()
	}
	var debit, credit int64
	for _, l := range lines {
		debit, credit = debit+l.Debit, credit+l.Credit
	}
	if debit != credit {
		src.Problem = pgtype.Text{String: "unbalanced", Valid: true}
		return save()
	}
	if len(lines) > 0 {
		if _, err := writeEntry(ctx, d.LegalEntityID, date, ref, &liveID.Int64, lines); err != nil {
			return err
		}
	}
	src.PostedRecordingID = liveID
	return save()
}

// journal maps ref's live posting lines through the rules, one line per account, side, org unit,
// partner and kind; missing lists, sorted, the kinds without a rule or whose account is not postable
// and active. An amount past MaxLineAmount stops at MaxLineAmount+1, for the caller to refuse.
func journal(ctx context.Context, ref record.Ref) ([]JournalLine, []string, error) {
	rows, err := store.LiveLines.Run(ctx, store.LiveLinesParams{DocType: ref.Type, DocID: ref.ID})
	if err != nil {
		return nil, nil, err
	}
	kinds := make([]string, 0, len(rows))
	for _, r := range rows {
		kinds = append(kinds, r.Kind)
	}
	rules, err := store.RuleOf.Run(ctx, kinds)
	if err != nil {
		return nil, nil, err
	}
	byKind := make(map[string]store.AccountingRule, len(rules))
	ids := make([]int64, 0, len(rules))
	for _, r := range rules {
		byKind[r.Kind] = r
		ids = append(ids, r.AccountID)
	}
	// Locked as writeEntry will, so a concurrent deactivation shows here as unmapped.
	slices.Sort(ids)
	if err := store.ShareAccounts.Run(ctx, ids); err != nil {
		return nil, nil, err
	}
	ok, err := postableActive(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	type key struct {
		account, unit, partner int64
		debit                  bool
		kind                   string
	}
	at := map[key]int{}
	var lines []JournalLine
	var missing []string
	for _, r := range rows {
		rule, found := byKind[r.Kind]
		if !found || !ok[rule.AccountID] {
			missing = append(missing, r.Kind)
			continue
		}
		if r.Amount == 0 {
			continue
		}
		// A negative amount goes on the opposite side.
		amount, debit := r.Amount, rule.Side == "debit"
		if amount < 0 {
			amount, debit = -amount, !debit
		}
		// The negation of the smallest int64 stays negative.
		if amount < 0 || amount > MaxLineAmount {
			amount = MaxLineAmount + 1
		}
		k := key{rule.AccountID, r.OrgUnitID, r.PartnerID.Int64, debit, r.Kind}
		i, ok := at[k]
		if !ok {
			i, at[k] = len(lines), len(lines)
			lines = append(lines, JournalLine{AccountID: rule.AccountID, OrgUnitID: &r.OrgUnitID, PartnerID: platform.Int8Ptr(r.PartnerID), Kind: r.Kind})
		}
		l := &lines[i]
		// Both terms are at most MaxLineAmount+1, so the sum cannot overflow.
		if debit {
			l.Debit = min(l.Debit+amount, MaxLineAmount+1)
		} else {
			l.Credit = min(l.Credit+amount, MaxLineAmount+1)
		}
	}
	slices.Sort(missing)
	return lines, slices.Compact(missing), nil
}
