# Platform

Concerns shared by every product. Deployment is mainly on-premise on the organisation's own server, with a hosted multi-tenant deployment as an option (see [tenancy.md](./tenancy.md)). Each part is only built when it is needed, not before. Exception: the minimum level of operations below is mandatory before production use.

## Identity and permissions

- Each tenant has a single `users` table, owned by `iam`, used by every product. Each person has one login account.
- Roles are assigned per product and per org unit scope: `iam.user_roles (id, user_id, product, role, org_unit_id)`.
  - `product = 'core'` is for core roles (tenant administration, user management).
  - Any product can declare roles that are only assigned tenant-wide (e.g. `core.admin`, `hrm.approval_admin`); assigning them at an org unit gives the error `role_tenant_wide`.
  - An empty `org_unit_id` means tenant-wide. A permission assigned at an org unit applies to that unit's whole subtree.
  - The unique constraint uses `NULLS NOT DISTINCT` (Postgres 15+) so there are no two identical rows when `org_unit_id` is empty.
  - Role names and each role's permissions are declared in code by the product module (the `iam` service's `RegisterRoles(product, roles, tenantWide...)` method); `iam` stores role grants and answers the question "at which org units does the user have permission Q of product P" (`Scope`) ([ADR-0010](./docs/adr/0010-org-tree-and-roles.md)).
