# Data Engine over sqlc — Implementation Plan (ADR, spike, vertical slice)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `internal/platform/data` and its enforcement, then migrate one list (leave requests) end to end, stopping at the product owner's gate.

**Architecture:** sqlc keeps generating every query. A small sqlc process plugin emits name-only descriptors that wrap sqlc's own methods, so Go generics carry the types. Dynamic lists compile a parameterless base query plus code-declared fragments into `(sql, args)`, scoped by a sealed IAM scope. Postgres enforces "writes only in `InTx`" through read-only default sessions, and River gets its own pool.

**Tech Stack:** Go, pgx v5.11.0, sqlc v1.30.0 (pinned `go run`), River v0.49.0, Postgres 18, golangci-lint (depguard, forbidigo).

**Spec:** `docs/superpowers/specs/2026-10-21-data-engine-design.md` (revision 2). Read it before any task.

**Scope of this plan:** spec migration steps 0–3 (ADR, spike, vertical slice, gate). Steps 4–7 (the other 9 lists, the Sales batched insert, module-by-module migration, docs) get their own plan after the gate, from what the slice taught.

## Global Constraints

- One Go binary + Postgres. No new Go or npm dependency.
- No ORM. SQL is hand-written in `queries.sql`. The engine never parses or rewrites SQL text.
- No change to transaction boundaries or the document lock order (`CLAUDE.md`, "Transactions and locking").
- Money stays `int64`. Never `float32/64`.
- API errors are `*platform.Error{Code, Params}`, never sentences.
- `platform` must not import `core`, `shared`, `modules` or `app`.
- Never mock the database. Tests use `pgtest`.
- Logs never contain SQL text or parameter values, at any level.
- Commits follow Conventional Commits, end with the `Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>` line, and stage explicit paths (never `git add -A`).
- All work is on branch `feat/data-engine`, except Phase 1, which is on throwaway branch `spike/data-engine` and is never merged.
- Comments are short and say why. They never reference ADRs, docs or milestones.

## Review Focus

1. **A list request from a user with no permission anywhere and no linked employee** must return an empty page and `Total = 0`, not an error and not every row. Pinned in Task 8 (`TestRunZeroScopeNoViewer`).
2. **A search string containing `%`, `_`, `\` or a quote** must match literally and never alter the SQL. Pinned in Task 7 (`TestCompileSearchEscapes`, `FuzzCompile`).
3. **A page number far past the end** must return no items and the true total. Pinned in Task 8 (`TestRunPagePastEnd`).
4. **A write in code that forgot `InTx`**, including a data-modifying CTE or `SELECT … FOR UPDATE`, must fail loudly with `25006`, not succeed silently. Pinned in Task 2 (`TestReadOnlyOutsideTx`).
5. **A job enqueued from a request handler with no transaction** must error rather than be inserted outside the business commit. Pinned in Task 2 (`TestEnqueueRequiresTx`).

---

## Phase 0: ADR

### Task 0: ADR-0038

**Files:**
- Create: `docs/adr/0038-data-engine-over-sqlc.md`

- [ ] **Step 1: Write the ADR** in the format of `docs/adr/0036-approval-flows.md`:
  - Title `# 0038. Data engine over sqlc: dynamic parts compiled from code-declared fragments, no ORM`.
  - Status line: `Status: accepted · 2026-10-21 · amends ADR-0008 (data access)`.
  - Sections Context, Decision, Consequences. Decision bullets come from spec §Architecture, §2 (scope, fragments), §3, §4, §5 and §6. Consequences include the behaviour changes from spec §Behaviour changes. Add an "Options not taken" table copied from spec §Why not an ORM.
- [ ] **Step 2: Commit**

```bash
git add docs/adr/0038-data-engine-over-sqlc.md
git commit -m "docs(adr): data engine over sqlc"
```

---

## Phase 1: Feasibility spike (branch `spike/data-engine`, throwaway)

Spike code gets no tests beyond what each question needs, and none of it is kept. Each task answers a yes/no question. Task S5 records the answers on `feat/data-engine`.

```bash
git switch -c spike/data-engine feat/data-engine
```

### Task S1: Can sqlc run a process plugin next to `gen.go`, offline, with name-only descriptors that compile for every query?

**Files (spike only):**
- `cmd/sqlc-gen-data/main.go`
- `sqlc.yaml`
- `Makefile`
- `internal/platform/data/fixed.go`

- [ ] **Step 1:** Write a minimal `cmd/sqlc-gen-data`. It reads the protobuf `GenerateRequest` from stdin and, for each query, emits one line by `cmd` and `len(params)`, using `github.com/sqlc-dev/sqlc/pkg/plugin` (already in the module cache with sqlc v1.30.0; check that `go.mod` does not need a new require, and record it if it does):

  ```go
  var GetEmployee = data.One1("hrm.GetEmployee", newQ, (*Queries).GetEmployee)
  ```

  It also emits `func newQ(c platform.Conn) *Queries { return New(c) }` once per file.
