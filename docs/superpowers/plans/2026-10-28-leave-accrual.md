# Leave Accrual and Computed Leave Days Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Annual leave builds up monthly by a catch-up system job (seniority, pro rata, carry-over with expiry), a leave request computes its days from the work calendar with half-day flags, and the termination payout counts days earned but not yet credited.

**Architecture:** A new `accrual.go` area in `internal/modules/hrm` holds the pure entitlement arithmetic, the `hrm.leave_accrual` system job (River periodic, hourly, run on start) and carry-over/expiry. A ledger table `hrm.leave_accruals` makes each month's credit idempotent; every balance change goes through the existing `AddLeaveBalance` row lock. Computed days extend `leaveAllocation` with half flags and replace the typed count in `leaveHeader`/`checkLeaveDates`.

**Tech Stack:** Go, sqlc, Postgres 18, River, shopspring/decimal, React + react-hook-form, Playwright.

**Spec:** `docs/superpowers/specs/2026-10-28-leave-accrual-design.md`; ADR `docs/adr/0047-leave-accrual-and-computed-days.md`.

## Global Constraints

- Worktree `/home/jarviisha/development/mmerp-wt/leave-accrual`, branch `feat/leave-accrual`. Never run `make e2e`, `make dev`, `make reset-db`, `make seed`. `make lint` and e2e only when the user or lead asks.
- One migration only, created with `make migration name=hrm_leave_accrual`; it touches `hrm.leave_balances` and `hrm.leave_requests` (say so in the PR body).
- Day figures are `numeric` / `decimal.Decimal` in half-day steps; never float.
- Services query only through `store.<Query>.Run(ctx, …)`, write inside `platform.InTx`; one employee per transaction in the job.
- Settings: `hrm.leave_accrual` (`monthly` default, `manual`); `hrm.leave_carry_over_months` (`0`, `3` default, `6`, `12`).
- Error codes: `leave_days_changed`, `leave_half_days`; audit actions `hrm.leave_accrued`, `hrm.leave_carried_over`, `hrm.leave_carry_expired` (restricted in history like `hrm.leave_balance_adjusted`).
- `TestPayrollSamples` and every existing test in `internal/modules/hrm/leave*_test.go`, `payroll*_test.go`, `internal/app/leave_test.go` pass **unchanged** after every task that touches leave or payroll code: run `go test ./internal/modules/hrm/ ./internal/app/` at the end of Tasks 2–6.
- Code, identifiers and docs in English; UI text through `vi` and `en` i18n files. Commits: `feat(hrm): …`, ending with the `Co-Authored-By` line.

## Review Focus

- An employee whose `hire_date` is after the month being decided, or in the future: no ledger row for months before hire, no credit (test in Task 1/2).
- A month the work calendar gives no working days for an employee's span (hired on the last day, a holiday month end): uncounted, 0 days, no division by zero (test in Task 1).
- A request whose start or end date falls on a weekend with a half flag set: the half lands on the first/last *working* date, not the weekend (test in Task 5).
- Approving a request whose stored count was computed before a holiday was added: refused `leave_days_changed`, nothing deducted (test in Task 5).
- A hand adjustment that takes the balance below its carried part: `carried` capped, no check violation (test in Task 3).

---

### Task 1: Schema and entitlement arithmetic

**Files:**
- Create: `migrations/<timestamp>_hrm_leave_accrual.sql` (via `make migration name=hrm_leave_accrual`)
- Create: `internal/modules/hrm/accrual.go`
- Create: `internal/modules/hrm/accrual_calc_test.go` (package `hrm`, like `leave_allocation_test.go`)
- Modify: `internal/modules/hrm/queries/leave.sql`

