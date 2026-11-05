package accounting

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/accounting/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

const journalEntryType = "accounting.journal_entry"

func (s *Service) registerJournalEntry() {
	s.d.Record.Register(record.Type{Code: journalEntryType, Product: "accounting", Kind: record.Document, NumberPrefix: "PK",
		Perms:        record.Perms{View: PermView, Edit: PermEntryEdit, ViewFiles: PermView, Attach: PermEntryEdit},
		OnTransition: journalEntryTransition,
		BeforeSubmit: func(ctx context.Context, d record.Doc) error { _, err := checkJournalEntry(ctx, d); return err },
		OnDelete:     deleteJournalEntry,
	})
}

// JournalEntryActions lists what the actor may do before picking an entry: create.
func (s *Service) JournalEntryActions(ctx context.Context) (JournalEntryActions, error) {
	acts, err := s.d.Record.TypeActions(ctx, journalEntryType)
	return JournalEntryActions{AllowedActions: acts}, err
}

func (s *Service) JournalEntries(ctx context.Context, f JournalEntryFilter) (JournalEntryList, error) {
	out := JournalEntryList{Items: []JournalEntryListItem{}}
	sc, err := s.d.IAM.DataScope(ctx, "accounting", PermView)
	if err != nil {
		return out, err
	}
	page, err := journalEntryList.Run(ctx, journalEntryQuery(sc, f))
	if err != nil {
		return out, err
	}
	out.Total = page.Total
	for _, r := range page.Items {
		out.Items = append(out.Items, JournalEntryListItem{ID: r.ID, Number: r.Number, Status: r.Status, Date: *platform.DatePtr(r.Date),
			LegalEntityID: r.LegalEntityID, Description: r.Description, Total: r.Total})
	}
	return out, nil
}

// JournalEntry returns one entry; one the actor may not view does not exist.
func (s *Service) JournalEntry(ctx context.Context, id int64) (JournalEntry, error) {
	d, err := s.d.Record.Read(ctx, record.Ref{Type: journalEntryType, ID: id})
	if err != nil {
		return JournalEntry{}, err
	}
	out, err := readJournalEntry(ctx, id)
	if err != nil {
		return out, err
	}
	out.AllowedActions, err = s.d.Record.Actions(ctx, d)
	return out, err
}

func readJournalEntry(ctx context.Context, id int64) (JournalEntry, error) {
	h, err := store.GetJournalEntry.Run(ctx, id)
	if err != nil {
		return JournalEntry{}, err
	}
	lines, err := store.GetJournalEntryLines.Run(ctx, id)
	if err != nil {
		return JournalEntry{}, err
	}
	out := JournalEntry{ID: id, Number: h.Number, Status: h.Status, Version: h.Version, Date: *platform.DatePtr(h.Date),
		LegalEntityID: h.LegalEntityID, Description: h.Description, Lines: make([]JournalEntryLineOut, len(lines))}
	for i, l := range lines {
		out.Lines[i] = JournalEntryLineOut{AccountCode: l.AccountCode, AccountName: l.AccountName, JournalEntryLine: JournalEntryLine{
			AccountID: l.AccountID, OrgUnitID: platform.Int8Ptr(l.OrgUnitID), Description: platform.TextPtr(l.Description), Debit: l.Debit, Credit: l.Credit}}
		out.TotalDebit, out.TotalCredit = out.TotalDebit+l.Debit, out.TotalCredit+l.Credit
	}
	return out, nil
}

