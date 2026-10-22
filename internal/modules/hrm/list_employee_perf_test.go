//go:build perf

package hrm

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
	"github.com/taoworklabs/mmerp/internal/platform/data/datatest"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

// The fixed query the employee list used before the data engine, kept as the
// baseline. Copied verbatim (placeholder order included) from the last commit
// that had it: c656677, internal/modules/hrm/queries.sql / queries.sql.go.
const oldEmployeeListSQL = `WITH RECURSIVE under (id) AS (
    SELECT $10::bigint WHERE $10::bigint IS NOT NULL
    UNION
    SELECT s.id FROM hrm.employees s JOIN under ON s.manager_id = under.id
)
SELECT e.id, e.code, e.full_name, e.org_unit_id, u.name AS org_unit_name, e.email, e.phone,
       e.hire_date, e.termination_date, count(*) OVER () AS total
FROM hrm.employees e JOIN iam.org_units u ON u.id = e.org_unit_id
WHERE ($1::bool OR e.org_unit_id = ANY($2::bigint[]))
  AND ($3::text = '' OR e.code ILIKE '%' || $3 || '%' OR e.full_name ILIKE '%' || $3 || '%')
  AND ($4::bigint IS NULL OR e.org_unit_id = $4)
  AND ($5::text = ''
       OR ($5 = 'active' AND (e.termination_date IS NULL OR e.termination_date >= $6::date))
       OR ($5 = 'terminated' AND e.termination_date < $6::date))
  AND e.id NOT IN (SELECT id FROM under)
ORDER BY
    CASE WHEN $7::text = 'code' THEN lower(e.code) END,
    CASE WHEN $7 = '-code' THEN lower(e.code) END DESC,
    CASE WHEN $7 = 'full_name' THEN e.full_name END,
    CASE WHEN $7 = '-full_name' THEN e.full_name END DESC,
    CASE WHEN $7 = 'hire_date' THEN e.hire_date END,
    CASE WHEN $7 = '-hire_date' THEN e.hire_date END DESC,
    e.id
LIMIT $9 OFFSET $8`

type employeePerfSeed struct {
	db        context.Context // carries the app pool, read-only outside a transaction
	url       string
	units     []int64 // five of the fifty org units, for scoped HR
	orgUnit   int64   // one org unit, for the org_unit_id filter
	managerOf int64   // EMP2, a manager a few levels below the root with a sizable subtree
	today     pgtype.Date
}

