# Partner Subledger (M16) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Business lines may name a partner, accounts may be tracked by partner, and accounting reports balances and ledgers per partner.

**Architecture:**
- `shared/posting` gains a nullable partner on lines and a declaration per kind saying whether its lines name a partner.
- `accounting` gains a `by_partner` flag on accounts. It enforces partner presence on every journal line in `writeEntry`, the single place an entry is written, and names the bad line earlier in rule saving and manual-entry checks.
- Two read-only reports join `partner.partners` for names.

**Tech Stack:** Go, Huma, sqlc, Postgres 18, React + `@mmerp/ui`, Playwright.

**Spec:** `docs/superpowers/specs/2026-11-12-partner-subledger-design.md`, with decisions in `docs/adr/0051-business-lines-name-a-partner.md`. Read both, plus `AGENTS.md`.

## Global Constraints

- Branch `feat/accounting-partner-ledger`. One PR into `main` at the end.
- Commits follow Conventional Commits, scopes `posting`, `accounting` and `web`. Stage explicit paths, never `git add -A`.
- New migrations come from `make migration name=<module>_<description>` and are SQL only.
- Money is `int64`. Balances are signed: debit minus credit.
- Services query only through `store.<Query>.Run(ctx, …)`, and write only inside `platform.InTx`.
- Error codes are stable strings with params, never sentences. A line error carries `line_no` (use `lineErr`).
- User-facing text goes into `web/src/accounting/i18n/{vi,en}/main.json`, in both languages.
- Screens follow the `mmerp-ui` and `new-screen` skills and finish with the `docs/ui.md` checklist.
- After any SQL or API type change, run `make gen`. `make lint` and `make test` must pass before each commit that touches code.
- Never mock the database. Tests use the `pgtest` databases, as the existing tests do.

## Review Focus

1. **A reversal keeps the partner.** Cancelling an entry with partner lines must bring each partner back to zero, not leave the partner on the original and NULL on the reversal. Test in Task 3.
2. **A partner on a line of an untracked account, saved in a draft.** Saving is allowed, so drafts can be incomplete. Sending or posting refuses `journal_line_partner_not_allowed` with `line_no`. Test in Task 4.
3. **An unknown `partner_id` on a manual entry line.** Saving returns 404, as unknown accounts do today. Test in Task 4.
4. **A tracked account gaining a child.** Refused with `account_parent_invalid`. Otherwise a tracked account would become non-postable while rules still point at it. Test in Task 2.
5. **Partner balances with no account filter, when one partner has lines on both 131 and 331.** The report shows two rows, one per (account, partner); it does not net them into one. Test in Task 5.

---

### Task 1: Partner on business lines (`shared/posting`)

**Files:**
- Create: `migrations/<ts>_posting_partner.sql` (via `make migration name=posting_partner`)
- Modify: `internal/shared/posting/types.go`, `internal/shared/posting/service.go`, `internal/shared/posting/queries/posting.sql`
- Modify: `internal/modules/accounting/rule.go:29` (adapt to the new `Kinds()` return type)
- Test: `internal/shared/posting/service_test.go`

**Interfaces:**
- Produces:

  ```go
  type KindDecl struct { Kind Kind; ByPartner bool }
  func Kinds() []KindDecl                      // declaration order; the five payroll kinds have ByPartner false
  func Declared(k Kind) (KindDecl, bool)
  func DeclareKind(k Kind, byPartner bool)     // appends; panics on a duplicate; call before serving (init or TestMain)
  type Line struct { Kind Kind; OrgUnitID int64; PartnerID *int64; Amount int64 }
  var ErrPartnerMismatch = &platform.Error{Status: http.StatusInternalServerError, Code: "posting_partner_mismatch"}
  ```

