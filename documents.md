# Documents and shared features

Customers of a business-management suite expect, by default, features such as approval, attachments, discussion, print templates, Excel import/export and custom fields on every document type. These features are provided once by modules in `core/` (`record`, `approval`, `attachment`, `discussion`, `printing`, `dataio`, `customfield`). Business modules only declare that they use them; they do not rebuild them.

## Record types

A module registers its record types with `record` at initialisation (see [backend.md](./backend.md#module-shape)):

- **Type code**: `<module>.<type>`, for example `sales.order`, `hrm.employee`.
- **Kind**: *document* (has a number, a date and a lifecycle) or *master data* (no lifecycle).
- **Number prefix** (`NumberPrefix`), for documents: document numbers have the form `<PREFIX>-YYYY-NNNNN`, for example `NP-2026-00001`.
- **Approval fields**, for documents: a short list (at most about 5) of fields used in approval conditions. Each field has a key, a kind (number, choice list, org unit) and a translation key for its label. For example, the leave request declares `days` and `leave_type`.
- **Callbacks**:
  - `Can(ctx, id, action)`: whether the user in `ctx` may perform `action` on record `id` of this type (the type is known at registration); other modules ask through `record.Can(ctx, type, id, action)`. `action` is one of `view`, `edit`, `post`, `cancel`, `export`, `view_files`, `attach`. An action the type does not answer is not allowed;
  - fetch the data for printing and export;
  - `OnTransition`, for documents: the module's work when the document changes status (see below);
  - `BeforeSubmit(ctx, doc)`, optional: runs when a `draft` document is submitted, before the approval gate is asked; if it returns an error the document stays `draft` (for example, a payroll whose source data has changed);
  - `Approvers(ctx, ref, level)`, optional: returns the approvers for a step of kind "by module" (see Approval below);
  - `Subjects(ctx, id)`, optional: the users the document is about (for example, the employee requesting leave); like the submitter, they may never approve that document.

Every shared feature attaches to a record through the pair `(doc_type, doc_id)`, with no foreign key to the module's tables. Core modules do not know the business tables, and always ask the owning module's `Can` before showing attachments, discussion or history, or allowing printing and export.

Each record type belongs to the product of the module that registers it. Every core shared route that touches a record (attachments, discussion, history, export, approval actions) goes through the same product gate as the module's own routes, by action class ([platform.md](./platform.md#enabled-products)). When the product is disabled: viewing history, downloading attachments and exporting still work; commenting, uploading attachments and approving do not.

### Attachments and discussion

Decision: [ADR-0024](./docs/adr/0024-attachments-and-discussion.md).

- `attachment` and `discussion` apply to every record type, both documents and master data. If a record cannot be viewed, every route returns not found, even when the id is guessed correctly.
- **Attachments** need two separate actions in `Can`: `view_files` (read class: list, download) and `attach` (write class: add, delete other people's files). A record type uses `view_files` to require an extra permission, for example the contract requires `hrm.salary.view`. The uploader can delete their own files. Each listing and download re-checks the current permissions.
- **Discussion** uses `view` to read, and `view` plus the write-class product gate to comment. Comments cannot be edited or deleted.
- `cancelled` documents: read-only. A locked period does not block attachments and comments, because they do not change figures; they also do not increase `version`.
- Deleting a draft: `record`'s `Deleted` hook deletes the attachments and discussion in the same transaction; files on disk are deleted later by `attachment`'s cleanup pass.
- Every attachment add, delete and download, and every comment, writes audit on the record.

### Permissions

- `record`'s `Edit`, `Delete` and `Transition` call `Can` themselves (below), so the module does not re-check the `edit`, `post`, `cancel` permissions in the service. **`Create` does not**: at that point there is no record yet to ask `Can` about, so `record.Create` only applies the product gate. The module must check the create permission itself before calling `Create` (for example, `hrm` checks `PermLeaveEdit` on the employee before creating a leave request).
- **Every API that returns a document includes `allowed_actions`**, computed by `record` from `Can`, the status, the period lock, the submitter and the product gate. The list only contains write actions: `edit`, `delete`, `submit` when `draft`; `withdraw` when `pending_approval` (submitter only); `cancel` when `posted`. A document dated in a locked period, or belonging to a disabled product, has an empty list. Export is not in this list. The frontend only renders buttons from this list and never derives it itself ([frontend.md](./frontend.md#permissions-the-frontend-computes-nothing)).
- Approval permission is decided by `approval`'s rules; the approver must still have `view`.
- **Sensitive fields** have their own view permission ([platform.md](./platform.md#personal-data)). Permission to view a document does not mean being able to see the values of its sensitive fields, including old values: the audit log stores sensitive field values encrypted, and the timeline always masks sensitive field values, for every user.
- **Reads using cross-schema joins**: the reading module is responsible for filtering by the user's permission scope, and must not select another module's sensitive columns. Sensitive columns are read only through the owning module's service.
- **Exporting data** with sensitive columns requires the permission to view that field. The export file is a temporary file with an expiry, and every export writes audit.

### Jobs: moving to the background does not raise permissions

- **System jobs** are the job kinds listed explicitly in code: posting reconciliation, temporary file cleanup, River maintenance. These jobs run as the system actor and skip `Can`.
- **Jobs on behalf of a user** are the default for every other job: import/export, bulk operations, and anything a user clicks that then runs in the background. The payload carries the requester's id. The worker runs with that same user as actor, and at run time re-checks `Can`, `Scope`, the sensitive-field view permission and the product gate for the job's class (export still runs when the product is disabled, import does not), instead of reusing the checks made when the job was created. If a permission is revoked while the job is waiting, the job fails with a permission error and reports back to the requester.

## Document lifecycle

Decision: [ADR-0002](./docs/adr/0002-record-owns-document-lifecycle.md).

### `record` status: in effect and editable

`record` keeps a table `record.documents (id, doc_type, number, status, version, date, legal_entity_id, org_unit_id, amount, fields, approval_ticket, submitted_by)`. The module's table reuses this `id` as the primary key of the document row. This is the source of truth for a document's status, date and legal entity. The status here only answers two questions: is the document in effect yet, and can it still be edited.

| Status | In effect | Editable |
| --- | --- | --- |
| `draft` | Not yet | Yes |
| `pending_approval` | Not yet | No |
| `posted` | Yes | No |
| `cancelled` | No longer | No |

Valid transitions, fixed for every document type:

```
draft ──► posted ──► cancelled
  │ ▲        ▲
  ▼ │        │
pending_approval   (approved → posted; rejected or withdrawn → draft)
```

A `draft` can be deleted, not cancelled. To change a `posted` document, cancel it or issue an adjusting document. Modules do not add statuses to this table.

### Progress status: owned by the module

Business progress (partly delivered, fully paid, in production…) is kept by the module that owns the document, in its own table, with its own names and meanings: for example `sales.orders.delivery_status`, `sales.orders.payment_status`. These statuses may change after the document is `posted`, because they are usually caused by other documents (goods issues, receipts). Two limits:

- Do not use the names `draft`, `posted`, `cancelled` or `pending_approval`.
- They do not decide whether the document is editable; only `record` decides that.

### `record` functions

Modules call the following functions **inside the service's transaction, before writing their own tables**.

**Lock order, the same for every operation:**

1. The legal entity's row in `record.period_locks (legal_entity_id, locked_until)`: every document write takes `FOR SHARE`; only the operation that moves the lock date takes `FOR UPDATE`.
2. The document's row in `record.documents`: `FOR UPDATE`. `Create` has no document row yet: it locks the `numbering` counter row (`FOR UPDATE`), then inserts the document row.
3. Only then the rows the module locks in `OnTransition` (for example, the leave balance).

The lock date is always read from the row just locked in step 1, never earlier. Document writes for the same legal entity do not block each other (both are `FOR SHARE`). Thanks to the lock in step 2, operations on the same document run one after another.

A document's legal entity cannot change after `Create`, so each operation locks exactly one row in step 1.

A core module that needs to lock its own rows after a document calls `record.Lock(ctx, ref)`: this function takes the step 1 and step 2 locks and returns the document, with no further checks. For example, `approval` calls it before locking its row in `approval.instances`.

**Each transaction writes exactly one document.** Bulk operations (approving many requests, importing many documents) are jobs that process one document at a time, one transaction per document. As a result, a transaction never locks two rows at the same step, and no lock-ordering convention between documents is needed.

**The document number** is assigned in `Create`, from the `numbering` counter for the fixed scope `(doc_type, legal_entity_id, year of the document date)`, in the form `<PREFIX>-YYYY-NNNNN`. The counter increases in the same transaction, so a rollback does not lose a number; only deleting a draft leaves a gap. A document type that requires gapless numbers (for example, invoices) settles its own numbering in the owning module's ADR.

| Function | Checks | Effect |
| --- | --- | --- |
| `Create(ctx, type, header)` | Product gate, period lock on the date; does **not** call `Can` (the module checks the create permission first) | Assigns the number, inserts a `draft` row, `version = 1`, returns the id |
| `Edit(ctx, ref, version, header)` | `Can(edit)`; status `draft`; `version` matches; period lock on **both the old and the new date**; legal entity unchanged (if changed: error `legal_entity_changed`) | Updates the header, increases `version` |
| `Delete(ctx, ref, version)` | as `Edit` | Deletes the row |
| `Transition(ctx, ref, version, to)` | `Can(post/cancel)`; valid transition; `version` matches; period lock | See below |

`header` holds the date, org unit, amount and the values of the declared approval fields. The module does not pass the legal entity: `record` derives it from the org unit (`iam.LegalEntityOf`). `record` checks that values have the right kind, rejects undeclared fields, then stores them in the `fields` column (jsonb). `version` is sent by the client and is the version the user is looking at. If it does not match, a conflict error is returned and the user must reload the document.

`Transition` runs in this order:

1. The checks in the table above, based on data read from `record.documents`, not on data passed in by the module. When submitting a `draft` document, it also calls `BeforeSubmit` if the type declares it; on error the document stays `draft`.
2. Ask the approval gate. If approval is needed, move to `pending_approval` and stop.
3. Write the new status, increase `version`, write the audit log.
4. Call the document type's `OnTransition`, in the same transaction. The module does its work here (deducting stock, writing business lines…). An error in `OnTransition` rolls back the whole transition; this is intended.

Because of the row lock and the `version` check, if two requests post the same document, only one of them runs `OnTransition`. The other gets a conflict error.

**The approval gate** is an interface defined by `record` in its `hooks.go`; `approval` is the adapter, and `internal/app` wires them. `record` does not import `approval`.

Each submission is a separate **approval instance**, with an id:

- When the gate answers "approval needed", `approval` creates an approval instance (`approval.instances`), records the submitted `version`, and returns the id. `record` stores this id in `record.documents.approval_ticket`.
- **Withdrawn** (by the submitter, needs `Can(edit)`) or **rejected**: the document returns to `draft`, `approval_ticket` is cleared, `version` increases, and that approval instance is closed. On rejection, `approval` calls `record.Reject(ctx, ref, ticket, version)`, with the same checks as `CompleteApproval` below.
- **Completed**: when the last approval step is approved, `approval` calls `record.CompleteApproval(ctx, ref, ticket, version)` (`version` is the version at submission), in the same transaction that writes the approval decision. `record` locks in the order above and checks, then runs steps 3 and 4 of `Transition`. This step does not call `Can(post)`, because approval permission is decided by `approval`'s rules. On failure, handling depends on the kind of error:

  | Kind of error | Example | Handling |
  | --- | --- | --- |
  | The approval instance is stale | The document is no longer `pending_approval`; `ticket` does not match; `version` differs from the submitted `version` | Reject and **close** that approval instance |
  | Any other error | Period lock; an error in `OnTransition` (not enough leave days, payroll sources changed); a DB error | **Full rollback**: the last-step approval decision is not written, the approval instance stays open at the last step, the document stays `pending_approval`. The approver gets the error code |

  After an error of the second kind, the approver can approve again once the conditions are met, or reject; the submitter can withdraw to edit.
- Each step-level approval action in `approval` also checks that the approval instance is still open. A late approval action on an old submission is rejected, even when the document is waiting for approval on a new submission.

**Period lock:** each legal entity has a lock date. Documents dated on or before the lock date cannot be created, edited, deleted or transitioned. The period lock applies even when the tenant does not use the accounting product.

Moving the lock date: `record` takes `FOR UPDATE` on the legal entity's row, checks for documents pending approval (below), then writes the new date, all in one transaction. `FOR UPDATE` must wait for every transaction holding `FOR SHARE` on that row, and blocks new transactions until it commits. Consequences:

- A document operation that started earlier commits first; the period lock checks see its result.
- A document operation that starts later waits for the period lock to commit, then reads the new lock date and is rejected if the document falls in the newly locked period.

There is no `BeforeLock(legal_entity, date)` hook yet for other modules to refuse a period lock; this hook is added together with `accounting` (see Posting).

**No period lock while documents are pending approval.** `record` itself refuses to move the lock date if the part of the period about to be locked still contains `pending_approval` documents, and lists them. If the lock were allowed, these documents would be stuck: they could not finish approval because the period is locked, and could not be withdrawn because withdrawing is a write in a locked period. This check runs while holding `FOR UPDATE` on the legal entity's row, so no document can move to `pending_approval` in that period between the check and the commit.

**Approvers by module** (`Approvers`):

- `approval` defines the interface; record types that need it implement it. `approval` knows nothing about "direct manager" or any other business relationship.
- Signature: `Approvers(ctx, ref, level) ([]int64, error)`. `level = 1` is the nearest level (for example, the direct manager), `level = 2` is the level above that, and so on. The result depends only on the parameters, not on how many times it is called. A step is approved by anyone in the list.
- **Approvers are fixed when the step starts** and stored in the approval instance. A manager changing midway does not change the approvers of the open step.
- **No self-approval:** `approval` removes the submitter, the people in `Subjects` and people without `view` permission on the document from the result. If the list is still empty it calls again with `level + 1`, up to the number of levels declared in the rule (default 3).
- **No approver found:** the step goes to the fallback role declared in the rule (filtered the same way). If still nobody is left, the step cannot start and the operation (submitting, or approving the previous step) fails with the error `no_approver` (`ErrNoApprover`). Never auto-approve.
- **Reassigning the approver:** a person with the rule's fallback role can reassign the approver of the open step, for example when the approver has left the company. Each reassignment writes audit. The submitter can also withdraw and resubmit so the approvers are recomputed.

### Guarantees

No tool can statically check whether a module calls `record` before writing its own tables. Instead, `record` provides the `recordtest` contract suite, and every document type must pass it in CI. The suite checks that:

- editing a `posted` or `pending_approval` document through the service is rejected;
- editing a document dated in a locked period, or moving its date into a locked period, is rejected;
- two concurrent `post` calls run `OnTransition` only once;
- a stale `version` is rejected;
- an approval action on a withdrawn submission is rejected;
- a completed approval that fails because of an error in `OnTransition` leaves the document `pending_approval` and the approval instance open;
- the lock date cannot be moved while the period still has `pending_approval` documents;
- document writes running concurrently with a period lock never leave a new document in the locked period.

## Shared features

| Feature | Scope | Deliberate limits |
| --- | --- | --- |
| **Approval** | A rule is a chain of steps. Each step has a simple condition on the header or on a declared approval field (amount > X, org unit = Y, `days` > 3, `leave_type` = unpaid leave) and an approver. Approvers are by role at an org unit, by a specific user, or **by module** (returned by the module through `Approvers`, for example the direct manager). Customers configure it themselves. | No parallel branches, loops, scripts or BPMN |
| **Attachments** | Files attached to a record. Metadata is stored in the DB; file contents are stored on disk, in `attachment`'s own directory | The file directory must be within backup scope. Write the file before writing the metadata. Files are deleted only by the cleanup job, when no longer referenced and older than the backup cycle, so a DB backup never points to a lost file |
| **Discussion and history** | Comments, mentions, a timeline of status changes and field changes (taken from the audit log, filtered by field view permission) | No real-time chat |
| **Notifications** | In-app notifications (approval needed, mentioned). Email and Zalo are sent through integration jobs | — |
| **Print templates** | Each document type has a default template shipped with the module. Customers can edit templates; edited templates are stored in the DB. The rendering and PDF export engine is settled by an ADR | Templates only display data, they do not run code |
| **Excel import/export** | The module declares the columns. `dataio` handles reading, writing, validating data and reporting errors per row. Imported data is always written through the owning module's service | — |
| **Custom fields** | Field definitions (record type, key, kind, label, required, choice list) are stored by `customfield`. Values are stored in a `custom jsonb` column on the module's table (the module adds this column if it wants to support them). Custom fields appear on forms, lists, filters, print templates and import/export | Customers cannot create new tables or record types |

Each feature is only built when the first module needs it. But once built, it must be built for every record type to share, not just for one module.

## Posting

Decision: [ADR-0003](./docs/adr/0003-business-lines-are-data.md). Accounting is an **optional product**. Tenants that do not enable accounting can still use every other product in full.

- Business modules never write to the general ledger and know nothing about accounting accounts. For each document, a module only describes **business lines**: line kind (revenue, cost of goods sold, tax, payment…), amount, counterparty, org unit.
- **Business lines are data, captured exactly once.** The `internal/shared/posting` module owns the `posting.lines` table and the line kinds. In `OnTransition` to `posted`, the module calls `posting.Record(ctx, ref, legalEntity, date, lines)` with the figures at that moment (cost of goods sold, tax rate…). On moving to `cancelled`, the module calls `posting.Void(ctx, ref)`: this marks the lines as voided, without deleting them. Both run in the status transition's transaction, so for document types that produce business lines, a `posted` document always has business lines. The record type does not declare this to `record`; the owning module calls `posting` itself. Document types that produce no business lines (leave requests, contracts…) do not call `posting`.
- **Business lines contain no sensitive data tied to an individual.** `posting.lines` and the general ledger are not encrypted, and accounting reads them with SQL. The module must aggregate such lines, for example salaries payable aggregated by department. Per-person detail stays in the owning module's tables, encrypted.
- Business lines are written even when the tenant has no accounting. `record` knows nothing about posting.
- `posting` exposes the hook `posting.Hooks{OnChanged func(ctx, ref) error}`, which runs in the transaction that wrote or voided the lines; a `nil` hook is a no-op. `accounting` (in `modules/`, only running when the accounting product is enabled) implements the hook by enqueueing a `(doc_type, doc_id)` job in the same transaction. Without accounting the hook stays `nil`.
- **Account mapping** (from line kind to account) is done by `accounting` using configurable rules, with defaults following Circular 200 (TT200) and Circular 133 (TT133).

### Posting jobs: reconcile, do not replay

A job carries no change content, only `(doc_type, doc_id)`. The worker always brings the general ledger in line with the **current state** of `posting.lines`:

1. Take `FOR SHARE` on the legal entity's period-lock row (the same lock order as `record`), then lock the document's row in `accounting`'s tracking table (`FOR UPDATE`). Never write journal entries into a locked period, except for the first posting run when accounting is enabled.
2. Read the current `posting.lines`. If they are in effect, ensure exactly the matching journal entries exist; if voided, ensure the journal entries have been removed. If already in line, do nothing.

So job order does not matter. If the cancellation's job runs before the posting's job, the cancellation job has already brought the ledger to the "cancelled" state, and the posting job that runs later finds the ledger already in line and does nothing. A failed job retries; running it any number of times gives the same result.

### Period lock and re-posting

- **No period lock while documents are not yet posted to the ledger.** `accounting` implements `BeforeLock` (added to `record` at the same time), and refuses to move the lock date if the period still has documents whose `posting.lines` do not match the general ledger (a job waiting or failing). An admin screen lists those documents.
- **A locked period is never re-posted automatically.** Changing account mapping rules does not affect existing journal entries. To apply new rules, the user runs "re-post" for a date range, and that range must lie entirely in the open period.
- **Tenants that enable accounting later:** when it is enabled, `accounting` posts all `posting.lines` from the start-of-use date, without calling the business modules again. This is the only exception allowed to write into a locked period, because the general ledger was empty before.
- The general ledger is kept separately per legal entity (see [backend.md](./backend.md#org-units)).

## Sample flow: leave and payroll

This flow validates the core with the first product. Every step must be covered by `recordtest`, `hrm`'s tests and the frontend–API end-to-end tests ([techstack.md](./techstack.md#testing-and-quality)).

Setting: legal entity C, department P, employee E with a leave balance of 12 days. Approval rule: every leave request needs approval by the head of department P.

| # | Action | What happens |
| --- | --- | --- |
| 1 | E creates a 2-day leave request from 10/03 | `hrm` opens a transaction → `record.Create`: `FOR SHARE` on C's period-lock row, checks the date 10/03 → writes `hrm.leave_requests`. `draft`, `version = 1` |
| 2 | E edits it to 3 days; another tab of E also edits it, sending `version = 1` | The first succeeds, `version = 2`. The second waits for the row lock, sees `version = 2` → conflict error |
| 3 | E submits for approval | `Transition(→posted)` → the gate creates approval instance I1 with `version = 2` → `pending_approval`, `approval_ticket = I1`. From here on every edit is rejected |
| 4 | E withdraws, edits it to 5 days, resubmits | Back to `draft`, I1 closed, `version = 3` → edit, `version = 4` → submit: approval instance I2 with `version = 4` |
| 5 | The head of department clicks approve on an old screen for I1 | `approval` sees I1 is closed → rejected. Even if it slipped through, `record.CompleteApproval` sees the ticket does not match I2 → rejected |
| 6 | The head and deputy head of department both approve I2 | The first: `posted`, `OnTransition` locks the leave balance row, 12 → 7. The second waits for the row lock, sees `posted` → error. The leave balance is deducted only once |
| 7 | Another 8-day leave request of E is approved at the same time | `OnTransition` waits for the leave balance row lock, sees 7 left → non-negative constraint → rollback. The request stays pending approval; the approver gets the error "Không đủ ngày phép" ("not enough leave days") |
| 8 | HR imports P's March timesheet from Excel | Job on behalf of a user: runs with HR as actor, re-checks permissions at run time. Timesheet `draft` → approved → `posted` |
| 9 | Prepare C's March payroll | Needs a `posted` timesheet for every org unit with employees in the payroll. Captures the inputs together with the version of each source document. `draft` |
| 10 | A March overtime request is approved afterwards | Opening the payroll: warning "Dữ liệu nguồn đã thay đổi" ("source data has changed"). Submitting is rejected until the user clicks "Tính lại" ("Recalculate") |
| 11 | Recalculate, submit, approve | `posted` → `OnTransition`: `posting.Record` with lines aggregated by department, with no per-person amounts → posting reconciliation job |
| 11' | After the payroll is `posted`, the head of department approves another March overtime request | The overtime request's `OnTransition` sees the March pay period is closed → rollback. The request stays pending approval, the approval instance stays open; the approver gets the error `payroll_period_closed` |
| 12 | Accounting locks March at the same moment HR cancels a March leave request | If the cancellation gets the lock first: the period lock waits for the cancellation to commit, then checks and writes the lock date. If the period lock gets the lock first: the cancellation waits, then reads the new lock date → rejected |
| 12' | Accounting locks March while the overtime request from step 11' is still pending approval | Rejected, with the list of documents pending approval in the period. The submitter withdraws or the approver rejects first, then the lock goes through |
| 13 | HR tries to cancel the March timesheet | Rejected: the period is locked, and the timesheet is referenced by a `posted` payroll |
