# Roadmap Phase 0: from core to the first HRM version (archived)

> **Archived** (2026-10-07). Phase 0 ended at M9. The following milestones, including M10 and the production-readiness gate (on hold), are in [roadmap.md](../../roadmap.md).

Scope: from an empty repo until the first HRM version ([products/hrm.md](../../products/hrm.md)) runs in production on an organisation's own server. Excludes accounting, cloud model B and other products.

## How milestones are split

- **Vertical slices.** Each milestone cuts through from migration, service and API to the screen, and ends with something that runs. No milestone is backend-only or frontend-only.
- **Core built first, proven by HRM.** Up to M9, core modules are built exactly as far as that milestone needs. From M11, the core is built ahead of need, but each core capability only passes when at least one HRM screen really uses it ([ADR-0023](../../docs/adr/0023-build-core-first.md)).
- **Proven by the sample flow.** A milestone's exit criteria are the steps of the [leave and payroll sample flow](../../documents.md#sample-flow-leave-and-payroll) running in tests, plus the matching Playwright tests ([techstack.md](../../techstack.md#testing-and-quality)).
- **Big risks first.** Document lifecycle, approval and concurrency scenarios are in milestone 3, early enough that a design mistake is still cheap to fix.
- A milestone only starts once the previous one has met all its exit criteria. Therefore a milestone's exit criteria may only rely on what that milestone and earlier ones have built.
- **Data:** from M0 to M6 use only development data, simulated data, or a copy of real data anonymised under agreement with its owner. Real data enters the system only after M10 is complete and the production-readiness gate has been passed, that is, after backup and restore have been proven. If an organisation wants an earlier trial run with real data, the pre-migration backup and the restore procedure of M10 must be pulled ahead of that trial run.

```
M0 Foundation ─► M1 Sign-in and app shell ─► M2 Organisation, permissions, employee profiles
   ─► M3 Document lifecycle: leave requests ─► [Gate: settle payroll calculation] ─► M4 Contracts and overtime ─► M5 Timekeeping and jobs
   ─► M6 Payroll calculation ─► M8 Business administration permissions ─► M9 Navigation shell and HRM overview
(M7 obsolete, split into M10 and the production-readiness gate)
```

## M0. Foundation

**Goal:** the repo, tooling and isolation rules work on an empty app.

