package accounting

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/modules/accounting/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// reportRange checks the view permission and parses the period.
func (s *Service) reportRange(ctx context.Context, legalEntity int64, from, to string) (f, t pgtype.Date, err error) {
	if err = s.d.IAM.Require(ctx, "accounting", PermView, legalEntity); err != nil {
		return
	}
	a, e1 := time.Parse("2006-01-02", from)
	b, e2 := time.Parse("2006-01-02", to)
	if e1 != nil || e2 != nil || a.After(b) {
		err = ErrInvalidRange
		return
	}
	return pgtype.Date{Time: a, Valid: true}, pgtype.Date{Time: b, Valid: true}, nil
}

// Ledger lists the lines of an account and its descendants in a period, with running balances.
// ponytail: no paging; a ledger of one account over a year is a few thousand lines at this size. Page it when a tenant outgrows it.
func (s *Service) Ledger(ctx context.Context, legalEntity, accountID int64, from, to string) (Ledger, error) {
	f, t, err := s.reportRange(ctx, legalEntity, from, to)
	if err != nil {
		return Ledger{}, err
	}
	ok, err := store.LedgerAccountExists.Run(ctx, accountID)
	if err != nil {
		return Ledger{}, err
	}
	if !ok {
		return Ledger{}, platform.ErrNotFound
	}
	opening, err := store.LedgerOpening.Run(ctx, store.LedgerOpeningParams{AccountID: accountID, LegalEntityID: legalEntity, FromDate: f})
	if err != nil {
		return Ledger{}, err
	}
	rows, err := store.LedgerLines.Run(ctx, store.LedgerLinesParams{AccountID: accountID, LegalEntityID: legalEntity, FromDate: f, ToDate: t})
	if err != nil {
		return Ledger{}, err
	}
	out := Ledger{Opening: opening, Closing: opening, Lines: make([]LedgerLine, 0, len(rows))}
	for _, r := range rows {
		out.Closing += r.Debit - r.Credit
		ln := LedgerLine{Date: r.Date.Time.Format("2006-01-02"), EntryID: r.EntryID, DocType: r.DocType, DocID: r.DocID, DocNumber: r.DocNumber,
			AccountCode: r.AccountCode, Debit: r.Debit, Credit: r.Credit, Balance: out.Closing}
		if r.Kind.Valid {
			ln.Kind = &r.Kind.String
		}
		if r.Description.Valid {
			ln.Description = &r.Description.String
		}
		if r.OrgUnitID.Valid {
			ln.OrgUnitID = &r.OrgUnitID.Int64
		}
		if r.PartnerID.Valid {
			ln.PartnerID, ln.PartnerCode, ln.PartnerName = &r.PartnerID.Int64, &r.PartnerCode.String, &r.PartnerName.String
		}
		out.Lines = append(out.Lines, ln)
	}
	return out, nil
}

// TrialBalance lists every account with an opening balance or movement in the period, parents rolled up.
func (s *Service) TrialBalance(ctx context.Context, legalEntity int64, from, to string) (TrialBalance, error) {
	f, t, err := s.reportRange(ctx, legalEntity, from, to)
	if err != nil {
		return TrialBalance{}, err
	}
	rows, err := store.TrialBalanceRows.Run(ctx, store.TrialBalanceRowsParams{LegalEntityID: legalEntity, FromDate: f, ToDate: t})
	if err != nil {
		return TrialBalance{}, err
	}
	out := TrialBalance{Rows: make([]TrialRow, 0, len(rows))}
	for _, r := range rows {
		row := TrialRow{AccountID: r.ID, Code: r.Code, Name: r.Name, Opening: r.Opening, Debit: r.Debit, Credit: r.Credit, Closing: r.Opening + r.Debit - r.Credit}
		if r.ParentID.Valid {
			row.ParentID = &r.ParentID.Int64
		}
		if r.Postable {
			out.TotalDebit += r.Debit
			out.TotalCredit += r.Credit
		}
		out.Rows = append(out.Rows, row)
	}
	return out, nil
}