- [ ] **Step 2:** In `data/fixed.go`, write the eight constructors as stubs with the final signatures from Task 5.
- [ ] **Step 3:** Wire it in:
  - `sqlc.yaml`: a top-level `plugins: [{name: data, process: {cmd: ./bin/sqlc-gen-data}}]`, plus `codegen: [{plugin: data, out: <same store dir>}]` on each `sql` entry;
  - `Makefile` `gen`: `go build -o bin/sqlc-gen-data ./cmd/sqlc-gen-data` before `$(SQLC) generate`.
- [ ] **Step 4:** Run `GOFLAGS=-mod=mod GOPROXY=off make gen && go build ./...`.
  - **Yes** if it exits 0 and every `data.gen.go` compiles.
  - Record whether `./bin/...` resolved, given sqlc's `exec.LookPath` and cleared environment. If it did not, record the absolute-path workaround used.

### Task S2: Does `RowToStructByName` scan every base shape we need?

- [ ] **Step 1:** In a spike test, run `SELECT b.* FROM (<leave base from spec §2>) b LIMIT 5` on a `pgtest` database seeded via the HRM test helpers. Scan with `pgx.CollectRows(rows, pgx.RowToStructByName[store.LeaveListBaseRow])`.
- [ ] **Step 2:** Repeat for the base shapes of employees, contracts, customers and sales headers. These have `pgtype.Date`, `pgtype.Text`, `numeric::text` and `bigint` money columns.
- [ ] **Step 3:** **Yes** if all scan with no error and the values equal those from the matching sqlc method. Record any column that needed an alias.

### Task S3: Do read-only default sessions catch every write form, and how many existing writes run outside `InTx`?

- [ ] **Step 1:** Build a pool with `RuntimeParams["default_transaction_read_only"] = "on"`. Outside any transaction, run each statement and record the SQLSTATE:
  1. `INSERT`;
  2. `UPDATE`;
  3. `DELETE`;
  4. `WITH x AS (INSERT … RETURNING id) SELECT * FROM x`;
  5. `SELECT … FOR UPDATE`;
  6. `SELECT … FOR SHARE`;
  7. `SELECT nextval('<a real sequence>')`;
  8. a `plpgsql` function that inserts;
  9. `SELECT pg_advisory_xact_lock(1)` (expected allowed);
  10. `CREATE TEMP TABLE` (expected allowed).

  Repeat all ten inside `pgx.BeginTxFunc(..., pgx.TxOptions{AccessMode: pgx.ReadWrite}, …)`. Expected: no errors.
- [ ] **Step 2:** Make the pools read-only:
  - `pgtest.create` uses the read-only pool;
  - River gets a normal pool;
  - `platform.InTx` uses `ReadWrite`.

  Then run `go test ./...` and list every failing test whose error is SQLSTATE `25006`, with the code path behind it.
- [ ] **Step 3:** **Yes** if forms 1–8 fail with `25006` outside and succeed inside. Record the list from Step 2. Its length sizes Task 2.

### Task S4: Separate count in a batch, or window count?

- [ ] **Step 1:** On a `pgtest` database, insert 200 000 leave requests with SQL (`generate_series` into `record.documents` and `hrm.leave_requests`, 50 org units, 5 000 employees). Then `ANALYZE`.
- [ ] **Step 2:** For pages 1, 50 and a page past the end, with no filter and with one selective filter, measure p50 and p95 over 200 runs after 20 warm-up runs:
  - (a) page with `count(*) OVER ()`;
  - (b) page plus `SELECT count(*)` in one `pgx.Batch`;
  - (c) as (b), skipping the count when page 1 is not full.
- [ ] **Step 3:** **Choose** (b)/(c) unless (a) is more than 30% faster at p95 on every combination. If it is, choose (a) with a fallback count when the page is empty. Record the numbers.

### Task S5: Record the spike results

- [ ] **Step 1:** `git switch feat/data-engine`. Append a section `## Spike results (YYYY-MM-DD)` to the spec with the S1–S4 answers, the numbers, the S3 Step 2 list, and any change these force on Tasks 1–10.
- [ ] **Step 2:** Commit.

```bash
git add docs/superpowers/specs/2026-10-21-data-engine-design.md
git commit -m "docs: data engine spike results"
```

- [ ] **Step 3:** **Stop.** Report to the product owner. Continue to Phase 2 only if every answer is yes, or if the spec was amended and approved.

---

## Phase 2: Vertical slice (branch `feat/data-engine`, TDD)

### Task 1: Query naming and the tracer

**Files:**
- Create: `internal/platform/querylog.go`
- Create: `internal/platform/querylog_test.go`
- Modify: `internal/platform/config.go` (add `SlowQuery time.Duration`, env `SLOW_QUERY_MS`, default `500`)
- Modify: `internal/platform/config_test.go`
- Modify: `CLAUDE.md` (add `SLOW_QUERY_MS` (`500`) to the env list under Commands)

**Interfaces:**
- Produces:
  - `func WithQueryName(ctx context.Context, name, shape string) context.Context`
  - `func NewQueryTracer(log *slog.Logger, slow time.Duration) pgx.QueryTracer` (implements `TraceQueryStart`/`TraceQueryEnd`)

