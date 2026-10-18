# 0034. Record types declare their permissions; the core answers `Can`, deletes and creates once

Status: accepted · 2026-10-18 · amends [ADR-0012](./0012-record-type-registry.md) (the module's `Can`), [ADR-0014](./0014-documents-numbering-and-period-lock.md) (create permission checked by the module; `record.Lock` for core modules), [ADR-0024](./0024-attachments-and-discussion.md) and [ADR-0026](./0026-printing.md) (a type opts into attachments and printing by answering `Can`). The lock order of [ADR-0002](./0002-record-owns-document-lifecycle.md) and [ADR-0029](./0029-withdraw-right-and-tree-lock.md) is unchanged, and `record` is still called before the module writes its own rows.

## Context

Every record type answers `Can` by hand, and every module rebuilds the same layer around `record`. A survey of HRM and Sales found eight hand-written `Can` functions, seven "load if visible, else not found" helpers, the same create/edit/delete transaction shell per type, `allowed_actions` assembled three different ways (`Get`, then `DocumentActions`, then `print` and `export` appended by hand), a delete route per document type while every other transition uses the shared route, and create-once by `request_id` in Sales only.

Several rules that keep documents correct live only in convention:

- `Can` must answer false for an id of another type (Sales checks the document kind by hand; nothing forces the others).
- A type that forgets to answer `view_files`, `attach` or `print` silently loses the feature. Nobody can tell from the code whether the payroll's missing attachments are deliberate.
- The module must check the create permission before `record.Create`, because there is no id to ask `Can` about.
- `documents.md` reserves `record.Lock` for core modules, `backend.md` lists it for modules, and Sales already calls it.

Most of the hand-written answers are the same shape: one permission per action, checked at the record's org unit. The exceptions are few and real:

- Contracts also need `hrm.salary.view` to edit, see files or print; employee profiles need `hrm.employee.sensitive` for files; payrolls need `hrm.salary.view` to print.
- Leave and overtime requests are self-service: the employee's own account may view, edit and send its requests with no permission at all, and the employee's managers may view them.

## Decision

### The permission declaration

- `record.Type` declares, for each action, the permission that grants it: `view`, `edit`, `post`, `cancel`, `export`, `print`, `view_files`, `attach`, and the actions before a record exists, `create` and `import`.
- Defaults: `export` and `print` follow `view`; `post` and `cancel` follow `edit`; `create` follows `edit`. `view_files`, `attach` and `import` have no default: undeclared means the type has none.
- `print` is available only when a print template is registered for the type, `export` only when an export is registered for it. The permission alone grants nothing.
- Registration panics at startup on an inconsistent declaration: no `view`; a permission that no role of the type's product registers; document-only fields (number prefix, approval fields, `OnTransition`) on a catalog; a catalog without a locate hook.

### How the core answers

`record.Can(ctx, type, id, action)` evaluates, in order:

1. The type exists. For a document, the row in `record.documents` exists and its `doc_type` is the requested type; otherwise the answer is false (not found to the caller). A catalog has no such row: the type's **locate hook** returns the record's org unit, or not found.
2. The declared permission holds at the record's org unit (from `record.documents`, or from the locate hook) in the actor's scope (`iam`). Payroll stores its legal entity as its org unit, so its legal-entity scope works unchanged.
3. If the permission does not hold, the type's optional **relation grant** may still grant the action to a user related to the record: the employee's own account, the employee's managers. It is asked only for actions the type declares.
4. The type's optional **refine hook** receives the action and id after steps 2 and 3 granted it, and may only narrow the answer. This is where the contract's and the payroll's `hrm.salary.view` and the employee profile's `hrm.employee.sensitive` live.
5. The product gate by the action's operation class, as before.

A hand-written `Can` keeps working for types not yet moved, and is removed once every type declares.

`Create` has no id, so the relation grant cannot ask about the record. The module names the subject of the new document in the `Create` call (the employee's account, read from its own data); when the type's relation grant answers `create` for subjects and the actor is that subject, `create` is granted. Otherwise `Create` checks the declared `create` permission at the header's org unit. The module no longer checks the create permission itself.

### One-call reads

- **A visible document**: returns the header of a document the actor may view, or not found. Replaces the per-type visibility helpers.
- **A document's `allowed_actions`**: the lifecycle actions by status (`edit`, `delete`, `submit`; `withdraw` for the submitter; `cancel`), the period lock, the product gate, `print` and `export` when available, plus extras the module passes in for actions only it knows (the quotation's `create_order`). Replaces `AllowedActions`, `DocumentActions` and the hand-appended actions.
- **Actions before a record exists** (`create`, `import`): the declared permission in any unit of the actor's scope, then the product gate by class. Replaces the hand-rolled `scope.Any() && ProductGate(…)`. Self-service entry points (the employee's own requests) stay module calls, because they depend on the actor's own employee profile.

