---
name: new-document-type
description: Add a document type (a record with number, date, legal entity and lifecycle) to a module. Use when a module needs draft/posted/cancelled records, approval, or posting lines.
---

# New document type

The rules live in `documents.md`; this skill is the order to apply them in. Each step ends on its completion criterion.

1. **Specify.** Read `documents.md` (Record types, Document lifecycle, `record` functions, Posting) and the product doc for this type (e.g. `products/hrm.md`).
   Done when: you can state, for this type, what `posted` means, the date used for the period lock, any affected date range for module-level locks, its approval fields (at most about five, typed), its `OnTransition` effects, and whether it produces posting lines.

2. **Register.** Register the type with `record`: kind, product, `NumberPrefix`, approval fields, `Can(ctx, id, action)`, `OnTransition`; optionally `BeforeSubmit` (rejects sending a draft, e.g. stale sources), `Approvers(ctx, ref, level)` when a step needs module-chosen approvers, and `Subjects(ctx, id)` for users the document is about, who must never approve it.
   If the type takes attachments, `Can` also answers `view_files` (read; ask for more than `view` when a file may show what a sensitive field or permission guards, as contracts ask `hrm.salary.view`) and `attach` (write). An action `Can` does not answer is refused, so a type without them has no attachments; discussion needs nothing beyond `view`.
   A type that goes through approval gets its notifications (waiting for approval, approved, rejected) from `approval` with nothing to add.
   Done when: the type appears in `record`'s registry at startup.

3. **Write path.** Every service write calls `record.Create`, `Edit`, `Delete` or `Transition` first, inside the same `InTx`, then writes the module's tables. `record.Create` applies only the product gate, not `Can`: check the actor's create permission in the module before calling it (e.g. `hrm` checks `PermLeaveEdit` on the employee). Module rows are locked after the `record` rows, following the lock order in `CLAUDE.md`.
   A module-owned status column (e.g. delivery or payment progress) changes only through one transition function, and its column has a `CHECK`.
   Done when: every service method that writes this type's tables has a preceding `record` call in the same transaction, every create path checks permission first, and tests cover each forbidden module-status transition.

4. **Posting.** If the type produces business lines, call `posting.Record` on `posted` and `posting.Void` on `cancelled` inside `OnTransition`. Aggregate any per-person sensitive amount before it reaches `posting`.
   Done when: posting lines carry no per-person sensitive amount.

5. **API.** Every response that returns a document includes `allowed_actions` from `record`; failures return stable error codes with parameters.
   Done when: the handler tests assert `allowed_actions` for each status.

6. **Prove it.** Run `recordtest` for the type, and add tests for the module's own concurrent cases (e.g. two approvals racing on the same balance row).
   Done when: `recordtest` and the module tests are green under `make test`.

7. **Frontend.** Add the type to the area manifest's `recordTypes` with `path`, `invalidate`, and `preview` when it goes through approval. Add status label overrides under `<doc_type>.status.*` in the area's `meta` translations. Build the screen with the `new-screen` skill using `DocumentPage`, passing `sections` from `useDiscussionSection`, plus `useAttachmentSection` when `Can` answers `view_files`.
   Done when: the document opens from the approval inbox, and approving it refreshes every query listed in `invalidate`.

8. **Name it.** Add the type and any new status meaning to `CONTEXT.md`.
