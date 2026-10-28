# Termination workflow and severance allowance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** An `hrm.termination` document that alone sets an employee's termination date, warns about what is still open, and makes the termination-month payroll pay the severance or job-loss allowance.

**Architecture:** New business area `termination` inside `internal/modules/hrm` (files `termination*.go`, `queries/termination.sql`), registered with `record` like overtime/contracts. Allowance rules are pure functions in `termination_allowance.go`. The payroll reads the posted termination through its basis (hashed, non-sensitive) and the 6 months of contract terms in `payInputs` (encrypted, captured). Frontend pages in `web/src/hrm` follow the contract pages.

**Tech Stack:** Go, sqlc, Huma, Postgres; React, TanStack Query, `@mmerp/ui`; excelize for the samples.

**Spec:** `docs/superpowers/specs/2026-10-28-termination-design.md` (approved), ADR `docs/adr/0046-termination-document.md`.

## Global Constraints

- Work only in `/home/jarviisha/development/mmerp-wt/termination`. Never run `make e2e`, `make dev`, `make reset-db`, `make seed`. `go test`, `pnpm` unit tests, `tsc`, `make gen`, `make lint` are fine.
- Go tests: `make db` once (compose Postgres on `:5433`, or `TEST_DATABASE_URL` from `local.mk`), then `go test ./internal/modules/hrm -run <Name>`.
- Money `int64` in đồng; averages and years in `shopspring/decimal`; the one rounding is `platform.Round`. Never float.
- Services query only through `store.<Query>.Run(ctx, …)` and open transactions with `platform.InTx`.
- Lock order in `OnTransition`: record row (by `record`) → `hrm.payroll_locks` `FOR SHARE` (`checkPayrollPeriods`) → employee `FOR NO KEY UPDATE` (`store.LockEmployee`).
- One migration only, created with `make migration name=hrm_terminations`; it creates `hrm.terminations` and touches no other table.
- Error codes (stable): `termination_exists`, `termination_before_hire`, `termination_via_document`, `termination_legal_entity_changed`, `payroll_period_closed` (existing). Warning codes: `contract_in_force_after`, `leave_after_termination`, `overtime_after_termination`, `leave_balance_to_pay`.
- Reason codes exactly: `contract_expired, work_completed, mutual_agreement, imprisonment, foreigner_expelled, death, employer_ceased, dismissal, employee_resignation, employer_unilateral, absent_without_reason, restructuring, work_permit_expired, probation_failed`.
- Permissions `hrm.termination.view`, `hrm.termination.edit`, added to role `hr`. Number prefix `NV`.
- No allowance amount in `hrm.terminations`, audit data, posting lines or logs.
- All user text through i18n in `vi` and `en`; comments short, no references to docs/ADRs.
- Commits: Conventional Commits, scope `hrm` or `web`, ending with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

## Review Focus

- A termination whose date falls in a closed period (late entry): posting refused with `payroll_period_closed` naming that period; nothing changes on the employee. → Task 3 test `TestTerminationLateEntry`.
- Cancelling a termination after the payroll of its month closed: refused, `termination_date` unchanged. → Task 3 test `TestTerminationCancel`.
- A legacy employee (date set on create, no document): termination posts over it and cancel restores the old date. → Task 3 test `TestTerminationLegacyDate`.
- A changed termination date in an import file for an existing employee: row error, nothing written; same date or empty cell passes. → Task 4 test `TestEmployeeImportTerminationDate`.
- A draft payroll computed before the termination posted: becomes stale (hash changes), and recalculating adds the allowance. → Task 6 test `TestPayrollSeveranceStale`.

---

### Task 1: Allowance rules as data and pure functions

**Files:**
- Create: `internal/modules/hrm/termination_allowance.go`
- Test: `internal/modules/hrm/termination_allowance_test.go` (package `hrm`, no database)