- Repo: `git init`, without `.backup/`; update the commands section in `CLAUDE.md` to match the real Makefile.
- Backend: `go.mod`; directory structure per [backend.md](../../backend.md#layers); minimal `platform`: configuration (read and validated at startup), logging, coded errors, `DBFrom`/`InTx`, translation lookup function, the `platform.Module` type.
- Home-grown migration runner: embedded files, advisory lock, one transaction for everything ([tenancy.md](../../tenancy.md#versions-and-migrations)). No backup step yet.
- Frontend: Vite, React, strict TypeScript, Mantine, `theme.ts` with the tokens from [ui.md](../../ui.md#tokens), self-hosted Inter font, `react-i18next`.
- Tooling: `make gen`, `make lint`, `make test`, `make dev`, `make image`. CI runs `make gen` and fails on any diff, `make lint`, `make test`, `make image`, then smoke-tests the image: start `app` and `postgres` with compose, wait for `GET /healthz` and the frontend page to respond within a fixed timeout, then clean up containers and volumes. `make dev` runs continuously so it is not in CI.
- Isolation rules: `depguard`, `forbidigo`, dependency-cruiser with `web/lint-fixtures`, ESLint (Mantine imports, inline `style`, hard-coded text), colour-code scan, translation key check.
- Backend tests: a migrated template database, one database per test.
- `compose.yml` has only `app` and `postgres`; the image embeds the built frontend.

**Done when:**

- The CI pipeline above is green on the empty app, including the image smoke test.
- dependency-cruiser reports every violation in `web/lint-fixtures`.
- `docker compose up` on a dev machine starts, migrates by itself, and serves `GET /healthz` and the empty frontend page.

**Passed** (2026-10-05).

## M1. Sign-in and app shell

**Goal:** sign in, enter the app shell with a menu, switch language, sign out.

- `iam`: users, argon2id passwords, sessions stored in the DB, `/me` with language and `authz_version`, `X-Authz-Version` header.
- Minimal `setting` (the tenant's default language and time zone). Minimal `audit` (sign-in, sign-out).
- Product gate: `PRODUCTS`, `ProductGate` by action class ([platform.md](../../platform.md#enabled-products)).
- Frontend: bootstrap, sign-in screen, application shell ([ui.md](../../ui.md#app-shell)), manifests and `app/areas.ts` with a `core` area and an empty `hrm` area, full session lifecycle ([frontend.md](../../frontend.md#session-lifecycle)), a `/dev/ui` page with the first primitives.
- A command to create the first admin user at install time.

**Done when** Playwright runs:

- a wrong sign-in shows an error on the form; a correct sign-in enters the app shell;
- a session revoked on the server sends the open tab back to the sign-in screen exactly once, with no loop; a second tab follows too;
- switching language changes the whole interface, menu included;
- with `PRODUCTS` lacking `hrm`, the HRM menu is not shown. (Changed by M9: a disabled product with data is still shown, read-only with a banner.)

**Passed** (2026-10-05).

## M2. Organisation, permissions, employee profiles

**Goal:** build the org tree, permissions by org unit, and employee profile management with sensitive fields. This is the first master-data screen.

- `iam`: org unit tree, legal entities, roles by product and org unit, `Scope()`, bump `authz_version` when permissions change ([backend.md](../../backend.md#org-units)).
- `platform`: column encryption (AES-256-GCM, key version code), keys read from outside the database.
- `record`: only registration of master-data record types and `Can`. No document lifecycle yet.
- `audit`: field change history (write side only; viewing history in the UI is done in M3 together with `DocumentHistory`); sensitive field values stored encrypted; audit every view of a sensitive field.
- HRM: `hrm.employees`, `hrm.dependents`; HRM roles.
- Frontend: admin screens for the org tree, users and roles (`core`); `ListPage` and `RecordPage` for employees; `DataTable`, form fields, `SensitiveField`; small-screen layout of `ListPage`.

**Done when:**

- A user only sees employees within the scope of their org units, both in the API and in the UI.
- Revoking the permission to view a sensitive field (e.g. citizen ID number, CCCD) mid-session makes the UI clear old data on the next request.
- Sensitive fields are encrypted in the DB, masked in the UI, and every reveal has an audit row.
- Filters, sorting and page live in the URL; the browser's Back works correctly.

**Passed** (2026-10-05).

## M3. Document lifecycle: leave requests

**Goal:** the first document flow, done end to end across frontend and API. This milestone proves the hardest part of the core.

- `numbering`: sequences per legal entity and year, assigned in `record.Create`.
- `record`: `record.documents`, `Create`/`Edit`/`Delete`/`Transition`, lock order, `version`, period lock (`record.period_locks`) with a pending-approval document check, `allowed_actions`, approval fields; the `recordtest` contract test suite ([documents.md](../../documents.md#document-lifecycle)).
- `approval`: rules, approval instances, `CompleteApproval` with two kinds of error, `Approvers(level)`, approvers fixed when a step starts, fallback roles and approver substitution.
- HRM: leave requests, leave balances, direct manager (`manager_id`) and `Approvers`. **Granting and adjusting leave balances** per employee and year: a dedicated permission, mandatory reason, audited ([products/hrm.md](../../products/hrm.md#leave)). No automatic accrual rules yet.
- Frontend: `DocumentPage`, `shared/document` (`DocumentStatus`, `DocumentActions`, `ApprovalPanel`, `DocumentHistory`, `useDocumentMutation`), approval inbox with `RecordPreview`, `invalidate` in the manifest, approval rule configuration, period lock screen; small-screen layouts of `DocumentPage` and `InboxPage`.

**Done when:**

- Steps 1 to 7 and step 12 of the sample flow run in tests, concurrent steps included.
- Period lock while documents are pending approval, using a leave request in place of step 12' overtime request: locking March while a March leave request is pending approval is refused, with the list; once the requester withdraws the request, the lock succeeds.
- `recordtest` green for `hrm.leave_request`.
- Test: adjusting a leave balance concurrently with approving a leave request never makes the balance negative and loses no adjustment.
- The implementer grants start-of-year leave balances to employees through the UI, without seeding data.
- Playwright: submit, withdraw, resubmit, approve and reject requests; `version` conflict when two tabs edit at once; approving from the approval inbox updates the leave balance on the HRM screen by itself; an employee submits a request and a manager approves it at 375px width.

**Passed** (2026-10-05).

## Gate before M4: settle payroll calculation

No code. The aim is to find early what payroll calculation demands of the contract model, timekeeping and payroll inputs, before building them in M4 and M5.

- Build the **payroll samples**: about 20 employees with a full calculation of every item, built independently of the system's code. With no accountant yet, the samples are built by Claude from public regulations and cross-checked against two independently written open-source payroll tools ([docs/payroll-samples](../../docs/payroll-samples/README.md)); the accountant's review moves to M7 (now the [production-readiness gate](../../roadmap.md#production-readiness-gate)). The samples must include:
  - joining mid-period, leaving mid-period;
  - salary change mid-period (an addendum effective mid-month);
  - unpaid leave, sick leave, annual leave;
  - overtime on working days, rest days, public holidays;
  - with dependants; hitting the insurance contribution cap; income spanning several personal income tax (thuế TNCN) brackets;
  - an error found after the period is closed, adjusted in the next period (back pay, clawback).
- Settle the **payroll calculation** for the first version and record it in [products/hrm.md](../../products/hrm.md): standard working days, how salary is split on a salary change or joining/leaving mid-period, which allowances are and are not subject to insurance, tax. Also settle how errors in a closed period are adjusted: entered in the open period, without cancelling the payroll of the locked period; from that, decide whether `posting.lines` needs negative amounts.
- The samples go into the repo as test data and become the **M6 acceptance test**.
- Review the contract model (M4), timesheets (M5) and employee profiles (M2: start date, leaving date) against the settled calculation; fix the docs before starting M4.

**Done when:** the samples cover all the cases above, every part with a cross-check tool matches to the đồng, assumptions and open points are written down, and `products/hrm.md` is updated.

**Passed** (2026-10-05): 28 sample cases; calculation in [products/hrm.md](../../products/hrm.md#payroll-calculation).

## M4. Contracts and overtime

**Goal:** the remaining two source document types of the payroll, reusing the whole M3 framework.

- HRM: contracts and addenda (the contract in force on a date), overtime requests, approval fields of each type.
- Payroll lock at legal-entity level (`hrm.payroll_locks`, `hrm.payroll_periods`) and a check of the affected date range. No period is closed yet at this milestone, but every source document already goes through the right check path.
- Frontend: contract screens (a tab in the employee profile, and a company-wide contract list with an expiring-soon filter) and overtime requests. Contract types have a fixed-term flag, deciding whether a contract requires an end date.

**Done when:**

- `recordtest` green for `hrm.contract` and `hrm.overtime_request`.
- Test: the contract in force on a date is computed correctly with several addenda; the affected date range is computed correctly for a leave request spanning two months and for a contract without an end date.
- Adding a new document type needs no change in `record`, `approval` or `shared/document`. This is the test that the core is general enough.

**Passed** (2026-10-06): two new document types without changing a line in `record`, `approval`, `shared/document` or the frontend `core` area.

## M5. Timekeeping and jobs

**Goal:** timesheets imported from Excel; job infrastructure.

- `platform`: River in the database, a single job-creation function going through `ProductGate`, system jobs and jobs on behalf of a user, periodic jobs written to catch up by themselves ([documents.md](../../documents.md#jobs-moving-to-the-background-does-not-raise-permissions)).
- `dataio`: read Excel, validate each row, report errors per row, write through the owning module's service.
- HRM: timesheets per period and org unit, manual entry, Excel import and **Excel export**; bulk import of start-of-year leave balances from Excel.
- Frontend: timesheet screen, Excel import and export screens with `useJobStatus`, the user's job list.

**Done when:**

- Step 8 of the sample flow runs.
- Test: an Excel file with bad rows reports the right row and reason; permission revoked while the job is queued makes the job fail with a permission error; disabling the HRM product while an import job is queued makes the job fail, while an export job still runs.
- Playwright: import a timesheet from Excel and follow progress until done.

**Passed** (2026-10-06): jobs on River with one job-creation function and middleware running jobs on behalf of the user ([ADR-0019](../../docs/adr/0019-background-jobs-dataio-and-file-storage.md)); day-grid timesheets, Excel import and export, leave balance import from Excel ([ADR-0020](../../docs/adr/0020-timesheets.md)). The work calendar and the "company pays" flag moved to M6; initial import moved to M7 (now [M10](../../roadmap.md#m10-on-premise-operations)).

## M6. Payroll calculation

**Goal:** payrolls that can be closed, with correct figures, immutable after closing.

- `platform`: rounding function and the largest-remainder allocation algorithm ([ADR-0006](../../docs/adr/0006-money-rounding.md)); rounding setting per legal entity.
- `shared/posting`: `posting.lines`, `Record`/`Void`, `OnChanged` hook (no-op, since there is no accounting yet).
- HRM: work calendar per legal entity (weekly rest days, public holidays) and the "company pays" flag of leave types ([ADR-0020](../../docs/adr/0020-timesheets.md)); legal parameters with effective dates and a default set for the current year; payroll calculation per the method settled at the gate before M4 (compute and recompute run as jobs on behalf of the user), social, health and unemployment insurance (BHXH, BHYT, BHTN), personal income tax (thuế TNCN); input snapshot and `payroll_sources`; detecting source changes and recomputing; closing and cancelling with `posted_payroll_id`; business lines grouped by department; encryption of every per-person amount.
- Frontend: payroll screen (employees × items, totals by department), source-change warning, recompute button, payroll Excel export (needs salary view permission, audited).

**Done when:**

- Steps 9, 10, 11', 12' and 13 of the sample flow run.
- Step 11 runs in its **HRM-only variant**: closing the payroll creates the right `posting.lines` (grouped by department, no per-person amounts); the `OnChanged` hook is a no-op so there is no posting job.
- Test: two payrolls of the same period cannot both be closed; cancelling a payroll that is not the one holding the period is refused; `line` and `total` rounding both make the sum of the lines equal the document total.
- Test: disabling the HRM product while a payroll export job is queued, the job still runs; a payroll calculation job fails with `product_not_enabled`.
- **Acceptance test:** the payroll samples from the gate before M4 come out right to the đồng in every case.

**Passed** (2026-10-06): the 28 sample cases come out right to the đồng from data entered through services; payroll, work calendar, legal parameters, per-legal-entity settings and `shared/posting` per [ADR-0021](../../docs/adr/0021-payroll.md). Deriving the `day_kind` of overtime requests from the calendar, cash rounding and severance allowance are deferred.

## M7. Ready for production use

**Obsolete** (2026-10-06). The self-doable part moved to [M10](../../roadmap.md#m10-on-premise-operations); the part needing a real installation moved to the [production-readiness gate](../../roadmap.md#production-readiness-gate).

## M8. Business administration permissions

**Goal:** the HR manager views the org structure and configures HRM approval rules without the `core.admin` role.

- `iam`: every signed-in user can view the whole org tree; editing the tree still needs `core.org.manage`. Scopes only apply to data attached to the tree (e.g. employees under `hrm.employee.view`).
- `approval`: viewing and editing the approval rule of a document type needs a tenant-wide `<product>.approval.manage`, with the product taken from the document type ([ADR-0022](../../docs/adr/0022-approval-rule-permissions-per-product.md)). HRM declares this permission in the `hrm.approval_admin` role, grantable tenant-wide only.
- Frontend: the approval rule editor moves into `shared/document` and takes a product; HRM has an Approval rules screen (HRM document types) and an Org structure screen (read-only tree, with a link to the editing page for holders of `core.org.manage`). The admin area's approval rules page is removed.

**Done when:**

- Test: a user with only an HRM role can view the org tree but cannot create, edit or move nodes.
- Test: a tenant-wide `hrm.approval_admin` can list, save and delete rules of HRM document types; granting this role at a branch is refused at grant time; it cannot act on rules of another product.
- Test: editing or deleting a rule does not change the steps of open approval instances.
- Screens only show actions the backend allows.

**Passed** (2026-10-06): the org tree is readable by every signed-in user; tenant-wide-only roles (error code `role_tenant_wide`) and `hrm.approval_admin`; approval rule permissions per product per [ADR-0022](../../docs/adr/0022-approval-rule-permissions-per-product.md); Org structure and Approval rules screens in HRM. The Org structure item shows for every signed-in user (like Leave requests and Overtime), not only those with HRM permissions.

## M9. Navigation shell and HRM overview

**Goal:** users get into the right product, see the HRM menu by work area, and the figures that need handling.

- Home page, titled "Hệ thống quản trị doanh nghiệp" (Business management system): a product grid taken from the manifest, showing products that are enabled or already have data, filtered by permission. After signing in, users land on the home page.
- The header takes its title from the open product ("Quản lý nhân sự", HR management).
- The HRM menu is grouped (the Org structure and Approval rules items exist since M8): Overview; Personnel (Employees, Org structure, Contracts); Timekeeping & leave (Timesheets, Leave requests, Overtime); Payroll (Payrolls); Configuration (Contract types, Leave types, Work calendar, Legal parameters, Approval rules). A group left with no items after permission filtering is hidden.
- HRM overview, each block hidden when the user lacks the corresponding permission:

  | Figure | Definition | Permission | Click goes to |
  | --- | --- | --- | --- |
  | Active staff | Employees `active` as of today (tenant time zone) | `hrm.employee.view` | Employee list, filtered `active` |
  | Contracts expiring soon | Original `posted` contracts expiring from today to 30 days ahead | Contract view permission | Contract list, filtered `expiring` |
  | Awaiting approval (leave, overtime, contracts) | Every `pending_approval` document of that type within view scope, not only those awaiting your approval | View permission of the type | List of that type, filtered awaiting approval |

**Done when:**

- Each figure equals the total of the target list with the same filter.
- A user with permissions only in department A does not see department B's figures; revoking a permission mid-session makes the corresponding block disappear after the reload triggered by `authz_version`.
- Deep links, Back and mobile work correctly; a disabled product with data is still reachable from the home page in read-only mode.

**Passed** (2026-10-07): home page product grid, header title per area, grouped HRM menu (Configuration collapsible), HRM overview with five count blocks using exactly the target list APIs; `/me` returns `products_with_data`, and a disabled product with data shows with a read-only banner. "Has data" only counts documents in `record.documents`, not master data such as employees. The home page keeps the greeting as its page title; the title "Hệ thống quản trị doanh nghiệp" sits in the header. After reviewing on sample data, the layout changed: the home page has no navigation bar, each area shows only its own menu, Administration becomes a tile on the home page, and the approval inbox and jobs move to the header.