**Interfaces:**
- Produces: `func entitlementThrough(cal calendar, hire string, termination *string, year, month int) decimal.Decimal` — cumulative entitlement of `year` through `month` (0 → 0), rounded down to a half day; `func monthCredit(cal calendar, hire string, termination *string, year, month int) decimal.Decimal` = `entitlementThrough(m) − entitlementThrough(m−1)`; `func monthCounted(cal calendar, hire string, termination *string, year, month int) bool`.
- Produces (sqlc): `InsertLeaveAccrual(employee_id, year, month, days)` `ON CONFLICT DO NOTHING` returning rows affected (`:execrows`); `LeaveAccrualMonths(employee_id, year) → []int32`; `LeaveAccruedDays(employee_id, year) → text` (sum, `'0'` when none).

- [ ] **Step 1: Write the migration**

```sql
CREATE TABLE hrm.leave_accruals (
    employee_id bigint NOT NULL REFERENCES hrm.employees,
    year        int NOT NULL,
    month       int NOT NULL CHECK (month BETWEEN 1 AND 12),
    days        numeric(4, 1) NOT NULL CHECK (days >= 0),
    PRIMARY KEY (employee_id, year, month)
);
ALTER TABLE hrm.leave_balances
    ADD COLUMN carried numeric(5, 1) NOT NULL DEFAULT 0,
    ADD COLUMN carried_until date,
    ADD CONSTRAINT leave_balances_carried_check CHECK (carried >= 0 AND carried <= days);
ALTER TABLE hrm.leave_requests
    ADD COLUMN start_half boolean NOT NULL DEFAULT false,
    ADD COLUMN end_half boolean NOT NULL DEFAULT false,
    ADD COLUMN computed boolean NOT NULL DEFAULT false,
    ADD COLUMN deducted_carried numeric(4, 1) NOT NULL DEFAULT 0;
```
With a short comment per table/column, as in the baseline.