// CreateJournalEntry adds a draft; the same request id returns the first one.
func (s *Service) CreateJournalEntry(ctx context.Context, in NewJournalEntry) (id int64, err error) {
	return id, platform.InTx(ctx, func(ctx context.Context) error {
		if err := s.d.Record.MayCreate(ctx, journalEntryType, record.Header{OrgUnitID: in.LegalEntityID, Date: in.Date}); err != nil {
			return err
		}
		h, err := prepareJournalEntry(ctx, s, &in.JournalEntryFields)
		if err != nil {
			return err
		}
		h.RequestID = in.RequestID
		d, err := s.d.Record.Create(ctx, journalEntryType, h)
		if id = d.ID; err != nil || d.Reused {
			return err
		}
		if err := store.InsertJournalEntry.Run(ctx, store.InsertJournalEntryParams{ID: id, Description: in.Description}); err != nil {
			return err
		}
		return s.writeJournalEntryLines(ctx, id, in.JournalEntryFields)
	})
}

func (s *Service) UpdateJournalEntry(ctx context.Context, id int64, in JournalEntryUpdate) error {
	ref := record.Ref{Type: journalEntryType, ID: id}
	return platform.InTx(ctx, func(ctx context.Context) error {
		if _, err := s.d.Record.Read(ctx, ref); err != nil {
			return err
		}
		h, err := prepareJournalEntry(ctx, s, &in.JournalEntryFields)
		if err != nil {
			return err
		}
		if _, err := s.d.Record.Edit(ctx, ref, in.Version, h); err != nil {
			return err
		}
		if err := store.UpdateJournalEntry.Run(ctx, store.UpdateJournalEntryParams{ID: id, Description: in.Description}); err != nil {
			return err
		}
		if err := store.DeleteJournalEntryLines.Run(ctx, id); err != nil {
			return err
		}
		return s.writeJournalEntryLines(ctx, id, in.JournalEntryFields)
	})
}

// prepareJournalEntry checks what a draft can check before posting: the legal entity, one
// side per line, accounts and org units that exist. Balance and postability wait for posting.
func prepareJournalEntry(ctx context.Context, s *Service, f *JournalEntryFields) (record.Header, error) {
	if f.Description = strings.TrimSpace(f.Description); f.Description == "" {
		return record.Header{}, ErrEntryDescriptionRequired
	}
	ok, err := store.IsLegalEntity.Run(ctx, f.LegalEntityID)
	if err != nil {
		return record.Header{}, err
	}
	if !ok {
		return record.Header{}, platform.ErrNotFound
	}
	if len(f.Lines) > MaxLines {
		return record.Header{}, ErrTooManyLines
	}
	var accounts, units []int64
	var total int64
	for i, l := range f.Lines {
		if l.Debit > MaxLineAmount || l.Credit > MaxLineAmount {
			return record.Header{}, lineErr(ErrLineAmountInvalid, i+1)
		}
		if l.Debit < 0 || l.Credit < 0 || (l.Debit == 0) == (l.Credit == 0) {
			return record.Header{}, lineErr(ErrLineSideInvalid, i+1)
		}
		total += l.Debit
		accounts = append(accounts, l.AccountID)
		if l.OrgUnitID != nil {
			units = append(units, *l.OrgUnitID)
		}
	}
	if n, err := store.CountAccounts.Run(ctx, accounts); err != nil {
		return record.Header{}, err
	} else if int(n) != len(uniq(accounts)) {
		return record.Header{}, platform.ErrNotFound
	}
	if n, err := store.CountOrgUnits.Run(ctx, units); err != nil {
		return record.Header{}, err
	} else if int(n) != len(uniq(units)) {
		return record.Header{}, platform.ErrNotFound
	}
	for i, l := range f.Lines {
		if l.OrgUnitID == nil {
			continue
		}
		if le, err := s.d.IAM.LegalEntityOf(ctx, *l.OrgUnitID); err != nil || le != f.LegalEntityID {
			return record.Header{}, lineErr(ErrLineOrgUnitInvalid, i+1)
		}
	}
	return record.Header{Date: f.Date, OrgUnitID: f.LegalEntityID, Amount: &total}, nil
}

func uniq(ids []int64) []int64 {
	ids = slices.Clone(ids)
	slices.Sort(ids)
	return slices.Compact(ids)
}

