package data

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/taoworklabs/mmerp/internal/platform"
)

var update = flag.Bool("update", false, "rewrite testdata/*.golden")

type fixtureRow struct{}

func fixtureMethod(struct{}, context.Context) ([]fixtureRow, error) { return nil, nil }

const fixtureSQL = `SELECT id, org_unit_id, user_id, number, code, start_date, end_date FROM t`

func fixtureBase(cols ...string) Base[fixtureRow] {
	if cols == nil {
		cols = []string{"id", "org_unit_id", "user_id", "number", "code", "start_date", "end_date"}
	}
	return NewBase("fixtureBase", fixtureSQL, cols, fixtureMethod)
}

func fixtureList() List[fixtureRow] {
	return List[fixtureRow]{
		Name:   "test.Fixture",
		Base:   fixtureBase(),
		Scope:  ByColumn("org_unit_id"),
		Search: []string{"number", "code"},
		Sorts: map[string][]Order{
			"start_date":  {Asc("start_date")},
			"-start_date": {Desc("start_date")},
			"code":        {Asc("code").Lower()},
			"-end_date":   {Desc("end_date").NullsLast()},
			"end_date":    {Asc("end_date").NullsFirst()},
		},
		Default:  []Order{Desc("start_date")},
		Tiebreak: Desc("id"),
	}
}

func withVisibility(l List[fixtureRow]) List[fixtureRow] {
	l.Visibility = func(v Viewer) Cond { return SQL(`b."user_id" = `).Arg(v.Actor).Cond() }
	return l
}

func paging(page, size int) platform.Paging { return platform.Paging{Page: page, PageSize: size} }

func TestRegisterRejects(t *testing.T) {
	cases := []struct {
		name, col string
		edit      func(*List[fixtureRow])
	}{
		{"search", "nope_search", func(l *List[fixtureRow]) { l.Search = []string{"code", "nope_search"} }},
		{"sorts", "nope_sort", func(l *List[fixtureRow]) { l.Sorts["x"] = []Order{Asc("nope_sort")} }},
		{"default", "nope_default", func(l *List[fixtureRow]) { l.Default = []Order{Asc("nope_default")} }},
		{"tiebreak", "nope_tiebreak", func(l *List[fixtureRow]) { l.Tiebreak = Asc("nope_tiebreak") }},
		{"by-column", "nope_scope", func(l *List[fixtureRow]) { l.Scope = ByColumn("nope_scope") }},
		{"cond", "nope_cond", func(l *List[fixtureRow]) {
			l.Visibility = func(Viewer) Cond { return Or(Eq("user_id", 1), And(IsNull("nope_cond"))) }
		}},
		{"missing-scope", "scope", func(l *List[fixtureRow]) { l.Scope = ScopeRule{} }},
		{"collision", "orgunitid", func(l *List[fixtureRow]) {
			l.Base = fixtureBase("id", "org_unit_id", "user_id", "number", "code", "start_date", "end_date", "orgunitid")
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := fixtureList()
			c.edit(&l)
			defer func() {
				r := recover()
				if r == nil {
					t.Fatal("Register did not panic")
				}
				msg := fmt.Sprint(r)
				if !strings.Contains(msg, "test.Fixture") || !strings.Contains(msg, c.col) {
					t.Fatalf("panic %q should name the list and %q", msg, c.col)
				}
			}()
			Register(l)
		})
	}
}

func TestRegisterAcceptsFixture(t *testing.T) {
	Register(withVisibility(fixtureList()))
}