**Interfaces:**
- Produces:
  - `type allowanceKind string` with `allowanceNone = "none"`, `allowanceSeverance = "severance"`, `allowanceJobLoss = "job_loss"`.
  - `var terminationReasons = []struct{ Code string; Kind allowanceKind }{…}` in the order of the Global Constraints list, with the kinds from the spec table.
  - `func reasonAllowance(code string) (allowanceKind, bool)` (false for an unknown code).
  - `func monthsWorked(hire, termination string) int` — whole months from hire to termination inclusive, leftover days dropped.
  - `func allowanceYears(months int) decimal.Decimal` — whole years + ½ for remainder 1–6, +1 for 7–11.
  - `type allowanceBasis struct { Kind allowanceKind `json:"kind"`; Worked, Insured, Paid int `json:"…"`; Pension bool }` with `func (b allowanceBasis) kind() allowanceKind` (none when Worked < 12, or severance with Pension) and `func (b allowanceBasis) years() decimal.Decimal` (allowanceYears(max(0, Worked−Insured−Paid))).
  - `func allowanceAmount(kind allowanceKind, years, average decimal.Decimal) decimal.Decimal` — severance years×½×average; job loss max(years, 2)×average; none 0. Unrounded; callers round.

- [ ] **Step 1: Write the failing tests**

```go
func TestReasonAllowanceTable(t *testing.T) {
	want := map[string]allowanceKind{
		"contract_expired": allowanceSeverance, "work_completed": allowanceSeverance, "mutual_agreement": allowanceSeverance,
		"imprisonment": allowanceSeverance, "foreigner_expelled": allowanceNone, "death": allowanceSeverance,
		"employer_ceased": allowanceSeverance, "dismissal": allowanceNone, "employee_resignation": allowanceSeverance,
		"employer_unilateral": allowanceSeverance, "absent_without_reason": allowanceNone, "restructuring": allowanceJobLoss,
		"work_permit_expired": allowanceNone, "probation_failed": allowanceNone,
	}
	// every row, and no other row
}
func TestMonthsWorked(t *testing.T) // ("2007-07-01","2026-04-15")=225; ("2024-03-01","2026-04-30")=26; ("2025-04-16","2026-04-15")=12; ("2025-04-17","2026-04-15")=11
func TestAllowanceYears(t *testing.T) // 0→0, 1→0.5, 6→0.5, 7→1, 12→1, 18→1.5, 19→2
func TestAllowanceBasis(t *testing.T) // Worked 11 → none; severance+Pension → none; job_loss+Pension → job_loss; Worked 225 Insured 207 → years 1.5
func TestAllowanceAmount(t *testing.T) // severance 1.5×22_000_000 → 16_500_000; job_loss years 0 avg 15_000_000 → 30_000_000; job_loss years 3 → 45_000_000
```

- [ ] **Step 2: Run** `go test ./internal/modules/hrm -run 'TestReasonAllowance|TestMonthsWorked|TestAllowance'` — expect compile failure.
- [ ] **Step 3: Implement** the functions above in `termination_allowance.go`; one comment line per rule naming the article (e.g. `// Art 8.3 Decree 145/2020: a remainder up to 6 months is half a year.` is allowed: it cites law, not repo docs).
- [ ] **Step 4: Run** the same command — PASS.
- [ ] **Step 5: Commit** `feat(hrm): severance and job-loss allowance rules`

### Task 2: The termination document (table, CRUD, record type, recordtest)

**Files:**
- Create: `migrations/<timestamp>_hrm_terminations.sql` (via `make migration name=hrm_terminations`)
- Create: `internal/modules/hrm/queries/termination.sql`, `termination.go`, `termination_types.go`, `termination_handler.go`, `termination_list.go`, `termination_test.go`
- Modify: `internal/modules/hrm/service.go` (const `terminationType = "hrm.termination"`, register type), `types.go` (perms, errors), `handler.go` (register handlers), `sqlc.yaml` only if queries are listed per file (check), `internal/modules/hrm/i18n/*.json` (reason labels used by approval options)

