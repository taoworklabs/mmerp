# Data Engine — Remaining HRM Lists Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move the HRM contract, overtime and timesheet lists onto the data engine, each proven identical to its old query.

**Architecture:** Same pattern as `internal/modules/hrm/list_leave.go`, `list_employee.go` and `list_payroll.go`:
- a parameterless `-- data:base` query;
- a `data.Register`ed `List` in `list_<name>.go`;
- a `<name>Query(…)` builder;
- the service calls `IAM.DataScope` then `List.Run`.

Each list gets an old-versus-new comparison test before its old query is deleted. The engine does not change.

**Tech Stack:** Go, pgx v5, sqlc v1.30.0 plus `sqlc-gen-data`, Postgres 18.

**Spec:** `docs/superpowers/specs/2026-10-21-data-engine-design.md` (§2, §Behaviour changes).

## Global Constraints

- No engine changes, no migrations, no indexes, no performance measurement. If a list needs an engine change, stop and report.
- **Behaviour is identical to the old queries**, except the deliberate changes:
  - search trims `q` and escapes `%`, `_` and `\`;
  - a page past the end returns the true total;
  - for internal callers only, an empty sort uses the list's default order.
- Never mock the database. The app pool is read-only outside `platform.InTx`. Tests seed through the fixture's writable pool or through the service.
- Commits follow Conventional Commits, stage explicit paths, and end with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- Comments are short and say why. They never reference specs, plans, tasks or commits.
- Branch `feat/data-engine-hrm-lists`, stacked on `feat/data-engine-slice-2` (PR #123).
- Offline `make gen`: `GOPROXY=file://$(go env GOMODCACHE)/cache/download make gen`.

## Review Focus

1. **Expiring contracts** include a posted original ending today and one ending on today + 30, and exclude appendices, drafts and contracts with no end date. Pinned in Task 1 (`TestContractListExpiringBounds`).
2. **The `-end_date` sort** puts open-ended contracts last. Covered by the Task 1 comparison matrix.
3. **Overtime visibility**: a manager sees their own requests and those of reports at any depth, never a sibling's. Pinned in Task 2 (`TestOvertimeListVisibility`).
4. **Overtime `from`/`to`** filter on the single `date` and are inclusive. Covered by the Task 2 comparison matrix.
5. **Timesheets in the same period** are ordered by org unit name, then `id` descending, for every sort key. Pinned in Task 3 (`TestTimesheetListOrgUnitTiebreak`).

---

### Task 1: Contract list on the engine

**Files:**
- `internal/modules/hrm/queries.sql`: add `ContractListBase`; later delete `ListContracts`.
- Create `internal/modules/hrm/list_contract.go`.
- Create `internal/modules/hrm/list_contract_test.go`.
- `internal/modules/hrm/service_contract.go:215-246` (`Contracts`).
- `make gen`.

**Interfaces:**
- Consumes:
  - `iam.(*Service).DataScope(ctx, "hrm", PermContractView)`;
  - `s.d.Setting.Today(ctx)`.
- Produces:
  - `var contractList = data.Register(data.List[store.ContractListBaseRow]{…})`;
  - `func contractQuery(sc data.Scope, f ContractFilter, today pgtype.Date) data.Query`.

- [ ] **Step 1: Base query.** Then run `make gen`.

  ```sql
  -- name: ContractListBase :many
  -- data:base
  SELECT c.id, d.number, d.status, d.org_unit_id, c.parent_id, p.number AS parent_number, c.contract_type_id,
         t.name AS contract_type_name, c.start_date, c.end_date, c.employee_id, e.code AS employee_code,
         e.full_name AS employee_name
  FROM hrm.contracts c
  JOIN record.documents d ON d.id = c.id
  JOIN hrm.contract_types t ON t.id = c.contract_type_id
  JOIN hrm.employees e ON e.id = c.employee_id
  LEFT JOIN record.documents p ON p.id = c.parent_id;
  ```

- [ ] **Step 2: Failing tests** in `list_contract_test.go`:
  - `TestContractListMatchesOld`: compare the old sqlc query with `Contracts()` over this matrix:
    - **Actors:** admin, unit-scoped HR, no role.
    - **Filters:**
      - none;
      - each `status`;
      - `contract_type_id`;
      - `org_unit_id`;
      - `employee_id`;
      - `expiring`;
      - `q` on number, code and name;
      - combinations of these.
    - **Sorts:** all 6.
    - **Pages:** pages 1 and 2, plus a page past the end.

    Seed enough contracts for 2+ pages, including open-ended ones, appendices and drafts. Assert the same ids, in the same order, with the same `Total`, except on a page past the end.
  - `TestContractListExpiringBounds`. With `expiring`:
    - included: a posted original ending today, and one ending today + 30;
    - excluded: one ending today + 31, one ending yesterday, an appendix, a draft, and an open-ended contract.
  - `TestContractListSearchLiteralPercent`.

  Expected: FAIL.
