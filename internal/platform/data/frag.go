package data

import "slices"

// Static is SQL text written in code. Untyped string constants convert to it
// implicitly; a string variable does not, so user input cannot become SQL text.
type Static string

// Frag is trusted SQL: static text with values bound as placeholders.
type Frag struct{ parts []fragPart }

type fragPart struct {
	text  Static
	arg   any
	isArg bool
}

// SQL starts a fragment.
func SQL(s Static) Frag { return Frag{parts: []fragPart{{text: s}}} }

// Arg appends a bound value.
func (f Frag) Arg(v any) Frag { return f.with(fragPart{arg: v, isArg: true}) }

// Text appends more static text.
func (f Frag) Text(s Static) Frag { return f.with(fragPart{text: s}) }

// with never writes into a shared backing array, so a Frag is a value.
func (f Frag) with(p fragPart) Frag { return Frag{parts: append(slices.Clip(f.parts), p)} }

// Cond turns the fragment into a condition. Its column references are not
// checked: the text is static and trusted.
func (f Frag) Cond() Cond { return Cond{op: opFrag, frag: f} }

// Cond is a WHERE condition. The zero Cond is no condition and vanishes.
type Cond struct {
	op   string
	col  string
	val  any
	kids []Cond
	frag Frag
}

const (
	opAnd  = "and"
	opOr   = "or"
	opFrag = "frag"
	opIn   = "in"
	opNull = "null"
)

// Eq is col = v.
func Eq(col string, v any) Cond { return Cond{op: "=", col: col, val: v} }

// In is col = ANY(v); v is a slice.
func In(col string, v any) Cond { return Cond{op: opIn, col: col, val: v} }

// Gte is col >= v.
func Gte(col string, v any) Cond { return Cond{op: ">=", col: col, val: v} }

// Lte is col <= v.
func Lte(col string, v any) Cond { return Cond{op: "<=", col: col, val: v} }

// IsNull is col IS NULL.
func IsNull(col string) Cond { return Cond{op: opNull, col: col} }

// And joins conditions; vanished ones are dropped, and it vanishes when all are.
func And(cs ...Cond) Cond { return Cond{op: opAnd, kids: cs} }

// Or joins conditions like And. An Or of nothing vanishes rather than
// becoming FALSE: optional filters built with When must not empty the page.
func Or(cs ...Cond) Cond { return Cond{op: opOr, kids: cs} }

// When keeps c only if ok.
func When(ok bool, c Cond) Cond {
	if !ok {
		return Cond{}
	}
	return c
}
