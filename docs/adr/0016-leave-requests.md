# 0016. Leave requests, leave types and leave balances

Status: accepted · 2026-10-05

## Context

The leave request is the first document (`products/hrm.md`). There is no work calendar yet, and organisations want to define their own leave types.

## Decision

- `hrm.leave_types (id, name, deducts_balance, active)`: a catalogue entered by the organisation, not seeded. Only types with `deducts_balance` deduct from the leave balance. A type that has been used is not deleted, only deactivated.
- `hrm.leave_requests (id, employee_id, leave_type_id, start_date, end_date, days, reason)`, `id` is the id in `record.documents`. `days` is `numeric(4,1)` (`shopspring/decimal` in Go), a multiple of 0.5, **entered by hand** and no more than the number of calendar days in the range; it will be computed once a work calendar exists. (2026-10-07: the work calendar has existed since M6 but `days` is still entered by hand; computing it is deferred, see the roadmap.) A request does not span two years (`leave_spans_years`), so each request deducts from only one leave balance.
- Document type `hrm.leave_request`: number prefix `NP`, the document date is the first day of leave, the org unit is the employee's department at creation, approval fields `days` (number) and `leave_type` (choice, the value is the leave type id). The employee cannot change after creation.
- `OnTransition`: moving to `posted` locks the year's leave balance row (`FOR UPDATE`), deducts, and records the deducted days in `leave_requests.deducted`; `posted` to `cancelled` refunds exactly that amount, even if the leave type's deduct flag has since changed. Not enough balance gives `insufficient_leave_balance`. No pay period check yet, since there are no pay periods yet.
- `hrm.leave_balances (employee_id, year, days)`, `CHECK (days >= 0)`. Granting and adjusting are one operation: add a number (negative or positive) with a mandatory reason, locking the same row as a deduction, one audit row each time. No separate adjustment history table.
- **Permissions by relationship, no role needed:**
  - The user linked to the employee (`employees.user_id`) creates, views, edits, submits and withdraws their own requests. Only the submitter can withdraw.
  - People on the employee's `manager_id` chain can view that employee's requests.
  - HR staff by role and the document's org unit scope: `hrm.leave.view`, `hrm.leave.edit` (added to the `hr` role); cancelling an approved request needs `hrm.leave.edit`.
  - New role `leave_admin`: `hrm.leave_balance.adjust` (grant, adjust leave balances) and `hrm.leave_type.manage`.
  - Viewing a leave balance: the owner, or `hrm.leave.view` or `hrm.leave_balance.adjust` at the employee's org unit.
- `Approvers(level)`: the user of the `level`-th manager up the `manager_id` chain; a manager without an account returns empty.
- `Subjects`: the account of the employee requesting leave, so employees do not approve their own requests when HR submits on their behalf.

## Consequences

- Self-service employees need no role grant; the "Đơn nghỉ" (leave requests) menu always shows when the HRM product is enabled.
- Leave requests have no sensitive fields; health information for sick leave is added when needed, with its own view permission.
