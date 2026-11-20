package accounting

import (
	"fmt"
	"strings"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// A statement formula is terms joined by + and -, each a function of comma-separated account
// code prefixes (or row codes for R):
//
//	B   balance, debit minus credit        DB, CB  debit balances, credit balances (positive),
//	DT, CT  debit, credit turnover,                taken per partner on tracked accounts
//	        closing entries excluded       DK, CK  debit, credit turnover of closing entries only
//	R   sum of other rows
//
// The empty formula is zero.
type formula []term

type term struct {
	neg  bool
	fn   string
	args []string
}

var formulaFns = map[string]bool{"B": true, "DB": true, "CB": true, "DT": true, "CT": true, "DK": true, "CK": true, "R": true}

// formulaError points at the 1-based character where the formula stops making sense.
type formulaError struct{ pos int }

func (e *formulaError) Error() string { return fmt.Sprintf("formula invalid at %d", e.pos) }

// balanceRow is a balance at a date, debit minus credit, per postable account and, on accounts
// tracked by partner, per partner (0 otherwise).
type balanceRow struct {
	code    string
	partner int64
	amount  int64
}

// turnoverRow is the turnover of a postable account over a range, split by closing or not.
type turnoverRow struct {
	code          string
	closing       bool
	debit, credit int64
}

type statementInput struct {
	balances  []balanceRow
	turnovers []turnoverRow
}

func parseFormula(s string) (formula, error) {
	p := &formulaParser{s: s}
	var f formula
	p.space()
	if p.end() {
		return nil, nil
	}
	for first := true; ; first = false {
		var t term
		switch {
		case p.eat('+'):
		case p.eat('-'):
			t.neg = true
		case !first:
			return nil, p.err()
		}
		p.space()
		start := p.i
		for !p.end() && s[p.i] >= 'A' && s[p.i] <= 'Z' {
			p.i++
		}
		if t.fn = s[start:p.i]; !formulaFns[t.fn] {
			return nil, &formulaError{pos: start + 1}
		}
		if !p.eat('(') {
			return nil, p.err()
		}
		for {
			p.space()
			start := p.i
			for !p.end() && isCodeChar(s[p.i]) {
				p.i++
			}
			if p.i == start {
				return nil, p.err()
			}
			t.args = append(t.args, s[start:p.i])
			p.space()
			if p.eat(')') {
				break
			}
			if !p.eat(',') {
				return nil, p.err()
			}
		}
		f = append(f, t)
		p.space()
		if p.end() {
			return f, nil
		}
	}
}

type formulaParser struct {
	s string
	i int
}

func (p *formulaParser) end() bool { return p.i >= len(p.s) }

func (p *formulaParser) err() error { return &formulaError{pos: p.i + 1} }

func (p *formulaParser) space() {
	for !p.end() && p.s[p.i] == ' ' {
		p.i++
	}
}

func (p *formulaParser) eat(c byte) bool {
	if !p.end() && p.s[p.i] == c {
		p.i++
		return true
	}
	return false
}

func isCodeChar(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c == '.'
}

func matches(code string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(code, p) {
			return true
		}
	}
	return false
}

// value evaluates the formula; row resolves R terms (nil when there are none).
func (f formula) value(in statementInput, row func(code string) int64) int64 {
	var total int64
	for _, t := range f {
		var v int64
		switch t.fn {
		case "B", "DB", "CB":
			for _, b := range in.balances {
				if !matches(b.code, t.args) {
					continue
				}
				switch {
				case t.fn == "B":
					v += b.amount
				case t.fn == "DB" && b.amount > 0:
					v += b.amount
				case t.fn == "CB" && b.amount < 0:
					v -= b.amount
				}
			}
		case "DT", "CT", "DK", "CK":
			closing := t.fn == "DK" || t.fn == "CK"
			for _, r := range in.turnovers {
				if r.closing != closing || !matches(r.code, t.args) {
					continue
				}
				if t.fn[0] == 'D' {
					v += r.debit
				} else {
					v += r.credit
				}
			}
		case "R":
			for _, code := range t.args {
				v += row(code)
			}
		}
		if t.neg {
			v = -v
		}
		total += v
	}
	return total
}

// evaluateRows computes every row of a form; row codes must be unique, R terms must name rows
// of the form, and no row may reach itself.
func evaluateRows(rows []StatementRow, in statementInput) (map[string]int64, error) {
	parsed := map[string]formula{}
	for _, r := range rows {
		if _, dup := parsed[r.Code]; dup {
			return nil, rowErr(ErrStatementRowDuplicate, r.Code, 0)
		}
		f, err := parseFormula(r.Formula)
		if err != nil {
			return nil, rowErr(ErrStatementFormulaInvalid, r.Code, err.(*formulaError).pos)
		}
		parsed[r.Code] = f
	}
	out := map[string]int64{}
	const visiting = 1
	state := map[string]int{}
	var walk func(code string) error
	walk = func(code string) error {
		switch state[code] {
		case visiting:
			return rowErr(ErrStatementFormulaCycle, code, 0)
		case visiting + 1:
			return nil
		}
		state[code] = visiting
		for _, t := range parsed[code] {
			if t.fn != "R" {
				continue
			}
			for _, ref := range t.args {
				if _, ok := parsed[ref]; !ok {
					return rowErr(ErrStatementRowUnknown, code, 0)
				}
				if err := walk(ref); err != nil {
					return err
				}
			}
		}
		out[code] = parsed[code].value(in, func(c string) int64 { return out[c] })
		state[code] = visiting + 1
		return nil
	}
	for _, r := range rows {
		if err := walk(r.Code); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func rowErr(e *platform.Error, code string, pos int) error {
	p := map[string]any{"code": code}
	if pos > 0 {
		p["position"] = pos
	}
	return &platform.Error{Status: e.Status, Code: e.Code, Params: p}
}
