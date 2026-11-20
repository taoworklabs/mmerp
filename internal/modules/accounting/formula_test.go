package accounting

import (
	"errors"
	"testing"
)

// A small ledger: cash 111 and 112, customer A owing 500 and customer B paid 200 in advance on
// 131, a 300 credit on 2293, revenue 511 of 1 000 with a 50 deduction, closed to 911.
var formulaInput = statementInput{
	balances: []balanceRow{
		{code: "1111", amount: 700}, {code: "1121", amount: 150},
		{code: "131", partner: 1, amount: 500}, {code: "131", partner: 2, amount: -200},
		{code: "2293", amount: -300}, {code: "4212", amount: -950},
	},
	turnovers: []turnoverRow{
		{code: "511", debit: 50, credit: 1_000},
		{code: "511", closing: true, debit: 950},
		{code: "911", closing: true, credit: 950},
	},
}

func TestStatementFormulas(t *testing.T) {
	for formula, want := range map[string]int64{
		"B(111,112)":          850,
		"DB(131)":             500,
		"CB(131)":             200,
		"B(131)":              300,
		"-CB(2293)":           -300,
		"-B(421)":             950,
		"CT(511)":             1_000,
		"DT(511)":             50,
		"DK(511)":             950,
		"CK(511)":             0,
		"CT(511) - DT(511)":   950,
		"DB(131) + CB(131)":   700,
		"B(1)":                1_150,
		"B(9)":                0,
		"":                    0,
		" DB( 131 ) +B(111) ": 1_200,
	} {
		f, err := parseFormula(formula)
		if err != nil {
			t.Errorf("%q: %v", formula, err)
			continue
		}
		if got := f.value(formulaInput, nil); got != want {
			t.Errorf("%q = %d, want %d", formula, got, want)
		}
	}
}

func TestStatementFormulaRows(t *testing.T) {
	rows := []StatementRow{
		{Code: "110", Formula: "B(111,112)"},
		{Code: "130", Formula: "DB(131) - CB(2293)"},
		{Code: "200", Formula: "R(110,130)"},
		{Code: "210", Formula: "R(200) - R(110)"},
	}
	got, err := evaluateRows(rows, formulaInput)
	if err != nil {
		t.Fatal(err)
	}
	for code, want := range map[string]int64{"110": 850, "130": 200, "200": 1_050, "210": 200} {
		if got[code] != want {
			t.Errorf("row %s = %d, want %d", code, got[code], want)
		}
	}
}

func TestStatementFormulaInvalid(t *testing.T) {
	for formula, pos := range map[string]int{
		"X(111)":      1,
		"B(111":       6,
		"B()":         3,
		"B(111) B(2)": 8,
		"B(11 1)":     6,
		"+":           2,
		"B(1,)":       5,
	} {
		_, err := parseFormula(formula)
		var e *formulaError
		if !errors.As(err, &e) || e.pos != pos {
			t.Errorf("%q: %v, want position %d", formula, err, pos)
		}
	}
	for name, rows := range map[string][]StatementRow{
		"unknown row": {{Code: "100", Formula: "R(200)"}},
		"cycle":       {{Code: "100", Formula: "R(200)"}, {Code: "200", Formula: "B(1) + R(100)"}},
		"self":        {{Code: "100", Formula: "R(100)"}},
		"duplicate":   {{Code: "100"}, {Code: "100"}},
	} {
		if _, err := evaluateRows(rows, formulaInput); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}
