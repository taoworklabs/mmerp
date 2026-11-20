package accounting_test

import (
	"testing"

	"github.com/taoworklabs/mmerp/internal/modules/accounting"
	"github.com/taoworklabs/mmerp/internal/platform"
)

func (l *ledger) form(t *testing.T, form string, rows ...accounting.StatementRow) {
	t.Helper()
	if err := l.s.SaveStatementForm(l.core.Admin, form, accounting.StatementForm{Rows: rows}); err != nil {
		t.Fatal(err)
	}
}

func lines(s accounting.Statement) map[string][2]int64 {
	out := map[string][2]int64{}
	for _, l := range s.Lines {
		out[l.Code] = [2]int64{l.Current, l.Previous}
	}
	return out
}

func TestStatementsFromTheLedger(t *testing.T) {
	l := closingLedger(t)
	start := "2025-01-01"
	if err := l.s.SetStart(l.core.Admin, l.le, &start); err != nil {
		t.Fatal(err)
	}
	l.form(t, accounting.FormBalanceSheet,
		accounting.StatementRow{Code: "110", Label: "Tiền", Formula: "DB(111)"},
		accounting.StatementRow{Code: "200", Label: "Tổng cộng tài sản", Formula: "R(110)", Bold: true},
		accounting.StatementRow{Code: "300", Label: "Nợ phải trả", Formula: "CB(33)"},
		accounting.StatementRow{Code: "417", Label: "Lợi nhuận sau thuế chưa phân phối", Formula: "-B(421)"},
		accounting.StatementRow{Code: "500", Label: "Tổng cộng nguồn vốn", Formula: "R(300, 417)", Bold: true})
	l.form(t, accounting.FormIncomeStatement,
		accounting.StatementRow{Code: "01", Label: "Doanh thu", Formula: "CT(511)"},
		accounting.StatementRow{Code: "24", Label: "Chi phí quản lý kinh doanh", Formula: "CK(642)"},
		accounting.StatementRow{Code: "60", Label: "Lợi nhuận sau thuế", Formula: "DK(511) - CK(642)", Bold: true})

	// 2025: 400 of revenue, closed at year end. 2026: 1 000 of revenue, 300 of expense.
	l.book(t, "2025-06-10", "111", "511", 400, nil)
	closing, err := l.newClosing(l.core.Admin, "2025-12-31")
	if err != nil {
		t.Fatal(err)
	}
	if err := l.postClosing(closing); err != nil {
		t.Fatal(err)
	}
	l.book(t, "2026-03-10", "111", "511", 1_000, nil)
	l.book(t, "2026-03-15", "642", "111", 300, &l.unit)

	// Before closing March, the income statement misses what only closing entries carry.
	is, err := l.s.IncomeStatement(l.core.Admin, l.le, "2026-01-01", "2026-12-31")
	if err != nil {
		t.Fatal(err)
	}
	if got := lines(is); !is.NotClosed || got["01"] != [2]int64{1_000, 400} || got["24"] != [2]int64{0, 0} || got["60"] != [2]int64{0, 400} {
		t.Fatalf("before closing %v, not closed %v", got, is.NotClosed)
	}

	closing, err = l.newClosing(l.core.Admin, "2026-03-31")
	if err != nil {
		t.Fatal(err)
	}
	if err := l.postClosing(closing); err != nil {
		t.Fatal(err)
	}
	is, err = l.s.IncomeStatement(l.core.Admin, l.le, "2026-01-01", "2026-12-31")
	if err != nil {
		t.Fatal(err)
	}
	if got := lines(is); is.NotClosed || got["01"] != [2]int64{1_000, 400} || got["24"] != [2]int64{300, 0} || got["60"] != [2]int64{700, 400} {
		t.Fatalf("after closing %v", got)
	}

	// The balance sheet at the end of March against the end of the previous year.
	bs, err := l.s.BalanceSheet(l.core.Admin, l.le, "2026-03-31")
	if err != nil {
		t.Fatal(err)
	}
	got := lines(bs)
	if bs.NotClosed || got["110"] != [2]int64{1_100, 400} || got["417"] != [2]int64{1_100, 400} || got["200"] != got["500"] {
		t.Fatalf("balance sheet %v", got)
	}
	if bs.Lines[1].Label != "Tổng cộng tài sản" || !bs.Lines[1].Bold {
		t.Fatalf("rows keep the form's label and order: %+v", bs.Lines[1])
	}

	// Statements are scoped to the legal entity; forms are readable by any accounting viewer.
	outsider, err := l.core.IAM.CreateUser(l.core.Admin, "outsider", "Outsider", "long enough")
	if err != nil {
		t.Fatal(err)
	}
	ctx := platform.WithActor(l.core.Ctx, outsider)
	if _, err := l.s.BalanceSheet(ctx, l.le, "2026-03-31"); !isCode(err, "forbidden") {
		t.Fatalf("outsider: %v", err)
	}
	if _, err := l.s.StatementForm(ctx, accounting.FormBalanceSheet); !isCode(err, "forbidden") {
		t.Fatalf("outsider reads the form: %v", err)
	}
}

func TestStatementFormChecked(t *testing.T) {
	l := newLedger(t)
	for name, rows := range map[string][]accounting.StatementRow{
		"statement_formula_invalid": {{Code: "100", Label: "A", Formula: "B(111"}},
		"statement_row_unknown":     {{Code: "100", Label: "A", Formula: "R(200)"}},
		"statement_formula_cycle":   {{Code: "100", Label: "A", Formula: "R(200)"}, {Code: "200", Label: "B", Formula: "R(100)"}},
		"statement_row_duplicate":   {{Code: "100", Label: "A"}, {Code: "100", Label: "B"}},
	} {
		if err := l.s.SaveStatementForm(l.core.Admin, accounting.FormBalanceSheet, accounting.StatementForm{Rows: rows}); !isCode(err, name) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if err := l.s.SaveStatementForm(l.core.Admin, "cash_flow", accounting.StatementForm{}); !isCode(err, "not_found") {
		t.Errorf("unknown form: %v", err)
	}
}
