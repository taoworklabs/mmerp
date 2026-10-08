# 0020. Timesheets

Status: accepted · 2026-10-06

## Context

The timesheet (`hrm.timesheet`) is the last source document of payroll. M5 builds timesheets entered by hand, imported from Excel and exported to Excel ([ADR-0019](./0019-background-jobs-dataio-and-file-storage.md)). The settled payroll calculation needs paid working days per day, including paid leave days and public holidays, but there is no work calendar yet.

## Decision

- Table: `hrm.timesheets (id, org_unit_id, period_start, period_end, active)`, `id` is the id in `record.documents` (deferred foreign key like other documents). The period is a calendar month.
- **One non-cancelled timesheet per org unit per period**: unique index `(org_unit_id, period_start) WHERE active`. Cancelling sets `active = false` in `OnTransition`; a duplicate gives `timesheet_exists`.
- **Daily grid**: `hrm.timesheet_lines (timesheet_id, employee_id, date, days)`, one row per employee per day, `days` is 0.5 or 1. Empty cells are not stored. A timesheet records only **days actually worked**; leave days come from leave requests, public holidays from the work calendar, both read by payroll (M6).
- Employees in the grid: employees whose `org_unit_id` equals the timesheet's org unit and who are employed during the period (by hire date and termination date), plus anyone who already has rows in the timesheet. Each row must lie within the period, within the employee's employment, and belong to an employee in the grid.
- Editing a draft timesheet rewrites the period and **all** rows at once, at the version the user saw; a draft's period can change as long as it does not clash with another timesheet.
- Document: number prefix `BC`, the document date is `period_end`, the org unit is the timesheet's org unit. No approval fields and no `Approvers`: approval rules for timesheets go by role, like contracts.
- Permissions by the timesheet's org unit: viewing and exporting need `hrm.timesheet.view`; editing, submitting and cancelling need `hrm.timesheet.edit`. Both belong to the `hr` role.
- `OnTransition` on posting and cancelling checks for closed pay periods over `[period_start, period_end]`.
- **Excel import** `hrm.timesheet`: columns `Mã NV | Họ tên | 1 … N` (employee code, full name; N is the number of days in the month). The file replaces **all** rows of the draft timesheet, written at the current version. **Excel export** has the same layout, so the exported file is also the template to fill in.
- **Leave balance import** `hrm.leave_balance`: columns `Mã NV | Năm | Số ngày | Lý do` (employee code, year, days, reason); each row is one leave balance adjustment and needs `hrm.leave_balance.adjust` at the employee's org unit. The rows of a file must not, in total, make a leave balance negative.
- **The work calendar and the leave type's "company pays" flag move to M6**, with payroll as the first place that uses them.
- **Initial contract import moves to M7**, with the preparation for production use. (2026-10-07: M7 is obsolete; initial import now belongs to M10.)

## Consequences

- Timesheets do not depend on the work calendar, so M5 does not need to build one; in return, HR does not see public holidays on the grid until M6.
- An employee who changes department mid-month appears in the grids of both org units if they already have rows in the old one; payroll sums the rows per employee.
- Excel import replaces all rows, so to change part of it, export, edit, then import again.
