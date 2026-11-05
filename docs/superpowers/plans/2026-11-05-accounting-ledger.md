# Accounting general ledger Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Accounting as the third product: chart of accounts, mapping rules from business line kinds to accounts, automatic journal entries from `posting.lines` through a reconcile job, manual journal entries, general ledger and trial balance.

**Architecture:** New product module `internal/modules/accounting` (schema `accounting`) plugged into `posting.Hooks.OnChanged` in `internal/app`; the hook only enqueues a system job `(doc_type, doc_id)`, and the worker reconciles the ledger against the current business lines. `record` gains a `BeforeLock` hook that accounting implements. Frontend area `web/src/accounting`.

**Tech Stack:** Go, Huma, sqlc, River, Postgres 18; React, TanStack Query, `@mmerp/ui`.

**Spec:** `docs/superpowers/specs/2026-11-05-accounting-ledger-design.md` (read it first; this plan does not repeat its reasons).

## Global Constraints

- Everything in `AGENTS.md`. Project skills give the step order: `.claude/skills/new-module`, `new-document-type`, `new-screen`, `mmerp-ui`.
- ADR numbers: this branch may use only ADR-0050 to ADR-0059.
- Money is `int64`/`bigint` in the minor unit, one currency. No float.
- Services use the data engine (`store.<Query>.Run`, `data.List`) and `platform.InTx`; never `store.New`, `platform.DBFrom` or a `pgx.Tx` parameter.
- New migrations come from `make migration name=accounting_<description>`, one per task that needs schema; every kind/state column has a `CHECK` listing exactly the accepted values.
- No account number appears in Go code. Account numbers live only in seed data (`web/e2e`) and tests.
- Do not run `make dev` or `make e2e` (ports and the `mmerp_e2e` database are shared with other worktrees). Ask the user once, at the start of execution, whether `make lint`, `pnpm -C web typecheck` and `pnpm -C web lint` may run after each task; if not, run only Go tests and Vitest and report lint as not run.
- Go tests: `make db` once, then `go test ./internal/<path> -run <Name>`.
- All code, identifiers, error codes and docs in English; user-facing text only through i18n files, `vi` and `en`.
- Commits: Conventional Commits, scope `accounting` (or `record`, `posting`, `web`, `docs`), ending with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

## Review Focus