// seedEmployeePerf builds 50 org units and 50 000 employees (EMPn reports to
// EMP(n/10), a chain up to five levels deep), 10% of them terminated, then
// ANALYZEs.
func seedEmployeePerf(tb testing.TB) employeePerfSeed {
	tb.Helper()
	pool := pgtest.New(tb)
	sys := pgtest.System(tb, pool)
	ctx := tb.Context()
	s := employeePerfSeed{
		db: platform.WithDB(tb.Context(), pool), url: pool.Config().ConnString(),
		today: pgtype.Date{Time: time.Now().Truncate(24 * time.Hour), Valid: true},
	}
	var company, dept int64
	if err := sys.QueryRow(ctx, `INSERT INTO iam.org_units (kind, name) VALUES ('company', 'Co') RETURNING id`).Scan(&company); err != nil {
		tb.Fatalf("seed: %v", err)
	}
	if err := sys.QueryRow(ctx, `WITH ins AS (INSERT INTO iam.org_units (parent_id, kind, name)
		SELECT $1, 'department', 'Dept ' || gs FROM generate_series(1, 50) gs RETURNING id)
	  SELECT min(id) FROM ins`, company).Scan(&dept); err != nil {
		tb.Fatalf("seed: %v", err)
	}
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO hrm.employees (code, full_name, org_unit_id, hire_date, termination_date)
		  SELECT 'EMP' || gs, 'Nhân viên ' || gs, $1::bigint + gs % 50, date '2015-01-01' + (gs % 3650),
		         CASE WHEN gs % 10 = 0 THEN $2::date - 30 END
		  FROM generate_series(1, 50000) gs`, []any{dept, s.today}},
		{`UPDATE hrm.employees e SET manager_id = m.id FROM hrm.employees m
		  WHERE e.code <> 'EMP1' AND m.code = 'EMP' || greatest(substr(e.code, 4)::int / 10, 1)`, nil},
		{`ANALYZE`, nil},
	} {
		if _, err := sys.Exec(ctx, q.sql, q.args...); err != nil {
			tb.Fatalf("seed: %v", err)
		}
	}
	for i := range 5 {
		s.units = append(s.units, dept+int64(i))
	}
	s.orgUnit = dept + 7
	if err := sys.QueryRow(ctx, `SELECT id FROM hrm.employees WHERE code = 'EMP2'`).Scan(&s.managerOf); err != nil {
		tb.Fatal(err)
	}
	return s
}

// employeePerfCase is one request, run through the old path and the engine.
type employeePerfCase struct {
	name  string
	all   bool
	units []int64
	f     EmployeeFilter
	today pgtype.Date
}

func (s employeePerfSeed) cases() []employeePerfCase {
	first := platform.Paging{Page: 1, PageSize: 50}
	admin := func(name string, f EmployeeFilter) employeePerfCase {
		if f.Sort == "" {
			f.Sort = "code"
		}
		if f.Paging == (platform.Paging{}) {
			f.Paging = first
		}
		return employeePerfCase{name: name, all: true, f: f, today: s.today}
	}
	return []employeePerfCase{
		admin("no filter", EmployeeFilter{}),
		admin("status active", EmployeeFilter{Status: "active"}),
		admin("status terminated", EmployeeFilter{Status: "terminated"}),
		admin("org_unit_id", EmployeeFilter{OrgUnitID: s.orgUnit}),
		admin("q code prefix", EmployeeFilter{Q: "EMP777"}),
		admin("q name fragment", EmployeeFilter{Q: "viên 3"}),
		admin("sort code", EmployeeFilter{Sort: "code"}),
		admin("sort -code", EmployeeFilter{Sort: "-code"}),
		admin("sort full_name", EmployeeFilter{Sort: "full_name"}),
		admin("sort -full_name", EmployeeFilter{Sort: "-full_name"}),
		admin("sort hire_date", EmployeeFilter{Sort: "hire_date"}),
		admin("sort -hire_date", EmployeeFilter{Sort: "-hire_date"}),
		admin("manager_of", EmployeeFilter{ManagerOf: s.managerOf}),
		admin("offset 10000", EmployeeFilter{Paging: platform.Paging{Page: 201, PageSize: 50}}),
		{name: "scoped HR (5 units)", units: s.units, f: EmployeeFilter{Sort: "code", Paging: first}, today: s.today},
	}
}

func (c employeePerfCase) ctx(db context.Context) context.Context {
	return platform.WithActor(db, 1)
}

func (c employeePerfCase) query() data.Query {
	// Grant* are sealed to iam; a measurement builds its scope directly.
	sc := data.GrantUnits(c.units) //nolint:forbidigo
	if c.all {
		sc = data.GrantAll() //nolint:forbidigo
	}
	return employeeQuery(sc, c.f, c.today)
}

func (c employeePerfCase) oldArgs() []any {
	f := c.f
	return []any{c.all, c.units, f.Q,
		pgtype.Int8{Int64: f.OrgUnitID, Valid: f.OrgUnitID != 0}, f.Status, c.today, f.Sort, f.Offset(), f.Limit(),
		pgtype.Int8{Int64: f.ManagerOf, Valid: f.ManagerOf != 0}}
}

// runOld is the old service path: the fixed query, scanned as sqlc scanned it.
func (c employeePerfCase) runOld(ctx context.Context, conn platform.Conn) ([]int64, int64, error) {
	rows, err := conn.Query(ctx, oldEmployeeListSQL, c.oldArgs()...)
	if err != nil {
		return nil, 0, err
	}
	ids := []int64{}
	var id, orgUnitID, total int64
	var code, fullName, orgUnitName string
	var email, phone pgtype.Text
	var hireDate, terminationDate pgtype.Date
	_, err = pgx.ForEachRow(rows, []any{&id, &code, &fullName, &orgUnitID, &orgUnitName, &email, &phone, &hireDate, &terminationDate, &total},
		func() error { ids = append(ids, id); return nil })
	return ids, total, err
}

func (c employeePerfCase) runNew(ctx context.Context) ([]int64, int64, error) {
	p, err := employeeList.Run(ctx, c.query())
	ids := make([]int64, len(p.Items))
	for i, r := range p.Items {
		ids[i] = r.ID
	}
	return ids, p.Total, err
}

func TestEmployeeListPerf(t *testing.T) {
	s := seedEmployeePerf(t)
	runs, warmup := *perfRuns, 20
	var table strings.Builder
	table.WriteString("| case | mode | path | planning | execution | shared hit | shared read | rows | p50 | p95 | p50 ×8 | p95 ×8 |\n|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	statements := map[string]bool{}
	for _, mode := range []string{"force_custom_plan", "force_generic_plan"} {
		db := modeDB(t, s.url, mode)
		conn := platform.DBFrom(db)
		for _, c := range s.cases() {
			if !strings.Contains(c.name, *perfCase) {
				continue
			}
			ctx := c.ctx(db)

			wantIDs, wantTotal, err := c.runOld(ctx, conn)
			if err != nil {
				t.Fatal(err)
			}
			gotIDs, gotTotal, err := c.runNew(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(wantIDs, gotIDs) || wantTotal != gotTotal {
				t.Errorf("%s: new differs from old: %d ids total %d, want %d ids total %d", c.name, len(gotIDs), gotTotal, len(wantIDs), wantTotal)
			}

			oldText := datatest.Explain(t, conn, data.Compiled{Name: "old", SQL: oldEmployeeListSQL, Args: c.oldArgs()}, mode)
			oldPlan := parsePlan(oldText)

			// New: the page, and the count when Run sends it.
			v := data.Viewer{Actor: 1}
			page, count, _, err := employeeList.Compile(c.query(), v, true)
			if err != nil {
				t.Fatal(err)
			}
			statements[page.SQL], statements[count.SQL] = true, true
			newText := datatest.Explain(t, conn, page, mode)
			newPlan := parsePlan(newText)
			if c.f.Page > 1 || len(gotIDs) == c.f.PageSize {
				countText := datatest.Explain(t, conn, count, mode)
				newText += "\n-- count\n" + countText
				newPlan = newPlan.add(parsePlan(countText))
			}
			t.Logf("== %s / %s / old plan\n%s", c.name, mode, oldText)
			t.Logf("== %s / %s / new plan\n%s", c.name, mode, newText)

			oldRun := func() {
				if _, _, err := c.runOld(ctx, conn); err != nil {
					t.Error(err)
				}
			}
			newRun := func() {
				if _, _, err := c.runNew(ctx); err != nil {
					t.Error(err)
				}
			}
			for _, r := range []struct {
				path string
				p    plan
				fn   func()
			}{{"old", oldPlan, oldRun}, {"new", newPlan, newRun}} {
				var seq, par datatest.Stats
				if runs > 0 { // -perf.runs=0 only explains
					seq, par = datatest.Measure(t, warmup, runs, r.fn), concurrently(2, runs, r.fn)
				}
				fmt.Fprintf(&table, "| %s | %s | %s | %s | %s | %d | %d | %s | %s | %s | %s | %s |\n",
					c.name, strings.TrimPrefix(mode, "force_"), r.path, ms(r.p.planning), ms(r.p.execution), r.p.hit, r.p.read, r.p.rows,
					ms(seq.P50), ms(seq.P95), ms(par.P50), ms(par.P95))
			}
		}
	}
	t.Logf("employee list, 50 000 employees, page size 50, %d runs after %d warm-up\n%s", runs, warmup, table.String())
	t.Logf("distinct statements (page and count) across the matrix: %d", len(statements))
}

// BenchmarkEmployeeListRun measures the engine's own cost on a selective request.
func BenchmarkEmployeeListRun(b *testing.B) {
	s := seedEmployeePerf(b)
	for _, c := range s.cases()[:2] {
		b.Run(c.name, func(b *testing.B) {
			ctx := c.ctx(s.db)
			b.ReportAllocs()
			for b.Loop() {
				if _, _, err := c.runNew(ctx); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
