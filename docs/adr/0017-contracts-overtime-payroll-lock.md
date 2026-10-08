# 0017. Contracts, overtime requests and payroll lock

Status: accepted · 2026-10-06

## Context

M4 adds the two remaining source document types of payroll: labour contracts (with appendices) and overtime requests, together with the payroll lock at legal-entity level that every source document must check (`products/hrm.md`, Payroll lock section). The payroll calculation settled on 2026-10-05 determines which amounts a contract must carry and how an overtime request must record hours. There is no payroll and no work calendar yet.

## Decision

### Contracts

- **Contract types** are a catalogue entered by the organisation, `hrm.contract_types (id, name, active)`, not seeded, names unique case-insensitively; same pattern as leave types. A type that has been used is not deleted, only deactivated. Managing them needs `hrm.contract_type.manage`.
- `hrm.contracts (id, employee_id, contract_type_id, parent_id, start_date, end_date, terms)`, `id` is the id in `record.documents`. A base contract has an empty `parent_id`; an **appendix** points to a base contract (never to another appendix), has an empty `end_date` and carries the base contract's type. An appendix's effective date lies within `[start_date, end_date]` of the base contract. An appendix does not change the end date: changing the term is a new contract.
- An appendix carries **all** amounts, not only the changed ones. The contract in effect on a date is the `posted` base contract covering that date, with the amounts of the latest `posted` appendix whose effective date is no later than that date, or of the base contract itself if there is no such appendix.
- `terms` is **one encrypted JSON blob** (AAD `hrm.contracts.terms`), same pattern as `hrm.dependents.data`: `{salary, lines: [{kind: allowance|support, name, amount, taxable}]}`, money is `int64` in VND. Salary **never** goes into `record.Header.Amount`, because `record.documents` is not encrypted.
- Document type `hrm.contract`: number prefix `HD` (appendices too), the document date is the effective date, the org unit is the employee's department at creation, approval field `contract_kind` (choice, the value is the contract type id). The employee and the base contract cannot change after creation. No `Approvers`: approval rules for contracts go by role. `Subjects` is the employee's account.
- **Permissions:**
  - `hrm.contract.view`, `hrm.contract.edit` and `hrm.contract_type.manage` are added to the `hr` role, scoped by the document's org unit.
  - New role `payroll` with permission `hrm.salary.view`: view contract amounts (later payrolls too).
  - Creating, editing and deleting a draft need `hrm.contract.edit` **and** `hrm.salary.view` at the org unit; submitting and cancelling need only `hrm.contract.edit`.
  - Contract details return `terms` only with `hrm.salary.view`, and each return writes an audit row `hrm.contract_terms_viewed`. Audit on create and edit stores the encrypted `terms`, like every sensitive field.
- `OnTransition`:
  - To `posted`: check the pay period; lock the employee row (`FOR UPDATE`); for a base contract, refuse `contract_overlaps` if it overlaps another `posted` base contract of the same employee; for an appendix, refuse `contract_parent_not_posted` if the base contract is not `posted`, and re-check that the effective date lies within the base contract's term.
  - `posted` to `cancelled`: check the pay period; lock the employee row; refuse `contract_has_appendices` if the base contract still has `posted` appendices.
  - Locking the employee row serialises approvals and cancellations of contracts for the same employee, so two overlapping contracts cannot both be approved, and an appendix cannot be approved while its base contract is being cancelled.

### Overtime requests

- `hrm.overtime_requests (id, employee_id, date, day_kind, day_hours, night_hours, reason)`. `day_kind` is `weekday`, `weekly_off` or `holiday`, **entered by hand** until a work calendar exists. Hours are `numeric(4,1)`, multiples of 0.5, non-negative, total greater than 0 and at most 24. One request is one day.
- Document type `hrm.overtime_request`: number prefix `TC`, the document date is the overtime date, approval fields `hours` (number, day hours plus night hours) and `day_kind` (choice). The `day_kind` labels are server-side translations in the viewer's language, taken from the `hrm` translation files; core is unchanged.
- Permissions and approvers as for leave requests: the owner self-serves, managers up the chain can view, HR needs `hrm.overtime.view`, `hrm.overtime.edit` (added to `hr`). `Approvers` follows the `manager_id` chain, `Subjects` is the employee's account.
- `OnTransition` only checks the pay period. Monthly and yearly overtime limits are done in M6, with payroll.

### Payroll lock

- `hrm.payroll_locks (legal_entity_id)`: one row per legal entity, created on demand (`INSERT … ON CONFLICT DO NOTHING`), used only for locking.
- `hrm.payroll_periods (id, legal_entity_id, period_start, period_end, posted_payroll_id)`, unique key `(legal_entity_id, period_start)`. `posted_payroll_id` points to `record.documents`; the constraint to payroll is added in M6. In M4 no period is closed yet.
- **The pay period check** runs in the `OnTransition` of every source document, when moving to `posted` and from `posted` to `cancelled`: take `FOR SHARE` on the legal entity's row, find periods with a non-empty `posted_payroll_id` overlapping the affected date range, and if any, refuse `payroll_period_closed` with the list of periods.
- Affected date range: leave request `[start date, end date]`; overtime request `[date, date]`; contract `[effective date, end date or infinity]`; appendix `[effective date, base contract end date or infinity]`.
- **HRM lock order** after the two steps of `record`: legal entity row in `hrm.payroll_locks` → employee row → leave balance.
- Leave requests now also check the pay period, before deducting or refunding the leave balance. This is the part ADR-0016 recorded as "no pay period check yet"; ADR-0016 stays unchanged.

## Consequences

- All three source document types go through the same pay period check from M4, even though no period is closed yet; M6 only needs to write `posted_payroll_id`.
- A contract whose date range overlaps a closed period cannot be cancelled; a contract without an end date overlaps every period closed after its effective date. To change terms, issue an appendix from an unclosed period.
- Someone with `hrm.contract.edit` but without `hrm.salary.view` can still approve and cancel contracts, but cannot see or enter amounts.
- Contracts have no initial import path yet; done in M7 ([ADR-0020](./0020-timesheets.md)). (2026-10-07: M7 is obsolete; initial import now belongs to M10.)
