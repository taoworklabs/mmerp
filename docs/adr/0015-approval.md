# 0015. Module `approval`: rules, approval instances, approval inbox

Status: accepted · 2026-10-05

## Context

ADR-0002 settled the approval gate and approval instances; `documents.md` describes a rule as a chain of steps with simple conditions. Leave requests need this part done in full.

## Decision

- `approval` is a `core` module that imports `record` and `iam`; it implements the approval gate of `record` (`Submit`, `Withdrawn`), wired in `internal/app`.
- **Rules**: `approval.rules (doc_type, steps, max_levels, fallback_product, fallback_role)`, at most one rule per document type; `steps` is a JSON array, and the whole rule is edited at once. With no rule, submitting goes straight to `posted`.
  - Each step: an optional condition `{field, op, value}` and an approver.
  - Condition fields: `amount`, `org_unit` or a declared approval field. Comparisons: numbers `gt`, `gte`; choices `eq`; org units `within` (inside the subtree).
  - Approver: `role` (users holding that role at the document's org unit, including grants at an ancestor or tenant-wide), `user`, or `module` (the document type's `Approvers`, `level` 1 to `max_levels`).
  - A fallback role is mandatory; it is resolved at the document's org unit.
- **Approval instances**: `approval.instances (id, doc_type, doc_id, version, submitted_by, status, current_step)` and `approval.instance_steps (instance_id, position, approver, approvers, fallback, decided_by, decision, reason, decided_at)`. On submit, conditions are evaluated once over the header and the approval fields (which cannot change while pending approval); only steps whose condition holds are created. If no step holds, no approval is needed.
- **Approvers are fixed when a step starts.** The submitter and the people the document is about (the document type's `Subjects`, e.g. the employee requesting leave) are excluded, including when approvers are replaced; an empty list moves on to the next `level` (`module` approver), then to the fallback role (`fallback = true`). People without permission to view the document are also excluded, since they could never act on that step. Still empty: submitting is refused (`no_approver`); there is never self-approval.
- **Lock order of an approval action**: read the approval instance to find the document → `record.Lock` (period lock, document) → approval instance `FOR UPDATE` → checks: the instance is open, the step matches the one the client sent, the actor is in the list, `Can(view)`, the product gate for writes. On the last step, call `record.CompleteApproval` in the same transaction. A stale error (`approval_stale`): close the instance as `stale` and commit. Any other error: roll back everything.
- **Rejecting** requires a reason and calls `record.Reject`. **Replacing approvers**: a holder of the fallback role replaces the approver list of the open step with one user (entered by login name), audited.
- Permission to configure rules: `core.approval.manage` in `core.admin`. Superseded by [ADR-0022](./0022-approval-rule-permissions-per-product.md).
- Approval inbox: open approval instances where the actor is in the approver list of the current step, filtered by `Can(view)`. No pagination.

## Consequences

- Every approval action locks the document before the approval instance, in the same order as withdrawing, so the two never deadlock.
- A manager changing midway does not change the approvers of the open step; to recompute, withdraw and resubmit, or replace the approvers.
- No parallel branches, time-based delegation or due-date reminders.