- [ ] **Step 1: Write failing tests** in `querylog_test.go`, using a `slog` handler that captures records and a `pgtest` pool whose `ConnConfig.Tracer` is the tracer:
  - `TestTracerLogsNameOnce`: a query run with `WithQueryName(ctx, "hrm.GetEmployee", "")` produces exactly one record at `debug`, with attrs `name=hrm.GetEmployee`, `duration`, `rows`.
  - `TestTracerNeverLogsSQLOrArgs`: run `SELECT $1::text` with arg `"secret-value-123"`. No captured record's message or attrs contain `secret-value-123` or `SELECT`, at any level.
  - `TestTracerSlowWarn`: with `slow = 1ns`, the record is at `warn`.
  - `TestTracerUnnamed`: without a name, the record has `name=unnamed` and is at `debug` even when slow.
  - `TestTracerSQLState`: a failing query logs `sqlstate=42P01`.
- [ ] **Step 2:** Run `go test ./internal/platform -run Tracer`. Expected: FAIL (undefined).
- [ ] **Step 3: Implement.** Name and shape are kept in a ctx value. `TraceQueryStart` stores the start time in the returned ctx. `TraceQueryEnd` logs once.
- [ ] **Step 4:** Run `go test ./internal/platform -run 'Tracer|Config'`. Expected: PASS.
- [ ] **Step 5: Commit** `feat(platform): query tracer logging names, never SQL or args`.

### Task 2: Read-only app pool, River pool, `Enqueue` in a transaction

**Files:**
- Modify: `internal/platform/db.go` (`NewPool`, `InTx` read-write)
- Modify: `internal/platform/jobs.go:92-122` (`Enqueue`)
- Modify: `internal/platform/migrate.go` (doc comment: needs a system pool)
- Modify: `internal/platform/pgtest/pgtest.go` (`create` uses `NewPool(…, AppPool)`; add `System`)
- Modify: `cmd/server/main.go:43` (two pools; `Migrate` and River on the system pool)
- Modify: `internal/app/app.go:104-150` (`Env.SystemPool`; `riverpgxv5.New(env.SystemPool)`)
- Modify: every call site listed in spike S3 Step 2 (wrap in `platform.InTx`)
- Modify: every `platform.Enqueue` caller outside `InTx` (find with `grep -rn "platform.Enqueue" internal`)
- Test: `internal/platform/db_test.go`, `internal/platform/jobs_test.go`

**Interfaces:**
- Produces:
  - `type PoolRole int` with `const (AppPool PoolRole = iota; SystemPool)`
  - `func NewPool(ctx context.Context, url string, role PoolRole, tracer pgx.QueryTracer) (*pgxpool.Pool, error)`. `AppPool` sets `default_transaction_read_only=on`. Both roles set the tracer when it is non-nil.
  - `var ErrEnqueueOutsideTx = errors.New("platform: job enqueued outside a transaction")`
  - `func System(t testing.TB, app *pgxpool.Pool) *pgxpool.Pool` in `pgtest`: a `SystemPool` on the same database.

- [ ] **Step 1: Write failing tests.**
  - `TestReadOnlyOutsideTx` (table test in `db_test.go`): forms 1–8 from spike S3 fail outside `InTx` with SQLSTATE `25006`, and succeed inside `InTx`. Forms 9–10 succeed outside.
  - `TestEnqueueRequiresTx` (`jobs_test.go`): `platform.Enqueue` with no tx in ctx returns `ErrEnqueueOutsideTx` and inserts no row in `river_job`. Inside `InTx` it inserts.
  - `TestWriteAuditJobCommitTogether` (`internal/app`): in one `InTx`, write an HRM document through the service (audit row, number and enqueued job), then return an error. Assert none of the three rows exists. Without the error, all three exist.
- [ ] **Step 2:** Run `go test ./internal/platform ./internal/app -run 'ReadOnly|Enqueue|CommitTogether'`. Expected: FAIL.
- [ ] **Step 3: Implement** `NewPool`, `InTx` via `pgx.BeginTxFunc(ctx, pool, pgx.TxOptions{AccessMode: pgx.ReadWrite}, …)` (the join-existing-tx path unchanged), `Enqueue` without the non-tx fallback, the pool wiring in `main.go`, `app.go` and `pgtest`, and the S3 call-site wraps.
  - `main.go` passes `platform.NewQueryTracer(log, cfg.SlowQuery)` to both pools.
  - Keep the template pool in `pgtest.template` as it is: it must stay writable.
  - Test helpers that seed data by writing straight to the pool switch to `pgtest.System(t, pool)` or to `platform.InTx`. Prefer `InTx` when the helper goes through a service.
- [ ] **Step 4:** Run `make test`. Expected: PASS. Any `25006` is another missed call site: wrap it in `InTx` and rerun.
- [ ] **Step 5: Commit** `feat(platform): read-only app sessions, River on its own pool, enqueue only in a transaction`.

### Task 3: Error normalization

