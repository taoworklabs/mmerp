package accounting_test

import (
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/modules/accounting"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

// templateLedger is a company with the template applied and accounting started on 2026-01-01.
func templateLedger(t *testing.T, code string) *ledger {
	t.Helper()
	core, s := newAccounting(t)
	l := &ledger{core: core, s: s, acc: map[string]int64{}}
	var err error
	if l.le, err = core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{Kind: "company", Name: "C"}); err != nil {
		t.Fatal(err)
	}
	if err := s.ApplyTemplate(core.Admin, code); err != nil {
		t.Fatal(err)
	}
	accounts, err := s.Accounts(core.Admin)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range accounts {
		l.acc[a.Code] = a.ID
	}
	start := "2026-01-01"
	if err := s.SetStart(core.Admin, l.le, &start); err != nil {
		t.Fatal(err)
	}
	return l
}

// entry posts a manual entry of balanced lines dated 2026-03-xx.
func (l *ledger) postLines(t *testing.T, date string, lines ...accounting.JournalEntryLine) {
	t.Helper()
	if err := l.post(l.newJournal(t, accounting.JournalEntryFields{Date: date, LegalEntityID: l.le, Description: "Bút toán", Lines: lines})); err != nil {
		t.Fatal(err)
	}
}

func dr(id, amount int64, partner ...int64) accounting.JournalEntryLine {
	l := accounting.JournalEntryLine{AccountID: id, Debit: amount}
	if len(partner) > 0 {
		l.PartnerID = &partner[0]
	}
	return l
}

func cr(id, amount int64, partner ...int64) accounting.JournalEntryLine {
	l := dr(id, 0, partner...)
	l.Debit, l.Credit = 0, amount
	return l
}

func TestTemplatesList(t *testing.T) {
	_, s := newAccounting(t)
	var codes []string
	for _, tp := range accounting.Templates() {
		codes = append(codes, tp.Code)
	}
	if len(codes) != 2 || codes[0] != "vn-tt133" || codes[1] != "vn-tt99" {
		t.Fatalf("templates %v", codes)
	}
	_ = s
}

func TestApplyTemplateOnce(t *testing.T) {
	l := templateLedger(t, "vn-tt133")
	if err := l.s.ApplyTemplate(l.core.Admin, "vn-tt133"); !isCode(err, "chart_not_empty") {
		t.Fatalf("applied twice: %v", err)
	}
	core, s := newAccounting(t)
	if err := s.ApplyTemplate(core.Admin, "vn-tt200"); !isCode(err, "not_found") {
		t.Fatalf("unknown template: %v", err)
	}
	mustCreate(t, s, core.Admin, acct("111", "asset", nil))
	if err := s.ApplyTemplate(core.Admin, "vn-tt99"); !isCode(err, "chart_not_empty") {
		t.Fatalf("over an existing chart: %v", err)
	}
}

func TestTemplatesMapPayrollPerFund(t *testing.T) {
	for code, want := range map[string]map[posting.Kind]string{
		"vn-tt133": {posting.SalaryExpense: "6422", posting.EmployerInsuranceExpense: "6422", posting.SalaryPayable: "334",
			posting.SocialInsurancePayable: "3383", posting.HealthInsurancePayable: "3384", posting.UnemploymentInsurancePayable: "3385",
			posting.UnionFeePayable: "3382", posting.PitPayable: "3335", posting.InsurancePayable: "3383"},
		"vn-tt99": {posting.SalaryExpense: "6421", posting.EmployerInsuranceExpense: "6421", posting.SalaryPayable: "334",
			posting.SocialInsurancePayable: "3383", posting.HealthInsurancePayable: "3384", posting.UnemploymentInsurancePayable: "3386",
			posting.UnionFeePayable: "3382", posting.PitPayable: "3335", posting.InsurancePayable: "3383"},
	} {
		l := templateLedger(t, code)
		rules, err := l.s.Rules(l.core.Admin)
		if err != nil {
			t.Fatal(err)
		}
		got := map[posting.Kind]string{}
		for _, r := range rules {
			for c, id := range l.acc {
				if r.AccountID != nil && *r.AccountID == id {
					got[posting.Kind(r.Kind)] = c
				}
			}
		}
		for k, c := range want {
			if got[k] != c {
				t.Errorf("%s: %s → %q, want %s", code, k, got[k], c)
			}
		}
		ca, err := l.s.ClosingAccounts(l.core.Admin)
		if err != nil || ca.ResultCode != "911" || ca.RetainedCode != "4212" {
			t.Errorf("%s: closing accounts %+v %v", code, ca, err)
		}
	}
}

