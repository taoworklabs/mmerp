# Roadmap Phase 1: a complete core

Scope: build the core capabilities still missing before production use, each proven by HRM ([ADR-0023](./docs/adr/0023-build-core-first.md)). M10 and the production-readiness gate are on hold. Excludes accounting, custom fields, cloud model B, and other products.

Phase 0 (M0 to M9: foundation, sign-in, organisation and permissions, document lifecycle, contracts, overtime, timekeeping, payroll calculation, administrative permissions, navigation shell) is finished and archived in [docs/archive/roadmap-phase-0.md](./docs/archive/roadmap-phase-0.md). You do not need to read that file when working on Phase 1.

## How milestones are split

- **Vertical slices.** Each milestone cuts through from migration, service and API to the screen, and ends with something that runs.
- **Core built first, proven by HRM.** A core capability only passes when at least one HRM screen really uses it, with tests from the API to the screen (Playwright included).
- **ADR before code.** Every new core module needs its own ADR before any code is written.
- A milestone only starts once the previous one has met all its exit criteria.
- **Data:** use only development or simulated data. Real data enters the system only after M10 passes and the production-readiness gate has been passed.

## Criteria common to every milestone

A milestone only passes when what it adds keeps the contract intact from table to screen. The `new-module`, `new-document-type` and `new-screen` skills carry the same criteria; change one place and change both.

- **Tables:** every new table belongs to exactly one module, and only that module's service writes to it.
- **Fields:** status and type columns have a `CHECK` with exactly the set of values the service accepts; columns are `NOT NULL` unless empty has a meaning. The DB constraint and the service check say the same thing.
- **Money and dates:** money is `bigint` in the currency's minor unit (for VND, the đồng); intermediate calculations use `decimal` and round once through `platform.Round` or `platform.Allocate`. Business dates are `date`. There are no foreign currencies yet; when there are, each currency declares its number of decimal places.
- **IDs:** easily confused ID pairs crossing a module boundary (`types.go`, `deps.go`, hooks) use a dedicated type declared by the owning module (e.g. `iam.UserID` differs from `hrm.EmployeeID`); store and API stay `int64`. Apply when touching the code, not in one sweep.
- **Queries:** only through sqlc, no dynamically assembled SQL.
- **Frontend:** only use types generated from OpenAPI; never redeclare API types by hand.
- **Module-specific status:** each status column has a single transition function, with tests for forbidden transitions.

```
M11 Attachments and discussion ─► M12 Notifications ─► M13 PDF printing
   ┄► M10 On-premise operations ┄► [Gate: production readiness]   (on hold)
```

## M11. Attachments and discussion

**Done** (2026-10-07): [ADR-0024](./docs/adr/0024-attachments-and-discussion.md).

**Goal:** users attach files and discuss directly on documents and profiles, instead of email and shared folders.