- [ ] **Step 3: Implement.**
  - `contractList`:
    - `Name: "hrm.ContractList"`;
    - `Scope: data.ByColumn("org_unit_id")`;
    - `Search: number, employee_code, employee_name`;
    - `Sorts`:
      - `start_date` → `Asc`, `-start_date` → `Desc`;
      - `end_date` → `Asc`, `-end_date` → `Desc("end_date").NullsLast()`;
      - `number` → `Asc`, `-number` → `Desc`;
    - `Default: []data.Order{data.Desc("start_date")}`;
    - `Tiebreak: data.Desc("id")`.
  - `contractQuery`, `Where`:
    - `When(f.Status != "", Eq("status", f.Status))`;
    - `When(f.ContractTypeID != 0, Eq("contract_type_id", …))`;
    - `When(f.OrgUnitID != 0, Eq("org_unit_id", …))`;
    - `When(f.EmployeeID != 0, Eq("employee_id", …))`;
    - `When(f.Expiring, data.SQL(`(b."parent_id" IS NULL AND b."status" = 'posted' AND b."end_date" BETWEEN `).Arg(today).Text(`::date AND `).Arg(today).Text(`::date + 30)`).Cond())`.
  - `Contracts`: switch to `DataScope`, `Today` and `contractList.Run`. Keep the mapping, including `AllowedActions: []string{}`.
- [ ] **Step 4:** Run `go test ./internal/modules/hrm -count=1`. Expected: PASS, including the existing `TestContractList`.
- [ ] **Step 5:**
  - Delete `ListContracts`, then run `make gen`.
  - Delete the comparison test. Keep its last output in the report.
  - Run `make lint && go test ./internal/modules/hrm/... -count=1`.
- [ ] **Step 6:** Commit `refactor(hrm): contract list on the data engine`.

### Task 2: Overtime list on the engine

**Files:**
- `queries.sql`: add `OvertimeListBase`; later delete `ListOvertimeRequests` and `Reports`.
- Create `list_overtime.go`.
- Create `list_overtime_test.go`.
- `service_overtime.go:107-137` (`Overtimes`).
- `make gen`.

**Interfaces:**
- Consumes: `DataScope(ctx, "hrm", PermOvertimeView)`.
- Produces:
  - `var overtimeList`;
  - `func overtimeQuery(sc data.Scope, f OvertimeFilter) data.Query`.

- [ ] **Step 1: Base query** (`r.day_hours::text`, `r.night_hours::text` as today, plus `d.org_unit_id`). Then run `make gen`.

  ```sql
  -- name: OvertimeListBase :many
  -- data:base
  SELECT r.id, d.number, d.status, d.org_unit_id, r.employee_id, e.code AS employee_code, e.full_name AS employee_name,
         r.date, r.day_kind, r.day_hours::text AS day_hours, r.night_hours::text AS night_hours
  FROM hrm.overtime_requests r
  JOIN record.documents d ON d.id = r.id
  JOIN hrm.employees e ON e.id = r.employee_id;
  ```

- [ ] **Step 2: Failing tests** in `list_overtime_test.go`:
  - `TestOvertimeListMatchesOld`: compare the old query (with the `Reports` lookup, as `Overtimes` does today) with `Overtimes()`:
    - **Actors:** admin, scoped HR, a depth-2 manager, a plain employee, a user with no employee.
    - **Filters:**
      - none;
      - each `status`;
      - `org_unit_id`;
      - `employee_id`;
      - `from`, `to`, and both together;
      - `q`.
    - **Sorts:** all 4.
    - **Pages:** pages 1 and 2, plus a page past the end.
  - `TestOvertimeListVisibility`: exact id sets per actor, in the shape of `TestLeaveListVisibility`. Cover the manager, a mid-level employee (sees own and report, not sibling), a sibling, a leaf, scoped HR (unit plus own outside the unit), no-role, and admin.
  - `TestOvertimeListSearchLiteralPercent`.

  Expected: FAIL.
