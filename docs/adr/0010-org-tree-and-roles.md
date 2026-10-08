# 0010. Org tree, roles and permission scopes in `iam`

Status: accepted · 2026-10-05

## Context

The employee profile is the first screen that needs permissions per org unit. `iam` (ADR-0009) only has users and sessions so far; the org tree, legal entities and roles are added to this same module, following [backend.md](../../backend.md#org-units) and [platform.md](../../platform.md#identity-and-permissions).

## Decision

- `iam.org_units (id, parent_id, kind, name, tax_code, legal_name, address)`. The three legal columns only have values when `kind = 'company'` (CHECK constraint). `kind` is stored as a string; the API currently accepts `group`, `company`, `branch`, `department`.
- **Tree invariants**, checked in the service after every create, edit and node move, in the same transaction and under a `SHARE ROW EXCLUSIVE` table lock (two people moving nodes at once cannot create a cycle):
  - every node is reachable from a root (no cycles);
  - `company` and `group` have no `company` ancestor;
  - every other kind has exactly one legal entity, its nearest `company` ancestor.
- No node deletion yet. The check "a node that already has documents does not change legal entity" is added once `record.documents` exists. (Done in ADR-0014.)
- `iam.user_roles (id, user_id, product, role, org_unit_id)`, unique `NULLS NOT DISTINCT` on `(user_id, product, role, org_unit_id)`. An empty `org_unit_id` means tenant-wide.
- **Roles are declared in code.** A module calls `iam.RegisterRoles(product, map[role][]permission)` at initialisation. Permissions have the form `<product>.<object>.<action>` (`hrm.employee.view`). The display name of a role is the translation key `<product>.role.<role>`. `iam` ships the `core.admin` role with `core.org.manage` and `core.user.manage`; the `create-admin` command grants this role tenant-wide. (2026-10-07: core roles are granted tenant-wide only; the rule has been generalised to `RegisterRoles(product, roles, tenantWide ...string)`, and granting a tenant-wide-only role at an org unit is refused with `role_tenant_wide`. `core.admin` now also has `core.period.manage`, `core.approval.manage`, `core.setting.manage`.)
- `iam.Scope(ctx, product, permission)` returns `Scope{All bool; Units []int64}`: the roles holding that permission, expanded to their subtrees with a recursive CTE. The result is cached in the request context, so it is computed once per request.
- **`authz_version` = `<user number>.<tenant number>`.** The tenant number lives in the single-row table `iam.authz`. Granting or revoking a role bumps the user number; moving a node bumps the tenant number; both in the same transaction as the change. Changing a role's permissions is a code change shipped with an update; the frontend reloads when the version changes.
- `/me` also returns `permissions`: every permission the user holds at any org unit, by product. The frontend uses it only to show or hide menus and pages.

## Consequences

- Adding a role or permission needs no migration.
- A granted role that code no longer declares is ignored when computing permissions.
- The tree locks the whole table on write; enough for a few hundred nodes and a few edits a day.
