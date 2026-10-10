# 0029. Withdrawing needs the right that sent; creating a document holds tree moves off

Status: accepted · 2026-10-10 · amends [ADR-0014](./0014-documents-numbering-and-period-lock.md) ("withdraw … needs `Can(edit)`") and extends the lock order of [ADR-0002](./0002-record-owns-document-lifecycle.md)

## Context

Checking the core against HRM and Sales turned up two gaps in `record`:

- A contract answers `edit` only with salary access, since editing shows the terms, but sending and cancelling need only `hrm.contract.edit`. HR without salary access, and the tenant administrator (salary access is sensitive, [ADR-0028](./0028-tenant-admin-holds-business-permissions.md)), could send a contract and then not withdraw it: only the approver could unblock it.
- `Create` and `Edit` read the org unit's legal entity under no lock a tree write conflicts with. A move of the unit to another legal entity committing alongside left the document under the old one: the tree's "documents stay in their legal entity" check could not see the uncommitted document, and every later edit failed with `legal_entity_changed`.

## Decision

- **Withdraw needs `Can(post)`**, the right that sent the document, not `Can(edit)`. Still only the submitter withdraws. For every type except contracts the two answers are the same.
- **`Create` and `Edit` call `iam.ShareTree` before reading the legal entity**: a `SHARE` lock on `iam.org_units`, which only tree writes (`SHARE ROW EXCLUSIVE`) conflict with. Document writes do not block each other; a tree move waits for documents being written, and they wait for it. Tree writes take no period, document or module lock, so the lock order of ADR-0002 gains no cycle.

## Rejected alternatives

- **Contracts answering `edit` without salary access when the document is pending**: `Can` would depend on status, which every other type keeps out of `Can`.
- **An advisory lock shared by documents and tree writes**: the same effect with a key to keep in step in two modules; the table lock names what it protects.
- **Locking the document's org-unit row**: a move updates an ancestor, not the unit, so it would not conflict.

## Consequences

- A tree move waits while a payroll is being computed (its `Edit` holds the lock until the lines are written), and new document writes queue behind the move. Tree moves are rare administrator actions.