### Writes

- **Delete** goes through the core route `DELETE /documents/{type}/{id}?version=`: `record.Delete`, then the type's **delete hook** (the module deletes its own rows), then the `Deleted` hooks of `attachment` and `discussion`, in one transaction. The module delete routes for documents go away. Catalog deletes and deletes of non-records (dependants, calendar weeks, holidays) stay module routes.
- **Create once**: `Create` takes an optional `request_id`. A retry with the same `request_id` by the same user returns the existing document instead of a new one, and the module skips its own insert. The uniqueness is a constraint in `record.documents`, so every type gets it; the Sales copy goes away.
- **Touch**: locks a draft and increases its `version` without changing the header, with the checks of `Edit`. For module writes that change only child rows (payroll adjustments), so they stop re-sending an old header to `Edit`.
- `Edit` still writes no audit entry. The module audits content changes, because the field diff is business knowledge.

### `record.Lock` is module-facing

`record.Lock(ctx, ref)` takes the period-lock row (`FOR SHARE`) and the document row (`FOR UPDATE`) in the shared order and returns the document, with no other check. Any module may call it, under three rules:

- The caller has already checked the actor may see the document (the visible read above), since `Lock` checks nothing.
- It never changes the locked document: a header change is `Edit`, a child-row-only change on a draft is `Touch`.
- It serialises a decision that depends on the locked document's state, such as Sales returning the existing live order of a quotation rather than making a second one. Within the one-document-per-transaction rule, the locked document is only read; the one document written is the new or edited one.

Evidence: `backend.md` already lists it for modules; Sales' "create order from quotation" calls it, and `products/sales.md` requires that quotation row lock; `approval` and `printing` use it the same way. The "core modules only" wording in `documents.md` predates Sales. A narrower Sales-only call would take exactly the same locks.

### Payroll has no attachments

`products/hrm.md` lists attachments only on contracts, leave and overtime requests and employee profiles; payrolls and timesheets have none. A payroll covers every employee of a legal entity, and a file on it (a bank transfer list, a signed summary) would carry per-person amounts that its view permission does not cover. So the payroll declares no `view_files` or `attach`. The difference is now a visible line in its declaration instead of a missing `case` in a `Can` function. Payroll declares `print` (the payslip template), refined by `hrm.salary.view`.

## Rejected alternatives

- **Keep hand-written `Can` and check it with the contract suite**: the suite only catches what it tests, and eight copies have already drifted (print appended three ways, the cross-type check in one module).
- **One hook that may flip the declared answer either way**: it is a hand-written `Can` again, with no rule left that the core can rely on.
- **Only a narrowing hook, as first proposed**: cannot express self-service. An employee viewing their own leave request has no permission to narrow from.
- **Several permissions per action instead of a refine hook**: covers the salary and sensitive rules but not self-service, which still needs code; one hook per direction keeps both explicit.
- **The module keeps checking the create permission**: the one write the core does not check is the one a new type most easily forgets.
- **`request_id` kept in each module's table**: every type would copy the Sales code and its constraint.
- **Delete stays a module route**: delete is the only lifecycle write with its own route per type, and the only one whose frontend wiring differs per page.
- **`record.Lock` for core modules only**: Sales would need a second call taking the same locks, and nothing would be safer.

## Consequences

- A document type is a declaration plus its business rules and its own rows; its `Can`, visibility helper, create-permission check, delete route and `allowed_actions` assembly go away.
- Answers no longer load the module's row: the org unit comes from `record.documents`. The relation grant and refine hook still read module data, only for the types that have them.
- Startup fails on a type whose declaration contradicts its kind or names an unknown permission.
- Deleting a document is an API change: `DELETE /documents/{type}/{id}`, with the frontend and API types updated in the same change.
- HRM document creation becomes idempotent by `request_id`, like Sales.
- The `recordtest` contract suite checks the declared behaviour (cross-type ids, not found without view, print implies view, attachments only where declared) instead of trusting each `Can`.
