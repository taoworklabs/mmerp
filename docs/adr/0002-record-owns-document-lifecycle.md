# 0002. `record` owns the validity and editability of documents

Status: accepted · 2026-10-05

## Context

The first design had `approval` import `record`, while `record`'s status transition function started approval: a dependency cycle inside `core`. The first fix only routed status transitions through `record`; creating and editing draft documents was still done by each module on its own. As a result, "do not edit a document under approval", "do not edit in a locked period" and protection against double posting were still implemented differently by each module. That version also wanted a single shared status chain to express business progress as well (partly delivered, partly paid), and could not achieve it.

## Decision

- `record.documents` is the source of truth for a document's status, `version`, date and legal entity. The only statuses are `draft`, `pending_approval`, `posted`, `cancelled`, with a fixed transition table. This status only expresses the document's validity and editability.
- Business progress is the module's own status, with its own name, in the module's table.
- Every write on a document (`Create`, `Edit`, `Delete`, `Transition`) calls `record` before the module writes its own tables, in the same transaction. `record` locks the row (`FOR UPDATE`), checks the `version` sent by the client, checks permission through `Can`, checks the period lock against both the old and the new date, and only allows transitions from the fixed table.
- `Transition`: checks → approval gate → write status + audit → the module's `OnTransition`, in the same transaction. An error in `OnTransition` rolls everything back.
- The approval gate is an interface in `record`'s `hooks.go`; `approval` is the adapter. Each submission for approval is an approval instance with an id, recording the submitted `version`. `record` stores that id in `approval_ticket`. Completion is accepted only when the ticket matches and the `version` has not changed since submission. Withdrawal or rejection closes the approval instance, so a late approval action on an old submission is refused. If completion fails, the approval instance is closed only when it has become stale; every other error rolls back everything, the approval instance stays open and the document stays pending approval.
- Moving the lock date is not allowed while the period about to be locked still has `pending_approval` documents; otherwise those documents would be stuck forever.
- **Single lock order:** the legal entity's period-lock row (`FOR SHARE` when writing a document, `FOR UPDATE` when moving the lock date) → the document row (`FOR UPDATE`) → rows locked by the module. A lock on each document does not protect the period-lock configuration row, so a lock at the legal-entity level is required. A document's legal entity does not change after creation.
- No tool can statically check that `record` is called, so the `recordtest` contract suite is mandatory for every document type.

## Consequences

- Dependencies inside `core` form a DAG: `approval → record`.
- Concurrent posting runs `OnTransition` only once. A period lock running concurrently with document writes leaves no new document in the locked period.
- Moving a legal entity's lock date must wait for all running document writes of that legal entity. Acceptable, because moving the lock date is rare.
- Each document write adds a row lock in `record.documents`, so all writes on the same document run one after another. Acceptable, because a document is rarely edited by several people at once.
- Queries that filter by validity must join `record.documents`.
- Approval conditions can use the header (date, legal entity, org unit, amount) and the **approval fields** the document type declares in advance (about 5 at most, typed). No full Snapshot for now; the declared field list will be the starting point if a Snapshot is later needed for printing and export.
- Approvers can be returned by the module (`Approvers(ctx, ref, step, level)`), e.g. the direct manager; the result depends only on the parameters, and approvers are fixed when the step starts. Self-approval is not allowed; when no approver is found the step goes to a fallback role, never to automatic approval.
