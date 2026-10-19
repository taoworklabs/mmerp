# 0035. The employee detail shows the last four digits of a sensitive field to who may read it, unaudited

Status: accepted · 2026-10-19 · amends [ADR-0013](./0013-employee-profile.md) (sensitive fields are only flagged in the detail API)

## Context

The employee detail API says only whether each sensitive field (`national_id`, `social_insurance_no`, `tax_code`, `bank_account`) holds a value; the page shows "••••••" and a reveal per field, each reveal an audited read. HR checking which account or ID is on file (comparing with a paper form, telling two bank accounts apart) has to reveal the whole value, and every such glance lands in the audit log as a full read. A bank statement or a card shows `•••• 4821` for the same need without disclosing the number.

## Decision

- **The detail API returns a tail per sensitive field** (`sensitive_tail`): the last four digits of the value, only to an actor holding `hrm.employee.sensitive` at the employee's org unit, and only when the value holds at least eight digits. Otherwise the tail is `null` and the page shows the mask alone. Digits, not characters: a bank account entered as "Vietcombank 0123 456 789" shows `6789`, and a short value (a passport number of seven digits) is never four-fifths disclosed.
- **The tail is computed on read.** The service decrypts the value in memory, keeps the last four digits and drops the rest; nothing new is stored. Lists never carry tails.
- **A tail is not an audited read.** Only a reveal, which returns the whole value, writes `hrm.employee_sensitive_viewed`, as before. The tail is shown to the same people who may reveal the whole value, so it discloses nothing they could not already see.
- **"Show all" on the page is four reveals.** The frontend calls the existing per-field endpoint for each field with a value; each call is audited on its own, as before.

## Rejected alternatives

- **A stored, unencrypted tail column.** Writes part of every ID and bank account in clear at rest, so a database dump leaks it; needs a migration and a backfill that SQL cannot do, since only the binary holds the keys.
- **Masking in the frontend.** The whole value would reach the browser on every page view.
- **Auditing each tail.** Every page view by HR would write four audit rows, burying the full reveals an auditor looks for.

## Consequences

- An employee detail request by a sensitive viewer decrypts up to four values; the cost is negligible next to the queries it already runs.
- The exception to "every read is audited" is only for these four digits, and only for actors who may read the whole value.