// A month of a trading company, closed, then B01 and B02 against figures worked out by hand:
// 100 000 of capital; 8 000 of goods bought on credit; 10 000 + 1 000 VAT sold to customer A on
// credit, at a cost of 6 000; 2 000 paid in advance by customer B; 1 000 of overheads.
func TestTemplateStatementsWorkedExample(t *testing.T) {
	for code, want := range map[string]struct{ b01, b02 map[string]int64 }{
		"vn-tt133": {
			b01: map[string]int64{"110": 101_000, "131": 11_000, "130": 11_000, "141": 2_000, "140": 2_000, "200": 114_000,
				"311": 8_000, "312": 2_000, "313": 1_000, "300": 11_000, "411": 100_000, "417": 3_000, "400": 103_000, "500": 114_000},
			b02: map[string]int64{"01": 10_000, "02": 0, "10": 10_000, "11": 6_000, "20": 4_000, "24": 1_000, "30": 3_000, "50": 3_000, "60": 3_000},
		},
		"vn-tt99": {
			b01: map[string]int64{"110": 101_000, "111": 101_000, "131": 11_000, "130": 11_000, "141": 2_000, "140": 2_000, "100": 114_000,
				"280": 114_000, "311": 8_000, "312": 2_000, "314": 1_000, "310": 11_000, "300": 11_000, "411": 100_000, "420": 3_000,
				"420b": 3_000, "400": 103_000, "440": 114_000},
			b02: map[string]int64{"01": 10_000, "02": 0, "10": 10_000, "11": 6_000, "20": 4_000, "26": 1_000, "30": 3_000, "50": 3_000, "60": 3_000},
		},
	} {
		t.Run(code, func(t *testing.T) {
			l := templateLedger(t, code)
			a, b, sup := l.partner(t, "A"), l.partner(t, "B"), l.partner(t, "S")
			bank, capital := l.acc["1121"], l.acc["4111"]
			revenue, overheads := l.acc["5111"], l.acc["6422"]
			if code == "vn-tt99" {
				bank, capital, revenue, overheads = l.acc["112"], l.acc["41111"], l.acc["511"], l.acc["6428"]
			}
			l.postLines(t, "2026-03-01", dr(bank, 100_000), cr(capital, 100_000))
			l.postLines(t, "2026-03-02", dr(l.acc["156"], 8_000), cr(l.acc["331"], 8_000, sup))
			l.postLines(t, "2026-03-05", dr(l.acc["131"], 11_000, a), cr(revenue, 10_000), cr(l.acc["33311"], 1_000))
			l.postLines(t, "2026-03-05", dr(l.acc["632"], 6_000), cr(l.acc["156"], 6_000))
			l.postLines(t, "2026-03-10", dr(bank, 2_000), cr(l.acc["131"], 2_000, b))
			l.postLines(t, "2026-03-20", dr(overheads, 1_000), cr(bank, 1_000))
			ref, err := l.newClosing(l.core.Admin, "2026-03-31")
			if err != nil {
				t.Fatal(err)
			}
			if err := l.postClosing(ref); err != nil {
				t.Fatal(err)
			}

			bs, err := l.s.BalanceSheet(l.core.Admin, l.le, "2026-03-31")
			if err != nil {
				t.Fatal(err)
			}
			got := lines(bs)
			for row, v := range want.b01 {
				if got[row][0] != v {
					t.Errorf("B01 %s = %d, want %d", row, got[row][0], v)
				}
			}
			is, err := l.s.IncomeStatement(l.core.Admin, l.le, "2026-03-01", "2026-03-31")
			if err != nil {
				t.Fatal(err)
			}
			got = lines(is)
			for row, v := range want.b02 {
				if got[row][0] != v {
					t.Errorf("B02 %s = %d, want %d", row, got[row][0], v)
				}
			}
		})
	}
}

// A kind left without a rule is offered the applied template's account.
func TestTemplateSuggestsRules(t *testing.T) {
	l := templateLedger(t, "vn-tt133")
	if err := l.s.DeleteRule(l.core.Admin, string(posting.SalaryExpense)); err != nil {
		t.Fatal(err)
	}
	rules, err := l.s.Rules(l.core.Admin)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rules {
		switch {
		case r.Kind == string(posting.SalaryExpense):
			if r.AccountID != nil || r.Suggested == nil || r.Suggested.AccountID != l.acc["6422"] || r.Suggested.Side != "debit" {
				t.Fatalf("unmapped kind %+v", r)
			}
		case r.Suggested != nil:
			t.Fatalf("mapped kind suggested %+v", r)
		}
	}
}
