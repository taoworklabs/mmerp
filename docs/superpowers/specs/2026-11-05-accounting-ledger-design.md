# Accounting: a minimal general ledger

Date: 2026-11-05 · Status: approved by the product owner, 2026-11-05

## Intent

mmerp already records what each posted document means economically: a payroll writes **business lines** (`posting.lines`: salary expense, employer insurance expense, salary payable, insurance payable, PIT payable) whether or not anyone keeps books. Nothing turns those lines into a general ledger yet. Accounting has been "out of scope until there is a real need" (`docs/roadmap.md`).

The product owner wants accounting as the **third product** and wants it built **generic first, localised later**. This branch builds the parts that are the same everywhere: a chart of accounts held as data, double-entry journal entries, automatic journal entries from business lines through configurable mapping rules, manual journal entries, a general ledger and a trial balance. The Vietnamese layer comes later as data loaded on top of the generic core: the Circular 133/200 charts, the Ministry of Finance report forms and tax. That layer is not designed here.

The product owner has no accounting background, so the design follows the textbook model and the decisions already recorded in `docs/documents.md` (section "Posting") and ADR-0003, which describe how accounting plugs into `posting`. Calls marked *(my call)* were made while writing this document, not by the product owner.

## What "done" looks like

Using `make seed` data with `PRODUCTS=hrm,sales,accounting`:

1. The accountant opens the chart of accounts and sees a small sample chart (created by the seed through the API).
2. They set a mapping rule for each of the five payroll line kinds.
3. HR posts a payroll. Shortly after (a job), the general ledger shows a balanced journal entry for it, on the right accounts, dated with the payroll's date and split by org unit.
4. HR cancels that payroll. The ledger shows a reversing entry; the original entry stays.
5. A payroll posted while a kind has no rule shows up under "awaiting mapping"; HR was never blocked. Once the rule exists, the accountant runs "post pending" and the entry appears.
6. The accountant enters, posts and cancels a manual journal entry.
7. The trial balance for a period shows equal debit and credit totals.

## Goals

### Product and module

- Product name `accounting`, enabled through `PRODUCTS` like `hrm` and `sales`; `ProductGate` governs it like any product (disabled: readable and exportable, no writes, no jobs).
- Backend module `internal/modules/accounting`, schema `accounting`. It reads `posting.lines` and `posting.recordings` with SQL (shared tier, allowed) and never imports `hrm` or `sales`.
- Frontend area `web/src/accounting` with its own client `shared/api/accounting.ts`, a product tile on the home page, a grouped menu.
- New product description `products/accounting.md`, new section in `CONTEXT.md`, ADR-0050.

### Chart of accounts

- One chart per tenant, shared by every legal entity *(my call; most SMB tenants have one company, and the Vietnamese charts are national, not per company)*.
- An account: code (text, unique, e.g. `334` or `6421`), name, type (`asset`, `liability`, `equity`, `revenue`, `expense`), optional parent, active flag.
- Only accounts without children take journal lines (**postable** accounts). Parent accounts exist to roll up balances *(my call; standard practice)*.
- An account with journal lines cannot be deleted, have its type changed, or gain children; it can be deactivated, which stops new lines on it but keeps its history *(my call)*.
- No chart is hardcoded in Go. The demo seed creates a small chart through the API. A future localisation pack loads a full chart the same way.

### Mapping rules

- One rule per business line kind: kind → account + side (debit or credit). Example: `salary_expense` → expense account, debit; `salary_payable` → payable account, credit.
- A business line has one signed amount, so each kind maps to **one** account and **one** side. A negative amount goes on the opposite side.
- The org unit of the business line is carried onto the journal line.
- `posting` exposes the list of known kinds so the rules screen can list them, with labels through i18n *(my call)*.
- Changing a rule never changes existing journal entries (as `docs/documents.md` already says).
- Rules per legal entity or per org unit are not built *(my call)*.

### Automatic journal entries

This follows `docs/documents.md` "Posting jobs: reconcile, do not replay" and ADR-0003; nothing here re-decides them.