func lineErr(e *platform.Error, lineNo int) error {
	return &platform.Error{Status: e.Status, Code: e.Code, Params: map[string]any{"line_no": lineNo}}
}

func (s *Service) writeJournalEntryLines(ctx context.Context, id int64, f JournalEntryFields) error {
	p := store.InsertJournalEntryLinesParams{JournalEntryID: id}
	for _, l := range f.Lines {
		var unit int64
		if l.OrgUnitID != nil {
			unit = *l.OrgUnitID
		}
		var desc string
		if l.Description != nil {
			desc = strings.TrimSpace(*l.Description)
		}
		p.AccountIds = append(p.AccountIds, l.AccountID)
		p.OrgUnitIds = append(p.OrgUnitIds, unit)
		p.Descriptions = append(p.Descriptions, desc)
		p.Debits = append(p.Debits, l.Debit)
		p.Credits = append(p.Credits, l.Credit)
	}
	if err := store.InsertJournalEntryLines.Run(ctx, p); err != nil {
		return err
	}
	return s.d.Audit.RecordFor(ctx, "accounting.journal_entry_saved", audit.Ref{Type: journalEntryType, ID: id}, map[string]any{"lines": len(f.Lines)})
}

// deleteJournalEntry deletes a draft's rows, lines first.
func deleteJournalEntry(ctx context.Context, id int64) error {
	if err := store.DeleteJournalEntryLines.Run(ctx, id); err != nil {
		return err
	}
	return store.DeleteJournalEntry.Run(ctx, id)
}

// journalEntryTransition writes the ledger entry on posting and reverses it on cancelling.
func journalEntryTransition(ctx context.Context, d record.Doc, _ record.Status) error {
	switch d.Status {
	case record.Posted:
		return postJournalEntry(ctx, d)
	case record.Cancelled:
		id, err := store.LiveEntryOfDoc.Run(ctx, store.LiveEntryOfDocParams{DocType: d.Type, DocID: d.ID})
		if err != nil {
			return err
		}
		_, err = reverseEntry(ctx, id)
		return err
	}
	return nil
}

func postJournalEntry(ctx context.Context, d record.Doc) error {
	lines, err := checkJournalEntry(ctx, d)
	if err != nil {
		return err
	}
	_, err = writeEntry(ctx, d.LegalEntityID, platform.NullDate(&d.Date), d.Ref, nil, lines)
	return err
}

// checkJournalEntry runs what posting needs, so sending an entry refuses what posting would:
// accounting started, at least two balanced lines, postable accounts. It returns the lines.
func checkJournalEntry(ctx context.Context, d record.Doc) ([]JournalLine, error) {
	e, err := readJournalEntry(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	// Read through the starts row so a start date change waits for this posting.
	start, err := store.ShareStart.Run(ctx, d.LegalEntityID)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && d.Date < start.Time.Format(time.DateOnly) {
		return nil, ErrAccountingNotStarted
	}
	if err != nil {
		return nil, err
	}
	var debit, credit int64
	lines := make([]JournalLine, len(e.Lines))
	ids := make([]int64, len(e.Lines))
	for i, l := range e.Lines {
		debit, credit = debit+l.Debit, credit+l.Credit
		ids[i] = l.AccountID
		lines[i] = JournalLine{AccountID: l.AccountID, OrgUnitID: l.OrgUnitID, Debit: l.Debit, Credit: l.Credit}
		if l.Description != nil {
			lines[i].Description = *l.Description
		}
	}
	if len(lines) < 2 || debit != credit {
		return nil, ErrJournalEntryUnbalanced
	}
	// Names the first bad line; writeEntry checks again under the account locks.
	ok, err := postableActive(ctx, uniq(ids))
	if err != nil {
		return nil, err
	}
	for i, id := range ids {
		if !ok[id] {
			return nil, lineErr(ErrAccountNotPostable, i+1)
		}
	}
	return lines, nil
}