- [ ] **Step 1: Write the failing tests.**
  - `TestKindsListsEveryKind`: the five payroll kinds, in order, all with `ByPartner == false`.
  - `TestRecordRefusesPartnerMismatch`: declare a test kind `test_receivable` with `ByPartner` true in `TestMain`. Then check three things: recording that kind without a partner returns `ErrPartnerMismatch`; recording `SalaryExpense` with a partner returns `ErrPartnerMismatch`; and recording an undeclared kind with a partner returns `ErrPartnerMismatch`, because undeclared kinds name no partner.
  - `TestRecordKeepsPartner`: a `test_receivable` line with a partner id is stored with that `partner_id` (read it back through the system pool, as `TestRecordAndVoid` does). Create the partner by inserting a row into `partner.partners` through the test's system pool.
- [ ] **Step 2: Run them and see them fail.** `go test ./internal/shared/posting -run 'TestKinds|TestRecord'` fails to compile, because `KindDecl` is undefined.
- [ ] **Step 3: Add the migration and code.**
  - Migration: `ALTER TABLE posting.lines ADD COLUMN partner_id bigint NULL REFERENCES partner.partners;`
  - `InsertLines` takes `@partner_ids::bigint[]`, stored through `NULLIF(…, 0)`, the same as `org_unit_ids` in `accounting.InsertEntryLines`.
  - `Record` checks every line against `Declared(l.Kind)` before inserting anything.
  - `rule.go` loops over `d.Kind`.
- [ ] **Step 4: Run `make gen`, then `go test ./internal/shared/posting ./internal/modules/...`.** Expected: PASS. HRM and accounting are unchanged.
- [ ] **Step 5: Commit** `feat(posting): business lines may name a partner`.

### Task 2: Tracked accounts and matching rules (`accounting`)

**Files:**
- Create: `migrations/<ts>_accounting_partner.sql` (via `make migration name=accounting_partner`). It holds every accounting column of this plan: `accounts.by_partner boolean NOT NULL DEFAULT false`, and `partner_id bigint NULL REFERENCES partner.partners` on both `entry_lines` and `journal_entry_lines`.
- Modify: `internal/modules/accounting/account_types.go`, `account.go`, `rule_types.go`, `rule.go`, `types.go`, `queries/account.sql`, `queries/rule.sql`
- Test: `internal/modules/accounting/account_test.go`, `rule_test.go`

**Interfaces:**
- Consumes: `posting.Declared`, `posting.Kinds`, `posting.DeclareKind` (Task 1).
- Produces:
  - `AccountFields.ByPartner bool` (json `by_partner`).
  - `Rule.ByPartner bool` (json `by_partner`): the kind's declaration, so the screen can show it.
  - `ErrRulePartnerMismatch` (422, `rule_partner_mismatch`).
  - `ErrAccountRuleMismatch` (409, `account_partner_rule_mismatch`).
  - `func accountFlags(ctx context.Context, ids []int64) (map[int64]bool, error)`: id to `by_partner`.

- [ ] **Step 1: Write the failing tests.** The accounting package declares `test_receivable` (`ByPartner` true) in its own `TestMain`.
  - `TestRuleMustMatchPartnerTracking`: `SaveRule(salary_expense → tracked 131)` returns `ErrRulePartnerMismatch`; `SaveRule(test_receivable → untracked 642)` returns `ErrRulePartnerMismatch`; `SaveRule(test_receivable → tracked 131)` returns nil.
  - `TestAccountFlagLockedByLinesAndRules`:
    - Turning the flag on for an account with lines returns `ErrAccountInUse`.
    - Turning it off while `test_receivable` maps to it returns `ErrAccountRuleMismatch`.
    - Turning it on while `salary_expense` maps to it returns `ErrAccountRuleMismatch`.
    - Turning it on for an unused, unmapped leaf succeeds.
  - `TestTrackedAccountTakesNoChild` (Review Focus 4): creating an account whose parent is tracked returns `ErrAccountParentInvalid`. Setting the flag on a parent returns `ErrAccountNotPostable`.
  - `TestRulesListShowsPartnerKinds`: `Rules()` has `ByPartner == true` for `test_receivable` and false for the payroll kinds.
- [ ] **Step 2: Run them and see them fail.** `go test ./internal/modules/accounting -run 'TestRule|TestAccountFlag|TestTracked'`.
- [ ] **Step 3: Implement.**
  - The checks live in `UpdateAccount`, `CreateAccount`, `checkParent` and `SaveRule`, inside `writeAccount`, so the chart lock already held serializes them.
  - An undeclared kind counts as `ByPartner` false.
