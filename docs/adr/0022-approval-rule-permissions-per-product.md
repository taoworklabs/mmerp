# 0022. Approval rule permissions per product

Status: accepted · 2026-10-06 · replaces the "Permission to configure rules" line of [ADR-0015](./0015-approval.md)

## Context

ADR-0015 gave a single permission `core.approval.manage`, inside `core.admin`. To let the head of HR configure HRM approval rules, they had to be granted all of `core.admin`, including managing users and the org tree. An approval rule is one rule per document type, applied tenant-wide; product roles, on the other hand, can be granted at an org unit (ADR-0010).

## Decision

- Viewing, saving and deleting the approval rule of a document type needs **either**: `core.approval.manage` (all products), or `<product>.approval.manage` with the product taken from the document type's `record.Type.Product`.
- The per-product permission only takes effect when granted **tenant-wide** (`Scope.All`), because the rule applies to every org unit. The product declares the role holding this permission as a tenant-wide-only role; granting it at an org unit is refused at grant time, so menus based on "has the permission somewhere" stay correct.
- `approval` checks the permission by a name built from the product, without importing the product module. The product declares the permission in its own role; HRM uses the role `hrm.approval_admin`.
- Viewing and editing share one permission; approvers do not need to see the rule to approve.

## Rejected alternatives

- **Granting `core.approval.manage` at an org unit**: breaks the rule that core roles are tenant-wide (ADR-0010), and still opens the rules of every product.
- **Separate rules per legal entity or org unit, with inheritance**: right for a group whose companies have different processes, but changes the data model of `approval.rules`; nothing needs it yet.

## Consequences

- `core.admin` can still configure every rule as before.
- A new product declares `<product>.approval.manage` in one of its roles; `approval` needs no change.
- Editing or deleting a rule does not change open approval instances, because the steps are created at submit time (ADR-0015).
