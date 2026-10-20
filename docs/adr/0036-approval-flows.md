# 0036. Approval flows: steps, choose-one and parallel splits replace the step chain

Status: accepted · 2026-10-19 · supersedes the rule and instance model of ADR-0015; ADR-0022 unchanged

## Context

Customers need approval configured from very simple ("Mr A approves") to complex: a group of people with a count of approvals and rejections; extra approvers when an order passes an amount threshold; extra approvers when an order sells a given item. The engine of ADR-0015 cannot express this: one condition per step, one approver decides a step, steps strictly in a row, approval only on submitting a draft. Customers measure configurability against Odoo Studio's approval rules, a rule on any action of a document. `approval` must cover the cases most ERP customers ask for, configured by the customer, with no code per case.

## Decision

This records the shape of approval flows at decision level for the whole upgrade (parts 1–6); later tickets build on it without changing it again. It replaces the rule and instance model of ADR-0015. ADR-0022's permission model (who may configure a flow) is unchanged.

- **Flow shape.** A flow is a sequence of elements, run top to bottom. An element is one of:
  - an **approval step**;
  - a **choose-one split**: ordered branches, each with a condition; the first branch whose condition holds runs; a final "otherwise" branch always exists and may be empty;
  - a **parallel split**: every branch whose condition holds runs, at the same time; the join waits for all of them.

  A branch's body is itself a flow, so splits nest. There are no loops and no arrows between arbitrary elements.

- **Approval step.** An approver is a `user` (1 to 20 `user_ids`), a `role`, or `module` (the document type's `Approvers`). Two counts, `approvals` and `rejections`, both 1 to 20, default 1 each. One vote per person per step. A step passes as soon as `approvals` approvals are cast; it rejects at `rejections` rejections, or as soon as the remaining voters can no longer reach `approvals`.

- **Approvers are fixed when the step starts**, filtered as today: the submitter, the document type's `Subjects`, and people without `view` are removed. An empty list after filtering falls back to the rule's fallback role, with the counts unchanged. If fewer people are eligible than the step's `approvals`, the step cannot start: the operation fails with `no_approver`, naming the step; the count is never lowered to fit who is left.

- **Conditions** are an expression tree: `{all: […]}`, `{any: […]}`, or a leaf `{field, op, value}` (operators unchanged from ADR-0015). A leaf that cannot be evaluated (a missing or malformed field) is doubt, and doubt only ever adds approval: a branch whose outcome rests on doubt runs, and in a choose-one split the choice then continues to the next branches (and to the otherwise when no branch surely holds), the chosen bodies running in parallel. Doubt never lets a document skip a stricter branch.

- **Resolution at submit.** Conditions are evaluated once, when the document is submitted. The instance stores the resolved plan — the steps that will run, numbered 1..n in reading order, with their sequence/parallel structure — not a reference to the rule. Editing the rule afterwards never changes an open instance.

- **Rejection.** Rejecting any step rejects the whole instance: the other, still-open branches are cancelled. For `post`, the document returns to `draft`, as today. For any other gated action (part 4), rejection unlocks the document and leaves its status unchanged.

- **Later parts, decided now:**
  - *Part 4*: rules are keyed by (document type, action); every write action can be gated, and the document is locked — no write of any kind, including a dependent write from another document — while a gated action waits. Reading (`view`, `view_files`, `print`, `export`) is never gated.
  - *Part 3*: line fields usable in conditions are declared by the record type, the same way header approval fields are declared today. They are aggregate fields over the lines: a set field holds the distinct ids the lines reference and is tested with `has` ("sells item Z"); a numeric aggregate such as a maximum is tested with `gt`/`gte` ("a discount above X"). There are no per-line correlated conditions ("a line where item is Z and discount is above X"): the condition model, the doubt rule and the stored snapshot all stay per document, and the cases customers ask for need nothing more.
  - *Part 5*: delegation is one hop (a delegate's own delegation is not followed); one person casts at most one vote per step, whether as a member or as a delegate. Delegation is resolved when a vote is cast, not when a step starts: a delegation declared while a document already waits still works, and a step's approver list stays fixed. Whoever records a delegation for someone else may not name themself as the delegate, so managing delegations never grants votes.
  - *Part 6*: a step may have a due time in wall-clock hours; passing it without a decision notifies the step's pending approvers and the submitter, once, with no automatic approval, rejection or reassignment.

- **Migration.** Existing rules migrate into flows: each rule becomes a chain of approval steps, with each step's condition becoming a choose-one split with an empty otherwise, so nothing configured today is lost.

## Holding a document for an action

Part 4: approval on every write action, not only `post`.

- `post` keeps today's `pending_approval` status: the document's row and `version` change, as before.
- Any other gated action — `cancel`, or a module's own document action (e.g. Sales `create_order`) — is an **approval request** that holds the document without changing its row or its `version`. The hold is the open approval instance itself; `record` reads it through the gate. Why: holding, withdrawing and rejecting then write no document at all, and completing a module action writes only that action's own document.
- At most one open request per document: a partial unique index enforces it.
- While a document is held:
  - `Edit`, `Touch`, `Delete`, `Transition`, `Request` and `LockToDepend` all refuse with `awaiting_approval`;
  - `Lock`, used by core modules that only read a document's state, and `LockOpen`, used by attachments and discussion, still work;
  - `allowed_actions` keeps only `print` and `export`.
- Completion: `approval` calls `record.CompleteAction(ctx, ref, ticket)` before closing the instance. `record` re-locks the document and checks that the hold still matches (the ticket, and `version` equal to the one recorded when the request opened) and the period lock. It then runs the action as the submitter:
  - `cancel` re-checks `Can(cancel)` and applies the transition;
  - a module action calls the document type's `ModuleActions[name]`, which re-runs its own checks.

  A failing re-check is a normal error: full rollback, and the request stays open.
- Rejecting or withdrawing a request closes the instance only, writing no document. The submitter withdraws through `POST /approvals/{id}/withdraw`.
- Moving the period lock is refused while any document in the newly locked days is held, the same as for a document `pending_approval` to `post`. `record` asks the gate (`HeldInPeriod`) for the list.
- Rules are keyed by `(doc_type, action)`. The HTTP API takes `?action=` and defaults to `post`.

## Rejected alternatives

- **Odoo Studio's levels only** (a fixed ladder of approvers per level): cannot express amount tiers without duplicating the whole ladder once per tier.
- **A free BPMN graph** (arbitrary arrows, loops): lets a flow have unreachable elements or cycles a customer cannot validate, and the engine cannot guarantee terminates.
- **Exclusive branching only** (no parallel split): independent reasons to add approvers (a special item, a new customer, a high discount) would each need their own branch of every combination, a combinatorial blow-up.

## Consequences

- `approval.rules` and `approval.instances` of ADR-0015 are replaced by a flow definition (nested elements) and an instance that stores a resolved plan of steps; the migration above keeps every configured rule working with no manual re-entry.
- A step's approver is a group with two counts, not a single decider: AND/OR between people lives only on the step.
- Every write action on a document can gate on a flow, not only `post`; parts 3–6 build gating by action, line conditions, delegation and due times on this shape.
- `documents.md`'s "no parallel branches" limit is lifted; "no loops, scripts or free-form BPMN" remains.
- ADR-0022's per-product permission to manage approval rules is unaffected: a flow is still configured per document type, by the same permission.
