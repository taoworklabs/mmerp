# Data Engine Slice 2 — Employee and Payroll Lists Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move the HRM employee list and payroll list onto the data engine, proving it on a recursive-CTE filter, a date-relative status and a legal-entity scope, then stop at a gate with measurements.

**Architecture:** Same pattern as the leave list (`internal/modules/hrm/list_leave.go`): a parameterless `-- data:base` query in `queries.sql`, a `data.Register`ed `List` in a `list_<name>.go` file, a `<name>Query(sc, f, …) data.Query` builder, and the service calling `IAM.DataScope` then `List.Run`. Each migration is proven by an old-versus-new comparison test before the old query is deleted. The engine itself does not change.

**Tech Stack:** Go, pgx v5, sqlc v1.30.0 plus the `sqlc-gen-data` plugin, Postgres 18.

**Spec:** `docs/superpowers/specs/2026-10-21-data-engine-design.md` (§2 Dynamic lists, §Behaviour changes, §Performance and correctness verification, "Vertical slice results"). Reference implementation: `internal/modules/hrm/list_leave.go`, `list_leave_test.go`, `list_leave_perf_test.go`.

## Global Constraints

- No engine changes (`internal/platform/data`). If a list needs one, stop and report.
- No migrations and no new indexes. If performance needs an index, stop and report with numbers.
- **Behaviour is identical to the old queries**, except the spec's deliberate changes:
  - search escapes `%`, `_` and `\` and trims `q`;
  - a page past the end returns the true total.
- **Payroll scope stays exactly as today:** `legal_entity_id = ANY(units)`. A role granted below the company level sees no payroll, and that is existing behaviour, not a bug to fix here.
- Never mock the database.
  - The app pool is read-only outside `platform.InTx`.
  - Tests seed through the fixture's `system` pool.
- Commits follow Conventional Commits, stage explicit paths, and end with `Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`.
- Comments are short and say why. They never reference specs, ADRs, plans or tasks.
- Branch `feat/data-engine-slice-2`.

## Review Focus

1. **An employee terminated exactly today** shows as `active` and matches `status=active`, not `terminated` (the boundary is `termination_date < today`). Pinned in Task 3 (`TestEmployeeListTerminatedTodayIsActive`).
2. **`manager_of` set to an employee at the bottom of a cycle-free chain** excludes that employee and everyone below, and nobody else. Covered by the existing `manager_of` test plus the comparison matrix in Task 3.
3. **A payroll user whose role is granted at a department** sees no payroll, as today. Pinned in Task 1 (`TestPayrollListScope`).
4. **`month=2026-02`** returns only payrolls whose period starts on 2026-02-01, not periods that merely overlap February. Pinned in Task 1 (`TestPayrollListFilters`).
5. **Employee search with `%`** matches literally. Pinned in Task 3 (`TestEmployeeListSearchLiteralPercent`).

---

### Task 1: Pin the payroll list's current behaviour

There is no test of `Payrolls` today. These tests run against the **current** code first, so Task 2 has a net.

**Files:**
- Create: `internal/modules/hrm/list_payroll_test.go`

**Interfaces:**
- Consumes:
  - `(*Service).Payrolls(ctx, PayrollFilter) (PayrollList, error)` (`service_payroll.go:461`);
  - `PayrollFilter{LegalEntityID int64; Month string; Status string; platform.Paging}`;
  - the HRM test fixture in `service_test.go` / `payroll_test.go`. Reuse its helpers for legal entities, payroll creation and actors with roles.
- Produces: `newPayrollListFixture(t) *payrollListFixture`. It has:
  - two legal entities, `LE1` and `LE2`, each with a department below it;
  - payrolls for months 2026-01, 2026-02 and 2026-03 in both entities, in mixed statuses: at least one `draft`, one `posted` and one `cancelled`;
  - actors:
    - `admin` (tenant-wide);
    - `payLE1`, with the payroll role granted at `LE1`;
    - `payDept`, with the same role granted at a department under `LE1`;
    - `nobody`.

  Task 2 reuses this fixture.

- [ ] **Step 1: Write the tests** (all expected to PASS on current code):
  - `TestPayrollListScope`:
    - `admin` sees all payrolls;
    - `payLE1` sees exactly `LE1`'s payrolls;
    - `payDept` sees none (`Total == 0`);
    - `nobody` sees none.
  - `TestPayrollListFilters`, each filter checked alone and combined:
    - `Status` (each value);
    - `LegalEntityID`;
    - `Month: "2026-02"`, which matches only `period_start = 2026-02-01`.

    Assert exact id lists.
  - `TestPayrollListOrder`: with no filter, the order is `period_start DESC`, then legal entity name ascending, then `id DESC`. Assert the exact id sequence.
  - `TestPayrollListPaging`:
    - `PageSize 20` with a fixture larger than one page, if feasible; otherwise assert that page 1 holds all rows.
    - A page past the end returns `Items == []` and `Total == 0`. This is today's behaviour; Task 2 flips it.
- [ ] **Step 2:** Run `go test ./internal/modules/hrm -run 'PayrollList' -count=1`. Expected: PASS.
- [ ] **Step 3: Commit** `test(hrm): pin payroll list scope, filters, order and paging`.

### Task 2: Payroll list on the engine

**Files:**
- Modify: `internal/modules/hrm/queries.sql`:
  - add `PayrollListBase`;
  - later delete `ListPayrolls`.
- Create: `internal/modules/hrm/list_payroll.go`
- Modify: `internal/modules/hrm/service_payroll.go:461-477` (`Payrolls`)
- Modify: `internal/modules/hrm/list_payroll_test.go`
- Regenerate: `make gen`

**Interfaces:**
- Consumes:
  - Task 1's fixture;
  - `iam.(*Service).DataScope(ctx, "hrm", PermPayrollView) (data.Scope, error)`;
  - `platform.MonthPeriod(month) (start, end pgtype.Date, err error)`.
- Produces:
  - `var payrollList = data.Register(data.List[store.PayrollListBaseRow]{…})`;
  - `func payrollQuery(sc data.Scope, f PayrollFilter, start pgtype.Date) data.Query`.

- [ ] **Step 1: Base query.** It is `ListPayrolls`'s `SELECT`/`FROM`/`JOIN` without the `WHERE`, `ORDER BY`, `LIMIT` and window count:

  ```sql
  -- name: PayrollListBase :many
  -- data:base
  SELECT p.id, d.number, d.status, d.legal_entity_id, u.name AS legal_entity_name,
         p.period_start, p.period_end, p.computed_at
  FROM hrm.payrolls p
  JOIN record.documents d ON d.id = p.id
  JOIN iam.org_units u ON u.id = d.legal_entity_id;
  ```

  Run `make gen`.
- [ ] **Step 2: Write the failing comparison test** `TestPayrollListMatchesOld`. While `ListPayrolls` still exists, for each actor (admin, payLE1, payDept, nobody) × each filter combination from Task 1 × pages 1 and 2 (size 20) and a page past the end:
  - run the old sqlc query and `payrollList.Run`;
  - assert the same ids in the same order;
  - assert the same `Total`, except a page past the end, where new = true count and old = 0.

  Run it. Expected: FAIL to compile (`payrollList` undefined).
- [ ] **Step 3: Implement `list_payroll.go`:**
  - `Name: "hrm.PayrollList"`;
  - `Scope: data.ByColumn("legal_entity_id")`;
  - no `Visibility`, no `Search`, no `Sorts`;
  - `Default: []data.Order{data.Desc("period_start"), data.Asc("legal_entity_name")}`;
  - `Tiebreak: data.Desc("id")`.

  `payrollQuery`'s `Where`:
  - `When(f.Status != "", Eq("status", f.Status))`;
  - `When(f.LegalEntityID != 0, Eq("legal_entity_id", f.LegalEntityID))`;
  - `When(start.Valid, Eq("period_start", start))`.

  Switch `Payrolls` to `DataScope` + `payrollList.Run`, mapping rows as today.
- [ ] **Step 4:** Run `go test ./internal/modules/hrm -run 'PayrollList' -count=1`. Expected: all PASS, except `TestPayrollListPaging`'s past-end assertion. Change that assertion to the true total and note it as the deliberate change.
- [ ] **Step 5: Remove the old path.**
  - Delete `ListPayrolls` from `queries.sql` and run `make gen`.
  - Delete `TestPayrollListMatchesOld`; paste its last passing output into the task report.
  - Run `make lint && go test ./internal/modules/hrm/... ./internal/app/... -count=1`. Expected: PASS.
- [ ] **Step 6: Commit** `refactor(hrm): payroll list on the data engine`.

Performance: `hrm.payrolls` holds one row per legal entity per month, so a few hundred rows at most. No perf harness. Record one `EXPLAIN (ANALYZE, BUFFERS)` of the old and new query on the Task 1 fixture in the report.

### Task 3: Employee list on the engine

**Files:**
- Modify: `internal/modules/hrm/queries.sql`:
  - add `EmployeeListBase`;
  - later delete `ListEmployees`.
- Create: `internal/modules/hrm/list_employee.go`
- Create: `internal/modules/hrm/list_employee_test.go`
- Modify: `internal/modules/hrm/service_employee.go:43-66` (`Employees`)
- Modify: `internal/modules/hrm/service_test.go`. In `TestListFiltersSortAndPages`, only the page-past-end case changes, from `"/0"` to the true total.
- Regenerate: `make gen`

**Interfaces:**
- Consumes:
  - `iam.(*Service).DataScope(ctx, "hrm", PermView)`;
  - `s.d.Setting.Today(ctx) (pgtype.Date, error)`;
  - `status(termination, today pgtype.Date) string` (`service_employee.go:405`, unchanged).
- Produces:
  - `var employeeList = data.Register(data.List[store.EmployeeListBaseRow]{…})`;
  - `func employeeQuery(sc data.Scope, f EmployeeFilter, today pgtype.Date) data.Query`.

- [ ] **Step 1: Base query, then `make gen`:**

  ```sql
  -- name: EmployeeListBase :many
  -- data:base
  SELECT e.id, e.code, e.full_name, e.org_unit_id, u.name AS org_unit_name, e.email, e.phone,
         e.hire_date, e.termination_date
  FROM hrm.employees e JOIN iam.org_units u ON u.id = e.org_unit_id;
  ```

- [ ] **Step 2: Write the failing tests** in `list_employee_test.go`:
  - `TestEmployeeListMatchesOld`. While `ListEmployees` exists, compare the old sqlc query with `employeeList.Run`, asserting the same ids in the same order and the same `Total` (except a page past the end). Run it over:
    - actors: admin, unit-scoped HR, no role;
    - filters: none, `q` matching code / name / partial with diacritics, each `status`, `org_unit_id`, `manager_of` (a top manager, a middle one, a leaf), and combinations;
    - each of the 6 sorts;
    - pages 1 and 2 (size 20), and a page past the end.

    Seed enough employees for 2+ pages, with terminations before, on and after today, and a 3-level manager chain.
  - `TestEmployeeListTerminatedTodayIsActive`: an employee whose `termination_date` = today.
    - It is listed with `Status == "active"`.
    - It appears under `status=active` and not under `status=terminated`.
  - `TestEmployeeListSearchLiteralPercent`: `q = "%"` returns no employee unless one's code or name contains `%`. Seed none, so `Total == 0`.

  Run `go test ./internal/modules/hrm -run 'EmployeeList' -count=1`. Expected: FAIL to compile.
- [ ] **Step 3: Implement `list_employee.go`:**
  - `Name: "hrm.EmployeeList"`;
  - `Scope: data.ByColumn("org_unit_id")`;
  - `Search: []string{"code", "full_name"}`;
  - `Sorts`:

    | Key | Order |
    |---|---|
    | `code` | `Asc("code").Lower()` |
    | `-code` | `Desc("code").Lower()` |
    | `full_name` | `Asc("full_name")` |
    | `-full_name` | `Desc("full_name")` |
    | `hire_date` | `Asc("hire_date")` |
    | `-hire_date` | `Desc("hire_date")` |

  - `Default: []data.Order{data.Asc("code").Lower()}`;
  - `Tiebreak: data.Asc("id")`.

  `employeeQuery`'s `Where`:
  - `When(f.OrgUnitID != 0, Eq("org_unit_id", f.OrgUnitID))`.
  - Status:
    - `active` → `Or(IsNull("termination_date"), Gte("termination_date", today))`;
    - `terminated` → `data.SQL(`b."termination_date" < `).Arg(today).Cond()`.
  - `manager_of`, only when set: a fragment excluding that employee and everyone below it, with the same recursion as the old CTE:

    ```go
    data.SQL(`b."id" NOT IN (WITH RECURSIVE under(id) AS (SELECT `).Arg(f.ManagerOf).
        Text(`::bigint UNION SELECT s.id FROM hrm.employees s JOIN under ON s.manager_id = under.id) SELECT id FROM under)`).Cond()
    ```

  Switch `Employees` to `DataScope` + `Setting.Today` + `employeeList.Run`, keeping the per-row `status(...)` mapping.
- [ ] **Step 4:** Run `go test ./internal/modules/hrm -count=1`. Expected: PASS, after updating `TestListFiltersSortAndPages`' page-past-end case to the true total.
- [ ] **Step 5: Remove the old path.**
  - Delete `ListEmployees` and run `make gen`.
  - Delete `TestEmployeeListMatchesOld`; keep its last passing output in the report.
  - Run `make lint && make test`. Expected: PASS.
- [ ] **Step 6: Commit** `refactor(hrm): employee list on the data engine`.

### Task 4: Employee list performance, results and gate

**Files:**
- Create: `internal/modules/hrm/list_employee_perf_test.go` (`//go:build perf`). Follow `list_leave_perf_test.go`, and keep the old SQL as constants so it can be re-run.
- Modify: `docs/superpowers/specs/2026-10-21-data-engine-design.md` (append `## Slice 2 results (YYYY-MM-DD)`)

