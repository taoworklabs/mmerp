package accounting

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/core/dataio"
	"github.com/taoworklabs/mmerp/internal/modules/accounting/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

func knownForm(form string) bool { return form == FormBalanceSheet || form == FormIncomeStatement }

func (s *Service) StatementForm(ctx context.Context, form string) (StatementForm, error) {
	if err := s.requireView(ctx); err != nil {
		return StatementForm{}, err
	}
	if !knownForm(form) {
		return StatementForm{}, platform.ErrNotFound
	}
	rows, err := statementRows(ctx, form)
	return StatementForm{Rows: rows}, err
}

func statementRows(ctx context.Context, form string) ([]StatementRow, error) {
	rows, err := store.StatementRows.Run(ctx, form)
	out := make([]StatementRow, len(rows))
	for i, r := range rows {
		out[i] = StatementRow{Code: r.Code, Label: r.Label, Formula: r.Formula, Bold: r.Bold}
	}
	return out, err
}

// SaveStatementForm replaces every row of a form; the rows must compute (see evaluateRows).
func (s *Service) SaveStatementForm(ctx context.Context, form string, in StatementForm) error {
	if !knownForm(form) {
		return platform.ErrNotFound
	}
	for i := range in.Rows {
		in.Rows[i].Label = strings.TrimSpace(in.Rows[i].Label)
		in.Rows[i].Formula = strings.TrimSpace(in.Rows[i].Formula)
	}
	if _, err := evaluateRows(in.Rows, statementInput{}); err != nil {
		return err
	}
	return s.writeAccount(ctx, func(ctx context.Context) error {
		if err := saveStatementRows(ctx, form, in.Rows); err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "accounting.statement_form_saved", map[string]any{"form": form, "rows": len(in.Rows)})
	})
}

func saveStatementRows(ctx context.Context, form string, rows []StatementRow) error {
	if err := store.DeleteStatementRows.Run(ctx, form); err != nil {
		return err
	}
	p := store.InsertStatementRowsParams{Form: form}
	for _, r := range rows {
		p.Codes = append(p.Codes, r.Code)
		p.Labels = append(p.Labels, r.Label)
		p.Formulas = append(p.Formulas, r.Formula)
		p.Bolds = append(p.Bolds, r.Bold)
	}
	return store.InsertStatementRows.Run(ctx, p)
}

// BalanceSheet computes the balance sheet form at date, with the end of the previous year as
// the comparative column.
func (s *Service) BalanceSheet(ctx context.Context, legalEntity int64, date string) (Statement, error) {
	_, d, err := s.reportRange(ctx, legalEntity, date, date)
	if err != nil {
		return Statement{}, err
	}
	prev := pgtype.Date{Time: time.Date(d.Time.Year()-1, 12, 31, 0, 0, 0, 0, time.UTC), Valid: true}
	return s.statement(ctx, legalEntity, FormBalanceSheet, d, func(ctx context.Context, current bool) (statementInput, error) {
		at := d
		if !current {
			at = prev
		}
		b, err := statementBalances(ctx, legalEntity, at)
		return statementInput{balances: b}, err
	})
}

// IncomeStatement computes the income statement form over a range, with the same range a
// year earlier as the comparative column.
func (s *Service) IncomeStatement(ctx context.Context, legalEntity int64, from, to string) (Statement, error) {
	f, t, err := s.reportRange(ctx, legalEntity, from, to)
	if err != nil {
		return Statement{}, err
	}
	return s.statement(ctx, legalEntity, FormIncomeStatement, t, func(ctx context.Context, current bool) (statementInput, error) {
		a, b := f, t
		if !current {
			a, b = pgtype.Date{Time: a.Time.AddDate(-1, 0, 0), Valid: true}, pgtype.Date{Time: b.Time.AddDate(-1, 0, 0), Valid: true}
		}
		rows, err := store.StatementTurnovers.Run(ctx, store.StatementTurnoversParams{LegalEntityID: legalEntity, FromDate: a, ToDate: b})
		in := statementInput{turnovers: make([]turnoverRow, len(rows))}
		for i, r := range rows {
			in.turnovers[i] = turnoverRow{code: r.Code, closing: r.Closing, debit: r.Debit, credit: r.Credit}
		}
		return in, err
	})
}

