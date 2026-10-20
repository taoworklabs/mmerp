# Approval flows: design

Status: draft for review · 2026-10-19

## Intent

Customers need to configure approval from very simple ("Mr A approves") to complex: a group of people with a count of approvals and rejections; extra approvers when an order passes an amount threshold; extra approvers when an order sells a given item. The approval engine as built (ADR-0015, ADR-0022) can't do this: one condition per step, one approver decides a step, steps strictly in a row, approval only on submitting a draft. The customer's reference point is Odoo Studio's approval rules: a rule on any action of a document. The goal is that `approval` covers the cases most ERP customers ask for, configured by the customer, with no code per case.

This replaces the rule model of ADR-0015 and the "no parallel branches" limit in `documents.md`. A new ADR records it before code (it supersedes parts of ADR-0015; ADR-0022's permission model stays).

## Goals

1. **Approval on every action of a document**, not only `post`: the lifecycle actions (`post`, `cancel`) and a module's own write actions on a document (e.g. Sales `create_order` from a quotation).
2. **A flow is a structured block diagram**, not a free graph. A flow is a sequence of elements run top to bottom; an element is one of:
   - **Approval node**: approvers plus two counts (below).
   - **Choose-one split**: ordered branches, each with a condition; the first branch whose condition holds runs; a final "otherwise" branch always exists and may be empty. Used for amount tiers.
   - **Parallel split**: branches each with an optional condition; every branch whose condition holds runs at the same time; the join waits for all of them. Used for independent, additive reasons (a special item, a new customer, a high discount).
   - A branch's body is itself a sequence, so splits nest. Every split has exactly one join; there are no loops and no arrows between arbitrary nodes.
3. **Approval node**: approvers are one user, a list of users, holders of a role at the document's org unit, or chosen by the document type (`Approvers`, e.g. the direct manager, with level climbing as today). Two counts: approvals needed to pass, rejections that reject. Default 1 and 1, which is today's behaviour. AND/OR between people lives only here (1 of n is OR, n of n is AND).
4. **Conditions** are AND/OR expressions over the document header (amount, org unit, declared approval fields) **and over its lines** ("any line whose item is Z", "any line with discount > 10%"). AND/OR over data lives only in branch conditions.
5. **While an action waits for approval the document is locked**: no write of any kind (edit, other actions, progress such as deliveries or payments). Viewing, printing and exporting stay allowed.
6. **Rejection at any node** rejects the whole request, cancels the other running branches, and the action does not happen. For `post` the document returns to `draft` (as today). For any other action the document is unlocked and keeps its status (a rejected cancel leaves the document `posted`).
7. **Delegation**: a user declares "from date A to date B, user X approves for me"; an administrator can declare it for them. While active, X may decide wherever the user is an approver; history shows "X for Y".
8. **Overdue**: each approval node may have a due time. When it passes without the node deciding, both sides are notified: the node's pending approvers and the submitter.

## Non-goals

- Free-form BPMN: arbitrary arrows, loops, scripts, timers other than the due time.
- "Return to the previous node": rejection always goes back to the submitter.
- Automatic actions on overdue (auto-approve, auto-reject, reassign). Overdue only notifies.
- Approval on catalogue records (customers, items, employees).
- Separate flows per legal entity or org unit (rejected in ADR-0022; still not needed). Conditions on `org_unit` cover per-branch differences.
- Due times in working days. `approval` is core and the work calendar lives in `hrm`; due times are wall-clock hours. Moving the calendar to core is a separate decision.

## Anti-goals

- Self-approval through any path: delegation, groups, reassigning. The submitter and the document's `Subjects` never decide, and neither does anyone without `view` on the document.
- A flow change affecting requests already submitted.
- A simple rule ("Mr A approves posting") becoming harder to configure than today.

## Constraints (from the repo; not negotiable here)

- `approval` is core: business-agnostic, imports `record` and `iam`, never a product. Products declare what conditions may test; `approval` knows nothing of items or discounts.
- Lock order and "one transaction writes one document" (`documents.md`). An approval action locks the document (`record.Lock`) before its own rows.
- Each submission is its own approval instance tied to the submitted `version`; a late decision on an older submission is rejected (`approval_stale`).
- Approvers of a node are fixed when the node starts.
- `allowed_actions` stays the only source of what the frontend shows.
- Every document type passes `recordtest`; the suite grows to cover the new guarantees.
- Rules are configured per product with `<product>.approval.manage` or `core.approval.manage` (ADR-0022).
- No external services; notifications (in-app, email via jobs) use the existing `notification` module. The overdue check is a system periodic job.

## Decided in conversation

- Shape: structured blocks with two split kinds (choose-one, parallel), between Odoo Studio's levels and full BPMN.
- A group is the set of people configured on a node, with approval and rejection counts.
- Every action on a document can require approval; the document is locked while it waits.
- Rejection at a node returns the document to draft (for `post`).
- Delegation and overdue notification are both in this upgrade; overdue notifies both sides.
- A node whose eligible approvers (after removing the submitter, `Subjects` and people without `view`) are fewer than its approval count can't start: the operation fails with `no_approver`, naming the node. It neither falls back nor lowers the count.

## My calls (not confirmed by the user)

- Conditions are evaluated once, at submit; the document is locked, so its data can't change while waiting.
- The flow is snapshotted at submit; editing a flow doesn't touch open requests.
- Rejection of a non-`post` action unlocks the document and keeps its status.
- Reading actions (`view`, `print`, `export`, `view_files`) are never gated; only write actions are. "Every action" means every write action.
- Draft `edit` and `delete` are not gated: a draft is the submitter's own workspace and nothing is in effect yet.
- Delegation is one hop (X's own delegation is not followed), dated with business dates in the tenant time zone, applies to all products, and a delegate who fails the anti-self-approval or `view` checks is skipped (falling back as today).
- The overdue notification is sent once per node, at the due time; no repeated reminders.
- The fallback role and reassignment by its holders stay as today.
- "Locked" covers writes from other documents too: while an action waits, `record.Lock` on the document refuses for a dependent write (no delivery against an order waiting for `cancel`, no order from a quotation waiting for `cancel`). Comments and attachments stay allowed, as on `pending_approval` today, so approvers can discuss what they decide.
- A pending non-`post` action can be withdrawn by its submitter (as a pending `post` can), and documents waiting on any action block moving the period lock date, as `pending_approval` does today.
- A gated module action runs at completion as the submitter (creator and audit name them), with the action's own checks re-run then. A failing check is a normal error: full rollback, the request stays open, and the approver can reject or the submitter withdraw.
- A node that can no longer reach its approval count (too many rejections short of the rejection count, e.g. 3 of 3 needed after one rejection) rejects at once.
- Reassigning replaces one member of a node's approver list, not the whole list; the counts stay.
- One person casts at most one vote per node, whether as a member, as a delegate, or both.
- The fallback role stays one per flow.
- Existing rules are migrated into flows (a chain of nodes, each step's condition becoming a choose-one split with an empty otherwise) so nothing configured is lost.

## Open for the builder

- How `record` represents "locked while a non-`post` action waits": `pending_approval` today means "draft waiting to post". A posted document waiting for `cancel` needs a lock that is not a status change (e.g. an approval ticket on a posted row that `record` checks on every write). Settle this in the ADR; it must keep the lock order and the stale-request guarantees.
- How a module's own action is gated and later executed: the action has to be described (code and parameters) at submit and run when the flow completes, in the transaction of the last approval, with the same "full rollback on a non-stale error" rule as `CompleteApproval`.
- How record types declare line data for conditions (e.g. a line field list, with choice options that can be searched because catalogues are large), and how values are captured at submit.
- Storage of flows and instances (the tree of elements, per-node decisions, per-branch state), the API, and the flow editor screen (it must follow `ui.md` and the `mmerp-ui` skill).
- Splitting into tickets; the order of delivery.
