package data

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// Base is a list query's fixed SQL and output columns, checked against its
// sqlc method so the row type follows the query.
type Base[R any] struct {
	name, sql string
	cols      []string
}

// NewBase records a base query; the method is taken only to infer R. sql is
// Static so only SQL written in code can become a base.
func NewBase[Q, R any](name string, sql Static, cols []string, _ func(Q, context.Context) ([]R, error)) Base[R] {
	return Base[R]{name: name, sql: string(sql), cols: cols}
}

// Order is one ORDER BY term over a base column.
type Order struct {
	col         string
	desc, lower bool
	nulls       string
}

// Asc and Desc order by col.
func Asc(col string) Order  { return Order{col: col} }
func Desc(col string) Order { return Order{col: col, desc: true} }

// Lower orders by lower(col), for case-insensitive text.
func (o Order) Lower() Order { o.lower = true; return o }

// NullsLast and NullsFirst place NULLs explicitly.
func (o Order) NullsLast() Order  { o.nulls = " NULLS LAST"; return o }
func (o Order) NullsFirst() Order { o.nulls = " NULLS FIRST"; return o }

// ScopeRule says how a list is limited to the caller's data scope. There is no
// default: the zero rule fails registration.
type ScopeRule struct{ col, reason string }

// ByColumn limits rows to the scope's org units through col.
func ByColumn(col string) ScopeRule { return ScopeRule{col: col} }

// Unscoped declares a list visible to anyone allowed to call it.
func Unscoped(reason string) ScopeRule { return ScopeRule{reason: reason} }

// List is a dynamic list declared in code: a base query plus everything a
// request may add to it.
type List[R any] struct {
	Name       string
	Base       Base[R]
	Scope      ScopeRule
	Visibility func(Viewer) Cond // rows visible outside the scope's units, OR'd with them
	Search     []string
	Sorts      map[string][]Order
	Default    []Order
	Tiebreak   Order // a unique column, so pages are stable
}

// Query is one request against a List.
type Query struct {
	Scope  Scope
	Q      string
	Sort   string
	Paging platform.Paging
	Where  Cond
}

// Compiled is one statement ready to send. Shape names the active filters and
// sort for logs; it never holds values.
type Compiled struct {
	Name, Shape, SQL string
	Args             []any
}

// Register validates a list definition and panics on a bad one, so a mistake
// fails at startup.
func Register[R any](l List[R]) List[R] {
	if err := l.validate(); err != nil {
		panic(fmt.Sprintf("data: list %s: %v", l.Name, err))
	}
	return l
}

func (l List[R]) validate() error {
	seen := map[string]string{}
	for _, col := range l.Base.cols {
		// pgx matches columns to fields this way, so two names that normalize alike are ambiguous.
		k := strings.ToLower(strings.ReplaceAll(col, "_", ""))
		if prev, ok := seen[k]; ok {
			return fmt.Errorf("base columns %q and %q collide after normalization", prev, col)
		}
		seen[k] = col
	}
	if l.Scope == (ScopeRule{}) {
		return errors.New("missing scope rule: use ByColumn or Unscoped with a reason")
	}
	c := l.compiler()
	var cols []string
	if l.Scope.col != "" {
		cols = append(cols, l.Scope.col)
	}
	cols = append(cols, l.Search...)
	for _, terms := range l.Sorts {
		for _, o := range terms {
			cols = append(cols, o.col)
		}
	}
	for _, o := range l.Default {
		cols = append(cols, o.col)
	}
	cols = append(cols, l.Tiebreak.col)
	for _, col := range cols {
		if _, err := c.col(col); err != nil {
			return err
		}
	}
	if l.Visibility != nil {
		if _, err := c.cond(l.Visibility(Viewer{}), false); err != nil {
			return err
		}
	}
	return nil
}

func (l List[R]) compiler() *compiler {
	return &compiler{cols: l.Base.cols}
}