- ADR before writing code: `attachment` and `discussion` modules in `core`, attached by `(doc_type, doc_id)` to both documents and master data; view permission follows the record's `Can(view)`, permissions to add and delete; size and file-type limits; whether files of a `posted` or `cancelled` document may be added or deleted.
- Files are stored through `platform.Files`; every add, delete and download writes audit.
- **File storage split by owning module.** Each module writes to its own directory in file storage and only cleans orphan files in that directory, so the `dataio` cleanup job never sees attachment files.
- **Adding and deleting never lose a file.** Add: write the file first, then write the metadata in the transaction. Delete: the transaction only deletes the metadata; the file on disk is removed later by the cleanup pass. A rollback at any step never loses a file that is still referenced.
- **Extra permission per record type.** Each record type declares the extra permission needed to view its attachments (contracts declare `hrm.salary.view`, since the file may contain salary); the uploader does not mark it. Every listing and download re-checks the current permission, like `dataio` export files.
- The side column of `DocumentPage` gains attachments and discussion ([ui.md](./ui.md#page-templates)); the employee profile gets them too.
- Proven by HRM: attach the scanned contract and the papers for a leave request; discussion on a leave request between the requester and the approver.

**Done when:**

- A user who cannot view a record cannot list or download its files and cannot read its discussion, even when guessing the right id.
- Deleting a draft takes its files and discussion with it; data of a cancelled document stays readable.
- Disabled product: readable and downloadable, nothing can be added.
- Running the `dataio` cleanup job when attachment files are older than 24 hours: the attachment files remain.
- Adding an attachment whose transaction rolls back: no metadata remains, and the cleanup pass deletes the orphan file. Deleting an attachment that rolls back: the file can still be downloaded.
- A user who can view a contract but lacks `hrm.salary.view` cannot list or download the contract's attachments. Revoking the permission after the file is attached makes the next download refused.
- Playwright test: attach a file to a contract and download the same content back; discussion on a leave request shows on both sides.

## M12. Notifications

**Done** (2026-10-07): [ADR-0025](./docs/adr/0025-notifications.md).

**Goal:** users learn about their work straight away without opening every list.

- ADR before writing code: `notification` module in `core`; list of events (documents awaiting my approval, my documents approved or rejected, my jobs finished or failed, being mentioned in a discussion); in-app notifications, and optional email when the tenant configures a mail server.
- Only job kinds that declare it send a notification when they finish or fail. Email-sending jobs and maintenance jobs (file cleanup, …) never declare it.
- Holders of `core.admin` see system jobs (including email sending) on the jobs screen, filterable by failure; errors show as codes, without recipient address or message content. Other users still see only their own jobs.
- Notifications are written in the same transaction as the event. Email is sent by a River job enqueued in that transaction; no Internet or a mail failure does not affect business operations.
- Notification icon in the header with an unread count; clicking opens the right record.
- Notifications contain no amounts or sensitive fields; email contains only a link.

**Done when:**

- Submitting a leave request notifies the first-step approver; once approved, the requester is notified; for a user who can no longer view the record, the notification no longer shows the document number or the actor, and clicking it only marks it read, revealing nothing.
- When the mail server is unreachable, the document still changes status, the email job retries per River and the error shows on the administrator's jobs screen.
- An email job succeeding or failing produces no further notification or email.
- Playwright test: submit and approve a leave request between two users; the unread count changes correctly.

## M13. PDF printing

**Done** (2026-10-08): [ADR-0026](./docs/adr/0026-printing.md). The two measurements the ADR defers — rendering a 3,000-employee payroll's payslips, and how much longer the payroll lock is held while snapshots are written — belong to the production-readiness gate.

**Goal:** payslips and contracts can be printed from the system.

- ADR before writing code: `printing` module in `core`; the PDF engine lives in the Go binary (no Chromium, no sidecar container); Vietnamese support and self-embedded fonts; print templates per record type, and how far tenants can edit them.
- **The print of a `posted` document never changes.** Print data is frozen when the document moves to `posted` (in `OnTransition`, same transaction); the template version is fixed at the first print; reprints always use exactly the fixed set. Documents posted before the printing module existed are frozen at their first print. Drafts print from current data, with a draft watermark. The frozen copy never expires; the PDF is only a temporary result that can be regenerated.
- Printing shares the extra per-record-type permission used by attachments (contracts and payslips need `hrm.salary.view`); every PDF download re-checks the current permission.
- Printing is a job on behalf of the user, re-checking permissions when it runs; the temporary file expires, like `dataio` export files.
- Proven by HRM: individual payslips (need `hrm.salary.view`, every print writes audit) and employment contracts.

**Done when:**

- A printed payslip matches the payroll line on screen and in the Excel export to the đồng.
- A user without permission to view salary cannot print a payslip, even by calling the API directly.
- Editing the employee name, the legal-entity information and the print template, then reprinting a `posted` contract: the content is identical to the previous print.
- Revoking `hrm.salary.view` after the PDF was created: downloading it again is refused, even by calling the API directly.
- Vietnamese diacritics display correctly in the PDF on a machine without Internet.
- Bulk printing the payslips of a 3,000-employee payroll runs as a job and does not block the request.

## M10. On-premise operations

**On hold** (2026-10-07); nothing is pulled forward ([ADR-0023](./docs/adr/0023-build-core-first.md)).

**Goal:** install, operate, back up and restore on the organisation's own server without the developers being present.

- Safe updates: a 5-step procedure, backup before migrating with a disk-space check ([tenancy.md](./tenancy.md#update-procedure-on-premise)).
- Minimal operations ([platform.md](./platform.md#operations)): daily backup including the file directory, a warning on the admin screen when the backup is older than 24 hours, heartbeat (the operator can turn it off), diagnostic bundle.
- Distribution: signed image, offline bundle, install script, install and handover documentation (including backing up the encryption keys).
- Initial import through `dataio`: employees, dependants and contracts currently in force (contracts take effect immediately, without approval), only while the legal entity has no closed pay period.
- Personal data: exporting all of the tenant's data is reserved for tenant administrators with a dedicated permission; the export file expires and writes audit.

**Done when:**

- A fresh install on a clean machine without Internet, using only the offline bundle (containing both the app and the Postgres image) and the documentation.
- Updating from the previous version to the M10 version on simulated data; a deliberately failing migration stops the app, leaves data unchanged, and the old version can run again.
- Restoring a backup on another machine: complete data, complete attachment files, and sensitive fields decrypt with the key backed up per the handover record.
- When the daily backup fails (e.g. disk full), or the latest one is older than 24 hours, the admin screen warns correctly; when a backup succeeds again the warning clears by itself.
- The heartbeat sends the right content, with no personal data; the operator can turn it off, and once off it sends nothing more.
- Exporting all of the tenant's data: every table of every module; a user without the dedicated permission cannot export.

## Production-readiness gate

**On hold** (2026-10-07) together with M10.

Not numbered; the checks before real data goes in. Real data enters the system only after M10 passes and this gate has been passed.

- Payroll calculation: an accountant reviews the payroll samples and their assumptions, and settles the open points ([products/hrm.md](./products/hrm.md#still-open)). Any fix changes both the samples and the payroll code.
- Performance: settle the reference configuration against the actual production server and measure.

**Reference configuration and performance targets.** Initial proposal, to be settled against the production server:

| Item | Value |
| --- | --- |
| Server | 4 vCPU, 8 GB RAM, SSD; app and Postgres on the same machine |
| Data | 3,000 employees; 12 closed pay periods; about 20,000 leave and overtime requests |
| Concurrent users | 50 |
| Open a list (filtered, paginated) | p95 under 500ms at the API |
| Open a document | p95 under 300ms at the API |
| Compute or recompute a 3,000-employee payroll | Under 2 minutes (background job) |
| Close a 3,000-employee payroll | Under 10 seconds (time during which source changes for the legal entity are blocked), including writing one print snapshot per line |
| Print a 3,000-employee payroll's payslips | Under 30 seconds (background job); peak memory under 512 MB |

**Passed when:**

- Every performance target in the table above is met on the settled configuration.
- The accountant confirms in writing that the payroll samples have been corrected per their feedback; the M6 acceptance test is green on those samples.
- The accountant confirms the two pending points: the default `line` rounding, and the general ledger not tracking payroll payables per employee ([ADR-0007](./docs/adr/0007-hrm-product.md)).

## Not in this roadmap

| Item | When to revisit |
| --- | --- |
| Custom fields | When a tenant needs something specific |
| Pay periods other than the calendar month, time clocks, salary payment through banks, electronic social insurance (BHXH điện tử) | Per [products/hrm.md](./products/hrm.md#scope) |
| Probe product (one document type, not released) to find HRM-specific assumptions still in the core | When M13 is done; the points found so far are in issues #1 to #15 |
| Accounting, a second product | When there is a real need for it ([README.md](./README.md#principles)) |
| Cloud model B | When operating cloud model A becomes overloaded ([tenancy.md](./tenancy.md#cloud-path-from-a-to-b)) |
| Product switcher | When there is a second product |
| Shared work page | When users need one place to gather work beyond the approval inbox |
| Timekeeping and payroll status on the HRM overview | When users ask for it; needs splitting by salary view permission |
| Org structure with effective dates and history (moving units by date, keeping old context) | When a tenant needs restructuring; needs its own ADR. Blocking documents from changing legal entity when a node moves already exists |
| Approval rules per legal entity or org unit | When a tenant needs different processes across legal entities |
| Computing the number of days of a leave request and the `day_kind` of an overtime request from the work calendar (still entered by hand) | When manual entry causes errors or users ask for it |
