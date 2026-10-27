# 0040. A payroll posting line never covers fewer than 3 people

Status: accepted · 2026-10-27 · refines the payroll posting of [ADR-0021](./0021-payroll.md)

## Context

Posting a payroll writes its business lines to `posting.lines`: salary expense, employer insurance expense and salary payable summed per org unit, insurance and income tax owed summed per legal entity. `posting.lines` is not encrypted and carries no salary permission, since accounting must read it. A department with one or two people on the payroll therefore shows their pay to anyone who can read posting lines, which defeats the column encryption and the `hrm.salary.view` permission that guard the payroll itself.

## Decision

- **A per-unit posting line needs at least 3 people of that payroll.** People are counted per org unit of the employee, and only those with something to post (gross, net or employer contributions not zero): two people on unpaid leave must not hide the third.
- **A smaller unit rolls up.** Units are visited deepest first; a unit below 3 adds its count and its amounts to its parent. A parent may then reach 3 with what its children brought, or roll up in turn.
- **The legal entity ends the roll-up.** What reaches the legal-entity node is posted there, as the insurance and income tax lines already are. If that leaves it with one or two people while units kept their own lines, it takes in the smallest of those units, one at a time, until it has 3.
- **From the next posting on.** Lines already posted are never rewritten; cancelling and posting a payroll again applies the rule.
- The threshold is fixed in code.

## Rejected alternatives

- **Encrypting posting lines.** Accounting reads them to build journal entries; it would need the salary key and permission for every line it reads.
- **A threshold per tenant.** No one has asked; a setting is something to explain and test forever.
- **Rewriting lines already posted.** Posted data is never rewritten; the next posting follows the rule.
- **Dropping small units' amounts.** Expenses and payables must add up to the payroll's cost.

## Consequences

- A department's salary expense may appear under its parent unit or under the legal entity.
- A legal entity with fewer than 3 people on a payroll still shows its total, which its own statutory declarations already disclose. No line covers fewer than 3 people otherwise.
- Amounts are conserved: for each kind, the posting lines add up to the payroll lines.
