# 0051. Business lines may name a partner; accounting keeps balances per partner

Status: accepted · 2026-11-12

## Context

A business line today has only a kind, an org unit and a signed amount (`posting.lines`). That is enough for payroll, which never names a person ([ADR-0007](./0007-hrm-product.md), [ADR-0040](./0040-payroll-posting-minimum-group.md)). Sales invoices and receipts (M17) and Purchasing (M18) need more: a receivable is owed by a given customer, a payable is owed to a given supplier, and the accountant must see what each of them owes. Issue #1 named the gap. The ledger of [ADR-0050](./0050-accounting-product-and-general-ledger.md) has nowhere to keep a balance per party. This decision makes business lines able to name a partner and gives accounting a partner subledger. It is built and proven before any product writes such lines, so M17 and M18 only add kinds and documents. Design: `docs/superpowers/specs/2026-11-12-partner-subledger-design.md`.

## Decision

### Business lines

- **`posting.Line` gains an optional partner** (`partner_id`, referencing `partner.partners`). The party is a partner from `shared/partner`, never an employee or a user. Employees are not partners ([ADR-0039](./0039-shared-master-data.md)), so a line can never carry one person's pay.
- **Each kind declares whether its lines name a partner.** `posting` keeps the declared kinds with this flag. `posting.Record` refuses a line whose partner does not match its kind: missing on a partner kind, or present on any other kind. The five payroll kinds name no partner. `posting.Kinds()` returns the flag, so accounting can check rules against it.
- `posting` still writes lines once and voids them; nothing else about recording changes. M16 declares no new partner kinds. The first ones (receivable, payable) come with the documents that write them.

### Accounts and rules

- **An account may be tracked by partner** (`by_partner`, vi: "Theo dõi theo đối tác"). Only postable accounts take the flag. It can change only while the account has no lines, like the account type.
- **Every journal line on a tracked account names a partner, and no line on any other account does.** The service enforces this for automatic and manual entries. The tables cannot enforce it with a CHECK, because the flag is on another table.
- **A mapping rule must match**: a partner kind can map only to a tracked account, and any other kind only to an untracked one (`rule_partner_mismatch`). Changing an account's flag is refused while a rule pointing to it would stop matching. A badly matched rule is refused when it is set, so reconciling never meets one.

### Ledger

- `accounting.entry_lines` and `accounting.journal_entry_lines` gain a nullable `partner_id`.
- **Reconcile carries the partner onto the journal line.** Mapped lines are summed per account, side, org unit, kind and partner. The entry bounds of ADR-0050 apply to the result unchanged.
- **Manual entries**: a line on a tracked account needs a partner (`journal_line_partner_required`), and a line on any other account must not have one (`journal_line_partner_not_allowed`). Both are checked when the entry is sent or posted, like the balance check. Opening receivables and payables are a manual entry with one line per partner.
- **Choosing a partner on a manual entry line** works like choosing an item on a quotation line: accounting serves a partner lookup to its line editor under `accounting.entry.edit`. Accountants need no `partner.viewer` role.

### Reports

Both reports are read under `accounting.view` for the legal entity, and show the partner's current code and name.

- **Partner balances** (vi: "Công nợ theo đối tác"): for one legal entity, one tracked account (or all of them) and a date range, the opening balance, debit, credit and closing balance of each partner. The totals equal that account's row in the trial balance.
- **Partner ledger** (vi: "Sổ chi tiết công nợ"): the journal lines of one partner on one tracked account (or all of them) over a date range, with the opening balance, the running balance, the source document's number and the closing balance.

### Not in this decision

Open-item matching (which receipt pays which invoice), due dates and aging, partner roles beyond Sales' customer (the supplier role comes with M18), balances per employee (advances, payables to staff), multiple currencies, tax codes on lines (VAT is a kind until the Vietnamese layer needs a declaration), and quantity and item on lines (they belong to the stock ledger).

## Rejected alternatives

- **An open party reference (`party_type`, `party_id`).** It would let a line point at an employee or a user, which breaks ADR-0007. It would also leave accounting unable to join one identity table for names.
- **Products choosing the account, or tracking by partner per rule.** Business lines stay account-free ([ADR-0003](./0003-business-lines-are-data.md)). Tracking is a property of the account (131 is tracked whichever document posts to it), so the flag belongs on the account.
- **A separate receivables or payables module.** It would hold a second copy of the balances that the ledger already has. Open-item matching may need its own tables later, inside `accounting`.
- **Letting reconcile flag a mismatched document instead of refusing the rule.** The mismatch is known when the rule is set. Refusing it there leaves no new pending state.

## Consequences

- HRM is unchanged: its kinds name no partner, and its lines and entries stay the same.
- M17 and M18 add partner kinds and documents. They write nothing new in `posting` or the ledger.
- A partner's name in the reports is the current one. Renaming a partner changes how earlier lines are shown, not their amounts.
- An account that already has lines cannot start being tracked. A tenant that wants tracking later opens a new tracked account and moves the balance with a manual entry.
