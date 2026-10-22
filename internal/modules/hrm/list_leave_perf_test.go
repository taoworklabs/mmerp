//go:build perf

package hrm

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
	"github.com/taoworklabs/mmerp/internal/platform/data/datatest"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

var (
	perfRuns = flag.Int("perf.runs", 200, "measured runs per combination; 0 only explains")
	perfCase = flag.String("perf.case", "", "run only the cases whose name contains this")
)

// The fixed query and the reports lookup the leave list used before the data
// engine, kept as the baseline.
const (
	oldLeaveListSQL = `SELECT r.id, d.number, d.status, r.employee_id, e.code AS employee_code, e.full_name AS employee_name,
       t.name AS leave_type_name, r.start_date, r.end_date, r.days::text AS days, count(*) OVER () AS total
FROM hrm.leave_requests r
JOIN record.documents d ON d.id = r.id
JOIN hrm.employees e ON e.id = r.employee_id
JOIN hrm.leave_types t ON t.id = r.leave_type_id
WHERE ($1::bool OR d.org_unit_id = ANY($2::bigint[]) OR e.user_id = $3::bigint OR r.employee_id = ANY($4::bigint[]))
  AND ($5::text = '' OR d.status = $5)
  AND ($6::text = '' OR d.number ILIKE '%' || $6 || '%' OR e.code ILIKE '%' || $6 || '%' OR e.full_name ILIKE '%' || $6 || '%')
  AND ($7::bigint IS NULL OR d.org_unit_id = $7)
  AND ($8::bigint IS NULL OR r.employee_id = $8)
  AND ($9::date IS NULL OR r.end_date >= $9)
  AND ($10::date IS NULL OR r.start_date <= $10)
ORDER BY
    CASE WHEN $11::text = 'start_date' THEN r.start_date END,
    CASE WHEN $11 = '-start_date' THEN r.start_date END DESC,
    CASE WHEN $11 = 'number' THEN d.number END,
    CASE WHEN $11 = '-number' THEN d.number END DESC,
    r.id DESC
LIMIT $13 OFFSET $12`
	oldReportsSQL = `WITH RECURSIVE down AS (
    SELECT e.id FROM hrm.employees e JOIN hrm.employees me ON e.manager_id = me.id WHERE me.user_id = $1::bigint
    UNION
    SELECT e.id FROM hrm.employees e JOIN down ON e.manager_id = down.id
)
SELECT down.id FROM down`
)

type leavePerfSeed struct {
	db          context.Context // carries the app pool, read-only outside a transaction
	url         string
	hr, manager int64   // users: HR without an employee record; the manager of employee EMP7
	units       []int64 // five of the fifty departments
	employee    int64   // EMP123, 40 requests
}

