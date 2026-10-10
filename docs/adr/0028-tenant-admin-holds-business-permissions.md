# 0028. The tenant administrator holds every business permission, sensitive ones excepted

Status: accepted · 2026-10-10 · amends [ADR-0010](./0010-org-tree-and-roles.md) ("`core.admin` only administers")

## Context

`core.admin` was granted only core permissions: the install-time administrator saw no product area until someone granted it product roles one by one (the e2e setup does exactly that for HRM). With a second product, an owner who installs mmerp and signs in as `admin` sees an empty home page and must learn the role catalogue of every product first. The product owner wants `admin` to work as the system's root.

The rule that sensitive fields need their own permission, never implied by a manager role ([platform.md](../../platform.md#personal-data)), still stands: salaries and identity numbers must not open to whoever administers users.

## Decision

- A user holding `core.admin` **tenant-wide** holds, tenant-wide, every permission every product registers, except the permissions the product declares **sensitive**.
- A product declares its sensitive permissions with `iam.RegisterSensitive(product, permissions...)` while wiring. HRM declares `hrm.employee.sensitive` and `hrm.salary.view`. Sales declares none.
- `iam.Scope` answers `All` for those permissions; `/me` lists them, so menus and tiles show. Nothing else changes: `Can`, `allowed_actions`, the product gate and audit work as before.
- Role-based approvers and fallback roles still come from role grants only: holding a permission through `core.admin` does not make the administrator an approver of every rule.
- Sensitive permissions keep needing an explicit role (`hrm.sensitive_viewer`, `hrm.payroll`), granted and revoked like any other, and audited on every read.

## Consequences

- A fresh install's `admin` enters every enabled product at once.
- Who holds `core.admin` can read and change all business data of the tenant; grant it sparingly.
- A new product must declare its sensitive permissions, or they open to administrators. The `new-module` skill says so.
