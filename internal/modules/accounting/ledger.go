package accounting

import (
	"context"
	"slices"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/accounting/internal/store"
)

// writeEntry is the only way a new journal entry reaches the ledger: balanced, at least two
// lines, every account postable and active (share-locked, so the chart cannot change under it),
// a partner on exactly the lines of accounts tracked by partner.
func writeEntry(ctx context.Context, legalEntity int64, date pgtype.Date, src record.Ref, recording *int64, lines []JournalLine) (int64, error) {
	var debit, credit int64
	ids := make([]int64, 0, len(lines))
	for _, l := range lines {
		if l.Debit > MaxLineAmount || l.Credit > MaxLineAmount {
			return 0, ErrEntryUnbalanced
		}
		debit, credit = debit+l.Debit, credit+l.Credit
		ids = append(ids, l.AccountID)
	}
	if len(lines) > MaxLines || len(lines) < 2 || debit != credit {
		return 0, ErrEntryUnbalanced
	}
	slices.Sort(ids)
	ids = slices.Compact(ids)
	if err := store.ShareAccounts.Run(ctx, ids); err != nil {
		return 0, err
	}
	ok, err := postableActive(ctx, ids)
	if err != nil {
		return 0, err
	}
	for _, id := range ids {
		if !ok[id] {
			return 0, ErrAccountNotPostable
		}
	}
	tracked, err := accountFlags(ctx, ids)
	if err != nil {
		return 0, err
	}
	for _, l := range lines {
		if tracked[l.AccountID] != (l.PartnerID != nil) {
			return 0, ErrEntryPartnerMismatch
		}
	}
	rec := pgtype.Int8{}
	if recording != nil {
		rec = pgtype.Int8{Int64: *recording, Valid: true}
	}
	id, err := store.InsertEntry.Run(ctx, store.InsertEntryParams{LegalEntityID: legalEntity, Date: date, DocType: src.Type, DocID: src.ID, RecordingID: rec})
	if err != nil {
		return 0, err
	}
	p := store.InsertEntryLinesParams{EntryID: id}
	for _, l := range lines {
		var unit int64
		if l.OrgUnitID != nil {
			unit = *l.OrgUnitID
		}
		p.AccountIds = append(p.AccountIds, l.AccountID)
		p.OrgUnitIds = append(p.OrgUnitIds, unit)
		var partner int64
		if l.PartnerID != nil {
			partner = *l.PartnerID
		}
		p.PartnerIds = append(p.PartnerIds, partner)
		p.Kinds = append(p.Kinds, l.Kind)
		p.Descriptions = append(p.Descriptions, l.Description)
		p.Debits = append(p.Debits, l.Debit)
		p.Credits = append(p.Credits, l.Credit)
	}
	return id, store.InsertEntryLines.Run(ctx, p)
}

// reverseEntry writes the mirror of an entry: same date and source, sides swapped. It copies
// the original lines and never maps again, so accounts closed since are still reversed.
func reverseEntry(ctx context.Context, entryID int64) (int64, error) {
	id, err := store.ReverseEntry.Run(ctx, entryID)
	if err != nil {
		return 0, err
	}
	return id, store.ReverseEntryLines.Run(ctx, store.ReverseEntryLinesParams{Reversal: id, EntryID: entryID})
}