**Interfaces:**
- Consumes: `reasonAllowance` (Task 1) to validate `reason`.
- Produces:
  - Table `hrm.terminations(id bigint PK REFERENCES record.documents(id), employee_id bigint NOT NULL REFERENCES hrm.employees(id), reason text NOT NULL, decision_date date, pension_eligible bool NOT NULL DEFAULT false, insured_months int NOT NULL CHECK (insured_months >= 0), paid_months int NOT NULL DEFAULT 0 CHECK (paid_months >= 0), handover text, note text, previous_date date)`, index on `employee_id`. The termination date is `record.documents.date`.
  - Types: `TerminationFields{TerminationDate string; Reason string (enum of the 14 codes); DecisionDate *string; PensionEligible bool; InsuredMonths int (minimum 0); PaidMonths int (minimum 0); Handover *string (maxLength 4000); Note *string (maxLength 1000)}`, `NewTermination{RequestID string; EmployeeID int64; TerminationFields}`, `TerminationUpdate{Version int32; TerminationFields}`, `Termination{ID, Number, Status, Version, EmployeeID, EmployeeCode, EmployeeName, TerminationFields, AllowedActions []string}` (Task 5 adds warnings and allowance), `TerminationListItem{ID, Number, Status, EmployeeID, EmployeeCode, EmployeeName, OrgUnitName, TerminationDate, Reason}`, `TerminationFilter{Q, Status, OrgUnitID, EmployeeID, Sort ("-date" default), platform.Paging}`.
  - Service: `CreateTermination(ctx, NewTermination) (int64, error)`, `UpdateTermination(ctx, id int64, TerminationUpdate) error`, `Termination(ctx, id int64) (Termination, error)`, `Terminations(ctx, TerminationFilter) (TerminationList, error)`, `deleteTermination(ctx, id) error`.
  - Routes: `GET/POST /hrm/terminations`, `GET/PUT /hrm/terminations/{id}`, `GET /hrm/terminations/actions` (returns `record.TypeActions` for create).
  - Record type: `Kind Document, NumberPrefix "NV"`, `Fields: [{Key:"reason", Kind: record.Choice, Label:"hrm.termination.reason", Options: s.reasonOptions}]`, `Perms{View: PermTerminationView, Edit: PermTerminationEdit, ViewFiles: PermTerminationView, Attach: PermTerminationEdit}`, `Subjects: s.terminationSubjects`, `OnTransition: s.terminationTransition` (stub returning nil in this task), `OnDelete: deleteTermination`.
  - Header: `Date = TerminationDate`, `OrgUnitID = employee's org unit`, `Subject = employee's user`, `Fields{"reason": code}`.

- [ ] **Step 1: Write the failing tests** in `termination_test.go` (package `hrm_test`, reuse `newLeaveFixture`; HR context `f.hr` needs role `hr`, which gains the new perms):
  - `TestTerminationRecordContract`: `recordtest.Run` like `TestOvertimeRecordContract`, `Table: "hrm.terminations"`, `Create` makes a termination of `f.emp` dated `date` with `reason: "mutual_agreement", InsuredMonths: 0`; `Effects` counts posted terminations via `Terminations(…Status:"posted")`; `BreakPosting` closes March (`f.closePeriod("2026-03-01","2026-03-31", r.ID)`).
  - `TestTermination`: number `NV-2026-00001`; header field `reason`; second create for the same employee while the first is draft → `termination_exists`; after deleting the draft a new one succeeds; date before hire → `termination_before_hire`; unknown reason → 422 from Huma enum or `invalid_termination_reason` from the service (service-level check needed because tests call the service).
- [ ] **Step 2: Run** `go test ./internal/modules/hrm -run TestTermination` — FAIL (undefined).
- [ ] **Step 3: Implement** migration, `queries/termination.sql` (`GetTermination`, `CreateTermination`, `UpdateTermination`, `DeleteTermination`, `OpenTerminationOf(employee_id, except_id)` returning the number of a non-cancelled termination, `TerminationListBase` with `-- data:base`), run `make gen`, then the service following `overtime.go` (`CreateOvertime`/`UpdateOvertime`/`Overtime` shape, `record.MayCreate` before `record.Create`, `RecordChanges` audit `hrm.termination_created/updated`). `termination_exists` check in create (and on update when nothing else changes it is unnecessary: the employee is fixed after create).
- [ ] **Step 4: Run** `go test ./internal/modules/hrm -run TestTermination` — PASS; also `go test ./internal/modules/hrm -run TestDeclare` (role/perm declaration tests) — PASS.
- [ ] **Step 5: Commit** `feat(hrm): termination document`

### Task 3: Posting and cancelling set the employee's termination date

