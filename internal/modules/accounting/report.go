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