**Files:**
- Create: `internal/platform/data/errors.go`
- Create: `internal/platform/data/errors_test.go`

**Interfaces:**
- Produces:
  - `type Class int` with `const (None Class = iota; Unique; ForeignKey; Check; NotNull; Serialization; Deadlock; LockTimeout; Canceled; ReadOnly)`
  - `func Classify(err error) Class`
  - `type Constraints map[string]*platform.Error` (key `schema.constraint`)
  - `func RegisterConstraints(c Constraints)`: an identical re-registration is a no-op; a different error for the same key panics.
  - `func Normalize(err error) error`: a mapped constraint becomes a value whose `Unwrap() []error` returns the `*platform.Error` and the `*pgconn.PgError`; anything else is returned unchanged.
  - `func IsRetryable(err error) bool` (Serialization, Deadlock)

- [ ] **Step 1: Write failing tests** against real violations on a `pgtest` database. Create a scratch table inside the test transaction.
  - `TestClassify`: SQLSTATE `23505, 23503, 23514, 23502, 40P01, 55P03, 57014, 25006` → the matching `Class`. Deadlock is provoked with two transactions; lock timeout with `SET LOCAL lock_timeout = '10ms'`; cancel with a cancelled ctx.
  - `TestNormalizeMapped`: register `{"public.t_code_key": errTaken}`. After a unique violation, `errors.AsType[*platform.Error](Normalize(err))` returns `errTaken`, and `platform.Violates(Normalize(err), "t_code_key")` is true.
  - `TestNormalizeUnmapped`: an unmapped violation returns an error whose `errors.AsType[*platform.Error]` is false.
  - `TestRegisterConstraintsIdempotent` and `TestRegisterConstraintsConflictPanics`.
  - `TestNoRowsPassesThrough`: `errors.Is(Normalize(pgx.ErrNoRows), pgx.ErrNoRows)`.
- [ ] **Step 2:** Run `go test ./internal/platform/data -run 'Classify|Normalize|Register|NoRows'`. Expected: FAIL.
- [ ] **Step 3: Implement.** The registry is a package map guarded by a `sync.Mutex`, written only at startup.
- [ ] **Step 4:** Run the same command. Expected: PASS.
- [ ] **Step 5: Commit** `feat(platform): normalize database errors to coded API errors`.

### Task 4: Fixed-query descriptors

**Files:**
- Create: `internal/platform/data/fixed.go`
- Create: `internal/platform/data/fixed_test.go`

**Interfaces:**
- Consumes: `platform.WithQueryName` (Task 1), `Normalize` (Task 3), `platform.DBFrom`, `platform.Conn`.
- Produces: eight descriptor types, each with a `Run` method that sets the query name, calls the sqlc method on `newQ(platform.DBFrom(ctx))` and normalizes the error:

  | Constructor | `Run` signature |
  |---|---|
  | `One0[Q, R any](name string, newQ func(platform.Conn) Q, f func(Q, context.Context) (R, error)) One0[R]` | `Run(ctx) (R, error)` |
  | `One1[Q, P, R any](name, newQ, f func(Q, context.Context, P) (R, error)) One1[P, R]` | `Run(ctx, p P) (R, error)` |
  | `Many0[Q, R any](name, newQ, f func(Q, context.Context) ([]R, error)) Many0[R]` | `Run(ctx) ([]R, error)` |
  | `Many1[Q, P, R any](name, newQ, f func(Q, context.Context, P) ([]R, error)) Many1[P, R]` | `Run(ctx, p P) ([]R, error)` |
  | `Exec0[Q any](name, newQ, f func(Q, context.Context) error) Exec0` | `Run(ctx) error` |
  | `Exec1[Q, P any](name, newQ, f func(Q, context.Context, P) error) Exec1[P]` | `Run(ctx, p P) error` |
  | `ExecRows0[Q any](name, newQ, f func(Q, context.Context) (int64, error)) ExecRows0` | `Run(ctx) (int64, error)` |
  | `ExecRows1[Q, P any](name, newQ, f func(Q, context.Context, P) (int64, error)) ExecRows1[P]` | `Run(ctx, p P) (int64, error)` |

  Plus `func (d One1[P, R]) Name() string`, and the same on every type.

- [ ] **Step 1: Write failing tests** with a hand-written fake `Queries` type in the test file, whose methods run real SQL on `DBFrom(ctx)`:
  - `TestOne1RunsInCtxTx`: inside `InTx`, `Run` sees a row inserted earlier in the same tx.
  - `TestRunNormalizesErrors`: a registered unique violation through `Exec1.Run` yields the mapped `*platform.Error`.
  - `TestRunSetsQueryName`: with a capturing tracer, the logged name equals the descriptor's name.
- [ ] **Step 2:** Run `go test ./internal/platform/data -run 'One1|Normalizes|QueryName'`. Expected: FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4:** Run the same command. Expected: PASS.
- [ ] **Step 5: Commit** `feat(platform): fixed-query descriptors over sqlc methods`.

### Task 5: `sqlc-gen-data` plugin