// Compile turns q into the page and count statements. empty means the caller
// may see nothing, so no statement should run. Values only ever become args.
func (l List[R]) Compile(q Query, v Viewer, hasViewer bool) (page, count Compiled, empty bool, err error) {
	// A List built without Register must not run unscoped by accident.
	if l.Scope == (ScopeRule{}) {
		return page, count, false, l.wrap(errors.New("missing scope rule"))
	}
	orders, key := l.Default, "default"
	if q.Sort != "" {
		o, ok := l.Sorts[q.Sort]
		if !ok {
			return page, count, false, &platform.Error{
				Status: http.StatusUnprocessableEntity, Code: "invalid_request",
				Params: map[string]any{"fields": []string{"query.sort"}},
			}
		}
		orders, key = o, q.Sort
	}

	c := l.compiler()
	var where []string
	scoped := l.Scope.col != "" && !q.Scope.all
	if scoped {
		var ors []string
		if len(q.Scope.units) > 0 {
			col, err := c.col(l.Scope.col)
			if err != nil {
				return page, count, false, l.wrap(err)
			}
			ors = append(ors, col+" = ANY("+c.arg(q.Scope.units)+")")
		}
		// Without a viewer the actor is unknown, so no row is "theirs".
		if hasViewer && l.Visibility != nil {
			s, err := c.cond(l.Visibility(v), false)
			if err != nil {
				return page, count, false, l.wrap(err)
			}
			if s != "" {
				ors = append(ors, s)
			}
		}
		if len(ors) == 0 {
			return page, count, true, nil
		}
		where = append(where, "("+strings.Join(ors, " OR ")+")")
	}

	if s := strings.TrimSpace(q.Q); s != "" && len(l.Search) > 0 {
		n := c.arg("%" + likeEscaper.Replace(s) + "%")
		terms := make([]string, len(l.Search))
		for i, name := range l.Search {
			col, err := c.col(name)
			if err != nil {
				return page, count, false, l.wrap(err)
			}
			terms[i] = col + " ILIKE " + n
		}
		where = append(where, "("+strings.Join(terms, " OR ")+")")
		// "q" carries no value; Shape only ever names which filters are active.
		c.filters = append(c.filters, "q")
	}

	s, err := c.cond(q.Where, true)
	if err != nil {
		return page, count, false, l.wrap(err)
	}
	if s != "" {
		where = append(where, s)
	}

	terms := make([]string, 0, len(orders)+1)
	for _, o := range append(slices.Clip(orders), l.Tiebreak) {
		t, err := c.order(o)
		if err != nil {
			return page, count, false, l.wrap(err)
		}
		terms = append(terms, t)
	}

	slices.Sort(c.filters)
	shape := strings.Join(slices.Compact(c.filters), ",") + "|" + key
	from := "FROM (" + l.Base.sql + "\n) b"
	if len(where) > 0 {
		from += " WHERE " + strings.Join(where, " AND ")
	}
	count = Compiled{Name: l.Name, Shape: shape, SQL: "SELECT count(*) " + from, Args: slices.Clone(c.args)}
	limit, offset := c.arg(q.Paging.Limit()), c.arg(q.Paging.Offset())
	page = Compiled{
		Name: l.Name, Shape: shape, Args: c.args,
		SQL: "SELECT b.* " + from + " ORDER BY " + strings.Join(terms, ", ") + " LIMIT " + limit + " OFFSET " + offset,
	}
	return page, count, false, nil
}

func (l List[R]) wrap(err error) error { return fmt.Errorf("data: list %s: %w", l.Name, err) }

// LIKE's default escape character is the backslash.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// compiler numbers placeholders and checks column names against the base.
type compiler struct {
	cols    []string
	args    []any
	filters []string
}

func (c *compiler) arg(v any) string {
	c.args = append(c.args, v)
	return "$" + strconv.Itoa(len(c.args))
}

// col emits a column reference; only names declared by the base reach SQL.
func (c *compiler) col(name string) (string, error) {
	if !slices.Contains(c.cols, name) {
		return "", fmt.Errorf("unknown column %q", name)
	}
	return "b." + pgx.Identifier{name}.Sanitize(), nil
}

func (c *compiler) order(o Order) (string, error) {
	t, err := c.col(o.col)
	if err != nil {
		return "", err
	}
	if o.lower {
		t = "lower(" + t + ")"
	}
	if o.desc {
		t += " DESC"
	} else {
		t += " ASC"
	}
	return t + o.nulls, nil
}

// cond emits x, or "" when it vanishes. filter records column names for Shape.
func (c *compiler) cond(x Cond, filter bool) (string, error) {
	switch x.op {
	case "":
		return "", nil
	case opAnd, opOr:
		var parts []string
		for _, k := range x.kids {
			s, err := c.cond(k, filter)
			if err != nil {
				return "", err
			}
			if s != "" {
				parts = append(parts, s)
			}
		}
		switch len(parts) {
		case 0:
			return "", nil
		case 1:
			return parts[0], nil
		}
		return "(" + strings.Join(parts, " "+strings.ToUpper(x.op)+" ") + ")", nil
	case opFrag:
		var b strings.Builder
		for _, p := range x.frag.parts {
			if p.isArg {
				b.WriteString(c.arg(p.arg))
			} else {
				b.WriteString(string(p.text))
			}
		}
		return "(" + b.String() + ")", nil
	}
	col, err := c.col(x.col)
	if err != nil {
		return "", err
	}
	if filter {
		c.filters = append(c.filters, x.col)
	}
	switch x.op {
	case opNull:
		return col + " IS NULL", nil
	case opIn:
		return col + " = ANY(" + c.arg(x.val) + ")", nil
	}
	return col + " " + x.op + " " + c.arg(x.val), nil
}
