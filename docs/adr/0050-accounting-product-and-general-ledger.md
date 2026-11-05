# 0050. Accounting is the third product: a minimal general ledger over business lines

Status: accepted · 2026-11-05

## Context

Posted documents already record what they mean economically: a payroll writes business lines (`posting.lines`) whether or not anyone keeps books ([ADR-0003](./0003-business-lines-are-data.md), [documents.md](../documents.md#posting)). Nothing turns those lines into a general ledger. Accounting was out of scope until there was a real need; the product owner now wants it as the third product, built generic first and localised later. This decision covers the parts that are the same everywhere. The Vietnamese layer (Circular 133/200 charts and default rules, Ministry of Finance reports, VAT and CIT) is later data on top and is not decided here. Design: `docs/superpowers/specs/2026-11-05-accounting-ledger-design.md`.

## Decision

### Module

- **`internal/modules/accounting`**, layer `modules`, schema `accounting`, product `accounting`, enabled through `PRODUCTS`. It declares no product dependency and never imports `hrm` or `sales`. SQL reads `posting.lines` and `posting.recordings` (shared tier).
- It owns: the chart of accounts, mapping rules (line kind to account and side), the journal entry header and lines, one tracking row per source document, and the manual entry tables. It has its own Postgres schema and is the only writer there.
- Frontend area `web/src/accounting` with its own client; a tile on the home page.
- Namespace `accounting.` ([ADR-0034](./0034-record-types-declare-permissions.md), [ADR-0039](./0039-shared-master-data.md)). Permissions: `accounting.view` and `accounting.entry.edit` (scoped to legal entity, like documents), `accounting.post` (post pending documents), `accounting.manage` (chart, rules, start date; tenant-wide), `accounting.approval.manage` (approval rules for manual entries, inside the area, as `sales.approval.manage`; tenant-wide). Roles: `accounting.viewer` {view}, `accounting.accountant` {view, entry.edit, post}, `accounting.admin` {all of them}.
- Record type `accounting.journal_entry` (manual journal entry, prefix `PK`, numbered per legal entity and year) with the usual lifecycle, approval with the total as approval field, attachments, discussion, audit; it must pass `recordtest`. No printing.
- No posting line kinds of its own (it reads lines, never writes them). No personal or sensitive fields: business lines already aggregate per-person amounts and the ledger must not undo that.

### Ledger

- Every journal entry, automatic or manual, is a row in one entry header table (legal entity, date, source document ref, the entry it reverses) and one entry line table (account, org unit, debit, credit, `bigint` minor units, one currency). Written once, never edited or deleted. An entry whose debits differ from credits cannot be written.
- **Automatic entries and reversals are ledger rows, not `record` documents.** They take no number; the ledger shows the source document's number. Only manual entries are documents, so the job never calls `record.Create`. Its lock order is the legal entity's period-lock row (`FOR SHARE`), the source document's row (`FOR UPDATE`), the legal entity's start row (`FOR SHARE`), the tracking row (`FOR UPDATE`), then the accounts it uses (`FOR SHARE`, sorted by id).
- **One chart per tenant**, shared by all legal entities; accounts form a tree and only leaves (postable accounts) take lines. No chart is hardcoded in Go. An account with lines cannot be deleted, retyped, moved to another parent (that would change the rollups of balances already reported) or given children; it can be deactivated.
- **Mapping rules**: one rule per business line kind, to one account and one side; a negative amount goes on the opposite side; the line's org unit is carried onto the journal line. Mapped lines are summed per account, side, org unit and kind; a document still past the entry bounds (line count or line amount) after that is flagged too large, not retried. `posting` exposes the list of known kinds. Changing a rule never changes existing entries. Rules per legal entity or org unit are not built.

### Automatic entries

- The `posting` hook `OnChanged` is implemented in `accounting` and wired in `internal/app`. It only **enqueues** a system job carrying `(doc_type, doc_id)` in the same transaction. Because `platform.Enqueue` applies `ProductGate` (class write), the hook first checks that the accounting product is enabled and does nothing otherwise; without that check, HR's payroll posting would fail whenever accounting is off.
- The worker reconciles as documented in [documents.md](../documents.md#posting-jobs-reconcile-do-not-replay); this ADR does not change that. The tracking row records one of **posted**, **awaiting mapping** (kinds without a rule are listed) or **unbalanced**. The last two are not job failures and do not retry.
- **HR is never blocked** by accounting. A missing rule only puts the document under awaiting mapping; the accountant runs "post pending" (one job per document, one transaction each) once the rule exists.
- A document with business lines dated on or after the legal entity's start date and no tracking row is **pending**; this also covers documents posted while accounting was disabled. There is no separate backfill job.

### Changes to existing decisions

1. **Reversal instead of removal.** When lines are voided the worker writes a reversing entry dated with the original document's date and linked to the entry it reverses; the original stays. [documents.md](../documents.md#posting) said entries are "removed". Balances net the same, and the audit trail is kept. A document voided before it ever reached the ledger produces nothing.
2. **Defaults move to the localisation pack.** Default account mapping following TT200/TT133 is no longer part of `accounting`; the generic core ships no accounts and no rules.
3. **`record.BeforeLock(legal entity, date)` is added** (planned since ADR-0003). `accounting` implements it: moving the lock date is refused while the part of the period being locked holds pending, awaiting-mapping or unbalanced documents dated on or after the start date; the error lists them. It checks only while the accounting product is enabled, so a tenant that switches accounting off can still lock periods; those documents become pending again when it is switched back on.

### Start of use

- Each legal entity has an **accounting start date**, held in an `accounting` table, not as a `setting` legal-entity key. Documents dated before it never reach the ledger; opening balances are a manual entry.
- No start date means accounting is not in use for that legal entity: nothing is pending, the job writes nothing, `BeforeLock` lets every lock through, manual entries are refused (`accounting_not_started`).
- The start date can be set or changed **only while the legal entity has no ledger entry**. After the first entry it is fixed, which stops moving the date earlier from opening closed periods.
- A start date on or before the legal entity's lock date is refused (`start_date_in_locked_period`) while a document with business lines is dated between it and the lock date, since those documents would become pending inside the locked period. A start date after the lock date, or one with no business lines in the locked part, is allowed. Tenants enabling accounting on locked history set the start date after the lock date and enter opening balances as a manual journal entry. `SetStart` takes the legal entity's period-lock row (`FOR SHARE`) before the start row (`FOR UPDATE`).
- So the only way a document's **first** posting or a reversal lands in a locked period is a period locked while accounting was off ([documents.md](../documents.md#period-lock-and-re-posting)): with accounting on, `BeforeLock` keeps pending documents out of locked periods, and the start date cannot bring locked ones in.

### Manual entries

Date, legal entity, description, at least two lines (postable active account, a positive debit or credit, optional org unit and description). Posting requires debits equal to credits (`journal_entry_unbalanced`) and writes the entry to the ledger in the same transaction; cancelling writes the reversal.

### Not in this decision

The Vietnamese layer; financial statements and period closing entries; receivables and payables per party; multiple currencies; re-posting a date range after rule changes (the accountant corrects with manual entries); business lines from Sales; budgets, cost centres beyond org units, fixed assets, bank reconciliation.

## Rejected alternatives

- **Removing entries when lines are voided.** It erases the audit trail, and removal would have to write into locked periods.
- **The hook writing entries in the payroll's transaction.** A missing rule or a ledger error would block HR from posting; the hook only enqueues.
- **Start date as a `setting` legal-entity key.** Setting keys are enumerations and cannot carry the guard "fixed after the first entry".
- **Automatic entries as `record` documents.** They would need numbers, lock-order changes and period-lock handling for what is a derived row; the source document's number is what accountants expect.

## Consequences

- With accounting off, nothing changes for other products: the hook does nothing and business lines are still written.
- The accountant cannot close a period until rules are complete; only the accountant waits.
- A wrong rule is fixed with manual entries until re-posting exists.
- Sales, once it writes business lines, is picked up by the same mechanism with no change here.