**Interfaces:**
- Consumes:
  - `datatest.Measure(tb, warmup, runs, fn) Stats{P50, P95}`;
  - `datatest.Explain(tb, conn, c data.Compiled, mode string) string`;
  - `employeeList`, `employeeQuery` (Task 3).

- [ ] **Step 1: Write `TestEmployeeListPerf`.**
  - Seed 50 000 employees across 50 org units, with 10% terminated and manager chains up to depth 5. Then `ANALYZE`.
  - Matrix:
    - no filter;
    - `status=active`;
    - `status=terminated`;
    - `org_unit_id`;
    - `q` (code prefix);
    - `q` (name fragment);
    - each of the 6 sorts;
    - `manager_of` (a top manager);
    - scoped HR (5 units);
    - offset 10 000.
  - For old and new, under `force_custom_plan` and `force_generic_plan`, it first checks that both return the same ids and total. Then it records:
    - planning and execution time;
    - shared hit and read buffers;
    - rows;
    - p50 and p95 sequential (`Measure(tb, 20, 200, …)`);
    - p95 at 8 concurrent clients.
  - It counts distinct `Compiled.SQL` strings.
  - Add `BenchmarkEmployeeListRun` (`-benchmem`).
- [ ] **Step 2: Run it in the background** with `go test -tags perf ./internal/modules/hrm -run EmployeeListPerf -v -timeout 240m` against a Postgres with `--shm-size=1g` (a throwaway `postgres:18` container is fine; the compose default of 64 MB fails with `53100`). Save the output.
  - Regression means p95 worse by more than 20%, or buffers more than doubled, without a stated cause.
  - Do not add indexes. Report instead.
- [ ] **Step 3: Write the results section** in the spec:
  - the correctness evidence of Tasks 2–3;
  - the employee perf table in the format of "Vertical slice results";
  - the payroll `EXPLAIN` summary;
  - whether the custom-plan pattern of the leave list (the separate count costs more under custom plans) repeats.

  The section must end with **"Open: product-owner decision on the count strategy, now with data from 2 lists."**
- [ ] **Step 4: Commit** `test(hrm): employee list performance harness` (the perf file) and `docs: data engine slice 2 results` (the spec), as two commits.
- [ ] **Step 5: Stop.** Report to the product owner. The remaining 7 lists and the module migration are planned after this gate.