- `accounting` implements `posting.Hooks.OnChanged` (wired in `internal/app`) by enqueueing a system job carrying only `(doc_type, doc_id)`, in the same transaction.
- The worker brings the ledger in line with the current state of that document's business lines, taking locks in the documented order (legal-entity period-lock row `FOR SHARE`, then the document's tracking row `FOR UPDATE`). Running it any number of times, in any order, gives the same result.
- A tracking row per source document records where it stands: **posted** (its entry matches the lines), **awaiting mapping** (some kind has no rule; the missing kinds are listed), or **unbalanced** (the mapped lines do not balance; should not happen, but is shown rather than hidden). Awaiting and unbalanced are not job failures: the job finishes and does not retry.
- **HR is never blocked** by accounting (product owner's decision). A missing rule only puts the document under "awaiting mapping".
- "Post pending" (accountant action) enqueues one job per pending document, one transaction each.
- **Voided lines produce a reversing entry** dated with the original document's date, linked to the entry it reverses. The original is never deleted *(my call: `docs/documents.md` currently says the entries are "removed"; reversing keeps the audit trail and nets to the same balances. ADR-0050 records the change and `docs/documents.md` is updated)*. A document voided before it was ever posted to the ledger produces nothing.

### Start of use and enabling later

- Each legal entity has an **accounting start date** (a legal-entity setting). Documents dated before it never reach the ledger; opening balances are entered as a manual journal entry *(my call)*.
- **No start date: accounting is not in use for that legal entity.** Nothing is pending, the job writes nothing, `BeforeLock` lets every lock through, manual entries are refused (`accounting_not_started`) *(my call)*.
- **The start date can be set or changed only while the legal entity has no ledger entry.** After the first entry it is fixed. This keeps the locked-period exception below to the first run and stops moving the date earlier from opening a way into closed periods *(my call)*.
- A document with business lines dated on or after the start date and no tracking row counts as **pending**. So documents posted while accounting was disabled, or before the hook existed, show up in "awaiting posting" and go in through "post pending". No separate backfill job.
- Entries the ledger lacks for a document (its **first** posting, or the reversal of a document cancelled while accounting was off) may land in a locked period: they make the ledger match the document's true state, the exception `docs/documents.md` allows for enabling accounting later. With accounting on, `BeforeLock` prevents this state, so it only arises from a period locked while accounting was off.

### Period lock

- `record` gains the `BeforeLock(legal entity, date)` hook already planned in `docs/documents.md`. `accounting` implements it: moving the lock date is refused while the period being locked holds pending, awaiting-mapping or unbalanced documents dated on or after the start date. The error lists them.
- Consequence: the accountant cannot close a period until its rules are complete. Only the accountant waits; HR is never blocked.
- `BeforeLock` checks only when the `accounting` product is enabled. A tenant that switches accounting off can still lock periods; the documents of that time become pending again when it is switched back on, and their first posting is the same first-run exception *(my call)*.

### Manual journal entry

- A document type `accounting.journal_entry` (vi: "Phiếu kế toán", number prefix `PK` *(my call)*), numbered per legal entity and year, with the usual lifecycle (draft → posted → cancelled), period lock, attachments, discussion and audit through `record`. It must pass `recordtest`.
- Fields: date, legal entity, description, and at least two lines. Each line: account (postable, active), debit **or** credit amount (one of them, positive), optional org unit, optional line description.
- Posting requires debits to equal credits (`journal_entry_unbalanced`). Posting writes the entry to the ledger in the same transaction; cancelling writes its reversal.
- Approval: supported through `record`'s gate like other documents, with the total as an approval field, so a tenant can require approval above an amount *(my call)*.
- Printing is not built.

### Ledger storage

- Every journal entry, automatic or manual, lands in the same two tables: an entry header (legal entity, date, source document ref, the entry it reverses if any) and entry lines (account, org unit, debit, credit). Entries are written once and never edited.
- **Automatic entries and reversals are ledger rows, not `record` documents.** They take no number of their own; the ledger shows the source document's number (the payroll's, or the manual entry's `PK` number for its reversal). Only manual entries are documents. So the job never calls `record.Create` and its lock order stays period-lock row, then tracking row *(my call; Vietnamese software usually shows the source voucher's number)*.
- Amounts are `bigint` in the currency's minor unit; one currency (the tenant's). No float anywhere.