- [ ] **Step 2: Write the failing tests** `TestEntitlement` (table-driven, calendar with Sat/Sun off, no holidays unless stated):
  - hired 2020-01-01, 2026 full year: through month 12 = `12`; month 1 = `1`; `monthCredit` each month = `1`.
  - hired 2021-07-01 (5 years completed end of June 2026): through 12 = `12.5`; through 6 = `6`.
  - hired 2010-01-01 (seniority 3): through 12 = `15`; through 1 = `1` (1.25 rounded down), through 2 = `2.5`.
  - hired 2026-03-10 (≥ half of March's working days): March counted; hired 2026-03-20: March not counted; through 12 = `10` and `9`.
  - terminated 2026-04-05: April not counted, through 12 = `3` (hire 2020); terminated 2026-04-20: `4`.
  - hired 2026-11-30 (one working day of November): November uncounted, no panic.
  - hire after the year: `0`.
- [ ] **Step 3: Run** `go test ./internal/modules/hrm -run TestEntitlement` → FAIL (undefined).
- [ ] **Step 4: Implement** in `accrual.go`. A month counts when the working days of `[max(hire, month start), min(termination, month end)]` are ≥ half of the month's working days (`cal.workingDays`); a month with no working days at all is uncounted. Seniority for month m = whole years from `hire` to the month's last day, `/ 5`. Sum `(12 + seniority)/12` over counted months 1..m, then `floor(x*2)/2`.
- [ ] **Step 5: Add the three queries** to `queries/leave.sql`, run `make gen`.
- [ ] **Step 6: Run** `go test ./internal/modules/hrm -run TestEntitlement` → PASS.
- [ ] **Step 7: Commit** `feat(hrm): leave entitlement arithmetic and accrual ledger`.

### Task 2: The accrual job

**Files:**
- Modify: `internal/modules/hrm/accrual.go`, `module.go` (worker + `Periodic`), `service.go` (register settings, `RestrictHistory` for the three audit actions), `types.go` (setting constants), `queries/leave.sql`, `i18n` server files if setting labels live there
- Create: `internal/modules/hrm/accrual_test.go` (package `hrm_test`, `newFixture`)

**Interfaces:**
- Consumes: Task 1 functions and queries; `workCalendar`, `s.d.IAM.LegalEntityOf`, `s.d.Setting.GetFor`, `store.EnsureLeaveBalance`, `store.AddLeaveBalance`.
- Produces: `const SettingLeaveAccrual = "hrm.leave_accrual"`, `SettingCarryOverMonths = "hrm.leave_carry_over_months"`; `func (s *Service) AccrueLeave(ctx context.Context, today string) error` (exported for tests; processes every employee, each in its own `platform.InTx`, continues past a failing employee and returns the joined error); `leaveAccrualArgs` kind `hrm.leave_accrual`, `JobSpec{System: true}`, worker calls `AccrueLeave` with `setting.Today`; periodic `time.Hour`, `RunOnStart: true`. Sqlc `AccrualEmployees(year_start date) → id, org_unit_id, hire_date, termination_date` (not terminated before last year's 1 January).

- [ ] **Step 1: Write the failing tests** in `accrual_test.go`:
  - `TestAccrualCatchesUp`: employee hired 2020-01-01 in C; `AccrueLeave(ctx, "2026-04-01")` → 2026 balance `3`, ledger months 1–3; run again → still `3`; `AccrueLeave(ctx, "2026-07-02")` → `6`.
  - `TestAccrualManualUntouched`: C set to `manual` via `f.core.Setting.SetFor` → balance row absent / `0`, no ledger rows.
  - `TestAccrualAudited`: employee history holds `hrm.leave_accrued` with `{"year":2026,"month":3,"days":"1","balance":"3"}`.
  - `TestAccrualNoBackfillAfterJanuary`: `AccrueLeave(ctx, "2026-10-01")` credits 2026 Jan–Sep only, no 2025 ledger rows; `AccrueLeave(ctx, "2026-01-02")` on a fresh fixture credits all 2025 months.
  - `TestAccrualLeaverStops`: terminated 2026-03-20 → after `AccrueLeave(ctx, "2026-06-01")` ledger months 1–3 only.
- [ ] **Step 2: Run** `go test ./internal/modules/hrm -run TestAccrual` → FAIL.
- [ ] **Step 3: Implement** `AccrueLeave`: for each employee, legal entity from `LegalEntityOf(org_unit_id)`; skip if setting is `manual`. Last year (in January, or when `LeaveAccrualMonths` for last year is non-empty): months 1–12 not in the ledger. This year: months before `today`'s month not in the ledger, and never after the termination month. Per month: `InsertLeaveAccrual`; only if a row was inserted and the credit > 0, `EnsureLeaveBalance` + `AddLeaveBalance`, then `Audit.RecordFor(ctx, "hrm.leave_accrued", employee ref, …)`. One calendar per legal entity and year per run.
- [ ] **Step 4: Run** `go test ./internal/modules/hrm -run TestAccrual` → PASS; `go test ./internal/modules/hrm/ ./internal/app/` → PASS unchanged.
- [ ] **Step 5: Commit** `feat(hrm): monthly leave accrual as a catch-up system job`.

### Task 3: Carry-over, expiry and spending carried days

**Files:**
- Modify: `internal/modules/hrm/accrual.go`, `leave.go` (`leaveTransition`), `queries/leave.sql` (`AddLeaveBalance` caps carried), `leave_types.go` (`LeaveBalance`)
- Test: `internal/modules/hrm/accrual_test.go`

**Interfaces:**
- Produces (sqlc): `CarryLeaveBalance(employee_id, from_year, to_year, until date) → moved text` (moves `days` of `from_year` into `to_year`, adds them to `carried`, sets `carried_until`, row-locked, `0` when nothing); `ExpireCarried(employee_id, year, today) → expired text`; `SpendCarried(employee_id, year, delta, on date) → spent text`; `AddLeaveBalance` now also sets `carried = LEAST(carried, days + delta)`.
- Produces: `LeaveBalance{Year, Days, Carried string, CarriedUntil *string}` (json `carried`, `carried_until`).

- [ ] **Step 1: Write the failing tests**:
  - `TestCarryOver`: 2025 balance 5 (hand grant), C on `manual`, carry 3 → `AccrueLeave(ctx, "2026-01-02")`: 2025 `0`, 2026 `days 5, carried 5, carried_until 2026-03-31`, audit `hrm.leave_carried_over`; rerun unchanged.
  - `TestCarryExpires`: then 2 days of annual leave on 2026-03-02..03 approved → carried `3`, `deducted_carried 2`; `AccrueLeave(ctx, "2026-04-01")` → days `0`, carried `0`, audit `hrm.leave_carry_expired`.
  - `TestCarryOverZeroLapses`: setting `0` → on 2026-01-02 the moved days lapse at once (2026 `0`, expiry audited).
  - `TestCarriedRefund`: cancel the approved request before expiry → carried restored; after expiry → only ordinary days restored (carried part dropped).
  - `TestAdjustBelowCarried`: carried 5, adjust `-3` → days `2`, carried `2`, no error.
  - `TestLeaverNotCarriedNorExpired`: terminated 2026-03-15 with 4 carried days: `AccrueLeave(ctx, "2026-04-01")` leaves days and carried unchanged; `AccrueLeave(ctx, "2027-01-02")` carries nothing into 2027.
- [ ] **Step 2: Run** `go test ./internal/modules/hrm -run 'Carr|AdjustBelow|LeaverNot'` → FAIL.
- [ ] **Step 3: Implement** in the job, after last year's accrual and before this year's: carry (skipped for anyone whose `termination_date` is before this 1 January), with `carried_until` = last day of month N of this year (N = 0 → 31 December last year); after this year's accrual: expire when `today > carried_until` (skipped once `termination_date < today`). In `leaveTransition` on approval of a deducting type with `start_date <= carried_until`: `SpendCarried` then record `deducted_carried`; on cancel give `deducted_carried` back to `carried` only while `today <= carried_until`. Lock order unchanged: everything goes through the one balance row.
- [ ] **Step 4: Run** the new tests → PASS; `go test ./internal/modules/hrm/ ./internal/app/` → PASS unchanged.
- [ ] **Step 5: Commit** `feat(hrm): carry unused leave over with an expiry`.

### Task 4: Termination payout counts earned-not-credited days

**Files:**
- Modify: `internal/modules/hrm/accrual.go`, `payroll_inputs.go:131-143`
- Test: `internal/modules/hrm/accrual_test.go` (or `payroll_test.go` if its helpers fit better)

**Interfaces:**
- Produces: `func (s *Service) earnedNotCredited(ctx context.Context, legalEntity, employee int64, hire string, termination *string, through string) (decimal.Decimal, error)` — `entitlementThrough(year, month of through) − LeaveAccruedDays(year)`, floored at 0; `0` on a `manual` legal entity. `AccrueLeave` uses the same arithmetic, so the two agree.

- [ ] **Step 1: Write the failing test** `TestPayrollPaysTerminationMonthAccrual`: monthly entity, employee hired 2020, terminated 2026-04-20, job run on `2026-04-01` (balance 3); April payroll's `unused_leave` line pays `4` days; then `AccrueLeave(ctx, "2026-05-01")` → balance `4` and the payroll's inputs hash is unchanged (no "source changed").
- [ ] **Step 2: Run** → FAIL (pays 3).
- [ ] **Step 3: Implement**: in `readBasis`, for a leaver in the period, `e.LeaveBalance = balance + earnedNotCredited(…, through = termination date)`.
- [ ] **Step 4: Run** the test → PASS; `go test ./internal/modules/hrm -run TestPayrollSamples` unchanged → PASS; full `go test ./internal/modules/hrm/ ./internal/app/` → PASS.
- [ ] **Step 5: Commit** `feat(hrm): termination payout counts leave earned but not yet credited`.

### Task 5: Computed leave days with half flags

**Files:**
- Modify: `internal/modules/hrm/leave.go` (`leaveHeader`, `checkLeaveDates`, `Leave`), `leave_types.go` (`LeaveFields`), `payroll_inputs.go` (`leaveAllocation`, `basisLeave`), `queries/leave.sql`, `queries/payroll.sql` (`PayrollLeaves` selects the flags), `errors` file holding `ErrLeave*`
- Test: `internal/modules/hrm/leave_computed_test.go`, `leave_allocation_test.go`

**Interfaces:**
- Produces: `LeaveFields` gains `StartHalf bool json:"start_half"`, `EndHalf bool json:"end_half"`; `Days` becomes optional on input (`json:"days,omitempty"`, `required:"false"`), always returned on output. `func leaveAllocation(c calendar, start, end string, days decimal.Decimal, startHalf bool) map[string]decimal.Decimal` (the first working date takes at most 0.5 when `startHalf`; otherwise unchanged greedy). `func computedDays(c calendar, start, end string, startHalf, endHalf bool) (decimal.Decimal, error)` returning `ErrLeaveNoWorkingDays` or `ErrLeaveHalfDays` (`leave_half_days`, 422: both flags on a single working date). `basisLeave` gains `StartHalf bool json:"start_half,omitempty"` so old payroll hashes do not move.

- [ ] **Step 1: Write the failing tests**:
  - `TestComputedDays` (package `hrm`): Mon–Fri `5`; with `startHalf` `4.5`, both `4`; range Sat–Tue with `startHalf` → `1.5` and allocation `Mon=0.5 Tue=1`; single day with both → `ErrLeaveHalfDays`; Sat–Sun → `ErrLeaveNoWorkingDays`.
  - `TestLeaveDaysComputedOnSave`: create without `Days`, Mon–Fri, `EndHalf` → stored `4.5`; create with `Days: "3"` for Mon–Fri → `leave_days_changed`.
  - `TestApprovalRefusesChangedCount`: submit Mon–Fri (5), add a holiday on Wednesday, approve → `leave_days_changed`, balance untouched.
  - `TestHalvesOverlap`: approved Monday `EndHalf` single day (0.5) + Monday `StartHalf` single day → approved; a full Monday afterwards → `leave_overlaps`.
  - `TestLegacyRequestApprovedUnderOldRules`: a row written with `computed = false` and typed `1` over a three-day range (seed via `f.system` update) approves and deducts `1`.
- [ ] **Step 2: Run** `go test ./internal/modules/hrm -run 'Computed|ChangedCount|HalvesOverlap|Legacy'` → FAIL.
- [ ] **Step 3: Implement**: `leaveHeader` computes days with `computedDays`, refuses a given `Days` that differs (`leave_days_changed`, 409, param `days` = computed), stores flags and `computed = true`; `checkLeaveDates` on approval recomputes for `computed` rows (refuse on difference) and keeps the ADR-0030 cap for `computed = false`; every `leaveAllocation` caller passes the row's `start_half`. Drop `leave_days_exceed_working_days` only from the computed path.
- [ ] **Step 4: Run** new tests → PASS; `go test ./internal/modules/hrm/ ./internal/app/` → PASS unchanged (existing tests send full-range counts, which match).
- [ ] **Step 5: Commit** `feat(hrm): a leave request computes its days from the calendar`.

### Task 6: Screens, generated client and docs

**Files:**
- Modify: `web/src/hrm/components/LeaveForm.tsx`, `LeaveDetails.tsx`/`LeavePreview.tsx` (show halves), `BalancesTab.tsx` (columns `carried`, `carried_until`), `web/src/core/pages/SettingsPage.tsx` (description from `t(\`${setting.key}.hint\`, { defaultValue: '' })`), `web/src/hrm/i18n/{vi,en}/*.json`, `web/src/shared/i18n/locales/{vi,en}.json` (error codes), `api/openapi.json`, `web/src/shared/api/schema.gen.ts` (`make gen`), `products/hrm.md` (Leave section), `CONTEXT.md` (Leave accrual, Carried days, Leave day allocation updated)
- Test: Vitest beside any extracted pure helper (e.g. `web/src/hrm/leaveDays.test.ts` for `leaveDays(working, startHalf, endHalf)`)

**Interfaces:**
- Consumes: API fields `start_half`, `end_half`, `days` (output), `LeaveBalance.carried`, `carried_until`; settings `hrm.leave_accrual`, `hrm.leave_carry_over_months` with values and `.hint` keys.

- [ ] **Step 1: Write the failing Vitest** `leaveDays(5, true, false) === '4.5'`, `leaveDays(1, true, false) === '0.5'`, `leaveDays(0, false, false) === null`.
- [ ] **Step 2: Run** `pnpm -C web vitest run src/hrm/leaveDays.test.ts` → FAIL.
- [ ] **Step 3: Implement**, following the `mmerp-ui` skill and `docs/ui.md` checklist: the form drops the typed `days` field, shows the computed count read-only from `useLeaveWorkingDays` and the flags, with checkboxes "Ngày đầu: chỉ nghỉ buổi chiều" / "First day: afternoon only", "Ngày cuối: chỉ nghỉ buổi sáng" / "Last day: morning only", and one "Nghỉ nửa ngày" / "Half day" when start = end (maps to `start_half`); error fields `leave_days_changed` → `start_date`, `leave_half_days` → `end_date`. Setting hint (vi/en) for `hrm.leave_accrual.hint`: "Bật hằng tháng giữa năm sẽ cộng phép của các tháng đã qua, cộng dồn lên số đã cấp tay; chuyển về thủ công không cộng thêm và không trừ đi gì." / "Turning monthly on mid-year credits the months so far on top of any hand grant; switching back to manual credits nothing more and removes nothing." Update `products/hrm.md` Leave bullets (accrual, carry-over, computed days, the "no automatic accrual" sentence removed) and `CONTEXT.md`.
- [ ] **Step 4: Run** `make gen` (no unexpected diff), `pnpm -C web vitest run`, `pnpm -C web tsc --noEmit` → PASS; `go test ./internal/modules/hrm/ ./internal/app/` → PASS.
- [ ] **Step 5: Commit** `feat(web): computed leave days, half days and carried balance`.

### Task 7: E2E spec, existing specs on manual, demo seed

**Files:**
- Create: `web/e2e/leave-accrual.spec.ts`
- Modify: `web/e2e/m3.spec.ts` (its legal entity set to `manual` in `beforeAll`, right after it is created), `web/e2e/demo.seed.ts`

Why only m3: there is no shared e2e setup that creates legal entities; every spec creates its own company. Of the specs that file leave, only m3 uses a balance-deducting type and asserts balance figures; all the others (m9, m11, m12, approval-*, keyboard-inputs, notification-links, personal-space) create `deducts_balance: false` types, so a credit the hourly job makes during the run cannot change what they assert. Accrual itself is covered by the Go tests, because e2e cannot wait for the job. The new spec uses its own company set to `manual`. (drop the 2026 opening grants; keep the 2025 carried days as a 2025 grant so the job carries them; leave requests send no `days`)

- [ ] **Step 1: Write** `leave-accrual.spec.ts`: the leave form shows the computed count for a Mon–Fri range, ticking "Last day: morning only" shows 4.5, the saved request shows 4.5; the settings dialog for `hrm.leave_accrual` shows the hint. Through the API only for setup.
- [ ] **Step 2: Typecheck** `pnpm -C web tsc --noEmit -p e2e` (or the repo's e2e tsconfig) → PASS. Do not run Playwright; the lead runs `make e2e` and the seed.
- [ ] **Step 3: Commit** `test(web): e2e for computed leave days; specs granting by hand run on manual`.

### Finish

- [ ] Run `go test ./...` and `pnpm -C web vitest run` once more; push `feat/leave-accrual` to `github`; open the PR into `main` (house style: plain-text body, per-task commits, behaviour changes, the migration and that it touches `hrm.leave_balances` / `hrm.leave_requests`, shared files touched, e2e specs added and the seed change for the lead to run, `Closes` none unless an issue exists). Stop there.
