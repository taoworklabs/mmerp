# Vietnamese accounting (M21)

Decision record: [ADR-0120](../../adr/0120-vietnamese-accounting-layer.md). This document is what a builder plans from.

## Why

The ledger ships no chart, no rules and no statements, so a tenant builds 60 to 100 accounts and maps every business line kind by hand before the first entry, and still gets no balance sheet or income statement. M21 ships the two Vietnamese regimes as templates, closes periods the way accountants do, and computes B01 and B02.

## Goals

- Chart templates for Circular 133 and Circular 99, applied once to a tenant without accounts.
- Payroll insurance owed, one kind per fund.
- Closing entries, `accounting.closing`.
- Statement forms as data; B01 and B02 per legal entity, on screen and in Excel.

## Non-goals

- VAT declarations, CIT, HTKK XML, cash flow statement (B03), notes (B09).
- Due dates (everything is short-term), profit carry-forward (4212 to 4211 stays a manual entry).
- Re-applying or upgrading a template; converting a tenant from one regime to the other.

## Design

### Templates (`internal/modules/accounting/templates`)

- One JSON file per template, embedded with `go:embed`: `vn-tt133.json`, `vn-tt99.json`. Fields: `code`, `name` (vi and en), `accounts` (code, name, type, parent code, `by_partner`), `rules` (kind, account code, side), `closing` (result and retained earnings account codes), `forms` (`balance_sheet`, `income_statement`: rows of code, label, formula, `bold`).
- Sources: the account lists in Appendix II of Circular 99/2025 and the account list of Circular 133/2016, the official B01/B02 forms of each. 911 is typed `equity`, so closing never closes it. 131 and 331 are `by_partner`; no other account is, so the business lines that name a partner (sales and purchasing) find tracked accounts.
- A test loads every template and checks it: codes unique, parents exist, only leaves are mapped, every kind in `posting.Kinds()` has a rule (so a later kind cannot be forgotten) on an account tracked by partner exactly when the kind names a partner, closing accounts are leaves, every formula parses, every row code it references exists.
- `GET /accounting/templates`: code and name of each. `POST /accounting/templates/{code}/apply` (`accounting.manage`): under the chart lock (`LockChart`), refused with `chart_not_empty` when any account exists; inserts accounts parents first, the rules for kinds declared now, the closing accounts and the form rows; stores the template code in `accounting.chart_template`; one audit row.
- The rules screen shows, for a kind without a rule, the applied template's account when that code exists in the chart, with one click to save it.

### Insurance per fund (`shared/posting`, `modules/hrm`)

- `posting` declares `social_insurance_payable`, `health_insurance_payable`, `unemployment_insurance_payable`, `union_fee_payable`. `insurance_payable` stays declared.
- `postingLines` writes the four per legal entity: social = employee + employer social, health = employee + employer health, unemployment = employee + employer unemployment, union fee = employer union. Zero lines are dropped as today.
- Template rules (credit): Circular 133: 3383, 3384, 3385, 3382. Circular 99: per Appendix II (check the unemployment fund account).

### Closing entries

- Tables `accounting.closings (id → record.documents, description)` and `accounting.closing_accounts` (one row: `result_account_id`, `retained_account_id`, both NOT NULL). `PUT /accounting/closing-accounts` (`accounting.manage`), leaves only.
- Record type `accounting.closing`, prefix `KC`, numbered per legal entity and year, permissions as manual entries (`view`, `entry.edit`), no approval fields, attachments and discussion, no printing. `recordtest` passes.
- Draft: date, legal entity, description. Sending and posting refuse `accounting_not_started` (as manual entries) and `closing_accounts_unset`.
- Posting, under the same locks as a manual entry: one query sums every postable `revenue` and `expense` account's balance up to the date, per account and org unit (closing entries included, so only what is left moves). For each non-zero balance, a line clearing it and the opposite line on the result account with the same org unit; then per org unit the result account's net moves to retained earnings. Nothing left: `closing_nothing_to_close`. Past `MaxLines`: `closing_too_large`. Inactive accounts with a balance: `account_not_postable` naming the account. Entry via `writeEntry`, source `accounting.closing`.
- Cancelling reverses the entry (`reverseEntry`). A draft shows a preview of the lines it would write at its date (`GET /accounting/closings/{id}/preview`).