- [ ] **Step 4: Run `make gen` and `go test ./internal/modules/accounting`.** Expected: PASS.
- [ ] **Step 5: Commit** `feat(accounting): accounts tracked by partner; rules must match`.

### Task 3: Partner on ledger lines and automatic entries

**Files:**
- Modify: `internal/modules/accounting/ledger_types.go`, `ledger.go`, `reconcile.go`, `report.go`, `report_types.go`, `queries/ledger.sql`, `queries/report.sql`
- Test: `internal/modules/accounting/reconcile_test.go`, `report_test.go`

**Interfaces:**
- Consumes: `posting.Line.PartnerID` (Task 1), `accountFlags` (Task 2).
- Produces:
  - `JournalLine.PartnerID *int64`.
  - `ErrEntryPartnerMismatch` (422, `entry_partner_mismatch`): the `writeEntry` backstop.
  - `LedgerLine` gains `PartnerID *int64`, `PartnerCode *string` and `PartnerName *string` (json `partner_id`, `partner_code`, `partner_name`).

- [ ] **Step 1: Write the failing tests.**
  - `TestReconcileCarriesPartner`: rules `test_receivable → 131` (tracked, debit) and `salary_payable → 334` (credit). Record two `test_receivable` lines, for partners A (70) and B (30), plus a `salary_payable` line of 100. The entry has a 131 line for A with debit 70, a 131 line for B with debit 30, and a 334 line with no partner and credit 100.
  - `TestReconcileVoidReversesPartner` (Review Focus 1): void that document. The reversal's 131 lines carry A and B with the sides swapped.
  - `TestWriteEntryRefusesPartnerMismatch`: call `writeEntry` directly with a line on a tracked account and no partner, and with a line on an untracked account that has a partner. Both return `ErrEntryPartnerMismatch`.
  - `TestLedgerShowsPartner`: the general ledger of 131 has `partner_code` and `partner_name` on each line.
- [ ] **Step 2: Run them and see them fail.**
- [ ] **Step 3: Implement.**
  - `LiveLines` selects `partner_id`.
  - `journal()` adds the partner (0 for none) to its grouping key and sets `JournalLine.PartnerID`.
  - `InsertEntryLines` takes `@partner_ids::bigint[]` with `NULLIF(…, 0)`.
  - `ReverseEntryLines` copies `partner_id`.
  - `writeEntry` calls `accountFlags` after `ShareAccounts` and refuses a line whose `PartnerID != nil` differs from its account's flag.
  - The ledger query left-joins `partner.partners`.
- [ ] **Step 4: Run `make gen` and `go test ./internal/modules/accounting`.** Expected: PASS. Every existing reconcile test is unchanged.
- [ ] **Step 5: Commit** `feat(accounting): journal lines carry the partner`.

### Task 4: Manual entries with partners and the partner lookup

**Files:**
- Modify: `internal/modules/accounting/journal_types.go`, `journal.go`, `journal_handler.go`, `queries/journal.sql`
- Create: `internal/modules/accounting/partner.go`, `partner_types.go`, `partner_handler.go`, `queries/partner.sql`
- Test: `internal/modules/accounting/journal_test.go`, `partner_test.go`

**Interfaces:**
- Consumes: `accountFlags` (Task 2), `JournalLine.PartnerID` (Task 3).
- Produces:
  - `JournalEntryLine.PartnerID *int64` (json `partner_id,omitempty`).
  - `JournalEntryLineOut` gains `PartnerCode *string` and `PartnerName *string`.
  - `ErrLinePartnerRequired` (422, `journal_line_partner_required`) and `ErrLinePartnerNotAllowed` (422, `journal_line_partner_not_allowed`), both through `lineErr`.
  - `type AccountingPartner struct { ID int64; Code string; Name string }` (json `id`, `code`, `name`) and `type AccountingPartnerList struct { Items []AccountingPartner }`.
  - `func (s *Service) Partners(ctx context.Context, q string) (AccountingPartnerList, error)`: at most 20 rows, matching code, name or tax code with ILIKE, under `accounting.entry.edit` held anywhere.
  - Route `GET /accounting/partners?q=`.