1. **Accounting disabled while HR posts or cancels a payroll:** the payroll posts and cancels normally, no job is enqueued. (`platform.Enqueue` applies `ProductGate` with class write, so an unguarded hook would make HR's posting fail.) Test in Task 5.
2. **No start date, or a payroll dated before it:** nothing is pending, nothing reaches the ledger, and the period lock passes. Tests in Tasks 5 and 7.
3. **Rule changed after an entry exists, then the payroll is cancelled:** the reversal mirrors the original entry's lines (same accounts, swapped sides). It never re-maps with the new rules. Test in Task 5.
4. **Rule pointing at an account that later became inactive or gained a child:** the document goes to `unmapped` with that kind listed. Nothing is written on a non-postable or inactive account. Tests in Tasks 4 and 5.
5. **Two legal entities:** pending list, start dates, ledger and trial balance of one never show the other's rows. Test in Task 9.

---

### Task 1: ADR-0050 and docs

**Files:**
- Create: `docs/adr/0050-accounting-product-and-general-ledger.md`
- Create: `products/accounting.md`
- Modify: `CONTEXT.md` (new `## Accounting` section after `## Sales`; rewrite the "Journal entry mapping" entry under Core)
- Modify: `docs/documents.md` (section "Posting": reversal instead of removal, defaults move to the localisation pack, `BeforeLock` exists, start date rule; section "Period lock": drop "no `BeforeLock` yet")
- Modify: `docs/roadmap.md` (accounting no longer in "Not in this roadmap"; add a milestone entry "M15 accounting general ledger" with the spec's "done" list as exit criteria)
- Modify: `AGENTS.md` "Project status" (one sentence: accounting is the third product, started 2026-11-05)

- [ ] **Step 1:** Write ADR-0050 in the style of `docs/adr/0049-*.md` (Status: accepted · 2026-11-05). It covers the `docs/backend.md` "Adding a module" checklist: layer `modules`, product `accounting` with no product dependencies, owned tables (Tasks 2 to 8), the `posting` hook implementation (enqueue only, and only when the product is enabled), the new `record.BeforeLock`, record type `accounting.journal_entry` (prefix `PK`), no posting line kinds of its own, no personal or sensitive fields. Decisions copied from the spec: reversal instead of removal, start date per legal entity fixed after the first entry, first posting may land in a locked period, `BeforeLock` only while the product is enabled, automatic entries are ledger rows, not documents. Rejected alternatives: removing entries on void; hook writing entries in the payroll's transaction (it would block HR on a missing rule); start date as a `setting` legal-entity key (keys are enumerations and cannot carry the "fixed after first entry" guard).
- [ ] **Step 2:** Add the glossary terms with vi names: Account (Tài khoản), Chart of accounts (Hệ thống tài khoản), Postable account (Tài khoản chi tiết), Mapping rule (Quy tắc định khoản), Journal entry (Bút toán), Reversing entry (Bút toán đảo), Manual journal entry (Phiếu kế toán), Accounting start date (Ngày bắt đầu dùng kế toán), Pending document (Chứng từ chờ ghi sổ), General ledger (Sổ cái), Trial balance (Bảng cân đối số phát sinh).
- [ ] **Step 3:** Write `products/accounting.md` (scope, roles, screens, out of scope) from the spec, in the shape of `products/sales.md`.
- [ ] **Step 4:** Commit: `docs(accounting): ADR-0050 accounting product and minimal general ledger`.

### Task 2: Module skeleton, product registration, accounts schema

**Files:**
- Create: `internal/modules/accounting/{service.go,types.go,deps.go,module.go,handler.go,i18n/vi.json,i18n/en.json,queries/account.sql}`, `internal/modules/accounting/internal/store/` (generated)
- Create: `migrations/<ts>_accounting_init.sql`
- Modify: `sqlc.yaml` (entry for the module), `.golangci.yml` (depguard rule like `sales`), `internal/app/app.go` (`products` map gets `"accounting": nil`; construct and mount the module), `Makefile` (`dev` target `PRODUCTS=hrm,sales,accounting`), `scripts/e2e-server.sh` (`:8090` serves `PRODUCTS=hrm,sales,accounting`)
- Test: `internal/modules/accounting/service_test.go`, `internal/app/app_test.go`

**Interfaces:**
- Produces:
  - `type Deps struct { IAM *iam.Service; Audit *audit.Service; Record *record.Service }` (record numbers journal entries; nothing else is needed)
  - `func NewService(d Deps) *Service`, `func Module(s *Service) platform.Module` with `Name: "accounting", Product: "accounting"`
  - Permissions (consts in `types.go`): `PermView = "accounting.view"`, `PermEntryEdit = "accounting.entry.edit"`, `PermPost = "accounting.post"`, `PermManage = "accounting.manage"`
  - Roles via `d.IAM.RegisterRoles("accounting", …, "admin")`: `viewer {view}`, `accountant {view, entry.edit, post}`, `admin {view, entry.edit, post, manage}`
- Schema in the init migration: `CREATE SCHEMA accounting;` and `accounting.accounts (id bigint identity PK, code text NOT NULL UNIQUE CHECK (code ~ '^[0-9A-Za-z.]{1,20}$'), name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200), type text NOT NULL CHECK (type IN ('asset','liability','equity','revenue','expense')), parent_id bigint NULL REFERENCES accounting.accounts, active boolean NOT NULL DEFAULT true)`.

- [ ] **Step 1:** Failing tests: `TestCheckProductsAccepts accounting` (in `internal/app/app_test.go`, `CheckProducts([]string{"hrm","sales","accounting"})` returns nil); `TestAccountingRoles` (a user granted `accounting.accountant` holds `accounting.post` and not `accounting.manage`).
- [ ] **Step 2:** Run `go test ./internal/app -run TestCheckProducts` and `go test ./internal/modules/accounting`. Expected: FAIL (unknown product, package missing).
- [ ] **Step 3:** Create the module per `.claude/skills/new-module` steps 3 to 5, with the table above only. `TestFrontendKnowsEveryProduct` in `internal/app/app_test.go` will need the frontend area: add a minimal `web/src/accounting/index.ts` manifest (product, label, basePath `/accounting`, empty `nav`, i18n meta/main with the group label, `homeIcon: IconCalculator`) registered in `web/src/app/areas.ts` before `core`, plus `web/src/shared/api/accounting.ts` (`productClient<'/accounting/'>()`). Task 10 fills the area.
- [ ] **Step 4:** `make gen`, then run the two tests again. Expected: PASS. Also run `go test ./internal/app -run TestFrontendKnowsEveryProduct`. Expected: PASS.
- [ ] **Step 5:** Commit: `feat(accounting): module skeleton and product registration`.

### Task 3: Chart of accounts

**Files:**
- Create: `internal/modules/accounting/{account.go,account_types.go,account_handler.go,account_test.go}`
- Modify: `internal/modules/accounting/queries/account.sql`

**Interfaces:**
- Produces:
  - `type AccountFields struct { Code string; Name string; Type string; ParentID *int64; Active bool }`, `type Account struct { ID int64; AccountFields; Postable bool; HasLines bool }`
  - `func (s *Service) Accounts(ctx) ([]Account, error)`: full list ordered by code. `PermView` held at any unit (`IAM.Scope(...).Any()`); a chart this size needs no paging.
  - `func (s *Service) CreateAccount(ctx, f AccountFields) (int64, error)`, `func (s *Service) UpdateAccount(ctx, id int64, f AccountFields) error`, `func (s *Service) DeleteAccount(ctx, id int64) error`: `IAM.RequireTenantWide(ctx, "accounting", PermManage)`, audited `accounting.account_saved` / `accounting.account_deleted`.
  - Unexported for Tasks 4, 5, 8: `postableActive(ctx, ids []int64) (map[int64]bool, error)`: true for an account that exists, is active and has no child.
  - Routes: `GET/POST /accounting/accounts`, `PUT/DELETE /accounting/accounts/{id}`.
- Errors (`types.go`): `account_code_taken` (409, constraint `accounting.accounts_code_key`), `account_parent_invalid` (422: parent missing, itself, a descendant, of another type, or an account with lines), `account_in_use` (409: delete, type change or code change on an account with lines; delete of an account with children or referenced by a rule).

- [ ] **Step 1:** Failing tests in `account_test.go`:
  - `TestAccountCreateAndList`: create `642` expense and `334` liability; `Accounts` returns both by code, `Postable` true.
  - `TestAccountParentMakesParentNotPostable`: create `333`, then `3335` with parent `333`; `333.Postable == false`.
  - `TestAccountParentRules`: parent of another type gives `account_parent_invalid`; making `333` a child of `3335` (cycle) gives `account_parent_invalid`.
  - `TestAccountCodeTaken`: `account_code_taken`.
  - `TestAccountNeedsManage`: an `accounting.accountant` gets forbidden on create.
  - `TestAccountInUse` is added in Task 5, once lines exist.
- [ ] **Step 2:** Run `go test ./internal/modules/accounting -run TestAccount`. Expected: FAIL.
- [ ] **Step 3:** Implement. `HasLines` and the in-use checks read `accounting.entry_lines` once Task 5 creates it; until then return false (one `TODO(task5)` comment removed in Task 5).
- [ ] **Step 4:** Run again. Expected: PASS.
- [ ] **Step 5:** Commit: `feat(accounting): chart of accounts`.

### Task 4: Mapping rules and accounting start dates

**Files:**
- Create: `internal/modules/accounting/{rule.go,rule_types.go,rule_handler.go,rule_test.go,start.go,start_test.go}`, `queries/rule.sql`, `queries/start.sql`, `migrations/<ts>_accounting_rules.sql`
- Modify: `internal/shared/posting/types.go` (add `func Kinds() []Kind` returning the five kinds in declaration order)
- Test: `internal/shared/posting/service_test.go` (`TestKindsListsEveryKind`)

**Interfaces:**
- Schema: `accounting.rules (kind text PRIMARY KEY, account_id bigint NOT NULL REFERENCES accounting.accounts, side text NOT NULL CHECK (side IN ('debit','credit')))`; `accounting.starts (legal_entity_id bigint PRIMARY KEY REFERENCES iam.org_units, start_date date NOT NULL)`.
- Produces:
  - `type Rule struct { Kind string; AccountID *int64; Side *string }`: one per `posting.Kinds()`, plus any kind found in `accounting.rules` or `posting.lines` but not in `Kinds()`.
  - `func (s *Service) Rules(ctx) ([]Rule, error)` (`PermView` anywhere), `func (s *Service) SaveRule(ctx, kind string, accountID int64, side string) error`, `func (s *Service) DeleteRule(ctx, kind string) error` (`PermManage` tenant-wide; the account must be postable and active, else `account_not_postable` 422).
  - `type Start struct { LegalEntityID int64; StartDate *string; Fixed bool }`, `func (s *Service) Starts(ctx) ([]Start, error)` (one per legal entity), `func (s *Service) SetStart(ctx, legalEntity int64, date *string) error` (`PermManage` tenant-wide; `nil` clears). Refused with `start_date_fixed` (409) once `accounting.entries` has a row for that legal entity; this check lands in Task 5 with the table (same `TODO(task5)` pattern).
  - Unexported `startOf(ctx, legalEntity int64) (pgtype.Date, error)`: invalid date when unset.
  - Routes: `GET /accounting/rules`, `PUT/DELETE /accounting/rules/{kind}`, `GET /accounting/starts`, `PUT /accounting/starts/{legal_entity_id}`.

- [ ] **Step 1:** Failing tests: `TestKindsListsEveryKind`; `TestRulesListEveryKind` (5 rows, all unset); `TestSaveRuleRefusesParentAccount` (`account_not_postable`); `TestSaveRuleRefusesInactiveAccount`; `TestSetStartPerLegalEntity` (two legal entities, setting one leaves the other unset).
- [ ] **Step 2:** Run `go test ./internal/shared/posting ./internal/modules/accounting -run 'TestKinds|TestRule|TestSaveRule|TestSetStart'`. Expected: FAIL.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** Run again. Expected: PASS.
- [ ] **Step 5:** Commit: `feat(accounting): mapping rules and start dates`.

### Task 5: Ledger tables, reconcile worker, posting hook

**Files:**
- Create: `internal/modules/accounting/{ledger.go,ledger_types.go,reconcile.go,reconcile_test.go,jobs.go}`, `queries/ledger.sql`, `migrations/<ts>_accounting_ledger.sql`
- Modify: `internal/modules/accounting/module.go` (`Workers`), `internal/app/app.go` (replace `posting.NewService(posting.Hooks{})` and its comment; `pst` is built after `acc` or receives the hook via a closure over `acc`), `account.go` and `start.go` (remove the `TODO(task5)`)
- Test: `internal/modules/accounting/reconcile_test.go`, `internal/app/payroll_test.go` (new `TestPayrollReachesLedger`)

**Interfaces:**
- Schema:
  - `accounting.entries (id bigint identity PK, legal_entity_id bigint NOT NULL REFERENCES iam.org_units, date date NOT NULL, doc_type text NOT NULL, doc_id bigint NOT NULL, recording_id bigint NULL REFERENCES posting.recordings, reverses bigint NULL UNIQUE REFERENCES accounting.entries, created_at timestamptz NOT NULL DEFAULT now())`, index `(doc_type, doc_id)`, index `(legal_entity_id, date)`.
  - `accounting.entry_lines (entry_id bigint NOT NULL REFERENCES accounting.entries, line_no int NOT NULL, account_id bigint NOT NULL REFERENCES accounting.accounts, org_unit_id bigint NULL REFERENCES iam.org_units, kind text NULL, description text NULL, debit bigint NOT NULL CHECK (debit >= 0), credit bigint NOT NULL CHECK (credit >= 0), CHECK ((debit = 0) <> (credit = 0)), PRIMARY KEY (entry_id, line_no))`, index `(account_id)`.
  - `accounting.sources (doc_type text, doc_id bigint, PRIMARY KEY (doc_type, doc_id), legal_entity_id bigint NOT NULL, date date NOT NULL, posted_recording_id bigint NULL, problem text NULL CHECK (problem IN ('unmapped','unbalanced')), missing_kinds text[] NOT NULL DEFAULT '{}')`: the tracking row.
- Definition used by Tasks 6 and 7: a source document is **pending** when it has a posting recording dated on or after its legal entity's start date, and `live recording id IS DISTINCT FROM sources.posted_recording_id`. The live recording id is the one whose `voided_at` is null, or null if every recording is voided. A missing `sources` row counts as `posted_recording_id = null`.
- Produces:
  - `type JournalLine struct { AccountID int64; OrgUnitID *int64; Kind string; Description string; Debit, Credit int64 }`
  - unexported `writeEntry(ctx, legalEntity int64, date pgtype.Date, src record.Ref, recording *int64, lines []JournalLine) (int64, error)`: the **only** insert into `accounting.entries`/`entry_lines`. Refuses with `entry_unbalanced` (422) when debits ≠ credits or fewer than 2 lines, and with `account_not_postable` when an account is not postable and active.
  - unexported `reverseEntry(ctx, entryID int64) (int64, error)`: same date and source, `reverses = entryID`, each line's debit and credit swapped. Never re-maps.
  - `func (s *Service) Reconcile(ctx context.Context, ref record.Ref) error`: system entry point, no permission check (comment says so), runs its own `platform.InTx`.
  - `func (s *Service) PostingChanged(ctx context.Context, ref record.Ref) error`: the hook. If `platform.ProductGate(ctx, "accounting", platform.ClassWrite) != nil`, return nil. Otherwise `platform.Enqueue(ctx, reconcileArgs{DocType, DocID})`.
  - `reconcileArgs` kind `accounting.reconcile`, `Spec() = platform.JobSpec{Product: "accounting", Class: platform.ClassWrite, System: true}`; `river.InsertOpts` unique by args while available/pending, so a burst collapses to one job.

`Reconcile` algorithm (the order matters; the rest is the implementer's):

```
InTx:
  d := Record.Lock(ctx, ref)               // period lock FOR SHARE, then document FOR UPDATE
  start := startOf(d.LegalEntityID); if unset or d.Date < start: return nil
  lockedUntil := SELECT locked_until FROM record.period_locks (row already share-locked)
  live := live recording id + its lines (posting.lines of that recording)
  src := upsert sources row (doc_type, doc_id, legal_entity_id, date), SELECT … FOR UPDATE
  if src.posted_recording_id == live: clear problem; return nil      // in line
  if src.posted_recording_id != nil:                                 // ledger shows an older or voided recording
      if d.Date <= lockedUntil: return coded 409 `period_locked` (cannot happen: record refuses cancelling there)
      reverseEntry(entry of src.posted_recording_id); src.posted_recording_id = nil
  if live == nil: save src; return nil
  map each line: rule(kind) missing, or account not postable/active → missing_kinds
  if missing: src.problem = 'unmapped', missing_kinds sorted; save; return nil
  amount > 0 → rule side; amount < 0 → opposite side, abs(amount); org unit carried; kind set
  if debits != credits: src.problem = 'unbalanced'; save; return nil
  writeEntry(... recording = live) ; src.posted_recording_id = live; problem = nil; save
  // a first posting may land on or before lockedUntil: allowed by design
```

- [ ] **Step 1:** Failing tests in `reconcile_test.go`. Use a test document type and `posting.Service.Record`/`Void` directly, as `internal/shared/stock/ledger_test.go` does with its `testDoc`. Fixture: accounts `642` expense, `334`, `338`, `3335` liabilities; rules for all five kinds; start date `2026-01-01`.
  - `TestReconcilePostsBalancedEntry`: lines `salary_expense 100 (unit A)`, `employer_insurance_expense 20 (unit A)`, `salary_payable 80 (unit A)`, `insurance_payable 30 (LE)`, `pit_payable 10 (LE)`. After `Reconcile`: one entry, debit 642 = 120, credit 334 = 80, 338 = 30, 3335 = 10, org units carried, `sources.posted_recording_id` set.
  - `TestReconcileIdempotent`: second `Reconcile` adds no entry.
  - `TestReconcileVoidWritesReversal`: Void then Reconcile. Two entries, the second has `reverses` = first and swapped sides; net per account 0.
  - `TestReconcileOrderIndependent`: Record, Void, then Reconcile once. No entry at all (nothing live, nothing posted).
  - `TestReconcileUnmapped`: delete the `pit_payable` rule. No entry, `problem = 'unmapped'`, `missing_kinds = {pit_payable}`, `Reconcile` returns nil. Restore the rule, Reconcile, entry written, problem cleared.
  - `TestReconcileInactiveAccountIsUnmapped`: deactivate `3335`. `unmapped` with `pit_payable`.
  - `TestReconcileNegativeAmountFlipsSide`: `salary_expense -50` plus `salary_payable -50`. Credit 642 50, debit 334 50.
  - `TestReconcileUnbalanced`: lines `salary_expense 100`, `salary_payable 90`. `problem = 'unbalanced'`, no entry.
  - `TestReconcileBeforeStartDate`: document dated `2025-12-31`. No `sources` row, no entry.
  - `TestReconcileRuleChangeDoesNotRemap` (Review Focus 3): post, then point `salary_expense` at a new account `6422`, then Void and Reconcile. The reversal debits/credits `642`, never `6422`.
  - `TestReconcileFirstPostingIntoLockedPeriod`: lock the period through `2026-12-31` before reconciling. Entry is written.
  - `TestPostingChangedSkipsWhenProductOff` (Review Focus 1): ctx with products `hrm` only; `PostingChanged` inside `InTx` returns nil and `river_job` has no `accounting.reconcile` row.
  - `TestAccountInUse` (from Task 3): after an entry exists, deleting `642` or changing its type gives `account_in_use`.
  - `TestStartDateFixedAfterEntry` (from Task 4): `SetStart` gives `start_date_fixed`.
- [ ] **Step 2:** Run `go test ./internal/modules/accounting -run 'TestReconcile|TestPostingChanged|TestAccountInUse|TestStartDateFixed'`. Expected: FAIL.
- [ ] **Step 3:** Implement schema, `writeEntry`, `reverseEntry`, `Reconcile`, the worker (`Work` calls `Reconcile`), `PostingChanged`, and the wiring in `app.go`.
- [ ] **Step 4:** Run again. Expected: PASS.
- [ ] **Step 5:** Add `TestPayrollReachesLedger` to `internal/app/payroll_test.go`, reusing `jobsFixture` (`payrollSetup`, `createPayroll`, `approveAll`, `work`). Products `hrm,accounting`. Seed accounts, rules and a start date through the HTTP API, post a payroll, work jobs, assert one balanced entry for `hrm.payroll` whose total debit equals gross plus employer insurance; cancel, work jobs, assert the reversal. Then turn accounting off (products `hrm`), post and cancel another payroll, and assert both succeed. Run `go test ./internal/app -run TestPayrollReachesLedger`. Expected: PASS.
- [ ] **Step 6:** Commit: `feat(accounting): journal entries from business lines`.

### Task 6: Pending documents and "post pending"

**Files:**
- Create: `internal/modules/accounting/{pending.go,pending_types.go,pending_handler.go,pending_test.go}`, `queries/pending.sql`

**Interfaces:**
- Consumes: the pending definition and `reconcileArgs` from Task 5.
- Produces:
  - `type PendingDoc struct { DocType string; DocID int64; Number string; Date string; LegalEntityID int64; Reason string; MissingKinds []string }`. `Reason` is one of `queued` (no problem recorded), `unmapped`, `unbalanced`.
  - unexported `pendingIn(ctx, legalEntity int64, until *pgtype.Date) ([]PendingDoc, error)`: dated ≤ until when given. Used by Task 7.
  - `func (s *Service) Pending(ctx, legalEntity int64) ([]PendingDoc, error)`: `PermView` at the legal entity (`IAM.Require…` at that unit, as other legal-entity scoped reads do).
  - `func (s *Service) PostPending(ctx, legalEntity int64) (int, error)`: `PermPost` at the legal entity and `ProductGate` write. Inside one `InTx`, enqueues one `reconcileArgs` per pending document and returns the count; each job runs in its own transaction.
  - Routes: `GET /accounting/pending?legal_entity_id=`, `POST /accounting/pending/post` body `{legal_entity_id}`.

- [ ] **Step 1:** Failing tests:
  - `TestPendingListsUnreconciled`: Record without Reconcile. Listed with `queued`.
  - `TestPendingShowsUnmapped`: listed with `unmapped` and `missing_kinds`.
  - `TestPendingDocumentsFromBeforeEnable`: a recording written while products exclude accounting is listed.
  - `TestPendingIgnoresBeforeStartAndUnset`: not listed.
  - `TestPostPendingEnqueuesOnePerDocument`: two pending documents give 2, and the `river_job` rows exist.
  - `TestPostPendingNeedsPost`: `accounting.viewer` is forbidden.
- [ ] **Step 2:** Run `go test ./internal/modules/accounting -run 'TestPending|TestPostPending'`. Expected: FAIL.
- [ ] **Step 3:** Implement. The number comes from `record.documents`.
- [ ] **Step 4:** Run again. Expected: PASS.
- [ ] **Step 5:** Commit: `feat(accounting): pending documents and post pending`.

### Task 7: `record.BeforeLock` and accounting's implementation

**Files:**
- Modify: `internal/core/record/hooks.go`, `internal/core/record/service.go` (`SetPeriodLock`, around the `HeldInPeriod` block)
- Modify: `internal/app/app.go` (`rec.SetBeforeLock(acc.BeforeLock)`)
- Create: `internal/modules/accounting/lock.go`, `lock_test.go`
- Test: `internal/core/record/service_test.go` (`TestBeforeLockRefuses`)
- Modify: `docs/documents.md` "Guarantees" list (one bullet for `BeforeLock`)

**Interfaces:**
- Produces:
  - `record`: `type BeforeLock func(ctx context.Context, legalEntity int64, until pgtype.Date) ([]HeldDoc, error)` and `func (s *Service) SetBeforeLock(f BeforeLock)`. `SetPeriodLock` calls it only when the lock date moves later, after `HeldInPeriod`, under the same `FOR UPDATE`. Its documents join the same `period_has_pending_documents` error list.
  - `accounting`: `func (s *Service) BeforeLock(ctx, legalEntity int64, until pgtype.Date) ([]record.HeldDoc, error)`. Returns nil when `ProductGate(ctx, "accounting", ClassWrite) != nil` or no start date is set; otherwise `pendingIn(ctx, legalEntity, &until)`.

- [ ] **Step 1:** Failing tests:
  - `record`: `TestBeforeLockRefuses`. A stub hook returning one doc gives `period_has_pending_documents` listing it, and the lock date is unchanged. Moving the lock earlier does not call the hook.
  - `accounting`: `TestLockRefusedWhilePending`; `TestLockAllowedOnceReconciled`; `TestLockAllowedWhenProductOff`; `TestLockAllowedWithoutStartDate` (Review Focus 2).
- [ ] **Step 2:** Run `go test ./internal/core/record -run TestBeforeLock` and `go test ./internal/modules/accounting -run TestLock`. Expected: FAIL.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** Run again, plus `go test ./internal/core/record` (the whole package, since `SetPeriodLock` changed). Expected: PASS.
- [ ] **Step 5:** Commit: `feat(record): BeforeLock hook; accounting refuses locks over pending documents`.

### Task 8: Manual journal entry document

**Files:**
- Create: `internal/modules/accounting/{journal.go,journal_types.go,journal_list.go,journal_handler.go,journal_test.go}`, `queries/journal.sql`, `migrations/<ts>_accounting_journal_entries.sql`

**Interfaces:**
- Follow `.claude/skills/new-document-type`; `internal/shared/stock/adjustment*.go` is the closest model.
- Schema: `accounting.journal_entries (id bigint PRIMARY KEY REFERENCES record.documents, description text NOT NULL CHECK (char_length(description) BETWEEN 1 AND 500))`; `accounting.journal_entry_lines (journal_entry_id bigint REFERENCES accounting.journal_entries, line_no int, account_id bigint NOT NULL REFERENCES accounting.accounts, org_unit_id bigint NULL REFERENCES iam.org_units, description text NULL, debit bigint NOT NULL CHECK (debit >= 0), credit bigint NOT NULL CHECK (credit >= 0), CHECK ((debit = 0) <> (credit = 0)), PRIMARY KEY (journal_entry_id, line_no))`.
- Record type: `Code: "accounting.journal_entry", Product: "accounting", Kind: record.Document, NumberPrefix: "PK"`, `Perms{View: PermView, Edit: PermEntryEdit, ViewFiles: PermView, Attach: PermEntryEdit}`, approval field `total` (amount, the sum of debits), `OnTransition`, `OnDelete`.
- Produces:
  - `type JournalEntryFields struct { Date string; LegalEntityID int64; Description string; Lines []JournalEntryLine }`, `type JournalEntryLine struct { AccountID int64; OrgUnitID *int64; Description *string; Debit, Credit int64 }`
  - `JournalEntryActions`, `JournalEntries(ctx, f JournalEntryFilter) (JournalEntryList, error)` (a `data.List`, filters: legal entity, status, date range, number), `JournalEntry(ctx, id)`, `CreateJournalEntry(ctx, NewJournalEntry) (int64, error)` (with `RequestID`), `UpdateJournalEntry(ctx, id, JournalEntryUpdate) error`
  - Routes under `/accounting/journal-entries`; transitions and delete through the core document routes.
- Rules:
  - Saving a draft checks each line has exactly one side (`line_side_invalid` with `line_no`).
  - Posting checks: at least 2 lines, debits = credits (`journal_entry_unbalanced`), every account postable and active (`account_not_postable` with `line_no`), legal entity has a start date and the date is on or after it (`accounting_not_started`).
  - `OnTransition` to `posted` calls `writeEntry` with `recording = nil` and the entry's own ref; to `cancelled`, `reverseEntry` of that entry.

- [ ] **Step 1:** Failing tests: `TestJournalEntryRecordContract` (`recordtest.Run`; `Effects` counts unreversed entries of the ref; `BreakPosting` deactivates the account used by the lines); `TestJournalEntryUnbalancedRefused`; `TestJournalEntryLineSideInvalid`; `TestJournalEntryBeforeStartRefused`; `TestJournalEntryCancelReverses`.
- [ ] **Step 2:** Run `go test ./internal/modules/accounting -run TestJournalEntry`. Expected: FAIL.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** Run again. Expected: PASS.
- [ ] **Step 5:** Commit: `feat(accounting): manual journal entries`.

### Task 9: General ledger and trial balance

**Files:**
- Create: `internal/modules/accounting/{report.go,report_types.go,report_handler.go,report_test.go}`, `queries/report.sql`

**Interfaces:**
- Produces:
  - `type LedgerLine struct { Date string; EntryID int64; DocType string; DocID int64; DocNumber string; Kind *string; Description *string; OrgUnitID *int64; AccountCode string; Debit, Credit, Balance int64 }`
  - `type Ledger struct { Opening int64; Lines []LedgerLine; Closing int64 }`. Balances are signed debit-minus-credit; the frontend shows a positive value as a debit balance and a negative one as a credit balance.
  - `func (s *Service) Ledger(ctx, legalEntity, accountID int64, from, to string) (Ledger, error)`: an account with children includes every descendant's lines, ordered by date, entry id, line no.
  - `type TrialRow struct { AccountID int64; Code, Name string; ParentID *int64; Opening, Debit, Credit, Closing int64 }`, `type TrialBalance struct { Rows []TrialRow; TotalDebit, TotalCredit int64 }`
  - `func (s *Service) TrialBalance(ctx, legalEntity int64, from, to string) (TrialBalance, error)`: every account with a non-zero opening or movement, parents rolled up from their descendants. Totals sum postable accounts only, so they are not counted twice.
  - Both need `PermView` at the legal entity. `from > to` gives `invalid_range`. Routes: `GET /accounting/ledger`, `GET /accounting/trial-balance`.
  - `// ponytail: no paging; a ledger of one account over a year is a few thousand lines at this size. Page it when a tenant outgrows it.`

- [ ] **Step 1:** Failing tests (entries made through `writeEntry` helpers, or through Reconcile from Task 5's fixture):
  - `TestLedgerOpeningAndRunningBalance`: entries on 2026-01-15 and 2026-02-10, ledger for February. Opening equals the January movement, the running balance is correct, and closing = opening + debit − credit.
  - `TestLedgerParentIncludesChildren`.
  - `TestTrialBalanceTotalsEqual`: `TotalDebit == TotalCredit` after several entries, including a reversal.
  - `TestTrialBalanceRollsUpParents`.
  - `TestReportsStayInLegalEntity` (Review Focus 5): two legal entities, each with an entry, and each report shows only its own. Also `Pending` of one entity does not list the other's documents.
- [ ] **Step 2:** Run `go test ./internal/modules/accounting -run 'TestLedger|TestTrialBalance|TestReportsStay'`. Expected: FAIL.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** Run again. Expected: PASS.
- [ ] **Step 5:** Commit: `feat(accounting): general ledger and trial balance`.

### Task 10: Frontend screens

**Files:**
- Modify: `web/src/accounting/{index.ts,routes.tsx,keys.ts,i18n/{vi,en}/{meta,main}.json}`, `web/src/shared/api/accounting.ts` (types after `make gen`)
- Create: `web/src/accounting/pages/{AccountsPage,RulesPage,StartsPage,PendingPage,JournalEntriesPage,JournalEntryPage,NewJournalEntryPage,LedgerPage,TrialBalancePage}.tsx`, `web/src/accounting/components/JournalEntryPreview.tsx`, `web/src/accounting/balance.ts` + `balance.test.ts`

**Interfaces:**
- Consumes: every route from Tasks 3, 4, 6, 8, 9.
- Produces:
  - Nav, permissions from Task 2:
    - `journal-entries` (view)
    - `ledger` (view)
    - `trial-balance` (view)
    - `pending` (view)
    - group `accounting.nav.group_config`: `accounts`, `rules`, `starts`, `approval-rules` (manage)
  - `recordTypes['accounting.journal_entry']`: `path: id => 'journal-entries/' + id`, `preview`, `invalidate` covering journal entries, ledger, trial balance, pending.
  - `balance.ts`: `splitBalance(n: number): { debit: number; credit: number }` (positive → debit, negative → credit) and `kindLabel(t, kind)` (falls back to the raw kind when no i18n key exists).

- [ ] **Step 1:** Failing Vitest `balance.test.ts`: `splitBalance(120)` gives `{debit:120, credit:0}`; `splitBalance(-30)` gives `{debit:0, credit:30}`; `splitBalance(0)` gives zeros; `kindLabel` returns the key's text for `salary_expense` and the raw string for `unknown_kind`.
- [ ] **Step 2:** Run `pnpm -C web exec vitest run src/accounting/balance.test.ts`. Expected: FAIL.
- [ ] **Step 3:** Implement `balance.ts`, then build each screen with the `new-screen` and `mmerp-ui` skills and `docs/ui.md` (page templates from `shared/ui/page`, `DocumentPage` for the journal entry, money formatting from the shared helpers). Screen behaviour from the spec's "Screens" section:
  - Ledger and trial balance: legal entity picker, date range; the account picker on the ledger.
  - Pending rows: link to the source document through the owning area's `recordTypes[...].path`, the same way the approval inbox does. The "post pending" button appears only with `accounting.post`.
- [ ] **Step 4:** Run `pnpm -C web exec vitest run src/accounting`. Expected: PASS. Run typecheck and lint only if the user agreed (Global Constraints). Finish with the `docs/ui.md` checklist.
- [ ] **Step 5:** Commit: `feat(web): accounting screens`.

### Task 11: Demo seed and Playwright spec

**Files:**
- Create: `web/e2e/accounting.seed.ts`, `web/e2e/accounting.spec.ts`
- Modify: the seed project wiring in `web/playwright.config.ts` (or wherever `sales.seed.ts` is wired) so `make seed` runs it after the HRM and Sales seeds

**Interfaces:**
- Seed data, all created through the API, never in Go:
  - Accounts: `111` Tiền mặt (asset), `112` Tiền gửi ngân hàng (asset), `333` Thuế và các khoản phải nộp Nhà nước (liability) with child `3335` Thuế thu nhập cá nhân, `334` Phải trả người lao động (liability), `338` Phải trả, phải nộp khác (liability), `411` Vốn đầu tư của chủ sở hữu (equity), `642` Chi phí quản lý doanh nghiệp (expense).
  - Rules: `salary_expense → 642 debit`, `employer_insurance_expense → 642 debit`, `salary_payable → 334 credit`, `insurance_payable → 338 credit`, `pit_payable → 3335 credit`.
  - Start date `2026-01-01` for the demo company. One posted manual entry for opening balances: debit `112` 500 000 000, credit `411` 500 000 000.
  - Run "post pending" so the seeded payrolls reach the ledger.

- [ ] **Step 1:** Write the seed. Write `accounting.spec.ts` covering the spec's "done" list 1 to 7 against `:8090`. Checking existing e2e specs that count home-page product tiles is part of this step: if one exists, update its expected count.
- [ ] **Step 2:** Do **not** run Playwright or `make seed` here: both need shared ports or a running `make dev`. Report them as not run and ask the user.
- [ ] **Step 3:** Commit: `test(accounting): demo seed and end-to-end spec`.
