package accounting

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/accounting/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

const closingType = "accounting.closing"

func (s *Service) registerClosing() {
	s.d.Record.Register(record.Type{Code: closingType, Product: "accounting", Kind: record.Document, NumberPrefix: "KC",
		Perms:        record.Perms{View: PermView, Edit: PermEntryEdit, ViewFiles: PermView, Attach: PermEntryEdit},
		OnTransition: closingTransition,
		BeforeSubmit: func(ctx context.Context, d record.Doc) error { _, err := checkClosing(ctx, d); return err },
		OnDelete:     store.DeleteClosing.Run,
	})
}

// ClosingAccounts returns where closings post; not found until they are set.
func (s *Service) ClosingAccounts(ctx context.Context) (ClosingAccountsOut, error) {
	if err := s.requireView(ctx); err != nil {
		return ClosingAccountsOut{}, err
	}
	r, err := store.GetClosingAccounts.Run(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return ClosingAccountsOut{}, platform.ErrNotFound
	}
	return ClosingAccountsOut{ClosingAccounts: ClosingAccounts{ResultAccountID: r.ResultID, RetainedAccountID: r.RetainedID},
		ResultCode: r.ResultCode, ResultName: r.ResultName, RetainedCode: r.RetainedCode, RetainedName: r.RetainedName}, err
}

// SetClosingAccounts sets both accounts; each must be postable and active.
func (s *Service) SetClosingAccounts(ctx context.Context, in ClosingAccounts) error {
	return s.writeAccount(ctx, func(ctx context.Context) error {
		ids := []int64{in.ResultAccountID, in.RetainedAccountID}
		if n, err := store.CountAccounts.Run(ctx, ids); err != nil {
			return err
		} else if int(n) != len(uniq(ids)) {
			return platform.ErrNotFound
		}
		ok, err := postableActive(ctx, ids)
		if err != nil {
			return err
		}
		if !ok[in.ResultAccountID] || !ok[in.RetainedAccountID] {
			return ErrAccountNotPostable
		}
		if err := store.SaveClosingAccounts.Run(ctx, store.SaveClosingAccountsParams{ResultAccountID: in.ResultAccountID, RetainedAccountID: in.RetainedAccountID}); err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "accounting.closing_accounts_saved", map[string]any{"result_account_id": in.ResultAccountID, "retained_account_id": in.RetainedAccountID})
	})
}

func (s *Service) ClosingActions(ctx context.Context) (ClosingActions, error) {
	acts, err := s.d.Record.TypeActions(ctx, closingType)
	return ClosingActions{AllowedActions: acts}, err
}

func (s *Service) Closings(ctx context.Context, f ClosingFilter) (ClosingList, error) {
	out := ClosingList{Items: []ClosingListItem{}}
	sc, err := s.d.IAM.DataScope(ctx, "accounting", PermView)
	if err != nil {
		return out, err
	}
	page, err := closingList.Run(ctx, closingQuery(sc, f))
	if err != nil {
		return out, err
	}
	out.Total = page.Total
	for _, r := range page.Items {
		out.Items = append(out.Items, ClosingListItem{ID: r.ID, Number: r.Number, Status: r.Status, Date: *platform.DatePtr(r.Date),
			LegalEntityID: r.LegalEntityID, Description: r.Description})
	}
	return out, nil
}

// Closing returns one closing entry; one the actor may not view does not exist.
func (s *Service) Closing(ctx context.Context, id int64) (Closing, error) {
	d, err := s.d.Record.Read(ctx, record.Ref{Type: closingType, ID: id})
	if err != nil {
		return Closing{}, err
	}
	h, err := store.GetClosing.Run(ctx, id)
	if err != nil {
		return Closing{}, err
	}
	out := Closing{ID: id, Number: h.Number, Status: h.Status, Version: h.Version, Date: *platform.DatePtr(h.Date),
		LegalEntityID: h.LegalEntityID, Description: h.Description, Lines: []ClosingLine{}}
	if d.Status == record.Posted || d.Status == record.Cancelled {
		rows, err := store.OriginalEntryLines.Run(ctx, store.OriginalEntryLinesParams{DocType: closingType, DocID: id})
		if err != nil {
			return Closing{}, err
		}
		for _, r := range rows {
			out.Lines = append(out.Lines, ClosingLine{AccountID: r.AccountID, AccountCode: r.AccountCode, AccountName: r.AccountName,
				OrgUnitID: platform.Int8Ptr(r.OrgUnitID), PartnerID: platform.Int8Ptr(r.PartnerID), Debit: r.Debit, Credit: r.Credit})
		}
	} else {
		// The preview shows nothing where posting would refuse; sending says why.
		lines, err := closingLines(ctx, d.LegalEntityID, h.Date.Time)
		var perr *platform.Error
		if err != nil && !errors.As(err, &perr) {
			return Closing{}, err
		}
		if err == nil {
			out.Lines = lines
		}
	}
	out.AllowedActions, err = s.d.Record.Actions(ctx, d)
	return out, err
}

