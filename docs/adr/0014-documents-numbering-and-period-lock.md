# 0014. Documents table, document numbers and period lock

Status: accepted · 2026-10-05

## Context

ADR-0002 settled that `record` owns the document lifecycle; ADR-0012 only built record type registration. The leave request is the first document, so the table, numbering and period lock must be settled at code level.

## Decision

- `record.documents (id, doc_type, number, status, version, date, legal_entity_id, org_unit_id, amount, fields, approval_ticket, submitted_by)`. `id` is `record`'s identity, and **the module's table uses that same id as its primary key**, with a `DEFERRABLE INITIALLY DEFERRED` foreign key to `record.documents`. This lets `record.Create` run before the module's insert, and `record.Delete` before the module's delete, in the same transaction.
- A document's legal entity is derived from `org_unit_id` (nearest `company` ancestor, `iam.LegalEntityOf`) at `Create`, and never changes. An `Edit` that moves the org unit to another legal entity is refused.
- **Document numbers**: the `numbering` module (core) owns `numbering.counters (doc_type, legal_entity_id, year, last)`. `Create` increments the counter with `INSERT … ON CONFLICT DO UPDATE … RETURNING`, so the counter row stays locked until the end of the transaction. Numbers have the form `<prefix>-<year>-<5 digits>` (`NP-2026-00001`), with the prefix declared in `record.Type`. The numbering scope is type × legal entity × year of the document date, for every type; a type that needs another scope adds it when it arrives.
- **Create permission** is checked by the module itself before calling `Create`, since there is no id yet to ask `Can`. `Create` still goes through the product gate and the period lock.
- `Transition` accepts the targets `posted` (submit; the approval gate may move it to `pending_approval` instead), `draft` (withdraw: only the submitter, stored in `submitted_by`, and needs `Can(edit)`) and `cancelled` (needs `Can(cancel)`). `OnTransition` is called after every status change, except a move to `pending_approval`. Completing and rejecting an approval only go through `CompleteApproval` and `Reject`, called by `approval`.
- `record.Lock(ctx, ref)` locks in the common order (period lock `FOR SHARE` → document `FOR UPDATE`) for other core modules that need to lock their own rows after the document (`approval`).
- A document's `allowed_actions`: `draft` → `edit`, `delete`, `submit`; `pending_approval` → `withdraw`; `posted` → `cancel`; each action goes through `Can` and the product gate. A document whose date falls in a locked period has no write actions.
- One shared route `POST /documents/{type}/{id}/transitions {to, version}` for every document type; the history `GET /documents/{type}/{id}/history` comes from the audit log and needs `Can(view)`. Sensitive field values in the history are always masked; conditional display comes when a document type needs it.
- **Period lock**: `record.period_locks (legal_entity_id, locked_until)`. The row is created on demand (`INSERT … ON CONFLICT DO NOTHING` then `SELECT … FOR SHARE/UPDATE`). Permission `core.period.manage`, in the `core.admin` role. Moving the lock date forward is refused (`period_has_pending_documents`) while there are `pending_approval` documents dated no later than the new date, with the list returned. No `BeforeLock` hook yet; it comes with `accounting`.
- `iam` exposes a `TreeChanged` hook after every tree write; `record` refuses if any document's legal entity derived from its current org unit differs from the stored legal entity (completing the pending consequence of ADR-0010).
- Numeric approval fields are stored as decimal strings in `fields` and compared with `shopspring/decimal`; never `float`.
- A user without permission to view a document gets `not_found` on every write, so the document's status cannot be probed.

## Consequences

- Document ids are unique across all types; the frontend and core only need `(doc_type, id)`.
- Deleting drafts leaves gaps in numbering; accepted for HRM documents.
- The legal-entity check on tree writes scans every document; enough at the scale of small and medium businesses.
