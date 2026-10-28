# 0043. Printing: an actor may print the part they own

Status: accepted · 2026-10-27 · extends [ADR-0026](./0026-printing.md) only where printing is concerned

## Context

A payslip is a part of the payroll record (one employee's line). Printing needs `View` and `Print` on the record (`check` in `internal/core/printing/service.go`), which an ordinary employee must never have: the payroll holds everyone's pay. Yet each employee must be able to print their own payslip.

## Decision

- A print template may declare `Owner(ctx, id) (int64, error)`: the part key the actor owns in the record, 0 for none.
- An actor without `View` and `Print` on the record may print exactly `parts: [owner key]`, and only when the record is **posted**. Any other parts, or a non-posted record, is refused as if the record did not exist.
- Such a print renders from the frozen snapshot (`frozen`), never from current data, and is audited on the record like any other print.
- HRM payslips use it, with the employee id as part key and `hrm.employees.user_id = actor` as ownership.

## Consequences

- No new permission and no second print path: `render` is shared; only `check` gains the owner branch.
- Templates without `Owner` behave as before.
- Only a posted record exposes parts, so a draft payroll never leaks through this path.
