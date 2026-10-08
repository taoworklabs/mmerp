# Multi-product ERP framework and HRM governance

Compiled: 2026-10-06.

> **Archived** (2026-10-07).

Status: **decided**. The decisions below went into the [roadmap](./roadmap-phase-0.md) (M8, M9, M10, production-readiness gate) and [ADR-0022](../adr/0022-approval-rule-permissions-per-product.md). Where this document differs from the roadmap or the ADR, the roadmap and ADR are right.

## 1. Context

- The first version of the HRM core is done up to M6. "Having an HRM core" does not mean "ready for production use".
- The home page is still empty; the HRM menu puts profiles, day-to-day operations and configuration at the same level.
- Viewing and editing the org structure needs `core.org.manage`; configuring approval rules needs `core.approval.manage`. Both exist only in `core.admin`, so the HR manager has to be granted system-wide administration.
- M7 is stuck on the parts that need a real installation (an accountant reviewing the payroll samples, real hardware configuration). M7 is marked obsolete: the parts that can be done without a real installation become M10, the rest become an unnumbered gate.

## 2. Order

| Milestone | Scope |
| --- | --- |
| M8. Business administration permissions | View the org tree; approval-rule permissions per product |
| M9. Navigation shell and HRM overview | Home page, per-product title, HRM menu groups, HRM overview |
| M10. On-premise operations | The self-doable part of the old M7; starts as soon as M9 passes |
| Production-readiness gate | Accountant reviews payroll calculation; settle configuration and performance targets |

Real data enters the system only after M10 passes and the gate has been passed.

## 3. Org structure

- The org tree belongs to core, one shared source for every product. No product has its own tree.
- Every signed-in user can view the whole tree: unit names are not sensitive and already show on documents. Editing the tree still needs `core.org.manage`.
- Permission scopes apply to data attached to the tree (employees, documents), not to the tree itself.
- HRM has a "Cơ cấu tổ chức" (Org structure) screen inside HRM to view the tree (read-only); assigning departments and direct managers is still done in the employee profile. Holders of `core.org.manage` also see a link to the tree-editing page in the admin area. HRM permissions do not allow editing the tree.
- Moving a node in a way that changes the legal entity of existing documents is already blocked. Org structure with effective dates and history is outside the roadmap and needs its own ADR.

## 4. Approval rules

Per [ADR-0022](../adr/0022-approval-rule-permissions-per-product.md):

- Viewing and editing the approval rule of a document type needs `core.approval.manage`, or a tenant-wide `<product>.approval.manage`. HRM declares this permission in the `hrm.approval_admin` role.
- `hrm.approval_admin` can only be granted tenant-wide, since each document type has only one rule for the whole tenant.
- The permission to act on an approval step is separate from the permission to edit rules. Editing a rule does not change open approval instances.
- Data and the approval engine belong to core, but the editing screen lives in the product: HRM approval rules are created and edited right in HRM's Configuration menu (a shared component, filtered by product). The admin area no longer has an approval rules page; holders of `core.approval.manage` get in through each product's menu. The approval inbox stays shared.
- A job title or management position does not by itself carry administrative permissions.

## 5. Navigation shell

- **Home page**, with a greeting as its title ("Hệ thống quản trị doanh nghiệp", "Business management system", sits in the header): a grid of products taken from the manifest. Shows products that are enabled or already have data (same rule as areas in `platform.md`), then filtered by permission. After signing in, users land on the home page.
- **Per-product title** in the header: "Quản lý nhân sự" (HR management) when in HRM. Titles inside pages still follow the page content.
- No product switcher until there is a second product; no shared work page.

## 6. HRM menu

| Group | Items |
| --- | --- |
| (top of menu) | Overview |
| Personnel | Employees · Org structure · Contracts |
| Timekeeping & leave | Timesheets · Leave requests · Overtime |
| Payroll | Payrolls |
| Configuration | Contract types · Leave types · Work calendar · Legal parameters · Approval rules |

One level of navigation in the sidebar; a group left with no items after permission filtering is hidden; Configuration is collapsible. Dependants and leave balances stay inside the employee detail.

## 7. HRM overview

| Figure | Definition | Permission | Click goes to |
| --- | --- | --- | --- |
| Active staff | Employees `active` as of today (tenant time zone) | `hrm.employee.view` | Employee list, filtered `active` |
| Contracts expiring soon | Original `posted` contracts expiring from today to 30 days ahead | Contract view permission | Contract list, filtered `expiring` |
| Awaiting approval (leave, overtime, contracts) | Every `pending_approval` document of that type within view scope; documents awaiting your own approval are in the approval inbox | View permission of the type | List of that type, filtered awaiting approval |

Each figure is the total of the target list with the same filter. Without the permission the block is hidden. Timekeeping and payroll status come later, because they must be split by salary view permission.

## 8. Reference menu tree for an extended HRM

A long-term map of functions for discussion, not a decided scope. A menu does not mean a separate backend module is needed.

| Group | Possible menus when extending |
| --- | --- |
| Overview | Overview by role |
| Personal | My profile · My timesheet · My leave · My overtime · My payslips |
| Personnel | Employees · Org structure · Job positions · Contracts & addenda · Personnel changes · Onboarding & offboarding |
| Recruitment | Hiring requests · Open positions · Candidates · Interview schedule · Offer letters |
| Timekeeping & leave | Shifts & shift assignment · Attendance data · Timesheets · Leave requests · Leave balances · Overtime · Attendance explanations |
| Payroll & benefits | Payrolls · Salary adjustments · Payslips · Insurance · Personal income tax (thuế TNCN) · Benefits · Salary payment |
| Performance | Goals · Review cycles · Employee reviews · Review results |
| Training | Training needs · Courses · Training plans · Results & certificates |
| Reports | Personnel & changes · Contracts · Time & leave · Personnel costs · Recruitment · Training & performance |
| Configuration | HRM master data · Time & leave policies · Payroll configuration · Legal parameters · Approval rules · Integrations |

Principles when extending:

- Each person only sees functions according to their permissions and scope.
- Personal menus share the same operations and data as the admin screens.
- Pending approvals use the shared approval inbox; the entry from HRM only filters HRM documents.
- A job position is an HRM entity, distinct from org tree nodes used for permission scopes.
- Only split out a separate menu when there is a need for centralised processing, not just because a data entity exists.
- Salary payment, insurance and tax need their boundary with accounting settled before building; being able to compute these amounts does not mean filing, settlement or payment exist.
- Unbuilt functions do not show as menus. Areas outside the first version's scope are built only after the need is confirmed and the roadmap is updated.