// trackedAccount checks an optional account filter of the partner reports.
func trackedAccount(ctx context.Context, accountID *int64) (pgtype.Int8, error) {
	if accountID == nil {
		return pgtype.Int8{}, nil
	}
	flags, err := accountFlags(ctx, []int64{*accountID})
	if err != nil {
		return pgtype.Int8{}, err
	}
	tracked, ok := flags[*accountID]
	if !ok {
		return pgtype.Int8{}, platform.ErrNotFound
	}
	if !tracked {
		return pgtype.Int8{}, ErrAccountNotTracked
	}
	return pgtype.Int8{Int64: *accountID, Valid: true}, nil
}

// PartnerBalances lists each partner's balance per tracked account (or on one) in a period.
func (s *Service) PartnerBalances(ctx context.Context, legalEntity int64, accountID *int64, from, to string) (PartnerBalances, error) {
	f, t, err := s.reportRange(ctx, legalEntity, from, to)
	if err != nil {
		return PartnerBalances{}, err
	}
	acc, err := trackedAccount(ctx, accountID)
	if err != nil {
		return PartnerBalances{}, err
	}
	rows, err := store.PartnerBalanceRows.Run(ctx, store.PartnerBalanceRowsParams{LegalEntityID: legalEntity, FromDate: f, ToDate: t, AccountID: acc})
	if err != nil {
		return PartnerBalances{}, err
	}
	out := PartnerBalances{Rows: make([]PartnerBalanceRow, len(rows))}
	for i, r := range rows {
		row := PartnerBalanceRow{AccountID: r.AccountID, AccountCode: r.AccountCode, PartnerID: r.PartnerID, PartnerCode: r.PartnerCode, PartnerName: r.PartnerName,
			Opening: r.Opening, Debit: r.Debit, Credit: r.Credit, Closing: r.Opening + r.Debit - r.Credit}
		out.Rows[i] = row
		out.TotalOpening += row.Opening
		out.TotalDebit += row.Debit
		out.TotalCredit += row.Credit
		out.TotalClosing += row.Closing
	}
	return out, nil
}

// PartnerLedger lists one partner's lines on tracked accounts (or on one) in a period, with running balances.
func (s *Service) PartnerLedger(ctx context.Context, legalEntity, partnerID int64, accountID *int64, from, to string) (Ledger, error) {
	f, t, err := s.reportRange(ctx, legalEntity, from, to)
	if err != nil {
		return Ledger{}, err
	}
	acc, err := trackedAccount(ctx, accountID)
	if err != nil {
		return Ledger{}, err
	}
	if ok, err := store.PartnerExists.Run(ctx, partnerID); err != nil || !ok {
		return Ledger{}, platform.OrErr(err, platform.ErrNotFound)
	}
	opening, err := store.PartnerLedgerOpening.Run(ctx, store.PartnerLedgerOpeningParams{LegalEntityID: legalEntity, PartnerID: pgtype.Int8{Int64: partnerID, Valid: true}, FromDate: f, AccountID: acc})
	if err != nil {
		return Ledger{}, err
	}
	rows, err := store.PartnerLedgerLines.Run(ctx, store.PartnerLedgerLinesParams{LegalEntityID: legalEntity, PartnerID: pgtype.Int8{Int64: partnerID, Valid: true}, FromDate: f, ToDate: t, AccountID: acc})
	if err != nil {
		return Ledger{}, err
	}
	out := Ledger{Opening: opening, Closing: opening, Lines: make([]LedgerLine, 0, len(rows))}
	for _, r := range rows {
		out.Closing += r.Debit - r.Credit
		code, name := r.PartnerCode, r.PartnerName
		out.Lines = append(out.Lines, LedgerLine{Date: r.Date.Time.Format("2006-01-02"), EntryID: r.EntryID, DocType: r.DocType, DocID: r.DocID, DocNumber: r.DocNumber,
			Kind: platform.TextPtr(r.Kind), Description: platform.TextPtr(r.Description), OrgUnitID: platform.Int8Ptr(r.OrgUnitID), AccountCode: r.AccountCode,
			PartnerID: &partnerID, PartnerCode: &code, PartnerName: &name, Debit: r.Debit, Credit: r.Credit, Balance: out.Closing})
	}
	return out, nil
}