**Files:**
- Modify: `internal/modules/hrm/termination.go` (`terminationTransition`), `queries/termination.sql` (`SetTerminationDate(id, date)`, `SetPreviousDate(id, date)`, `PostedTerminationOf(employee_id, except_id)`)
- Test: `internal/modules/hrm/termination_test.go`

**Interfaces:**
- Consumes: `payrollEffect`, `checkPayrollPeriods(ctx, legalEntity, from, nil)`, `store.LockEmployee`, `s.d.IAM.LegalEntityOf`, `s.recordChanges` (employee audit `hrm.employee_updated` with the `termination_date` change).
- Produces: posted termination ⇒ `hrm.employees.termination_date = doc date`, `hrm.terminations.previous_date = old value`; cancelled ⇒ restored.

- [ ] **Step 1: Write the failing tests**
  - `TestTerminationPost`: post → `Employee(f.emp).TerminationDate == "2026-04-15"` and an `hrm.employee_updated` audit row with the `termination_date` change. Two drafts cannot exist (Task 2), so the "another posted" guard is pinned by inserting a second `hrm.terminations` row through `f.system` for a cancelled-then-recreated case: cancel the first, create and post a second → passes; then re-posting is impossible (cancelled is final), so the guard's test is: two drafts racing — create A, delete-free insert of B via `f.system.Exec` bypassing the create check, post A, post B → `termination_exists`.
  - `TestTerminationCancel`: post, cancel → date back to nil. Post, close April with a payroll period row (`f.closePeriod("2026-04-01","2026-04-30", id)`), cancel → `payroll_period_closed`, date still `2026-04-15`.
  - `TestTerminationLateEntry`: close March, create termination dated `2026-03-31`, post → `payroll_period_closed` with `periods` holding `{2026-03-01, 2026-03-31}`; employee date unchanged (nil).
  - `TestTerminationLegacyDate`: employee created with `TerminationDate "2026-05-31"`; termination dated `2026-04-15` posted → `2026-04-15`; cancelled → `2026-05-31`.
  - `TestTerminationLegalEntityChanged`: create draft, move the employee to an org unit of another legal entity (fixture has `f.b` under another company? check `newFixture`; if not, create one with `f.core`), post → `termination_legal_entity_changed`.
- [ ] **Step 2: Run** `go test ./internal/modules/hrm -run 'TestTermination(Post|Cancel|LateEntry|LegacyDate|LegalEntity)'` — FAIL.
- [ ] **Step 3: Implement** `terminationTransition(ctx, d record.Doc, from record.Status) error`: `payrollEffect`; `checkPayrollPeriods(ctx, d.LegalEntityID, d.Date, nil)`; `LockEmployee`; on post: refuse if `PostedTerminationOf` finds another, if date < hire (`termination_before_hire`), if `LegalEntityOf(employee unit) != d.LegalEntityID`; store previous, set date; on cancel: set date to `previous_date`. Audit the employee change.
- [ ] **Step 4: Run** the Step 2 command and `-run TestTerminationRecordContract` — PASS.
- [ ] **Step 5: Commit** `feat(hrm): a posted termination sets the employee's termination date`

### Task 4: Termination date read-only on profile and import

**Files:**
- Modify: `internal/modules/hrm/employee.go` (`prepare`), `types.go` (`ErrTerminationViaDocument`, 422, code `termination_via_document`), `employee_io.go` only if the row error needs mapping, i18n `hrm.import.*`/error keys in `web/src/hrm/i18n/{vi,en}/main.json`
- Test: `internal/modules/hrm/employee_test.go` or the existing employee test file holding `UpdateEmployee` tests; `employee_io_test.go`

**Interfaces:**
- Produces: `prepare` returns `ErrTerminationViaDocument` when `old.ID != 0` and the new `TerminationDate` differs from `old.TerminationDate` (nil vs value counts as different). Create (`old.ID == 0`) unchanged.

- [ ] **Step 1: Write the failing tests**
  - `TestEmployeeTerminationDateReadOnly`: create with `TerminationDate "2026-05-31"` succeeds; update with same date succeeds; update with `"2026-06-30"` or nil → `termination_via_document`.
  - `TestEmployeeImportTerminationDate`: export, change the termination cell of an existing employee → import job fails with a row error `termination_via_document` on that row and writes nothing; same file with the cell emptied or unchanged → completes; a new row with a termination date → created.