func (s *Service) CreateClosing(ctx context.Context, in NewClosing) (id int64, err error) {
	return id, platform.InTx(ctx, func(ctx context.Context) error {
		h := record.Header{OrgUnitID: in.LegalEntityID, Date: in.Date, RequestID: in.RequestID}
		if err := s.d.Record.MayCreate(ctx, closingType, h); err != nil {
			return err
		}
		if err := prepareClosing(ctx, &in.ClosingFields); err != nil {
			return err
		}
		d, err := s.d.Record.Create(ctx, closingType, h)
		if id = d.ID; err != nil || d.Reused {
			return err
		}
		return store.InsertClosing.Run(ctx, store.InsertClosingParams{ID: id, Description: in.Description})
	})
}

func (s *Service) UpdateClosing(ctx context.Context, id int64, in ClosingUpdate) error {
	ref := record.Ref{Type: closingType, ID: id}
	return platform.InTx(ctx, func(ctx context.Context) error {
		if _, err := s.d.Record.Read(ctx, ref); err != nil {
			return err
		}
		if err := prepareClosing(ctx, &in.ClosingFields); err != nil {
			return err
		}
		if _, err := s.d.Record.Edit(ctx, ref, in.Version, record.Header{Date: in.Date, OrgUnitID: in.LegalEntityID}); err != nil {
			return err
		}
		return store.UpdateClosing.Run(ctx, store.UpdateClosingParams{ID: id, Description: in.Description})
	})
}

func prepareClosing(ctx context.Context, f *ClosingFields) error {
	if f.Description = strings.TrimSpace(f.Description); f.Description == "" {
		return ErrEntryDescriptionRequired
	}
	ok, err := store.IsLegalEntity.Run(ctx, f.LegalEntityID)
	if err != nil {
		return err
	}
	if !ok {
		return platform.ErrNotFound
	}
	return nil
}

func closingTransition(ctx context.Context, d record.Doc, _ record.Status) error {
	switch d.Status {
	case record.Posted:
		lines, err := checkClosing(ctx, d)
		if err != nil {
			return err
		}
		journal := make([]JournalLine, len(lines))
		for i, l := range lines {
			journal[i] = JournalLine{AccountID: l.AccountID, OrgUnitID: l.OrgUnitID, PartnerID: l.PartnerID, Debit: l.Debit, Credit: l.Credit}
		}
		_, err = writeEntry(ctx, d.LegalEntityID, platform.NullDate(&d.Date), d.Ref, nil, journal)
		return err
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

// checkClosing refuses what posting would refuse and returns the lines posting writes.
func checkClosing(ctx context.Context, d record.Doc) ([]ClosingLine, error) {
	// Read through the starts row so a start date change waits for this posting.
	start, err := store.ShareStart.Run(ctx, d.LegalEntityID)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && d.Date < start.Time.Format(time.DateOnly) {
		return nil, ErrAccountingNotStarted
	}
	if err != nil {
		return nil, err
	}
	date, err := time.Parse(time.DateOnly, d.Date)
	if err != nil {
		return nil, err
	}
	return closingLines(ctx, d.LegalEntityID, date)
}

// closingLines clears every revenue and expense balance left at date into the result account,
// by org unit and partner, then moves the result of each org unit to retained earnings.
func closingLines(ctx context.Context, legalEntity int64, date time.Time) ([]ClosingLine, error) {
	acc, err := store.GetClosingAccounts.Run(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrClosingAccountsUnset
	}
	if err != nil {
		return nil, err
	}
	rows, err := store.ProfitAndLossBalances.Run(ctx, store.ProfitAndLossBalancesParams{LegalEntityID: legalEntity, Date: pgtype.Date{Time: date, Valid: true}})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrClosingNothingToClose
	}
	side := func(l ClosingLine, debitMinusCredit int64) ClosingLine {
		if debitMinusCredit > 0 {
			l.Debit = debitMinusCredit
		} else {
			l.Credit = -debitMinusCredit
		}
		return l
	}
	result := func(unit *int64) ClosingLine {
		return ClosingLine{AccountID: acc.ResultID, AccountCode: acc.ResultCode, AccountName: acc.ResultName, OrgUnitID: unit}
	}
	var out []ClosingLine
	byUnit := map[int64]int64{}
	for _, r := range rows {
		if !r.Active {
			return nil, &platform.Error{Status: ErrAccountNotPostable.Status, Code: ErrAccountNotPostable.Code, Params: map[string]any{"account_code": r.AccountCode}}
		}
		unit := platform.Int8Ptr(r.OrgUnitID)
		out = append(out,
			side(ClosingLine{AccountID: r.AccountID, AccountCode: r.AccountCode, AccountName: r.AccountName, OrgUnitID: unit, PartnerID: platform.Int8Ptr(r.PartnerID)}, -r.Balance),
			side(result(unit), r.Balance))
		byUnit[r.OrgUnitID.Int64] += r.Balance
	}
	// No org unit travels as 0, which sorts first.
	for _, u := range slices.Sorted(maps.Keys(byUnit)) {
		net := byUnit[u]
		if net == 0 {
			continue
		}
		var unit *int64
		if u != 0 {
			unit = &u
		}
		out = append(out, side(result(unit), -net),
			side(ClosingLine{AccountID: acc.RetainedID, AccountCode: acc.RetainedCode, AccountName: acc.RetainedName, OrgUnitID: unit}, net))
	}
	if len(out) > MaxLines {
		return nil, ErrClosingTooLarge
	}
	return out, nil
}