// seedLeavePerf builds 50 departments, 5 000 employees in a manager tree (EMPn
// reports to EMP(n/10)) and 200 000 leave requests over two years, then ANALYZEs.
func seedLeavePerf(tb testing.TB) leavePerfSeed {
	tb.Helper()
	pool := pgtest.New(tb)
	sys := pgtest.System(tb, pool)
	ctx := tb.Context()
	s := leavePerfSeed{db: platform.WithDB(tb.Context(), pool), url: pool.Config().ConnString()}
	var company, dept int64
	steps := []struct {
		sql  string
		args func() []any
		dest []any
	}{
		{`INSERT INTO iam.org_units (kind, name) VALUES ('company', 'Co') RETURNING id`, nil, []any{&company}},
		{`WITH ins AS (INSERT INTO iam.org_units (parent_id, kind, name)
			SELECT $1, 'department', 'Dept ' || gs FROM generate_series(1, 50) gs RETURNING id)
		  SELECT min(id) FROM ins`, func() []any { return []any{company} }, []any{&dept}},
		{`WITH ins AS (INSERT INTO iam.users (login, name, password_hash)
			VALUES ('perf_hr', 'HR', 'x'), ('perf_manager', 'Manager', 'x') RETURNING id)
		  SELECT min(id), max(id) FROM ins`, nil, []any{&s.hr, &s.manager}},
	}
	for _, st := range steps {
		var args []any
		if st.args != nil {
			args = st.args()
		}
		if err := sys.QueryRow(ctx, st.sql, args...).Scan(st.dest...); err != nil {
			tb.Fatalf("seed: %v", err)
		}
	}
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO hrm.employees (code, full_name, org_unit_id, hire_date)
		  SELECT 'EMP' || gs, 'Nhân viên ' || gs, $1::bigint + gs % 50, date '2020-01-01' FROM generate_series(1, 5000) gs`, []any{dept}},
		{`UPDATE hrm.employees e SET manager_id = m.id FROM hrm.employees m
		  WHERE e.code <> 'EMP1' AND m.code = 'EMP' || greatest(substr(e.code, 4)::int / 10, 1)`, nil},
		{`UPDATE hrm.employees SET user_id = $1 WHERE code = 'EMP7'`, []any{s.manager}},
		{`INSERT INTO hrm.leave_types (name, deducts_balance) VALUES ('Phép năm', true), ('Ốm', true), ('Không lương', false)`, nil},
		// The number carries the series value, so the second insert finds the
		// employee and dates whatever ids the documents got.
		{`WITH ins AS (
			INSERT INTO record.documents (doc_type, number, status, version, date, legal_entity_id, org_unit_id, approval_ticket)
			SELECT 'hrm.leave_request', 'NP' || lpad(gs::text, 6, '0'), st, 1, make_date(2025 + gs % 2, 1, 1) + (gs % 360)::int, $1,
			       $2::bigint + (gs % 5000 + 1) % 50, CASE WHEN st = 'pending_approval' THEN gs END
			FROM generate_series(1, 200000) gs,
			     LATERAL (SELECT (ARRAY['draft', 'pending_approval', 'cancelled', 'posted', 'posted', 'posted', 'posted', 'posted', 'posted', 'posted'])[gs % 10 + 1] AS st) s
			RETURNING id, number)
		  INSERT INTO hrm.leave_requests (id, employee_id, leave_type_id, start_date, end_date, days)
		  SELECT ins.id, e.id, t.id, make_date(2025 + (n % 2)::int, 1, 1) + (n % 360)::int, make_date(2025 + (n % 2)::int, 1, 1) + (n % 360 + n % 3)::int, n % 3 + 1
		  FROM ins, LATERAL (SELECT substr(ins.number, 3)::bigint AS n) x
		  JOIN hrm.employees e ON e.code = 'EMP' || (x.n % 5000 + 1)
		  JOIN hrm.leave_types t ON t.name = (ARRAY['Phép năm', 'Ốm', 'Không lương'])[x.n % 3 + 1]`, []any{company, dept}},
		{`ANALYZE`, nil},
	} {
		if _, err := sys.Exec(ctx, q.sql, q.args...); err != nil {
			tb.Fatalf("seed: %v", err)
		}
	}
	for i := range 5 {
		s.units = append(s.units, dept+int64(i))
	}
	if err := sys.QueryRow(ctx, `SELECT id FROM hrm.employees WHERE code = 'EMP123'`).Scan(&s.employee); err != nil {
		tb.Fatal(err)
	}
	return s
}

// leavePerfCase is one request, run through the old path and the engine.
type leavePerfCase struct {
	name  string
	actor int64
	all   bool
	units []int64
	f     LeaveFilter
}

func (s leavePerfSeed) cases() []leavePerfCase {
	first := platform.Paging{Page: 1, PageSize: 50}
	admin := func(name string, f LeaveFilter) leavePerfCase {
		if f.Sort == "" {
			f.Sort = "-start_date"
		}
		if f.Paging == (platform.Paging{}) {
			f.Paging = first
		}
		return leavePerfCase{name: name, actor: s.hr, all: true, f: f}
	}
	return []leavePerfCase{
		admin("no filter", LeaveFilter{}),
		admin("employee", LeaveFilter{EmployeeID: s.employee}),
		admin("status draft", LeaveFilter{Status: "draft"}),
		admin("q EMP4242", LeaveFilter{Q: "EMP4242"}),
		admin("sort start_date", LeaveFilter{Sort: "start_date"}),
		admin("sort number", LeaveFilter{Sort: "number"}),
		admin("sort -number", LeaveFilter{Sort: "-number"}),
		admin("offset 10000", LeaveFilter{Paging: platform.Paging{Page: 201, PageSize: 50}}),
		{name: "scoped HR (5 units)", actor: s.hr, units: s.units, f: LeaveFilter{Sort: "-start_date", Paging: first}},
		{name: "manager (no units)", actor: s.manager, f: LeaveFilter{Sort: "-start_date", Paging: first}},
	}
}

func (c leavePerfCase) ctx(db context.Context) context.Context {
	return platform.WithActor(db, c.actor)
}

func (c leavePerfCase) query() data.Query {
	// Grant* are sealed to iam; a measurement builds its scope directly.
	sc := data.GrantUnits(c.units) //nolint:forbidigo
	if c.all {
		sc = data.GrantAll() //nolint:forbidigo
	}
	return leaveQuery(sc, c.f)
}

func (c leavePerfCase) oldArgs(reports []int64) []any {
	f := c.f
	return []any{c.all, c.units, c.actor, reports, f.Status, f.Q,
		pgtype.Int8{Int64: f.OrgUnitID, Valid: f.OrgUnitID != 0}, pgtype.Int8{Int64: f.EmployeeID, Valid: f.EmployeeID != 0},
		platform.NullDate(platform.Optional(f.From)), platform.NullDate(platform.Optional(f.To)), f.Sort, f.Offset(), f.Limit()}
}

// runOld is the old service path: the reports lookup, then the fixed query,
// scanned as sqlc scanned it.
func (c leavePerfCase) runOld(ctx context.Context, conn platform.Conn) ([]int64, int64, error) {
	rows, err := conn.Query(ctx, oldReportsSQL, c.actor)
	if err != nil {
		return nil, 0, err
	}
	reports, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return nil, 0, err
	}
	rows, err = conn.Query(ctx, oldLeaveListSQL, c.oldArgs(reports)...)
	if err != nil {
		return nil, 0, err
	}
	ids := []int64{}
	var id, emp, total int64
	var number, status, code, name, typeName, days string
	var start, end pgtype.Date
	_, err = pgx.ForEachRow(rows, []any{&id, &number, &status, &emp, &code, &name, &typeName, &start, &end, &days, &total},
		func() error { ids = append(ids, id); return nil })
	return ids, total, err
}

func (c leavePerfCase) runNew(ctx context.Context) ([]int64, int64, error) {
	p, err := leaveList.Run(ctx, c.query())
	ids := make([]int64, len(p.Items))
	for i, r := range p.Items {
		ids[i] = r.ID
	}
	return ids, p.Total, err
}

// modeDB opens an app pool whose sessions plan every statement under mode, and
// returns a context carrying it.
func modeDB(tb testing.TB, dbURL, mode string) context.Context {
	tb.Helper()
	u, err := url.Parse(dbURL)
	if err != nil {
		tb.Fatal(err)
	}
	q := u.Query()
	q.Set("plan_cache_mode", mode)
	u.RawQuery = q.Encode()
	pool, err := platform.NewPool(tb.Context(), u.String(), platform.AppPool, nil)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(pool.Close)
	return platform.WithDB(tb.Context(), pool)
}

// concurrently runs fn from 8 clients, runs calls in all, and reports the
// percentiles of every call's latency.
func concurrently(warmup, runs int, fn func()) datatest.Stats {
	const clients = 8
	var mu sync.Mutex
	var d []time.Duration
	var wg sync.WaitGroup
	for range clients {
		wg.Go(func() {
			for range warmup {
				fn()
			}
			for range runs / clients {
				start := time.Now()
				fn()
				mu.Lock()
				d = append(d, time.Since(start))
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	slices.Sort(d)
	return datatest.Stats{P50: d[len(d)*50/100], P95: d[len(d)*95/100]}
}

var (
	planningRe  = regexp.MustCompile(`Planning Time: ([0-9.]+) ms`)
	executionRe = regexp.MustCompile(`Execution Time: ([0-9.]+) ms`)
	buffersRe   = regexp.MustCompile(`Buffers: shared(?: hit=(\d+))?(?: read=(\d+))?`)
	rowsRe      = regexp.MustCompile(`actual time=[0-9.]+\.\.[0-9.]+ rows=([0-9.]+)`)
)

// plan is what one EXPLAIN reports; a page and its count add up.
type plan struct {
	planning, execution time.Duration
	hit, read           int64
	rows                string
}

func parsePlan(text string) plan {
	var p plan
	if m := planningRe.FindStringSubmatch(text); m != nil {
		p.planning, _ = time.ParseDuration(m[1] + "ms")
	}
	if m := executionRe.FindStringSubmatch(text); m != nil {
		p.execution, _ = time.ParseDuration(m[1] + "ms")
	}
	// The first Buffers line belongs to the top node and covers the whole tree.
	if m := buffersRe.FindStringSubmatch(text); m != nil {
		p.hit, _ = strconv.ParseInt(m[1], 10, 64)
		p.read, _ = strconv.ParseInt(m[2], 10, 64)
	}
	if m := rowsRe.FindStringSubmatch(text); m != nil {
		p.rows = m[1]
	}
	return p
}

func (p plan) add(o plan) plan {
	return plan{planning: p.planning + o.planning, execution: p.execution + o.execution, hit: p.hit + o.hit, read: p.read + o.read, rows: p.rows}
}

func TestLeaveListPerf(t *testing.T) {
	s := seedLeavePerf(t)
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

			// Old: explain the fixed query with the reports it would have been given.
			var reports []int64
			rows, err := conn.Query(ctx, oldReportsSQL, c.actor)
			if err == nil {
				reports, err = pgx.CollectRows(rows, pgx.RowTo[int64])
			}
			if err != nil {
				t.Fatal(err)
			}
			oldText := datatest.Explain(t, conn, data.Compiled{Name: "old", SQL: oldLeaveListSQL, Args: c.oldArgs(reports)}, mode)
			oldPlan := parsePlan(oldText)

			// New: the page, and the count when Run sends it.
			v := data.Viewer{Actor: c.actor}
			page, count, _, err := leaveList.Compile(c.query(), v, true)
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
	t.Logf("leave list, 200 000 requests, page size 50, %d runs after %d warm-up\n%s", runs, warmup, table.String())
	t.Logf("distinct statements (page and count) across the matrix: %d", len(statements))
}

func ms(d time.Duration) string {
	us := d.Microseconds()
	return fmt.Sprintf("%d.%02d ms", us/1000, us%1000/10)
}

// BenchmarkLeaveListRun measures the engine's own cost on a selective request.
func BenchmarkLeaveListRun(b *testing.B) {
	s := seedLeavePerf(b)
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