- [ ] **Step 1: Write the failing tests.**
  - `TestJournalEntryPartnerChecks`:
    - A draft with a 131 line without a partner saves fine. Sending it returns `journal_line_partner_required` with `line_no` 1.
    - A draft with a partner on a 411 line saves fine. Sending it returns `journal_line_partner_not_allowed` with that `line_no` (Review Focus 2).
    - With partners on the 131 lines, posting succeeds, and the ledger lines carry the partners.
  - `TestJournalEntryUnknownPartner` (Review Focus 3): saving with `partner_id` -1 returns `platform.ErrNotFound`.
  - `TestAccountingPartnersLookup`: an accountant without `partner.viewer` finds a partner by part of its name. A `viewer` (no `entry.edit`) gets `platform.ErrForbidden`.
- [ ] **Step 2: Run them and see them fail.**
- [ ] **Step 3: Implement.**
  - `prepareJournalEntry` counts the partner ids, as it counts accounts and org units.
  - `checkJournalEntry` names the first bad line, using `accountFlags`.
  - `writeJournalEntryLines` stores `partner_id`.
  - `readJournalEntry` joins `partner.partners` for the code and name.
- [ ] **Step 4: Run `make gen` and `go test ./internal/modules/accounting`.** Expected: PASS, including the `recordtest` contract for `accounting.journal_entry`.
- [ ] **Step 5: Commit** `feat(accounting): manual entry lines name a partner`.

### Task 5: Partner balances and partner ledger (API)

**Files:**
- Modify: `internal/modules/accounting/report.go`, `report_types.go`, `report_handler.go`, `queries/report.sql`
- Test: `internal/modules/accounting/report_test.go`

**Interfaces:**
- Consumes: `reportRange` (existing), and the tracked lines from Tasks 3 and 4.
- Produces:

  ```go
  type PartnerBalanceRow struct { AccountID int64; AccountCode string; PartnerID int64; PartnerCode, PartnerName string; Opening, Debit, Credit, Closing int64 }
  type PartnerBalances struct { Rows []PartnerBalanceRow; TotalOpening, TotalDebit, TotalCredit, TotalClosing int64 }
  func (s *Service) PartnerBalances(ctx context.Context, legalEntity int64, accountID *int64, from, to string) (PartnerBalances, error)
  func (s *Service) PartnerLedger(ctx context.Context, legalEntity, partnerID int64, accountID *int64, from, to string) (Ledger, error)
  ```

  - Routes: `GET /accounting/partner-balances?legal_entity_id&account_id&from&to` and `GET /accounting/partner-ledger?legal_entity_id&partner_id&account_id&from&to`.
  - JSON field names are snake_case.
  - `account_id` is optional, but when present it must be a tracked account. Otherwise the request fails with `ErrAccountNotTracked` (422, `account_not_tracked`).
  - Rows are sorted by account code, then partner code. A row that is zero throughout the range is left out.

- [ ] **Step 1: Write the failing tests.**
  - `TestPartnerBalancesMatchTrialBalance`: post opening entries for A and B on 131 dated before `from`, and movements inside the range. Each row's opening, debit, credit and closing are correct, and the totals equal 131's row in `TrialBalance` for the same range.
  - `TestPartnerBalancesOneRowPerAccount` (Review Focus 5): with A on both 131 and 331 and no account filter, A has two rows.
  - `TestPartnerLedgerRunningBalance`: A's lines in date order, with the opening balance, the running balance, `doc_number`, and a closing balance equal to the balances row.
  - `TestPartnerReportsStayInLegalEntity`: lines of another legal entity never show, and a user without `accounting.view` there gets `platform.ErrForbidden`. Mirror `TestReportsStayInLegalEntity`.
- [ ] **Step 2: Run them and see them fail.**
- [ ] **Step 3: Implement** with two sqlc queries that sum `entry_lines` joined to `entries` and `partner.partners`, filtered by legal entity and date, the same way the trial balance query does.
- [ ] **Step 4: Run `make gen` and `go test ./internal/modules/accounting`.** Expected: PASS.
- [ ] **Step 5: Commit** `feat(accounting): partner balances and partner ledger`.