func TestCompileGolden(t *testing.T) {
	scoped := withVisibility(fixtureList())
	unscoped := fixtureList()
	unscoped.Scope = Unscoped("tenant-wide catalogue")
	all := GrantAll()
	units := GrantUnits([]int64{3, 7})
	p1 := paging(1, 50)

	cases := []struct {
		name      string
		list      List[fixtureRow]
		q         Query
		hasViewer bool
	}{
		{"all-scope", scoped, Query{Scope: all, Paging: p1}, true},
		{"units-scope", scoped, Query{Scope: units, Paging: p1}, true},
		{"units-scope-no-viewer", scoped, Query{Scope: units, Paging: p1}, false},
		{"no-units-with-visibility", scoped, Query{Paging: p1}, true},
		{"no-units-no-visibility", fixtureList(), Query{Paging: p1}, true},
		{"no-units-no-viewer", scoped, Query{Paging: p1}, false},
		{"unscoped", unscoped, Query{Paging: p1}, false},
		{"search", unscoped, Query{Q: " abc ", Paging: p1}, false},
		{"cond-eq", unscoped, Query{Where: Eq("code", "A1"), Paging: p1}, false},
		{"cond-in", unscoped, Query{Where: In("user_id", []int64{1, 2}), Paging: p1}, false},
		{"cond-gte", unscoped, Query{Where: Gte("end_date", "2026-01-01"), Paging: p1}, false},
		{"cond-lte", unscoped, Query{Where: Lte("start_date", "2026-12-31"), Paging: p1}, false},
		{"cond-is-null", unscoped, Query{Where: IsNull("end_date"), Paging: p1}, false},
		{"cond-frag", unscoped, Query{Where: SQL(`b."end_date" >= `).Arg("2026-01-01").Text(` + 1`).Cond(), Paging: p1}, false},
		{"cond-nested", scoped, Query{
			Scope: units, Q: "x", Paging: p1,
			Where: And(
				Or(Eq("code", "A"), Eq("number", "N")),
				When(false, Eq("user_id", 9)),
				Or(),
				And(),
				When(true, Gte("start_date", "2026-01-01")),
			),
		}, true},
		{"order-asc", unscoped, Query{Sort: "start_date", Paging: p1}, false},
		{"order-desc", unscoped, Query{Sort: "-start_date", Paging: p1}, false},
		{"order-lower", unscoped, Query{Sort: "code", Paging: p1}, false},
		{"order-nulls-last", unscoped, Query{Sort: "-end_date", Paging: p1}, false},
		{"order-nulls-first", unscoped, Query{Sort: "end_date", Paging: p1}, false},
		{"default-sort", unscoped, Query{Paging: p1}, false},
		{"paging-offset", unscoped, Query{Paging: paging(3, 20)}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			page, count, empty, err := c.list.Compile(c.q, Viewer{Actor: 42}, c.hasViewer)
			if err != nil {
				t.Fatal(err)
			}
			var b strings.Builder
			fmt.Fprintf(&b, "-- empty --\n%v\n", empty)
			if !empty {
				fmt.Fprintf(&b, "-- page --\n%s\n-- page args --\n", page.SQL)
				for _, a := range page.Args {
					fmt.Fprintf(&b, "%#v\n", a)
				}
				fmt.Fprintf(&b, "-- count --\n%s\n-- count args --\n", count.SQL)
				for _, a := range count.Args {
					fmt.Fprintf(&b, "%#v\n", a)
				}
				fmt.Fprintf(&b, "-- shape --\n%s\n", page.Shape)
				checkPlaceholders(t, page)
				checkPlaceholders(t, count)
			}
			path := filepath.Join("testdata", c.name+".golden")
			if *update {
				if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if b.String() != string(want) {
				t.Fatalf("got:\n%s\nwant:\n%s", b.String(), want)
			}
		})
	}
}

var placeholder = regexp.MustCompile(`\$(\d+)`)

// checkPlaceholders asserts every $n has an arg and every arg is referenced.
func checkPlaceholders(t testing.TB, c Compiled) {
	t.Helper()
	hi := 0
	seen := map[int]bool{}
	for _, m := range placeholder.FindAllStringSubmatch(c.SQL, -1) {
		n, _ := strconv.Atoi(m[1])
		seen[n] = true
		hi = max(hi, n)
	}
	if hi != len(c.Args) || len(seen) != len(c.Args) {
		t.Fatalf("placeholders up to $%d (%d distinct) but %d args in %q", hi, len(seen), len(c.Args), c.SQL)
	}
}

func TestCompileSearchEscapes(t *testing.T) {
	l := fixtureList()
	l.Scope = Unscoped("test")
	page, count, _, err := l.Compile(Query{Q: "  50%_a\\b'  ", Paging: paging(1, 20)}, Viewer{}, false)
	if err != nil {
		t.Fatal(err)
	}
	want := `%50\%\_a\\b'%`
	if len(count.Args) != 1 || count.Args[0] != want {
		t.Fatalf("count args %#v, want [%q]", count.Args, want)
	}
	if page.Args[0] != want {
		t.Fatalf("page arg %#v, want %q", page.Args[0], want)
	}
	if strings.Contains(page.SQL, "50") || strings.Contains(page.SQL, "'") {
		t.Fatalf("search text leaked into SQL: %s", page.SQL)
	}
}

