# 0120. Vietnamese charts, rules and statement forms are templates applied once; closing is a document

Status: accepted · 2026-11-20

## Context

The general ledger is generic ([ADR-0050](./0050-accounting-product-and-general-ledger.md)): no chart, no rules and no reports ship with it, so every tenant builds a chart and maps every business line kind by hand. The seed shows the cost: 338 sits at the top level because the payroll's single `insurance_payable` kind needs a postable account, so 3381 and 3388 become top-level accounts too. Vietnamese companies keep books under one of two regimes. Circular 200/2014 was replaced by Circular 99/2025 from the financial year starting 2026-01-01. Circular 133/2016 stays in force for small and medium companies, and a replacement is in draft. Both regimes require the balance sheet (B01) and the income statement (B02). Accountants close each period by transferring revenue and expense accounts to 911 and the result to 421, and B02 is usually read off those transfers.

## Decision

### Templates

- **A chart template** is a data file embedded in the binary: accounts (code, name, type, parent, tracked by partner), a default mapping rule per business line kind, the closing accounts, and the statement forms. M21 ships two templates, Circular 133 and Circular 99. No template is fetched from the Internet.
- **Applying a template** is a tenant-wide action (`accounting.manage`), allowed only while the tenant has no account. It writes the accounts, the rules for every kind declared at that moment, the closing accounts and the forms as ordinary data, in one transaction. Afterwards nothing links the data to the template: the accountant edits everything, and a newer template version never changes a tenant.
- **One chart per tenant** stays ([ADR-0050](./0050-accounting-product-and-general-ledger.md)): every legal entity of a tenant keeps books under the template the tenant applied.
- A kind declared after the template was applied (a product enabled later) gets no rule and waits under awaiting mapping, as today. The template's rule for it serves as the suggestion on the rules screen.

### Insurance per fund

- HRM writes what is owed to the insurance funds as one kind per fund: `social_insurance_payable`, `health_insurance_payable`, `unemployment_insurance_payable` and `union_fee_payable`, each the employee and employer shares together, by legal entity. Each fund has its own account, and the account numbers differ between the regimes (the unemployment fund is 3385 under Circular 133).
- `employer_insurance_expense` stays one kind: a rule maps a kind to one account, and every fund's expense lands on the same expense account.
- `insurance_payable` stays a known kind for payrolls posted before this change; new payrolls no longer write it.

### Closing entries

- **A closing entry** (`accounting.closing`, vi: "Kết chuyển", prefix `KC`) is a document per legal entity, dated the last day of the period it closes, with the usual lifecycle and `recordtest`. Posting computes the closing balance at its date of every postable `revenue` and `expense` account and writes one ledger entry. That entry moves each balance to the result account (911), then moves 911's net to the retained earnings account (4212 in both templates). Cancelling writes the reversal.
- The closing accounts (result and retained earnings) are tenant data set by the template, editable under `accounting.manage`. Posting is refused while either is unset.
- A closing moves whatever balance is left at its date, so a second closing of the same date, after a backdated document, moves only the remainder. Periods are not fixed: monthly, quarterly or yearly closings are all valid.
- Closing never blocks the period lock, and the period lock never requires a closing.
- Moving the year's profit from 4212 to 4211 at the start of a new year stays a manual entry.

### Statements

- **A statement form** (B01, B02) is data: ordered rows, each with a code (the "mã số" on the official form), a label and one formula. A formula is a sum of terms over account code prefixes, or a sum of other rows. A term reads one of the following, signed:
  - the net balance at a date;
  - the debit balances only or the credit balances only at a date, summed per partner on tracked accounts (131 debit balances are a receivable, credit balances an advance from the customer);
  - the debit or credit turnover over a date range, either excluding closing entries or limited to them.
- B01 is computed at a date, B02 over a date range, per legal entity, on screen with an Excel export. B02 rows read the closing entries' turnover against 911, as accountants do. While a revenue or expense account still has a balance in the range, the statement says the period is not closed.
- Short-term and long-term splits that depend on due dates read everything as short-term, because the ledger carries no due dates.

## Rejected alternatives

- **Templates linked to the tenant and upgraded in place.** A template upgrade would rewrite accounts and rules that the accountant has edited, and every edit would conflict with the next release. A new regime is applied by an accountant, not by a deploy.
- **Applying a template over an existing chart.** Merging codes, names and parents into a chart that already has lines has no safe answer. Tenants with a chart keep it.
- **One template, the accountant converts.** The two regimes differ in accounts (611 and 631 exist only under Circular 133, 521 only under Circular 99) and in forms. Making every Circular 133 tenant rebuild the chart is the cost this milestone removes.
- **Statements that compute profit on the fly, no closing entries.** Simpler, but 911 would never show in the ledger, B01 would read a 421 that is not the one on the trial balance, and accountants check one against the other.
- **Closing as automatic ledger rows without a document.** A closing has no source document to take a number from, needs the period lock and needs to be cancellable by the accountant. A document gives all three.
- **Statement forms in Go.** Forms change with each circular. As data, a corrected form is a template change and an accountant's fix, not a release.
- **VAT declarations (01/GTGT), CIT and HTKK export.** The ledger does not carry VAT per invoice or per supplier, and the declarations need data that this milestone does not add. They are a later milestone.

## Consequences

- `accounting` gains the template files, the apply action, the closing accounts setting, the form tables, the `accounting.closing` record type and the statements. The generic ledger is unchanged for tenants that never apply a template.
- A business line kind added by a later milestone also needs its row in each template that maps it, in the same change.
- HRM's payroll lines change kinds. No tenant is in production, so the dev and demo data are reseeded, and the seed applies the Circular 133 template instead of building a chart by hand.
- The forms are only as exact as their formulas. The milestone checks B01 and B02 against a worked example per template, by hand.
- The draft replacement of Circular 133 will be a third template when it takes effect. Tenants switch by starting a new tenant or by editing their chart, never automatically.