- [ ] **Step 2: Run** `go test ./internal/modules/hrm -run 'TestEmployeeTerminationDate|TestEmployeeImportTerminationDate'` — FAIL.
- [ ] **Step 3: Implement** the check in `prepare` (a few lines next to `ErrTerminationBeforeHire`).
- [ ] **Step 4: Run** `go test ./internal/modules/hrm -run 'TestEmployee'` — PASS (existing employee tests still green; fix any that edited a termination date by moving that date to create).
- [ ] **Step 5: Commit** `feat(hrm): only a termination changes an employee's termination date`

### Task 5: Warnings and the allowance preview on the termination

**Files:**
- Modify: `termination.go`, `termination_types.go`, `queries/termination.sql`
- Test: `termination_test.go`

**Interfaces:**
- Consumes: Task 1 functions; `termsAt`, `openTerms`, `ContractTerms.insured()` from payroll files; `s.allowed(ctx, PermSalaryView, unit)`.
- Produces:
  - `func (s *Service) allowanceFor(ctx, employee int64, hire, termination string, b allowanceBasis) (allowanceKind, decimal.Decimal years, []int64 monthly, int64 average, error)` — reads posted contracts of the employee overlapping the 6 calendar months before the termination month (query `TerminationContracts(employee_id, from, to)`), audits each read `hrm.contract_terms_viewed`, takes `termsAt(month end).Terms.insured()` per month (months without a contract skipped), averages, returns. Task 6 reuses it.
  - `Termination` gains `Warnings []TerminationWarning{Code string; Params map[string]any}` (nullable false) and `Allowance TerminationAllowance{Kind string; MonthsWorked, MonthsCounted int; Years string; Average *int64; Amount *int64}` — `Average`/`Amount` only when the actor holds `hrm.salary.view` at the document's unit; otherwise contracts are not read at all.
  - Queries: `TerminationContractsAfter(employee_id, date)` (posted originals with end null or > date: number, end), `TerminationLeavesAfter`, `TerminationOvertimeAfter` (draft/pending/posted, end/date > date), balance from `hrm.leave_balances` for the termination year.

- [ ] **Step 1: Write the failing tests**
  - `TestTerminationWarnings`: employee with an indefinite posted contract, a pending leave 2026-04-20..21, a posted overtime 2026-04-22, balance 4 → termination 2026-04-15 detail has the four codes with params (contract number; leave number and dates; overtime number; `days: "4"`). A cancelled leave after the date is not listed.
  - `TestTerminationAllowancePreview`: contract 20_000_000 from 2025-01-01, addendum 24_000_000 from 2026-01-01; hire 2007-07-01; termination 2026-04-15 `mutual_agreement`, insured 207 → payroll-role actor sees `kind severance, months_worked 225, months_counted 18, years "1.5", average 22_000_000, amount 16_500_000`; HR without `hrm.salary.view` sees kind/years and nil amounts, and no `hrm.contract_terms_viewed` audit row is written for that read.
- [ ] **Step 2: Run** `go test ./internal/modules/hrm -run 'TestTerminationWarnings|TestTerminationAllowancePreview'` — FAIL.
- [ ] **Step 3: Implement** as specified; amount = `platform.Round(allowanceAmount(...))`.
- [ ] **Step 4: Run** — PASS.
- [ ] **Step 5: Commit** `feat(hrm): termination warnings and allowance preview`

### Task 6: The payroll pays the allowance

**Files:**
- Modify: `payroll_inputs.go` (basis + inputs), `payroll_calc.go` (amount), `payroll_types.go` (`PayrollAmounts.Severance int64 json:"severance" doc:"Severance or job-loss allowance on leaving"`), `queries/payroll.sql` (`PayrollSourceDocs` adds `hrm.termination` posted with date in the period; new `PayrollTerminations(ids, period_start, period_end)`), `payroll.go`/export code where amounts are listed (export column), `add()` in `payroll_calc.go`
- Test: `payroll_test.go` (new tests)