### Screens

- **Chart of accounts**: tree list, create, edit, deactivate.
- **Mapping rules**: one row per known business line kind, pick account and side.
- **Awaiting posting**: documents pending, awaiting mapping or unbalanced, with the reason and a "post pending" action; each row links to its source document through the record type's `path`.
- **Journal entries**: list and form for manual entries; automatic entries are viewable, not editable.
- **General ledger**: pick legal entity, account, date range; shows opening balance, each line (date, document, description, org unit, debit, credit), running balance, closing balance. A parent account shows its children's lines.
- **Trial balance**: pick legal entity and date range; per account: opening balance, debit, credit, closing balance, parents rolled up; totals row where debits equal credits.
- Every screen follows `docs/ui.md` and the `mmerp-ui` skill. Excel export of the ledger and the trial balance through `dataio` if it fits cheaply; otherwise leave it out *(builder's call)*.

### Permissions

- Namespace `accounting.`. Viewing the ledger, trial balance and entries, and entering manual entries, are scoped to legal entity like other documents. The chart and the rules are tenant-wide.
- Roles *(my call)*: `accounting.viewer` (read), `accounting.accountant` (read, manual entries, post pending), `accounting.admin` (also chart, rules, start date). Exact permission names are the builder's, following ADR-0034 and ADR-0039.

## Non-goals

- The Vietnamese layer: the Circular 133/200 charts, default rules following them, Ministry of Finance reports (B01, B02…), VAT and CIT. `docs/documents.md` says defaults follow TT200/TT133; that moves to the localisation pack.
- Financial statements (balance sheet, income statement), period closing entries, profit carry-forward.
- Receivables and payables per customer, supplier or employee (ADR-0007 already says no per-employee payables).
- Multiple currencies, exchange rates.
- Re-posting a date range after rules change (`docs/documents.md` mentions it). If a rule was wrong, the accountant fixes the ledger with manual entries for now *(my call)*.
- Business lines from Sales. Sales writes none today; when it does, the same mechanism picks them up.
- Budgets, cost centres beyond org units, fixed assets, bank reconciliation.

## Anti-goals

These would make the work a failure even if it ran:

- HR (or any other product) blocked by anything accounting does or lacks.
- A ledger that can hold an entry whose debits differ from its credits.
- A posted entry edited or deleted in place.
- Any account number written in Go code.
- Per-person salary amounts reaching the ledger (business lines already aggregate them; the ledger must not undo that).
- The ledger diverging from business lines silently: every document that has not reached the ledger is visible on "awaiting posting" and blocks the period lock.

## Constraints

Everything in `AGENTS.md` applies. The ones that bite here:

- One Go binary and Postgres; the worker is a River job enqueued in the same transaction.
- Services use the data engine and `platform.InTx`; one transaction writes one document; bulk actions are jobs, one document per transaction.
- Lock order as documented; the tracking row sits with "other module-owned rows".
- Money is `int64` in the minor unit.
- New ADR numbers come from the range 0050 to 0059 only.
- Do not run `make e2e` or `make dev` without asking: other worktrees share the ports and the e2e database.

## Tests that prove it

- Worker: idempotent (running twice changes nothing), order-independent (void job before post job ends in the same state), awaiting mapping does not retry, negative amounts flip side, org unit carried.
- A Go test from payroll to ledger: post a payroll and get a balanced entry; cancel it and get a reversal; the trial balance nets to zero.
- `BeforeLock` refuses a lock while documents are pending and allows it once they are posted.
- First posting into a locked period is allowed; so is a reversal, which only arises when the document was cancelled and the period locked while accounting was off.
- `recordtest` for `accounting.journal_entry`; unbalanced entry refused.
- Frontend: Vitest for the screens' logic; one Playwright spec covering the "done" scenario, run only when the product owner says so.

## Left to the builder

- Table and column names, the tracking table's exact shape, indexes.
- Permission and role names within the scheme above; i18n keys and wording (vi and en).
- How the ledger query computes opening balances (on the fly is fine at this size).
- Whether the seed's sample chart sits in `web/e2e/` next to the other seeds.
- Error codes beyond those named here.
- Splitting the work into PRs.