- [ ] **Step 3: Implement.**
  - `overtimeList`:
    - `Name: "hrm.OvertimeList"`;
    - `Scope: data.ByColumn("org_unit_id")`;
    - `Visibility`: the same fragment as `leaveList`'s (actor's employee plus everyone below it, actor bound with `Arg`);
    - `Search: number, employee_code, employee_name`;
    - `Sorts`: `date` / `-date`, `number` / `-number`;
    - `Default: []data.Order{data.Desc("date")}`;
    - `Tiebreak: data.Desc("id")`.
  - `overtimeQuery`, `Where`:
    - `status`, `org_unit_id`, `employee_id` as `Eq`;
    - `from` → `Gte("date", from)`;
    - `to` → `Lte("date", to)`;
    - parse both dates with `platform.NullDate(platform.Optional(…))`.
  - Factor the shared fragment only if it removes a verbatim copy. Put it in a small helper in `list_leave.go` and use it from both lists.
- [ ] **Step 4:** Run `go test ./internal/modules/hrm -count=1`. Expected: PASS.
- [ ] **Step 5:**
  - Delete `ListOvertimeRequests` and `Reports`. First confirm with `grep -rn "Reports(" internal` that nothing else calls `Reports`.
  - Run `make gen`.
  - Delete the comparison test. Keep its output in the report.
  - Run `make lint && go test ./internal/modules/hrm/... -count=1`.
- [ ] **Step 6:** Commit `refactor(hrm): overtime list on the data engine`.

### Task 3: Timesheet list on the engine

**Files:**
- `queries.sql`: add `TimesheetListBase`; later delete `ListTimesheets`.
- Create `list_timesheet.go`.
- Create `list_timesheet_test.go`.
- `service_timesheet.go:42-63` (`Timesheets`).
- `make gen`.

**Interfaces:**
- Consumes:
  - `DataScope(ctx, "hrm", PermTimesheetView)`;
  - `platform.MonthPeriod`.
- Produces:
  - `var timesheetList`;
  - `func timesheetQuery(sc data.Scope, f TimesheetFilter, start pgtype.Date) data.Query`.

- [ ] **Step 1: Base query.** Then run `make gen`.

  ```sql
  -- name: TimesheetListBase :many
  -- data:base
  SELECT t.id, d.number, d.status, t.org_unit_id, u.name AS org_unit_name, t.period_start, t.period_end
  FROM hrm.timesheets t
  JOIN record.documents d ON d.id = t.id
  JOIN iam.org_units u ON u.id = t.org_unit_id;
  ```

- [ ] **Step 2: Failing tests** in `list_timesheet_test.go`:
  - `TestTimesheetListMatchesOld`:
    - **Actors:** admin, scoped HR, no role.
    - **Filters:**
      - none;
      - each `status`;
      - `org_unit_id`;
      - `month`;
      - combinations of these.
    - **Sorts:** all 4, plus empty (must equal `-period`).
    - **Pages:** pages 1 and 2, plus a page past the end.

    Seed several units with timesheets in the same periods, with unit names whose order differs from their id order.
  - `TestTimesheetListOrgUnitTiebreak`: for the same period, rows are ordered by unit name then `id` descending, under `-period` and under `number`.

  Expected: FAIL.
- [ ] **Step 3: Implement.**
  - `timesheetList`:
    - `Name: "hrm.TimesheetList"`;
    - `Scope: data.ByColumn("org_unit_id")`;
    - no `Search`;
    - `Sorts`, each followed by `data.Asc("org_unit_name")`:
      - `period` → `{Asc("period_start"), Asc("org_unit_name")}`;
      - `-period` → `{Desc("period_start"), Asc("org_unit_name")}`;
      - `number` → `{Asc("number"), Asc("org_unit_name")}`;
      - `-number` → `{Desc("number"), Asc("org_unit_name")}`;
    - `Default: {Desc("period_start"), Asc("org_unit_name")}`;
    - `Tiebreak: data.Desc("id")`.
  - `timesheetQuery`, `Where`: `status`, `org_unit_id`, and `When(start.Valid, Eq("period_start", start))`.
  - `Timesheets`: switch to `DataScope` and `Run`. A `Month` parse error still returns, as today.
- [ ] **Step 4:** Run `go test ./internal/modules/hrm -count=1`. Expected: PASS, including `TestListsSearchAndFilter`, `timesheet_test.go` and `payroll_test.go`.
- [ ] **Step 5:**
  - Delete `ListTimesheets`, then run `make gen`.
  - Delete the comparison test. Keep its output in the report.
  - Run `make lint && make test`.
- [ ] **Step 6:** Commit `refactor(hrm): timesheet list on the data engine`.
