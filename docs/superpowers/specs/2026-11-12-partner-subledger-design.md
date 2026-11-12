# Partner subledger (M16)

Decision record: [ADR-0051](../../adr/0051-business-lines-name-a-partner.md). This document is what a builder plans from.

## Why

mmerp is to close the order-to-cash and procure-to-pay loops: Sales invoices and receipts (M17), then Purchasing (M18), then full stock (M19). Both loops end in what a party owes or is owed. Today a business line cannot name a party, and the ledger cannot hold a balance per party. M16 builds that foundation once, generic, so that M17 and M18 only add kinds and documents. Nothing writes partner lines yet. M16 is proven through manual journal entries: opening balances per customer and supplier.

## Goals

- Business lines may name a partner. Each kind declares whether it does, and `posting` enforces that.
- Accounts may be tracked by partner. Rules, automatic entries and manual entries respect the flag.
- Two reports: partner balances and partner ledger.
- Seed and Playwright show the flow end to end.

## Non-goals

Open-item matching, aging and due dates, a supplier role, balances per employee, currencies, tax codes on lines, and any new document type. No Sales or Purchasing code changes.

## Anti-goals

- A line that names an employee or one person's pay.
- Any change in how HRM payrolls post.
- A new pending state in the awaiting-posting screen.

## Slices

Each slice runs from migration to screen where it has a screen.

1. **Partner on business lines** (`shared/posting`)
   - Migration: `posting.lines.partner_id bigint NULL REFERENCES partner.partners`.
   - Kinds become declarations with a `ByPartner` flag. The existing constants stay. `Kinds()` returns the declarations.
   - `Line.PartnerID` is a nullable `partner.PartnerID` (or `*int64`, if the shared type does not exist yet).
   - `Record` refuses a mismatch with `posting_partner_mismatch`. That is a programming error, not a user error.
   - Tests need a partner kind that is not shipped. *Builder's call:* register it from an `export_test.go` in `posting`, or keep a test-only kind behind a build tag. Pick the smaller one.
2. **Tracked accounts and matching rules** (`accounting`)
   - Migration: `accounting.accounts.by_partner boolean NOT NULL DEFAULT false`.
   - The account editor gets a switch, disabled when the account has lines or children.
   - Rule set refuses `rule_partner_mismatch`. Flag change refuses `account_partner_rule_mismatch` while a rule would stop matching.
   - The rules screen shows which kinds name a partner.
3. **Partner on ledger lines**
   - Migration: `partner_id` on `accounting.entry_lines` and `accounting.journal_entry_lines`.
   - Reconcile groups by partner too, and reversals copy it.
   - The general ledger shows a partner column when the account is tracked.
4. **Manual entries with partners**
   - Partner picker on the line editor, required on tracked accounts. It is fed by `GET /accounting/partners?q=` under `accounting.entry.edit`, returning id, code and name.
   - Checks `journal_line_partner_required` and `journal_line_partner_not_allowed` when the entry is sent or posted.
5. **Reports**
   - `GET /accounting/partner-balances?legal_entity_id&account_id?&from&to` and `GET /accounting/partner-ledger?legal_entity_id&partner_id&account_id?&from&to`, both under `accounting.view`.
   - Two screens in the accounting area: "Công nợ theo đối tác" (list page, a row opens the partner ledger) and "Sổ chi tiết công nợ".
   - Both reports sum through the data engine, the way the existing reports do.
6. **Seed and end to end**
   - `accounting.seed.ts` adds 131 Phải thu của khách hàng and 331 Phải trả cho người bán, both tracked.
   - It also adds a supplier partner and an opening entry for 2026-07-01 with two customers on 131 and the supplier on 331, against 411.
   - The Playwright spec covers the exit criteria below.

## Exit criteria (roadmap M16)

With `make seed` and `PRODUCTS=hrm,sales,accounting`:

1. The accountant marks an account as tracked by partner. Mapping a payroll kind to it is refused.
2. An opening entry with a line on 131 without a partner is refused when sent. With partners, it posts.
3. Partner balances for 131 list each customer with their balance, and the total equals 131's closing balance in the trial balance.
4. The partner ledger of one customer shows the opening line with a running balance. Cancelling the entry adds a reversal, and the balance returns to zero.
5. A payroll posts and reaches the ledger exactly as before: no partner on its lines, and the HRM and accounting suites are unchanged and green.
6. Go tests: a partner kind's line reaches the journal line with its partner, a mismatched line is refused by `posting.Record`, and the existing `recordtest` passes for `accounting.journal_entry`.

## Calls made without asking

These are my calls; the product owner can override them.

- Only postable accounts take the flag. Partner reports do not roll parents up.
- Partner names in reports are current, not snapshots.
- The partner lookup lives in accounting (item precedent), not behind `partner.viewer`.
- A partner's activity is not checked: partners have no active flag today.
- The flag cannot change on an account with lines. To change tracking, open a new account and move the balance.