**Interfaces:**
- Consumes: `allowanceBasis`, `allowanceFor` (Task 5).
- Produces:
  - `basisEmployee.Allowance *allowanceBasis json:"allowance,omitempty"` — set for an employee with a posted termination dated in the period (so it is in `inputs_hash`; `omitempty` keeps the hash of every existing payroll unchanged).
  - `payInput.Severance *severance json:"severance,omitempty"` with `type severance struct{ Kind allowanceKind; Years decimal.Decimal; Monthly []int64; Average int64 }`.
  - Calc: `a.Severance = round(allowanceAmount(...))`; `Gross` and `Exempt` add it; insurance base untouched.

- [ ] **Step 1: Write the failing tests**
  - `TestPayrollSeverance`: payroll fixture, employee hire 2007-07-01, contract 20M from 2025-01-01 + addendum 24M from 2026-01-01, timesheet April, posted termination 2026-04-15 `mutual_agreement` insured 207 → line `Severance 16_500_000`, `Gross` and `Exempt` include it, `InsuranceBase` = 24_000_000, captured input `severance.monthly == [20M,20M,20M,24M,24M,24M]` (read via the payroll line's inputs as the payroll-role actor; use the same accessor `payroll_test.go` uses for inputs, or decode from the line detail endpoint).
  - `TestPayrollSeveranceStale`: compute the April payroll before posting the termination, then post it → payroll shows stale (`inputs changed`, same assertion existing stale tests use), submit refused; recalculate → `Severance` set.
  - `TestPayrollSeveranceNone`: `dismissal` → 0; `restructuring`, hire 2024-03-01, terminated 2026-04-30, insured 26, salary 15M → 30_000_000.
- [ ] **Step 2: Run** `go test ./internal/modules/hrm -run 'TestPayrollSeverance'` — FAIL.
- [ ] **Step 3: Implement**: in `readBasis` load `PayrollTerminations` and set `Allowance` (with `Worked` from `monthsWorked`); in `payInputs` call `allowanceFor` for those employees only; in `calculate` add the column. Payroll export: add a "Trợ cấp thôi việc" column at the end of the existing columns.
- [ ] **Step 4: Run** `go test ./internal/modules/hrm -run 'TestPayroll'` — PASS, including `TestPayrollSamples` unchanged.
- [ ] **Step 5: Commit** `feat(hrm): the termination-month payroll pays the severance allowance`

### Task 7: Payroll samples C18 and C19

**Files:**
- Modify: `docs/payroll-samples/build.mjs` (two rows, a column "Trợ cấp thôi việc/mất việc" appended after AF as AG, Giả định entries), `docs/payroll-samples/payroll-input-samples.xlsx` (rebuilt), `docs/payroll-samples/README.md` (30 cases, AG column), `internal/modules/hrm/payroll_samples_test.go`

**Interfaces:**
- Consumes: Task 6.
- Rows: C18 `mutual_agreement`, salary 24_000_000 in April (20_000_000 before 2026-01-01 via addendum), hire 2007-07-01, quit 2026-04-15, insured 207 → AG 16_500_000; C19 `restructuring`, salary 15_000_000, hire 2024-03-01, quit 2026-04-30, insured 26 → AG 30_000_000. Gross (Q), exempt (R), net (AA), cost (AF) include AG; tax unchanged by AG. Notes cite Art 46/47 BLLĐ 2019, Art 8 NĐ 145/2020, Art 2.2(b) TT 111/2013.

- [ ] **Step 1: Extend the test**: `sample` gains `hire`/`quit` (exist), `reason string`, `insured int`, `before int64` (salary before 2026-01-01); for such rows build the contract from 2025-01-01 with an addendum from 2026-01-01 and post a termination; compare `"AG": l.Severance`; for C18 also assert the six captured `monthly` figures.
- [ ] **Step 2: Run** `go test ./internal/modules/hrm -run TestPayrollSamples` — FAIL (no C18/C19 rows).
- [ ] **Step 3: Rebuild the workbook** with build.mjs as the README says (`ORACLE_ROOT=/tmp`). If cloning the oracles is impossible offline, stop and report rather than hand-edit the xlsx.
- [ ] **Step 4: Run** `go test ./internal/modules/hrm -run TestPayrollSamples` — PASS; `git diff --stat docs/payroll-samples` shows the 28 old rows' values unchanged (check with a quick excelize dump if needed).
- [ ] **Step 5: Commit** `test(hrm): payroll samples for severance and job-loss allowances`

### Task 8: Screens

**Files:**
- Create: `web/src/hrm/pages/TerminationsPage.tsx`, `NewTerminationPage.tsx`, `TerminationPage.tsx`, `web/src/hrm/components/TerminationDetails.tsx` (fields panel, warnings block, allowance block), `web/src/hrm/hooks/useTermination.ts`, `web/e2e/termination.spec.ts`
- Modify: `web/src/hrm/index.ts` (nav item under group Nhân sự `hrm.nav.terminations`, `recordTypes[terminationDocType]` with `path: (id) => \`terminations/${id}\``, `preview`, `invalidate` termination detail/list and employee), `web/src/hrm/keys.ts`, `web/src/hrm/routes.tsx`, `web/src/hrm/pages/EmployeePage.tsx` / `components/EmployeeForm.tsx` (termination date read-only on edit with a link; action "Cho nghỉ việc" → `terminations/new?employee=<id>`), `web/src/hrm/pages/PayrollPage.tsx` (severance column/line when non-zero), i18n `web/src/hrm/i18n/{vi,en}/{main,meta}.json`

**Interfaces:**
- Consumes: routes from Task 2, `warnings`/`allowance` from Task 5, `severance` from Task 6 (after `make gen`).

- [ ] **Step 1:** Invoke the `mmerp-ui` and `new-screen` skills; follow `ContractsPage`/`NewContractPage`/`ContractPage` (HR documents, not self-service).
- [ ] **Step 2: Write the e2e spec** `web/e2e/termination.spec.ts`: HR creates a termination from an employee page, sees the `leave_balance_to_pay` warning, submits/posts, the employee profile shows the date read-only with the link; the payroll role sees the allowance amount. (Lead runs it.)
- [ ] **Step 3: Implement** the pages; reason select shows label + article; pension checkbox hint "Nghỉ hưu: chọn lý do hết hạn hợp đồng hoặc thỏa thuận và đánh dấu ô này".
- [ ] **Step 4: Run** `cd web && pnpm tsc --noEmit && pnpm vitest run` and `make lint` — PASS.
- [ ] **Step 5: Commit** `feat(web): termination screens`

### Task 9: Docs

**Files:**
- Modify: `products/hrm.md` (Entities row `hrm.terminations`; Record types row `hrm.termination`; Approval fields `reason`; new "Termination" section: reason codes table for reports, allowance rules, late-entry way out; replace "Severance allowance … not in the first release"; Still open items from the spec), `CONTEXT.md` (Termination, Severance allowance, Job-loss allowance — appended to the HRM section), `docs/adr/0046-termination-document.md` (already written; commit if not yet)

- [ ] **Step 1:** Edit; append only, keep existing wording.
- [ ] **Step 2: Run** `make lint` — PASS.
- [ ] **Step 3: Commit** `docs(hrm): termination and severance allowance`

### Task 10: Print the termination decision (last)

**Files:**
- Modify: `internal/modules/hrm/print.go` (register `printing.Template{Code: terminationType, Name: "hrm.print.termination.name", DocType: terminationType, Blocks: grounds, articles, footer with placeholders employee_name, employer_name, termination_date, reason}`, data func `terminationPrint`, one layout like `contractLayout1`), i18n defaults (vi/en)
- Test: `termination_test.go` `TestTerminationPrint` (render data holds number, employee, date, reason label with its article, no amount)

- [ ] **Step 1: Write the failing test** following the contract print test.
- [ ] **Step 2: Run** `go test ./internal/modules/hrm -run TestTerminationPrint` — FAIL.
- [ ] **Step 3: Implement**; the detail page shows the print action from `allowed_actions`.
- [ ] **Step 4: Run** `go test ./internal/modules/hrm` and `make lint` — PASS.
- [ ] **Step 5: Commit** `feat(hrm): print the termination decision`

### Finish

- [ ] `make gen` (no diff afterwards), `make lint`, `make test` — all PASS.
- [ ] Push `feat/termination` to `github`; open the PR into `main` per house style (plain body, per task with commits, behaviour changes, migration, shared files touched: employee.go, employee_io.go, payroll files, products/hrm.md, CONTEXT.md, i18n, web/src/hrm/index.ts; e2e spec added for the lead to run). Do not merge.