- A product module can add its own login method (for example, a PIN on a shared device), but sessions are always issued by `iam`.
- No SSO across tenants: a person working for two organisations has two accounts.
- **Permission version** (`authz_version`): each user has a number that increases whenever that user's roles, org unit scopes or sensitive-field view permissions change. A change that affects many users at once (editing a role's permissions, moving a node in the org tree) increases a tenant-wide number. The `authz_version` sent to the frontend combines these two numbers, and is included in `/me` and in the `X-Authz-Version` header of every response. The frontend uses it to clear stale data when permissions change ([frontend.md](./frontend.md#session-lifecycle)).

## Enabled products

**Enabled products** is the list of products the tenant may use. Each product maps to a set of modules. The architecture depends only on this list, not on where the list comes from.

- **Taken from configuration.** The list is read from the `PRODUCTS` environment variable (for example `PRODUCTS=sales,inventory`), set by the operator at install time.
- Dependencies between products are declared in code. The app refuses to start if the list is missing a product that another product depends on ([backend.md](./backend.md#rules-between-modules)).
- **Checked per request, not at startup.** `internal/app` always initialises and mounts every module. A middleware puts the tenant's list of enabled products in the context. A product's hook adapter reads the list from the context and becomes a no-op if that product is not enabled. This way, on-premise and cloud model B (one list per tenant, taken from a registry) run the same code path; only the source of the list differs.
- **Product gate by action class.** Every operation belongs to exactly one of three classes:

  | Class | Includes | Product not enabled or disabled |
  | --- | --- | --- |
  | Read | View, list, history, read discussion, download attachments | Allowed |
  | Export | Excel/PDF export, full data export | Allowed |
  | Write | Create, edit, delete, transition, approve, comment, upload attachments, import data, bulk operations | Blocked, error code `product_not_enabled` |

  A single `platform` function, `ProductGate(ctx, product, class)`, decides for **every place**: the module's own routes (a route declares its class; a route that does not is read for GET and write otherwise); core shared routes (the product is taken from the record type); `allowed_actions` (drops write actions, keeps export actions); job creation and job execution (each job kind declares its class). This way users can always look up and export their data, and documents of other products that reference it still open. The user's roles and permissions stay unchanged, so re-enabling restores everything as before.
- **Jobs and enabled products:**
  - A job runs with the enabled products of the tenant that owns the database holding the job. Every place that creates a job (hooks, direct enqueue in a service, periodic jobs) goes through one `platform` function, which calls `ProductGate` for the job's class.
  - **System jobs** (posting reconciliation, cleanup…) always run to completion once enqueued, even if the product has just been disabled: cutting them off would leave data inconsistent. Hooks of a disabled product are no-ops, so no new system jobs arise.
  - **Jobs on behalf of a user** go through `ProductGate` twice, at creation and at execution, together with re-checking the requester's permissions ([documents.md](./documents.md#jobs-moving-to-the-background-does-not-raise-permissions)). If the product is disabled while the job is waiting, export jobs still run; import jobs and bulk operations fail with `product_not_enabled` and report back to the requester.
  - **Periodic jobs** of a product that is not enabled skip the run. So periodic jobs must be written as "process everything not yet processed", not "process since the last run", so they catch up by themselves when re-enabled.
- **Disabling a product never deletes data.** Data is only deleted on a published schedule when the organisation stops using the system entirely (see Personal data).
- The frontend only shows a product's area when that product is enabled or already has data.
- The frontend hides the other areas based on `GET /me`, but access is always blocked in the backend.
- Modules in `core/` and `shared/` are always enabled.

## Operations

| Item | On-premise | Cloud |
| --- | --- | --- |
| Distribution | Signed Docker image + `compose.yml` + install script. For installations without Internet: an offline bundle (Docker image as `docker save`) | Same as on-premise; model B: one app, rolling restart |
| Updates | The operator runs the update command, or enable automatic updates in a maintenance window. The app backs itself up before migrating (see [tenancy.md](./tenancy.md#versions-and-migrations)). The system stops serving for the whole backup and migration; on-premise has no zero-downtime updates | Done by the operator |
| Backup | Daily `pg_dump` plus the attachments directory to another disk on the server, keeping 30 copies, plus one copy off the server (the organisation's NAS, or an encrypted upload to an off-site store). The app warns on the admin screen when the latest backup is older than 24 hours | Daily `pg_dump` stored off the server, plus WAL archiving (WAL-G or pgBackRest); object storage with versioning enabled |
| Restore testing | At installation and at each handover: test-restore a backup | Monthly, rotating through tenants |
| Monitoring | Optional outbound heartbeat to a monitoring endpoint chosen by the operator: version, status, age of the latest backup, error count. No business data is sent. It can be turned off. A lost heartbeat alerts the operator | `GET /healthz`, external uptime checks, error tracking labelled by tenant |
| Logs | Container stdout, Docker log rotation. A command exports a diagnostic bundle (logs + version + configuration with secrets masked) to send to whoever provides support | Labelled by tenant |
| Remote support | Only when the organisation opens it, through a channel it controls. No pre-installed backdoor | — |

Organisations use the system during working hours to run their business. On-premise, nobody may be watching the server from outside, so the heartbeat and the in-app backup warning are the mandatory minimum before production use.

## Personal data

Every module that stores personal data falls under the Law on Personal Data Protection (Luật Bảo vệ dữ liệu cá nhân, No. 91/2025/QH15, in force from 01/01/2026) and its implementing documents.

- A module's ADR must list its personal data fields and its sensitive fields (identity documents, bank accounts, income, health…).
- Sensitive fields are column-encrypted at rest, with the encryption key kept outside the database. Trade-off: SQL can no longer filter or aggregate on these columns, so the ADR must state which computations have to be done in the service.
- Permission to view sensitive fields is a separate permission, not automatically included with manager roles. Every read of these fields is written to the audit log, and the audit row must be written before the data is returned: if writing audit fails, the request fails.
- Each tenant can export all of its data. Data is deleted on a published schedule after the organisation stops using the system.
- **Legal roles:** on-premise, the data lives on the organisation's server and the organisation is the data controller. Whoever touches the data on its behalf (remote support, receiving backups, running a hosted deployment) becomes a processor. The contract between them must spell out each case.
- **Encryption keys on-premise** live on the organisation's server, outside the database (a file with restricted permissions or a container environment variable). Losing the key means losing the encrypted data, so the key must be backed up separately and recorded in the handover minutes.
- **Storage location:** a hosted deployment, uploaded backups, and third-party services (error tracking, email, SMS), if located outside Vietnam, count as cross-border data transfer and need paperwork under the law. Default: cloud and backups are located in the country; heartbeats and error events contain no personal data.

## Integrations

Integrations with external providers (e-invoicing, payment gateways, SMS/Zalo…):

- Each provider is a package in `internal/integration/<provider>/`. The package knows nothing about business logic and can be shared by many modules.
- The module that owns the document decides when to call the provider, and always calls through a River job enqueued in the same transaction, so a provider outage or loss of Internet never breaks a business operation.
- On-premise can lose Internet for hours: jobs retry with backoff, and pending jobs are shown on the admin screen so the organisation knows what has not been sent yet.
- Provider credentials belong to the organisation, stored in `setting` (encrypted) on its server.
- Do not create a common provider interface until there are two providers for the same task.

## Third-party API

- The main public API is the API the frontend uses (OpenAPI generated by Huma), not a separate API.
- Third parties authenticate with an API key. Each key is tied to a system user, so the key's powers are exactly that user's permissions.
- Outgoing webhooks go through jobs, like every other integration.
- The compatibility commitment (API versioning, notice period before removing an endpoint) is settled by an ADR when the first integration partner arrives.