func statementBalances(ctx context.Context, legalEntity int64, at pgtype.Date) ([]balanceRow, error) {
	rows, err := store.StatementBalances.Run(ctx, store.StatementBalancesParams{LegalEntityID: legalEntity, Date: at})
	out := make([]balanceRow, len(rows))
	for i, r := range rows {
		out[i] = balanceRow{code: r.Code, partner: r.PartnerID, amount: r.Amount}
	}
	return out, err
}

// statement evaluates form on the current and the comparative input; end is the date whose
// open revenue and expense balances make the statement not closed.
func (s *Service) statement(ctx context.Context, legalEntity int64, form string, end pgtype.Date,
	input func(ctx context.Context, current bool) (statementInput, error)) (Statement, error) {
	rows, err := statementRows(ctx, form)
	if err != nil {
		return Statement{}, err
	}
	out := Statement{Lines: make([]StatementLine, len(rows))}
	var cols [2]map[string]int64
	for i, current := range []bool{true, false} {
		in, err := input(ctx, current)
		if err != nil {
			return Statement{}, err
		}
		if cols[i], err = evaluateRows(rows, in); err != nil {
			return Statement{}, err
		}
	}
	for i, r := range rows {
		out.Lines[i] = StatementLine{StatementRow: r, Current: cols[0][r.Code], Previous: cols[1][r.Code]}
	}
	out.NotClosed, err = store.ProfitAndLossOpen.Run(ctx, store.ProfitAndLossOpenParams{LegalEntityID: legalEntity, Date: end})
	return out, err
}

// StatementParams are what a statement export is asked for: Date for the balance sheet,
// From and To for the income statement.
type StatementParams struct {
	LegalEntityID int64  `json:"legal_entity_id"`
	Date          string `json:"date,omitempty"`
	From          string `json:"from,omitempty"`
	To            string `json:"to,omitempty"`
}

// The official forms' column headings.
var statementHeaders = map[string][]string{
	FormBalanceSheet:    {"Chỉ tiêu", "Mã số", "Số cuối kỳ", "Số đầu năm"},
	FormIncomeStatement: {"Chỉ tiêu", "Mã số", "Kỳ này", "Cùng kỳ năm trước"},
}

func (s *Service) registerStatementExports() {
	for form, name := range map[string]string{FormBalanceSheet: "B01", FormIncomeStatement: "B02"} {
		s.d.DataIO.RegisterExport(dataio.Export{Kind: "accounting.report." + form, Product: "accounting",
			Run: func(ctx context.Context, params json.RawMessage) (dataio.Sheet, error) {
				var p StatementParams
				if err := json.Unmarshal(params, &p); err != nil {
					return dataio.Sheet{}, ErrInvalidRange
				}
				var st Statement
				var err error
				file := name + " " + p.Date
				if form == FormBalanceSheet {
					st, err = s.BalanceSheet(ctx, p.LegalEntityID, p.Date)
				} else {
					st, err = s.IncomeStatement(ctx, p.LegalEntityID, p.From, p.To)
					file = name + " " + p.From + " " + p.To
				}
				if err != nil {
					return dataio.Sheet{}, err
				}
				rows := make([][]any, len(st.Lines))
				for i, l := range st.Lines {
					rows[i] = []any{l.Label, l.Code, l.Current, l.Previous}
				}
				return dataio.Sheet{Name: file, Header: statementHeaders[form], Rows: rows, TextColumns: []int{1}}, nil
			},
			Check: func(ctx context.Context, params json.RawMessage) error {
				var p StatementParams
				if err := json.Unmarshal(params, &p); err != nil {
					return platform.ErrForbidden
				}
				return s.d.IAM.Require(ctx, "accounting", PermView, p.LegalEntityID)
			},
		})
	}
}
