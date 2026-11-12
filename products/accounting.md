# Accounting product

A minimal general ledger, generic first: the parts that are the same in every country. Accounting is the third product. Hard-to-reverse decisions are in [ADR-0050](../docs/adr/0050-accounting-product-and-general-ledger.md); terminology is in the Accounting section of [CONTEXT.md](../CONTEXT.md#accounting).

## Scope

| In the first release | Not yet |
| --- | --- |
| Chart of accounts as data (one per tenant), postable accounts | The Vietnamese layer: Circular 133/200 charts, default rules, Ministry of Finance reports (B01, B02...), VAT and CIT |
| Mapping rules from business line kind to account and side | Financial statements, period closing entries, profit carry-forward |
| Automatic journal entries from `posting.lines`, reversing entries when lines are voided | Open-item matching, due dates and aging; balances per employee |
| Awaiting posting: pending, awaiting-mapping and unbalanced documents, "post pending" | Multiple currencies, exchange rates |
| Manual journal entries with approval | Re-posting a date range after rules change (fix with manual entries) |
| General ledger and trial balance per legal entity and date range | Business lines from Sales, budgets, cost centres beyond org units, fixed assets, bank reconciliation |
| Accounting start date per legal entity, refusal to lock a period with unposted documents | Printing manual entries |
| Accounts tracked by partner, partner balances and partner ledger (M16, [ADR-0051](../docs/adr/0051-business-lines-name-a-partner.md)) | |

The Vietnamese layer is mostly data (charts, default rules, report forms), but it also needs HRM to split its insurance business line kinds per fund: HRM folds BHXH, BHYT, BHTN and the union fee into `employer_insurance_expense` and `insurance_payable`, while Circular 133/200 books them to 3382, 3383, 3384 and 3386 separately, which needs new posting kinds in HRM (a code change). Splitting them is cheapest before any tenant has a ledger.

## Module

A single module `internal/modules/accounting`, schema `accounting`, product `accounting`. Depends on no other product. Reads `posting.lines` and `posting.recordings` with SQL; never imports `hrm` or `sales`. Implements `posting.Hooks.OnChanged` (enqueue only, and only while the product is enabled) and `record.BeforeLock`, both wired in `internal/app`.

## Record types

| Type code | Kind | `posted` means | Date for period lock | Approval fields |
| --- | --- | --- | --- | --- |
| `accounting.journal_entry` | document `PK` (vi: "Phiếu kế toán") | Entry written to the ledger | Entry date | `amount` (the total) |

Automatic entries and reversals are ledger rows, not documents: they have no number of their own and show the source document's number.

## Behaviour

- **Chart.** Code (unique text), name, type (`asset`, `liability`, `equity`, `revenue`, `expense`), optional parent, active flag. Only accounts without children take lines. An account with lines cannot be deleted, retyped or gain children; one with lines on it or below it cannot move to another parent. It can be deactivated.
- **Rules.** One per business line kind: account and side. A negative amount goes on the opposite side; the org unit is carried onto the journal line.
- **Automatic entries.** A job per `(doc_type, doc_id)` reconciles the ledger with the document's current business lines; any order and any number of runs give the same result. HR is never blocked: a missing rule puts the document under awaiting mapping.
- **Start date.** Per legal entity; no date means accounting is not in use there. It can be set or changed only while the legal entity has no ledger entry. A start date on or before the legal entity's lock date is refused (`start_date_in_locked_period`) while a document with business lines is dated between it and the lock date, since those documents would become pending inside the locked period; tenants enabling accounting on locked history set it after the lock date and enter opening balances as a manual journal entry. Documents dated before it never reach the ledger.
- **Period lock.** Locking is refused while the period holds pending, awaiting-mapping or unbalanced documents dated on or after the start date. The only way a document's first posting or a reversal lands in a locked period is a period locked while accounting was off.
- **Manual entry.** Date, legal entity, description, at least two lines (postable active account, one positive debit or credit, optional org unit and description). Posting needs debits equal to credits.

## Permissions

| Role | Permissions | Granted |
| --- | --- | --- |
| `accounting.viewer` | `accounting.view` | At a legal entity or tenant-wide |
| `accounting.accountant` | `accounting.view`, `accounting.entry.edit`, `accounting.post` | At a legal entity or tenant-wide |
| `accounting.admin` | all of the above, `accounting.manage`, `accounting.approval.manage` | Tenant-wide only |

Viewing the ledger, trial balance and entries, and entering manual entries, are scoped to legal entity. The chart, the rules and the start date are tenant-wide (`accounting.manage`). With accounting disabled everything stays readable; every write action and job disappears.

## Personal data

No personal or sensitive data. Business lines already aggregate per-person amounts; the ledger carries only account, org unit and amount.

## Frontend

Area `web/src/accounting`, menu: general ledger, trial balance, journal entries, awaiting posting; configuration: chart of accounts, mapping rules, approval rules. Screens: chart (tree), mapping rules (one row per known kind), awaiting posting (reason, link to the source document, "post pending"), journal entries (manual form; automatic entries viewable only), general ledger (opening balance, lines, running balance, closing balance; a parent shows its children), trial balance (opening, debit, credit, closing, parents rolled up, equal totals).

## Proving flow

Seeded sample chart and rules for the five payroll kinds, a payroll posted, a balanced entry appears split by org unit, the payroll cancelled and a reversing entry appears, a payroll posted before its rule exists waits under awaiting posting and enters after "post pending", a manual entry entered, posted and cancelled, and the trial balance totals are equal.
