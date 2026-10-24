---
name: new-document-type
description: Add a document type (a record with number, date, legal entity and lifecycle) to a module. Use when a module needs draft/posted/cancelled records, approval, or posting lines.
---

# New document type

The rules live in `documents.md`; this skill is the order to apply them in. Each step ends on its completion criterion.

1. **Specify.** Read `documents.md` (Record types, Document lifecycle, `record` functions, Posting) and the product doc for this type (e.g. `products/hrm.md`).
   Done when: you can state, for this type, what `posted` means, the date used for the period lock, any affected date range for module-level locks, its approval fields (at most about five, typed), its `OnTransition` effects, and whether it produces posting lines.

2. **Register.** Register the type with `record`: kind, product, `NumberPrefix`, approval fields, the permission declaration `Perms` (at least `View`; `Edit` covers `post`, `cancel` and `create` unless declared apart), `OnDelete` (deletes the module's rows of a deleted draft), `OnTransition`; optionally a relation grant `Grant` (users related to the record, such as the employee's own account; `SubjectCreates` lets the subject named in `Create`'s header create), a refine hook `Refine` (only narrows, e.g. a further sensitive permission), `BeforeSubmit` (rejects sending a draft, e.g. stale sources), `Approvers(ctx, ref, level)` when a step needs module-chosen approvers, and `Subjects(ctx, id)` for users the document is about, who must never approve it.
   Registration panics on a declaration without `View`, a permission no role of the product grants, or a missing `OnDelete`.
   If the type takes attachments, declare `ViewFiles` (read) and `Attach` (write); refine `view_files` when a file may show what a sensitive field or permission guards, as contracts ask `hrm.salary.view`. Undeclared, the type has no attachments; discussion needs nothing beyond `view`. `print` and `export` follow `view` and appear once a print template or an export is registered for the type.
   A type that goes through approval gets its notifications (waiting for approval, approved, rejected) from `approval` with nothing to add.
   Done when: the type appears in `record`'s registry at startup.

3. **Write path.** Every service write calls `record.Create`, `Edit` or `Transition` first, inside the same `InTx`, then writes the module's tables. Drafts are deleted only through the core route `DELETE /documents/{type}/{id}`, which calls the type's `OnDelete` to delete the module's rows; the module has no delete route. `record.Create` checks the declared `create` permission at the header's org unit (or lets the header's `Subject` create when the type sets `SubjectCreates`), so the module does not check it again. Module rows are locked after the `record` rows, following the lock order in `AGENTS.md`.
   A module-owned status column (e.g. delivery or payment progress) changes only through one transition function, and its column has a `CHECK`.
   Done when: every service method that writes this type's tables has a preceding `record` call in the same transaction, and tests cover each forbidden module-status transition.

4. **Posting.** If the type produces business lines, call `posting.Record` on `posted` and `posting.Void` on `cancelled` inside `OnTransition`. Aggregate any per-person sensitive amount before it reaches `posting`.
   Done when: posting lines carry no per-person sensitive amount.

5. **API.** Every response that returns a document includes `allowed_actions` from `record.Actions` (passing the module's own extra actions); the list is a `data.List` over a `-- data:base` query, scoped by the document's `org_unit_id` through `iam.DataScope`, and its `create`/`import` come from `record.TypeActions`; document lines are written in one `unnest` statement; failures return stable error codes with parameters.
   Done when: the handler tests assert `allowed_actions` for each status.

6. **Prove it.** Run `recordtest` for the type, and add tests for the module's own concurrent cases (e.g. two approvals racing on the same balance row).
   Done when: `recordtest` and the module tests are green under `make test`.

7. **Frontend.** Add the type to the area manifest's `recordTypes` with `path`, `invalidate`, and `preview` when it goes through approval. Add status label overrides under `<doc_type>.status.*` in the area's `meta` translations. Build the screen with the `new-screen` skill using `DocumentPage`: the values as `EditablePanel` cards edited in place while `allowed_actions` has `edit` (`fullWidth` with the form instead only for a wide table), `ApprovalDecision` in `actions`, `approval={<ApprovalPanel … decideElsewhere />}`, `activity={<ActivityFeed docType={…} id={…} />}` and, when the type declares `ViewFiles`, `sections` from `useAttachmentSection`.
   Done when: the document opens from the approval inbox, and approving it refreshes every query listed in `invalidate`.

8. **Name it.** Add the type and any new status meaning to `CONTEXT.md`.