### Task 6: Screens

Use the `mmerp-ui` and `new-screen` skills.

**Files:**
- Modify: `web/src/accounting/pages/AccountsPage.tsx` (a "tracked by partner" switch, disabled when `has_lines` is true or the account is not postable), `RulesPage.tsx` (a marker on partner kinds), `LedgerPage.tsx` (a partner column when the account is tracked), `components/JournalEntryForm.tsx` and `pages/JournalEntryPage.tsx` (a partner field per line, shown and required when the line's account is tracked), `index.ts`, `routes.tsx`, `hooks/useAccounting.ts`, `keys.ts`, `i18n/{vi,en}/main.json`
- Create: `web/src/accounting/pages/PartnerBalancesPage.tsx`, `web/src/accounting/pages/PartnerLedgerPage.tsx`, `web/src/accounting/components/AccountingPartnerPicker.tsx`
- Test: Vitest only where there is logic (none is expected beyond existing helpers). The screens are proven in Task 7.

**Interfaces:**
- Consumes: the API from Tasks 2 to 5, through the generated `api` in `shared/api/accounting.ts`.
- Produces:
  - Nav entries `accounting.nav.partner_balances` (path `partner-balances`, icon `IconUsers`, permission `accounting.view`). The partner ledger route `partner-ledger` is reached from a balances row; it has no nav entry.
  - `AccountingPartnerPicker` copies the shape of `shared/master/PartnerPicker`, but calls `GET /accounting/partners`.

- [ ] **Step 1: Build the screens.** The page templates come from `shared/ui/page`, and the filters (legal entity, account, range) copy those of `TrialBalancePage`. The i18n text, in both languages:
  - "Công nợ theo đối tác" / "Partner balances"
  - "Sổ chi tiết công nợ" / "Partner ledger"
  - "Theo dõi theo đối tác" / "Tracked by partner"
  - Error texts for the six new error codes.
- [ ] **Step 2: Run `make lint` and `pnpm -C web test`.** Expected: PASS, including the i18n key check.
- [ ] **Step 3: Run the `docs/ui.md` checklist** against both new pages and the changed form.
- [ ] **Step 4: Commit** `feat(web): partner balances, partner ledger and partners on entry lines`.

### Task 7: Seed, end to end, docs

**Files:**
- Modify: `web/e2e/accounting.seed.ts`, `web/e2e/accounting.spec.ts`, `products/accounting.md` (API and screens lists, if it has them)

- [ ] **Step 1: Extend the seed.**
  - Add `131 Phải thu của khách hàng` (asset) and `331 Phải trả cho người bán` (liability) with `by_partner: true`.
  - Add a supplier partner through `POST /api/partners`.
  - Give the opening entry of 2026-07-01 lines for two seeded customers' partners on 131, the supplier on 331, and the balance against 411.
- [ ] **Step 2: Add a Playwright test** `partners are tracked on receivables and show in partner balances and the partner ledger`. It covers exit criteria 1 to 4 of the spec, through the UI:
  - Mark an account as tracked, and see the payroll rule refused.
  - Send an entry without a partner and see `journal_line_partner_required` shown.
  - Post it with partners, and see the partner balances total equal 131 in the trial balance.
  - Open a partner's ledger, cancel the entry, and see the balance return to zero.
- [ ] **Step 3: Verify.** Run `make lint`, `make test` and `make e2e`. Expected: all green. The existing payroll tests in `accounting.spec.ts` must not change (exit criterion 5).
- [ ] **Step 4: Run the seed** with `make reset-db` and `make dev`, then `make seed`, on a free port and a separate database if :8080 is taken (check `ss -ltn` first). Expected: 4 passed.
- [ ] **Step 5: Commit** `test(accounting): partner subledger end to end`. Then open the PR with title `feat(accounting): partner subledger` and a body that follows house style: plain text, the work delivered, the deliberate behaviour changes and the two migrations.