**Files:**
- Create: `cmd/sqlc-gen-data/main.go`
- Create: `cmd/sqlc-gen-data/main_test.go`
- Create: `cmd/sqlc-gen-data/testdata/request.json`
- Create: `cmd/sqlc-gen-data/testdata/data.gen.go.golden`
- Modify: `sqlc.yaml` (top-level `plugins`; a `codegen` entry per `sql` entry, `out` = that entry's store dir)
- Modify: `Makefile` `gen` (build the plugin first, to the path chosen in S1)
- Modify: `.gitignore` (plugin binary path)
- Generated: `internal/**/internal/store/data.gen.go` (14 files)

**Interfaces:**
- Consumes: Task 4 constructors.
- Produces:
  - in every store package, one exported `var <QueryName> = data.<Kind><0|1>("<module>.<QueryName>", newQ, (*Queries).<QueryName>)` per query;
  - for queries marked `-- data:base`, instead a `var <QueryName> = data.NewBase("<module>.<QueryName>", <unexportedConst>, []string{<output column names in order>}, (*Queries).<QueryName>)`.

  `<module>` is the store's parent directory name (`hrm`, `sales`, `iam`, …).

- [ ] **Step 1: Write the failing golden test** `TestGenerate`. It feeds `testdata/request.json` (a JSON-format `GenerateRequest` covering `:one`, `:many`, `:exec`, `:execrows`, zero and one parameter, a `Params` struct, and one `-- data:base` query) and compares the output with `data.gen.go.golden`. Add these too:
  - `TestRejectsUnknownCmd`: `:execresult` returns an error naming the query.
  - `TestRejectsBaseWithParams`: a `-- data:base` query with a parameter returns an error naming it.
- [ ] **Step 2:** Run `go test ./cmd/sqlc-gen-data`. Expected: FAIL.
- [ ] **Step 3: Implement.** Read only `name`, `cmd`, `len(params)`, `comments` (for `data:base`) and `columns[].name`. Map no types. Format the output with `go/format`. Create `internal/platform/data/list.go` now with only `type Base[R any] struct{ name, sql string; cols []string }` and `NewBase` (signature in Task 7), which stores its arguments. Task 7 adds the rest of the file.
- [ ] **Step 4:** Run `go test ./cmd/sqlc-gen-data && make gen && go build ./... && git status --short`. Expected: tests PASS, build OK, and the only changes are the 14 new `data.gen.go` files plus the files listed above.
- [ ] **Step 5: Commit** `build: generate data engine descriptors from sqlc`.

### Task 6: Sealed scope and `iam.DataScope`

**Files:**
- Create: `internal/platform/data/scope.go`
- Create: `internal/platform/data/scope_test.go`
- Modify: `internal/core/iam/service.go` (add `DataScope` next to `Scope` at :342)
- Test: `internal/core/iam/scope_test.go` (add a case to the existing file, or create it)

**Interfaces:**
- Produces:
  - `type Scope struct{ /* unexported: all bool; units []int64 */ }` — the zero value grants nothing
  - `func GrantAll() Scope`
  - `func GrantUnits(units []int64) Scope`
  - `type Viewer struct{ Actor int64 }`
  - `func viewerFrom(ctx context.Context) (Viewer, bool)`, unexported: the actor from `platform.ActorFrom`
  - `func (s *iam.Service) DataScope(ctx context.Context, product, permission string) (data.Scope, error)`: calls `Scope` and converts (`All` → `GrantAll()`, units → `GrantUnits`, none → `Scope{}`)

- [ ] **Step 1: Write failing tests.**
  - `TestZeroScopeGrantsNothing`: `Scope{}` reports no access.
  - `TestDataScopeMatchesScope` (iam): for an admin, a unit-scoped user and a user with no role, `DataScope` agrees with `Scope`.
- [ ] **Step 2:** Run `go test ./internal/platform/data ./internal/core/iam -run 'Scope'`. Expected: FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4:** Run the same command. Expected: PASS.
- [ ] **Step 5: Commit** `feat(iam): sealed data scope built only by IAM`.

### Task 7: List definition, validator and compiler

**Files:**
- Create: `internal/platform/data/frag.go` (`Static`, `Frag`, `Cond`, condition constructors)
- Modify: `internal/platform/data/list.go` (add `List`, `Order`, `ScopeRule`, `Register`, `Compile` next to Task 5's `Base`)
- Create: `internal/platform/data/compile_test.go`
- Create: `internal/platform/data/testdata/*.golden`

**Interfaces:**
- Consumes: `Scope`, `Viewer` (Task 6); `platform.Paging`.
- Produces:
  - `type Static string`
  - `func SQL(s Static) Frag`, `func (f Frag) Arg(v any) Frag`, `func (f Frag) Text(s Static) Frag`
  - `type Cond` (opaque), with:
    - `func Eq(col string, v any) Cond`, and `In`, `Gte`, `Lte` with the same shape;
    - `func IsNull(col string) Cond`;
    - `func And(cs ...Cond) Cond`, `func Or(cs ...Cond) Cond`;
    - `func When(ok bool, c Cond) Cond`;
    - `func (f Frag) Cond() Cond`.
  - `type Base[R any] struct{ name, sql string; cols []string }` and `func NewBase[Q, R any](name, sql string, cols []string, _ func(Q, context.Context) ([]R, error)) Base[R]`
  - `type Order struct{…}`, with:
    - `func Asc(col string) Order`, `func Desc(col string) Order`;
    - `func (o Order) Lower() Order`, `func (o Order) NullsLast() Order`, `func (o Order) NullsFirst() Order`.
  - `type ScopeRule struct{…}`, `func ByColumn(col string) ScopeRule`, `func Unscoped(reason string) ScopeRule`
  - `type List[R any] struct{ Name string; Base Base[R]; Scope ScopeRule; Visibility func(Viewer) Cond; Search []string; Sorts map[string][]Order; Default []Order; Tiebreak Order }`
  - `type Query struct{ Scope Scope; Q, Sort string; Paging platform.Paging; Where Cond }`
  - `func Register[R any](l List[R]) List[R]`: validates and panics with a message naming the list and the bad column
  - `type Compiled struct{ Name, Shape, SQL string; Args []any }`
  - `func (l List[R]) Compile(q Query, v Viewer, hasViewer bool) (page Compiled, count Compiled, empty bool, err error)`. `empty` is true when the engine must return an empty page without querying. `err` is a `*platform.Error` with code `invalid_request` for an unknown sort.

- [ ] **Step 1: Write failing tests** with a fixture list over a base `SELECT id, org_unit_id, user_id, number, code, start_date, end_date FROM t`:
  - `TestRegisterRejects`: an unknown column in `Search`, `Sorts`, `Default`, `Tiebreak`, `ByColumn` or a `Cond` column; a missing `Scope` rule; and base columns that collide after normalization (`org_unit_id` vs `orgunitid`). Each panics, naming the column.
  - `TestCompileGolden` (table, comparing `page.SQL`, `count.SQL` and `Args` with `testdata/<case>.golden`):
    - `all-scope`
    - `units-scope`
    - `no-units-with-visibility`
    - `no-units-no-visibility` (expects `empty == true`)
    - `unscoped`
    - `search`
    - each condition kind, nested `And(Or(...), When(false, …))`
    - each `Order` modifier
    - `default-sort`
    - `paging-offset`
  - `TestCompileSearchEscapes`: `Q = "  50%_a\\b'  "` binds exactly `%50\%\_a\\b'%`, and `page.SQL` contains neither `50` nor `'`.
  - `TestCompileUnknownSort`: returns an error with code `invalid_request`.
  - `TestCompileShape`: `Shape` lists active filter column names sorted, plus the sort key (e.g. `org_unit_id,status|-start_date`), and never contains values.
  - `FuzzCompile`: random `Q`, `Sort` from the allowed set or random bytes, random `Eq` values and paging. `page.SQL` never contains the fuzzed `Q` or `Eq` values, and every `$n` in the SQL has a matching arg (`len(Args)` equals the highest `$n`).
  - `compile_fail` fixture: a file under build tag `staticcheck_fixture`, containing `s := "x"; data.SQL(s)`. Assert with `go vet -tags staticcheck_fixture` (or `go build`) that it does not compile, run from the test via `exec.Command`.
- [ ] **Step 2:** Run `go test ./internal/platform/data -run 'Register|Compile' && go test ./internal/platform/data -fuzz FuzzCompile -fuzztime 30s`. Expected: FAIL.
- [ ] **Step 3: Implement.**
  - Column references are emitted as `b."<name>"` after validation.
  - Placeholders are numbered by one counter shared by the page and the count compile, starting at `$1`.
  - The scope expression is `(b."<col>" = ANY($n) OR <visibility>)`. With `GrantAll` it is dropped. With no units it is just `<visibility>`. With no units and no visibility, `empty = true`.
  - Search binds `"%" + escaped + "%"` once and ORs `ILIKE $n` over the `Search` columns, with escaping `\` → `\\`, `%` → `\%`, `_` → `\_`.
  - `Sort == ""` uses `Default`. `Tiebreak` is always appended last.
  - `LIMIT $n OFFSET $n` comes from `Paging.Limit()` and `Paging.Offset()`.
- [ ] **Step 4:** Rerun the Step 2 commands. Expected: PASS, and the fuzz test finds nothing.
- [ ] **Step 5: Commit** `feat(platform): compile dynamic lists from code-declared fragments`.

### Task 8: `List.Run`: executor, scanner, count

**Files:**
- Create: `internal/platform/data/run.go`
- Create: `internal/platform/data/run_test.go`

**Interfaces:**
- Consumes: `Compile` (Task 7), `Normalize` (Task 3), `WithQueryName` (Task 1).
- Produces:
  - `type Page[R any] struct{ Items []R; Total int64 }`
  - `func (l List[R]) Run(ctx context.Context, q Query) (Page[R], error)`

  The count strategy is the one chosen in spike S4. The steps below assume (c), batch plus skip. If S4 chose (a), adapt Step 3 and keep the same tests.

- [ ] **Step 1: Write failing tests** on a `pgtest` database with a scratch table of 30 rows across 3 org units and 2 users:
  - `TestRunPage`: with `GrantUnits([1])` and page size 20, `Items` match the expected ids in order and `Total` is the true count.
  - `TestRunPagePastEnd`: page 99 returns `Items == []` and the true `Total`.
  - `TestRunSkipsCountOnShortFirstPage`: with a capturing tracer, a first page shorter than the limit logs one query, not two.
  - `TestRunZeroScopeNoViewer`: `Scope{}`, a list with `Visibility` and no actor in ctx returns an empty page and logs no query.
  - `TestRunVisibilityOnly`: `Scope{}` with an actor sees only that actor's rows.
  - `TestRunScansIntoBaseRow`: `R` is a struct with `pgtype.Date`, `pgtype.Text` and `int64` fields; the values equal the inserted ones.
- [ ] **Step 2:** Run `go test ./internal/platform/data -run Run`. Expected: FAIL.
- [ ] **Step 3: Implement.**
  - Return early on `empty`.
  - Build `viewerFrom(ctx)`.
  - Run the page with `pgx.CollectRows(rows, pgx.RowToStructByName[R])` on `platform.DBFrom(ctx)`, with `WithQueryName(ctx, l.Name, shape)`.
  - Run the count in the same `pgx.Batch` unless it is page 1 and `len(items) < limit`.
  - Normalize errors.
  - Return `Items` as a non-nil slice.
- [ ] **Step 4:** Run the same command. Expected: PASS.
- [ ] **Step 5: Commit** `feat(platform): run dynamic lists with a batched count`.

### Task 9: Lint enforcement and Go lint fixtures

**Files:**
- Modify: `.golangci.yml` (forbidigo patterns and depguard rule from spec §6, with the allowlist paths as `exclusions.rules`)
- Create: `scripts/lint-fixtures/go/fixture.go` (build tag `lintfixture`, one planted violation per rule)
- Create: `scripts/lint-fixtures/go/expected.txt` (the exact `file:line linter` list)
- Create: `scripts/check-go-lint-fixtures.sh`
- Modify: `Makefile` `lint` (run the script)

**Interfaces:**
- Consumes: the names from Tasks 2, 6 and 7 (`platform.JobsFrom`, `data.Static`, `data.GrantAll`, `data.GrantUnits`).

Rules, all with the spec §6 allowlist:

| Pattern | Scope |
|---|---|
| `store\.New` | business code |
| `\.Begin(Tx\|Func)?\(`, `pgx\.Begin(Tx)?Func` | business code |
| `platform\.JobsFrom`, River `Insert`/`InsertTx` | outside `internal/platform/**` |
| `data\.Static\(`, `data\.Grant(All\|Units)` | outside `internal/platform/data/**` and `internal/core/iam/**` |
| `riverqueue/river` client construction (depguard on `river.NewClient` users) | outside `internal/app/**` |

- **The `store.New` and `platform.DBFrom` bans** are added with path exclusions for every module except the ones already migrated. In this plan that is none fully migrated, so the rule is active only in `internal/platform/data/**` and fixtures. Each module enables it when it migrates, in the follow-up plan. Record this in a YAML comment.

- [ ] **Step 1: Write the fixture** with one planted violation per rule, and `expected.txt` listing each expected finding as `fixture.go:<line> forbidigo` / `depguard`.
- [ ] **Step 2:** Run `scripts/check-go-lint-fixtures.sh`. It runs `golangci-lint run --build-tags lintfixture ./scripts/lint-fixtures/go/...`, normalizes the output to `file:line linter`, and diffs it with `expected.txt`. Expected: FAIL (rules not configured yet).
- [ ] **Step 3: Implement** the `.golangci.yml` rules.
- [ ] **Step 4:** Run `scripts/check-go-lint-fixtures.sh && make lint`. Expected: the fixture matches exactly, and `make lint` passes on the real code.
- [ ] **Step 5: Commit** `build: lint rules for data access enforcement`.

### Task 10: Leave request list on the engine

**Files:**
- Modify: `internal/modules/hrm/queries.sql` (add `LeaveListBase` with `-- data:base`, the SQL in spec §2; later remove `ListLeaveRequests`, and `Reports` if `grep -rn "\.Reports(" internal` shows no other caller)
- Create: `internal/modules/hrm/list_leave.go` (the `leaveList` definition)
- Modify: `internal/modules/hrm/service_leave.go:190-218` (`Leaves`)
- Modify: `internal/modules/hrm/deps.go` (none if `IAM *iam.Service` already exposes `DataScope`)
- Create: `internal/modules/hrm/list_leave_test.go`
- Create: `internal/modules/hrm/list_leave_perf_test.go` (build tag `perf`)
- Create: `internal/platform/data/datatest/perf.go` (`Measure`, `Explain` helpers)
- Regenerate: `make gen`

**Interfaces:**
- Consumes: everything above.
- Produces:
  - `var leaveList = data.Register(data.List[store.LeaveListBaseRow]{…})` with:
    - `Name: "hrm.LeaveList"`
    - `Scope: data.ByColumn("org_unit_id")`
    - `Visibility`: `b."user_id" = actor OR b."employee_id" IN (<the Reports recursive SQL with the actor bound>)`
    - `Search: number, employee_code, employee_name`
    - `Sorts`: `start_date` → `Asc`, `-start_date` → `Desc`, `number` → `Asc`, `-number` → `Desc`
    - `Default: []Order{Desc("start_date")}`
    - `Tiebreak: Desc("id")`
  - `Where` exactly as in spec §2's service example: `status`, `org_unit_id`, `employee_id`, `from` → `Gte("end_date")`, `to` → `Lte("start_date")`.
  - In `datatest`:
    - `func Measure(tb testing.TB, warmup, runs int, fn func()) Stats` with `type Stats struct{ P50, P95 time.Duration }`
    - `func Explain(tb testing.TB, conn platform.Conn, c data.Compiled, mode string) string` (`mode` is `force_custom_plan` or `force_generic_plan`; runs `SET LOCAL plan_cache_mode` then `EXPLAIN (ANALYZE, BUFFERS)` inside a rolled-back tx)

- [ ] **Step 1: Write the failing correctness test** `TestLeaveListMatchesOld` in `list_leave_test.go`, while `ListLeaveRequests` still exists.
  - Seed with the existing HRM test helpers: two org units, an HR user scoped to one unit, a manager with two reports, an employee with own requests, and requests in every status spanning dates.
  - For each actor (admin, scoped HR, manager, plain employee, user with no employee) and each combination (no filter, each single filter, `from`+`to`, `q` matching number / code / name, each of the 4 sorts, page 2 with size 20), run the old sqlc query and `leaveList.Run`.
  - Assert the same ids in the same order. Assert the same `Total`, except page-past-end, where new = true count and old = 0.
- [ ] **Step 2: Add the deliberate-change tests.**
  - `TestLeaveListSearchLiteralPercent`: `q = "%"` returns only requests whose number, code or name contains `%`. With the seed that is none.
  - `TestLeaveListPagePastEndTotal`.
- [ ] **Step 3:** Run `go test ./internal/modules/hrm -run 'LeaveList'`. Expected: FAIL (`leaveList` undefined).
- [ ] **Step 4: Implement.** Add the base query, run `make gen`, write `list_leave.go`, and switch `Leaves` to `s.d.IAM.DataScope(ctx, "hrm", PermLeaveView)` + `leaveList.Run`. Mapping to `LeaveListItem` is as today.
- [ ] **Step 5:** Run `go test ./internal/modules/hrm -run 'LeaveList' && go test ./internal/modules/hrm`. Expected: PASS, including the existing `leave_test.go` and `list_filters_test.go`.
- [ ] **Step 6: Write the performance harness** `TestLeaveListPerf` (`//go:build perf`).
  - Seed 200 000 requests as in spike S4 and run `ANALYZE`.
  - The matrix: no filter; `employee_id`; `status`; `q`; each sort; scoped HR; a page at offset 10 000.
  - For old and new, under `force_custom_plan` and `force_generic_plan`, it records:
    - `Explain` (planning and execution time, shared hit and read, rows);
    - `Measure(tb, 20, 200, …)` sequential;
    - p50 and p95 with 8 goroutines.
  - It writes a Markdown table to `t.Log`, and counts distinct `Compiled.SQL` strings across the matrix.
  - Add `BenchmarkLeaveListRun` (`-benchmem`).
- [ ] **Step 7:** Run `go test -tags perf ./internal/modules/hrm -run LeaveListPerf -v -timeout 30m && go test -tags perf ./internal/modules/hrm -bench LeaveListRun -benchmem -run '^$'`.
  - Expected: no combination has p95 worse by more than 20%, or buffers more than doubled, without a cause explained in the PR notes.
  - Save the output for the PR.
  - If a combination regresses, reshape `LeaveListBase` and repeat. If it cannot be fixed, stop and report: per spec, the list stays on a fixed query.
- [ ] **Step 8: Remove the old path.**
  - Delete `ListLeaveRequests` (and `Reports` if it is unused) from `queries.sql`.
  - Delete `TestLeaveListMatchesOld`. Its result is recorded in the PR.
  - Run `make gen && make lint && make test && make e2e`. Expected: all PASS.
- [ ] **Step 9: Commit** `refactor(hrm): leave request list on the data engine`.

### Task 11: Gate

- [ ] **Step 1:** Append `## Vertical slice results` to the spec. Include:
  - Task 10's performance table;
  - the distinct-statement count;
  - the benchmark allocations;
  - the list of call sites wrapped in Task 2;
  - anything the follow-up plan must change.
- [ ] **Step 2:** Commit `docs: data engine vertical slice results`.
- [ ] **Step 3:** **Stop.** Report to the product owner with the acceptance checklist from spec §Acceptance criteria ticked against evidence. The remaining work (9 lists, Sales batched insert, module migrations, docs) is planned after approval.