func TestCompileUnknownSort(t *testing.T) {
	_, _, _, err := fixtureList().Compile(Query{Scope: GrantAll(), Sort: "id; DROP TABLE t", Paging: paging(1, 20)}, Viewer{}, false)
	var pe *platform.Error
	if !errors.As(err, &pe) || pe.Code != "invalid_request" {
		t.Fatalf("err = %v, want invalid_request", err)
	}
	if strings.Contains(fmt.Sprint(pe.Params), "DROP") {
		t.Fatalf("error params echo the sort: %v", pe.Params)
	}
}

func TestCompileUnknownWhereColumn(t *testing.T) {
	_, _, _, err := fixtureList().Compile(Query{Scope: GrantAll(), Where: Eq("nope", 1), Paging: paging(1, 20)}, Viewer{}, false)
	if err == nil || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("err = %v, want one naming the column", err)
	}
}

func TestCompileShape(t *testing.T) {
	l := fixtureList()
	l.Base = fixtureBase("id", "org_unit_id", "user_id", "number", "code", "start_date", "end_date", "status_x")
	q := Query{
		Scope: GrantAll(), Sort: "-start_date", Paging: paging(1, 20), Q: "secret-q",
		Where: And(
			Eq("status_x", "posted"),
			When(true, Eq("org_unit_id", int64(555))),
			Eq("status_x", "draft"),
			SQL(`b."id" > `).Arg(int64(9)).Cond(),
		),
	}
	page, count, _, err := l.Compile(q, Viewer{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if page.Shape != "org_unit_id,status_x|-start_date" || count.Shape != page.Shape {
		t.Fatalf("shape %q / %q", page.Shape, count.Shape)
	}
	q.Sort, q.Where = "", Cond{}
	page, _, _, _ = l.Compile(q, Viewer{}, false)
	if page.Shape != "|default" {
		t.Fatalf("default shape %q", page.Shape)
	}
}

func FuzzCompile(f *testing.F) {
	f.Add("50%_a\\b'", "-start_date", "x' OR 1=1 --", 1, 20)
	f.Add("", "code", "", 2, 50)
	f.Add("$1", "nope", "$2", 0, -1)
	sorts := []string{"", "start_date", "-start_date", "code", "-end_date", "end_date"}
	l := withVisibility(fixtureList())
	build := func(q, sort, eq string, page, size int) Query {
		return Query{
			Scope: GrantUnits([]int64{1}), Q: q, Sort: sort, Paging: paging(page, size),
			Where: And(Eq("code", eq), When(eq != "", In("number", []string{eq}))),
		}
	}
	f.Fuzz(func(t *testing.T, q, sort, eq string, page, size int) {
		if page%2 == 0 && len(sort) > 0 {
			sort = sorts[int(sort[0])%len(sorts)]
		}
		pg, count, empty, err := l.Compile(build(q, sort, eq, page, size), Viewer{Actor: 1}, true)
		if err != nil {
			var pe *platform.Error
			if !errors.As(err, &pe) || pe.Code != "invalid_request" {
				t.Fatalf("unexpected error %v", err)
			}
			return
		}
		if empty {
			t.Fatal("units scope compiled as empty")
		}
		// Same SQL text as for neutral values with the same presence: values never shape SQL.
		refQ, refEq := "", "zz"
		if strings.TrimSpace(q) != "" {
			refQ = "zz"
		}
		if eq == "" {
			refEq = ""
		}
		ref, refCount, _, err := l.Compile(build(refQ, sort, refEq, 1, 20), Viewer{Actor: 2}, true)
		if err != nil {
			t.Fatal(err)
		}
		if pg.SQL != ref.SQL || count.SQL != refCount.SQL || pg.Shape != ref.Shape {
			t.Fatalf("values changed the SQL:\n%s\nvs\n%s", pg.SQL, ref.SQL)
		}
		for _, s := range []string{strings.TrimSpace(q), eq} {
			if s != "" && !strings.Contains(ref.SQL, s) && strings.Contains(pg.SQL, s) {
				t.Fatalf("input %q leaked into SQL %q", s, pg.SQL)
			}
		}
		checkPlaceholders(t, pg)
		checkPlaceholders(t, count)
	})
}

// A string variable must not convert to Static implicitly.
func TestStaticRejectsVariables(t *testing.T) {
	if testing.Short() {
		t.Skip("runs go build")
	}
	out, err := exec.Command("go", "build", "-tags", "staticcheck_fixture", "./testdata/staticfixture").CombinedOutput()
	if err == nil {
		t.Fatal("fixture compiled; Static accepts a string variable")
	}
	if !strings.Contains(string(out), "fixture.go:9") || !strings.Contains(string(out), "Static") {
		t.Fatalf("unexpected build output:\n%s", out)
	}
}
