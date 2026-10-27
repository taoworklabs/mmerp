# Audit cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the drifts the 2026-10-16 audit found that are still open on `main`: the server decides leave status filters, the current contract and quotation expiry; jobs are system or user by declaration; two backend rule fixes.

**Architecture:** One branch `fix/audit-cleanup`, one commit per ticket, one PR closing #55 and #56–#63. Backend changes are test-first against real Postgres; frontend changes drop client-side rules in favour of the new server fields.

**Tech Stack:** Go (Huma, sqlc, data engine, River), React + TanStack Query, generated OpenAPI types.

**Spec:** GitHub issue #55 and tickets #56–#63 (`gh issue view <n>`). Written 2026-10-16; checked against `main` on 2026-10-27 (see below).

## Already done on main (close with the PR, no task)

- **#56** `hrm.employee` is in `web/src/hrm/index.ts` `recordTypes` with `path` and `invalidate`; no hand-written history invalidation remains.
- **#57** `web/src/sales/routes.tsx:35` guards `items` by `canItems`; `ItemsPage.tsx` only reads `item.manage` to decide writability (items moved to `shared/item`, ADR-0039).
- **#58** `PayrollsPage` uses `useListView`; `AppLayout.tsx` and `NotificationsPage.tsx` use `shared/ui` sizes only (`ThemeIcon size` is the kit's own pixel prop).
- **#63, first bullet:** `sales/service_item.go`/`writeItem` no longer exists; `shared/item` passes no store.

## Global Constraints