### Statement forms and statements

- Table `accounting.statement_rows (form, line_no, code, label, formula, bold)`, `form IN ('balance_sheet', 'income_statement')`, primary key `(form, line_no)`, unique `(form, code)`. Edited row by row under `accounting.manage`; a formula is parsed on save (`statement_formula_invalid`, params `code`, `position`).
- Formula: terms joined by `+` and `-`, each a function of comma-separated account prefixes or row codes:
  - `B(…)`: balance at the date, debit minus credit;
  - `DB(…)`, `CB(…)`: sum of debit balances, sum of credit balances (positive), taken per partner on tracked accounts and per postable account otherwise;
  - `DT(…)`, `CT(…)`: debit, credit turnover over the range, excluding closing entries;
  - `DK(…)`, `CK(…)`: debit, credit turnover over the range, closing entries only;
  - `R(…)`: sum of other rows of the same form (no cycles, checked on save).
- B01: at a date, with the same figures at the end of the previous year as the comparative column. B02: over a range, with the same range a year earlier. Per legal entity, `accounting.view` scoped to legal entity. Inputs read with fixed sqlc queries (balances per account and partner at a date, turnovers per account split by closing or not); formulas are evaluated in Go.
- `not_closed: true` when a revenue or expense account has a non-zero balance at the end date. Totals are rows of the form like any other; the worked-example tests check that they balance.
- Export: `dataio` exports `accounting.report.balance_sheet` and `accounting.report.income_statement` (xlsx: label, code, current, comparative).

### Frontend (`web/src/accounting`)

- Chart page: when empty, a choice of template and an apply button (confirm dialog).
- Rules page: the template suggestion per unmapped kind.
- Configuration: closing accounts (two account pickers) and statement forms (one table per form, row editing).
- Closing entries: list, new, document page with the preview while draft and the entry once posted.
- Balance sheet and income statement pages: legal entity, date or range, rows with the comparative column, bold rows, the not-closed notice, export.

### Seed and tests

- `accounting.seed.ts` applies the Circular 133 template instead of building its chart; the rules come with it. The purchasing e2e keeps its own chart (it runs alone and sets rules tenant-wide).
- Go: template checks; apply (empty and non-empty chart, rules only for declared kinds); payroll lines per fund; closing (zeroes revenue and expense, 911 nets to zero, second closing moves the remainder, cancel reverses, refusals); formula parser; B01 and B02 against a small hand-computed ledger per template; `recordtest` for `accounting.closing`.
- Playwright: apply a template, post a closing, open B01 and B02.

## Exit criteria (roadmap M21)

With `make seed` data and `PRODUCTS=hrm,sales,purchase,accounting`:

1. On a tenant without accounts, applying the Circular 133 template gives its chart, a rule for every declared kind and the closing accounts; applying it again, or with any account present, is refused. The same holds for Circular 99 on another tenant.
2. A payroll posts its insurance to one account per fund (3382, 3383, 3384, 3385 under Circular 133); payrolls posted before keep `insurance_payable`.
3. A closing entry at month end leaves every revenue and expense account at zero and 911 at zero, and moves the result to 4212; a document backdated into the month and a second closing move only the remainder; cancelling a closing reverses it. `recordtest` passes for `accounting.closing`.
4. B01 at month end balances (total assets equal total sources) and its 421 equals the trial balance; a customer with an advance shows under liabilities, not as a negative receivable.
5. B02 for the month matches a worked example per template, checked by hand; before the closing it says the period is not closed.
6. Both statements export to Excel. The seed applies the Circular 133 template instead of its hand-built chart.

## Left to the builder

Slicing and order; the exact template contents (from the official texts, cited in the commit); how the form editor looks; test names.