- All code in English; user-facing text only through i18n in `vi` and `en`.
- Services query only through `store.<Query>.Run(ctx, …)` / `data.List`; transactions via `platform.InTx`.
- Frontend uses only generated OpenAPI types; run `make gen` after any API change and commit `api/openapi.json` and `web/src/shared/api/schema.gen.ts`.
- Comments short, explain why, never name milestones, ADRs or review/ticket ids.
- Commits: Conventional Commits with scope (`hrm`, `sales`, `platform`, `web`), ending with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`. Stage explicit paths only.
- Run one Go test with `go test ./internal/<path> -run <Name>` after `make db`.

## Review Focus

- A leave list request with no `status` at all must still return every status (overlap panel aside, the leave list page sends none). Test in Task 1.
- A system job enqueued before this change (metadata `{}`) must still run after deploy. Task 4's design keys on the job kind, not on metadata; test in Task 4.
- An employee with only an appendix in force (original covers the date) must show the original contract, not "no contract". Test in Task 2.
- Creating an order from an expired quotation must still be refused (`expired()` at `sales/doc.go:417` keeps its meaning). Test in Task 3.
- A user who may view an employee but not contracts must not get `current_contract`. Test in Task 2.

---

### Task 1: Leave list accepts several statuses (#59)

**Files:**
- Modify: `internal/modules/hrm/leave_types.go:77`, `internal/modules/hrm/leave_list.go:45`
- Modify: `web/src/hrm/hooks/useLeave.ts:39-46`
- Test: `internal/modules/hrm/leave_list_test.go`

**Interfaces:**
- Produces: `LeaveFilter.Status []string` with tag `query:"status,explode" enum:"draft,pending_approval,posted,cancelled"`; repeated `?status=a&status=b`.

- [ ] **Step 1: Failing test** `TestLeaveListSeveralStatuses`: create one leave in each of `draft`, `pending_approval`, `posted`; list with `Status: []string{"pending_approval","posted"}` returns exactly those two; list with `Status: nil` returns all three.
- [ ] **Step 2: Run** `go test ./internal/modules/hrm -run TestLeaveListSeveralStatuses` → FAIL (compile: Status is string).
- [ ] **Step 3: Implement.** Change the field type; in `leave_list.go` use `data.When(len(f.Status) > 0, data.In("status", f.Status))`. Check Huma renders the enum on array items (`make gen`, inspect `api/openapi.json` for `/hrm/leaves` `status`); if Huma does not apply `enum` to items, validate in the service and return the existing invalid-parameter error.
- [ ] **Step 4: Run** the test and `go test ./internal/modules/hrm -run 'Leave'` → PASS. Fix any caller that set `Status: "…"` to `[]string{…}`.
- [ ] **Step 5: Frontend.** `make gen`; in `useOverlappingLeaves` pass `status: ['pending_approval', 'posted']` and filter only `x.id !== leave!.id`. Any other caller of `/hrm/leaves` that passed a single status now passes a one-element array (`pnpm -C web tsc --noEmit` finds them).
- [ ] **Step 6: Commit** `feat(hrm): leave list filters several statuses on the server`.

### Task 2: Current contract on the employee (#60)

**Files:**
- Modify: `internal/modules/hrm/queries/contract.sql` (new query), `internal/modules/hrm/employee_types.go:57`, `internal/modules/hrm/employee.go:95`
- Modify: `web/src/hrm/components/EmployeeDigest.tsx`, `web/src/hrm/hooks/useEmployeeRecords.ts:24-26` (delete `currentContract` if unused)
- Test: `internal/modules/hrm/contract_test.go`

**Interfaces:**
- Produces: `Employee.CurrentContract *CurrentContract` (`json:"current_contract"`, nullable) with `CurrentContract{ID int64 "id"; Number string "number"; ContractType string "contract_type"; StartDate string "start_date"; EndDate *string "end_date"}`. `ContractType` is the contract type's display name, as `ContractListItem.contract_type_name` shows today.
- New sqlc query `CurrentContract :one`: the posted original contract (`parent_id IS NULL`) of `@employee_id` covering `@at` — the same `WHERE` as `ContractTermsAt` — joined to its number and contract type name. `pgx.ErrNoRows` → `nil`.

- [ ] **Step 1: Failing tests** in `contract_test.go`:
  - `TestEmployeeCurrentContract`: no contract → `CurrentContract == nil`; posted contract covering today → its id, number, type name, dates; a posted appendix in force → still the original's id; an expired contract (end_date before today) → nil; a draft → nil.
  - `TestEmployeeCurrentContractNeedsContractView`: a user with `hrm.employee.view` but not `hrm.contract.view` gets `nil`.
- [ ] **Step 2: Run** `go test ./internal/modules/hrm -run TestEmployeeCurrentContract` → FAIL (field missing).
- [ ] **Step 3: Implement.** Add the query, `make gen`; in `Service.Employee`, only when `actions` includes `view_contracts` (the existing gate the screen uses), run it with "today" from `s.d.Setting.Today(ctx)`. No amounts, no audit.
- [ ] **Step 4: Run** both tests → PASS; `go test ./internal/modules/hrm` → PASS.
- [ ] **Step 5: Frontend.** `EmployeeDigest` reads `e.current_contract`; it no longer calls `useEmployeeContracts`/`currentContract` for this panel (keep the hook if other panels use it). The status badge goes: the field is a posted contract by definition, so show number and "since" line only. `pnpm -C web tsc --noEmit` and `pnpm -C web lint` pass.
- [ ] **Step 6: Commit** `feat(hrm): employee detail carries its current contract`.

### Task 3: Quotation expiry decided by the server (#61)

**Files:**
- Modify: `internal/modules/sales/doc.go:76`, `:136`
- Modify: `web/src/sales/components/DocProgress.tsx:5-9`
- Test: `internal/modules/sales/service_test.go`

**Interfaces:**
- `expired(validUntil, today)` keeps its meaning (date only) for the order-creation check at `doc.go:417`.
- Produces: the API `expired` field = `status == posted && !ordered && expired(valid_until, today)`, on both list rows and detail. Put that rule in one helper `shownExpired(status string, ordered bool, validUntil, today pgtype.Date) bool` used at both sites.

- [ ] **Step 1: Failing test** `TestQuoteExpiredOnlyWhenPostedAndNotOrdered`: with `valid_until` before today, a draft quote → `Expired false`; a posted quote → `true`; a posted quote with a live order → `false`. Check both `Quote(id)` and `Quotes(filter)`. Keep (or add) the assertion that creating an order from the expired posted quote is still refused with the existing error code.
- [ ] **Step 2: Run** `go test ./internal/modules/sales -run TestQuoteExpired` → FAIL on the draft case.
- [ ] **Step 3: Implement** `shownExpired` and use it at lines 76 and 136 (at 136 compute after `out.Ordered` is known).
- [ ] **Step 4: Run** → PASS; `go test ./internal/modules/sales` → PASS.
- [ ] **Step 5: Frontend.** `DocProgress` shows expired when `doc.expired`, no `status` check; drop `'status'` from its `Pick` if nothing else uses it. tsc passes.
- [ ] **Step 6: Commit** `fix(sales): the server decides when a quotation shows as expired`.

### Task 4: Jobs declare whether they are system jobs (#62)

**Files:**
- Modify: `internal/platform/jobs.go` (registry, `AddWorker`, `JobMiddleware`)
- Modify: every `river.AddWorker` call outside tests (7: `grep -rn "river.AddWorker" internal | grep -v _test`)
- Modify: `.golangci.yml` forbidigo: ban `river\.AddWorker` outside `internal/platform`, plus a case in `scripts/lint-fixtures/go` and its `expected.txt`
- Test: `internal/platform/jobs_test.go`

**Interfaces:**
- Produces: `func AddWorker[T JobArgs](w *river.Workers, worker river.Worker[T])` — calls `river.AddWorker` and records `kind → zero T.Spec().System` in a package-level `sync.Map` (kinds are fixed per binary; `System` never depends on arg values, see every `Spec()`).
- `JobMiddleware` decides by the recorded flag for `job.Kind`: system → run as today's `RequestedBy == 0` branch; not system and `meta.RequestedBy == 0` → log `job_kind` and `job_id` only at error level, `return river.JobCancel(ErrJobWithoutRequester)` with no notification; otherwise unchanged. A kind with no record is treated as not system.

- [ ] **Step 1: Failing tests** following `TestJobFailureNotified`'s setup:
  - `TestUserJobWithoutRequesterCancelled`: insert a user-kind job directly with empty metadata (bypassing `Enqueue`); after it is worked its state is `cancelled`, `attempt == 1`, the worker body never ran, and no notification row exists.
  - `TestSystemJobWithEmptyMetadataRuns`: a system-kind job with metadata `{}` runs and completes (covers jobs queued before deploy).
- [ ] **Step 2: Run** `go test ./internal/platform -run 'TestUserJobWithoutRequester|TestSystemJobWithEmptyMetadata'` → first FAILS (job runs as system today).
- [ ] **Step 3: Implement** the registry, `AddWorker`, the middleware branch and `ErrJobWithoutRequester` (unexported sentinel is fine). Replace the 7 `river.AddWorker` calls with `platform.AddWorker`. Add the forbidigo rule and lint fixture.
- [ ] **Step 4: Run** the tests → PASS; `go test ./internal/... ` → PASS; `make lint` → PASS (fixture reports exactly `expected.txt`).
- [ ] **Step 5: Commit** `fix(platform): jobs are system or user by declaration, not by a missing requester`.

### Task 5: Small backend rule fixes (#63)

**Files:**
- Modify: `internal/modules/hrm/import.go:186` (`leaveBalanceHeader`)
- Modify: `web/e2e/review-probe.spec.ts:5-10` (comment), rename file if its name is the only reference to the review
- Test: `internal/modules/hrm/export_test.go` or the existing import test file

- [ ] **Step 1: Failing test** `TestLeaveBalanceTemplateNeedsAdjustPermission`: a user without `hrm.leave_balance.adjust` (the `PermBalanceAdjust` the import checks at `import.go:201`) asking `dataio.Template(ctx, "hrm.leave_balance", nil)` gets the same error the import gives; a user with it gets the header.
- [ ] **Step 2: Run** → FAIL (template returned).
- [ ] **Step 3: Implement.** At the top of `leaveBalanceHeader`, the same `s.d.IAM.Scope(ctx, "hrm", PermBalanceAdjust)` check as the import, refusing with the same error when `!sc.Any()`.
- [ ] **Step 4: Run** → PASS.
- [ ] **Step 5: Comments.** Rewrite the header comment of `web/e2e/review-probe.spec.ts` without `TB-0x` ids (say what each case covers). Rename the file to `notification-links.spec.ts` (`git mv`) and fix any reference (`grep -rn review-probe web`). Confirm `grep -rnE "\bM1[0-9]\b|TB-0|NP-0" internal web/src web/e2e` finds nothing else.
- [ ] **Step 6: Commit** `fix(hrm): leave balance template checks the import's permission; drop review ids from comments`.

### Task 6: Whole branch

- [ ] **Step 1:** `make gen` → no diff; `make lint` → PASS; `make test` → PASS.
- [ ] **Step 2:** `make e2e` → PASS (leave overlap panel, employee digest, quotation progress are covered by existing specs; fix selectors only if a spec read the removed badge).
- [ ] **Step 3:** Push and open the PR: title `fix: audit cleanup, frontend and backend`, plain-text body per ticket with commits, the four tickets already done on main named as such, no migrations, then `Closes #55` … `Closes #63`. No "Generated with" line. Do not merge.
