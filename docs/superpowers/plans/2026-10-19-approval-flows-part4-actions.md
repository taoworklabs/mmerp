# Approval flows, part 4: approval on every write action — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Approval rules are keyed by document type and action. Besides `post`, the actions `cancel` and a module's own document actions (Sales `create_order`) can need approval. While such a request is open, the document is held: no write and no dependent write. When the flow completes, the action runs once, as the submitter.

**Architecture:** `post` keeps today's `pending_approval` status. Every other action leaves the document's status and row untouched. The hold is the open approval instance itself. `record` asks the approval gate (`HeldBy`) on every write path and refuses while a request is open. Completing a request calls `record.CompleteAction`, which re-checks and runs the action in the approver's transaction, acting as the submitter. Because the held document's row is never written:
- creating an order from a quotation still writes exactly one document, the order;
- cancelling writes exactly one, the cancelled document.

`record` never reads approval's tables. Approval reads `record.documents`, a lower tier, to list held documents for the period lock.

**Tech Stack:** Go, Postgres (sqlc, SQL migrations), Huma; React + TypeScript; Vitest; Playwright.

**Spec:** `docs/superpowers/specs/2026-10-19-approval-flows-design.md` (goal 1, goal 5, goal 6 for other actions; "My calls" on locking, withdrawal, period lock, actor at completion). Ticket #94. Parts 1–2 are done on this branch: read `internal/core/approval/{flow.go,service.go,types.go}`, `internal/core/record/{service.go,hooks.go,types.go}` and `documents.md` ("record functions").

## Global Constraints

- **Lock order, unchanged:**
  1. In `Create`, with a `request_id`, the advisory lock first.
  2. The legal entity row in `record.period_locks`, `FOR SHARE`.
  3. The document row, `FOR UPDATE`.
  4. Module rows.

  An approval action reads its instance, then calls `record.Lock`, then takes the instance `FOR UPDATE`.
- **One transaction writes exactly one document.** Holding, withdrawing or rejecting a non-`post` request writes no document row.
- **`record` never imports `approval` and never reads the `approval` schema.** It only goes through `ApprovalGate` (`hooks.go`), wired in `internal/app`.
- **Approval is never automatic.** Nobody votes on their own request or on a document they can't view.
- Draft `edit` and `delete` are not gated. Read actions (`view`, `view_files`, `print`, `export`) are never gated.
- Migrations are SQL only, in a new file `0042`. Never edit an existing one.
- APIs return stable error codes with params. `allowed_actions` is the only source of buttons.
- English identifiers. Comments short, say why, never reference docs or ADRs. i18n in `vi` and `en`.
- Conventional Commits, ending with `Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`. Stage explicit paths. Branch `approval-flows`.

## Review Focus

1. **Two requests racing on one document** (two users cancel at once, or cancel against `create_order`): exactly one opens a request and the other gets `awaiting_approval`. A partial unique index backs this. Test: `TestOneOpenRequestPerDocument` (Task 3).
2. **A held document's dependent write:** `create_order` from a quotation that waits for `cancel` is refused, and `cancel` of a quotation that waits for `create_order` is refused. Test: Sales `TestHeldQuoteRefusesDependentActions` (Task 4).
3. **The submitter loses the permission before completion:** the run's re-check fails. The whole decision rolls back and the request stays open. Test: `TestCompletionRechecksAsSubmitter` (Task 3).
4. **Moving the period lock over a held document** is refused, with `period_has_pending_documents` listing it. Test: `TestPeriodLockRefusesHeldDocuments` (Task 3).
5. **A rejected or withdrawn cancel request** leaves the document `posted`, its version unchanged and its actions back. Test: `TestRejectedCancelKeepsTheDocument` (Task 3).

---

### Task 1: ADR amendment and glossary

**Files:**
- Modify: `docs/adr/0036-approval-flows.md` (a new section "Holding a document for an action")
- Modify: `documents.md` ("record functions": the gate's new methods, `CompleteAction`, `LockToDepend`; the lifecycle text about pending requests)
- Modify: `CONTEXT.md` (add **Approval request** (vi: "Yêu cầu duyệt") and **Held document** (vi: "Chứng từ đang chờ duyệt yêu cầu"); reword **Approval instance** so it covers any action)

- [ ] **Step 1: Record the decisions.** Each bullet below goes into the ADR. Each item that has a "why" gives it in one sentence.
  - `post` keeps `pending_approval`.
  - Any other action is an approval request that holds the document without changing its row or `version`. The hold is the open instance, which `record` reads through the gate. Why: holding, withdrawing and rejecting then write no document, and completing a module action writes only its own document.
  - At most one open request per document; a partial unique index enforces it.
  - While held:
    - `Edit`, `Touch`, `Delete`, `Transition`, `Request` and `LockToDepend` refuse with `awaiting_approval`;
    - `Lock`, which core modules use, and `LockOpen`, for comments and attachments, still work;
    - `allowed_actions` keeps only `print` and `export`.
  - Completion: approval calls `record.CompleteAction(ctx, ref, ticket)` before closing the instance. `record` re-locks the document and checks that the hold still matches (ticket, and `version` equal to the one at request) and the period lock. It then runs the action as the submitter:
    - `cancel` re-checks `Can(cancel)` and applies the transition;
    - a module action calls the type's `ModuleActions[name]`, which re-runs its own checks.

    A failing re-check is a normal error: full rollback, and the request stays open.
  - Rejecting or withdrawing a request closes the instance only. The submitter withdraws through `POST /approvals/{id}/withdraw`.
  - Moving the period lock is refused while any document in the newly locked days is held. `record` asks the gate (`HeldInPeriod`).
  - Rules are keyed by `(doc_type, action)`. The HTTP API takes `?action=` and defaults to `post`.
- [ ] **Step 2:** Update `documents.md` and `CONTEXT.md` to match.
- [ ] **Step 3: Commit** `docs(approval): hold a document while an action awaits approval` (stage the three files).

---

### Task 2: Rules per action

**Files:**
- Create: `migrations/0042_approval_actions.sql`
- Modify: `internal/core/approval/queries.sql` (rules by `(doc_type, action)`), regenerate the store
- Modify: `internal/core/approval/{service.go,types.go,handler.go}`
- Modify: every Go caller of `SaveRule` and `DeleteRule`, mechanically (`grep -rn "\.SaveRule(\|\.DeleteRule(" internal`)
- Test: `internal/core/approval/service_test.go`

**Interfaces:**
- Produces:
  - `func (s *Service) SaveRule(ctx context.Context, docType string, action record.Action, in RuleInput) error`
  - `func (s *Service) DeleteRule(ctx context.Context, docType string, action record.Action) error`
  - `TypeRule` gains `Action string \`json:"action"\``. The frontend labels actions through i18n (`shared.rule.action.<name>`, or `<product>.action.<name>` for module actions). `Rules` returns one `TypeRule` per `(type, action)` the type offers:
    - `post`;
    - `cancel`;
    - then each key of `record.Type.ModuleActions`, sorted. The field is added in Task 3; until then, only post and cancel.
  - HTTP: `PUT` and `DELETE /approval-rules/{doc_type}` take the query `action`, default `post`, validated against the type's actions (`invalid_rule` with reason `unknown_action` otherwise). `GET /approval-rules` returns the expanded list.
- Migration `0042_approval_actions.sql`:

```sql
ALTER TABLE approval.rules ADD COLUMN action text NOT NULL DEFAULT 'post';
ALTER TABLE approval.rules DROP CONSTRAINT rules_pkey, ADD PRIMARY KEY (doc_type, action);
ALTER TABLE approval.instances ADD COLUMN action text NOT NULL DEFAULT 'post', ADD COLUMN params jsonb;
-- One open request per document holds it; a second one waits on this index and fails.
CREATE UNIQUE INDEX instances_one_open_idx ON approval.instances (doc_type, doc_id) WHERE status = 'open';
```

Check the real name of the rules primary key in `0009_approval_init.sql`; it is `rules_pkey` unless that file names it. Also check that no existing data breaks the unique index: two open instances for one document can't exist today, because `pending_approval` allows only one ticket.

- [ ] **Step 1: Write the failing tests:**
  - `TestRulesPerAction`: save a `cancel` rule for `test.doc`. Then `Rules` lists both `test.doc/post` and `test.doc/cancel`, each with its own flow or none. Delete the cancel rule and the post rule stays.
  - `TestUnknownActionRefused`: `SaveRule(…, "test.doc", "create_order", …)` returns `invalid_rule` with reason `unknown_action`.
- [ ] **Step 2: Run** `go test ./internal/core/approval -run 'TestRulesPerAction|TestUnknownActionRefused'` → FAIL.
- [ ] **Step 3: Implement** the migration, queries and service. `Submit` (post) reads the rule for `(d.Type, "post")`. Update every Go caller to pass `record.Post`. JSON callers (`internal/app/*_test.go`, `web/e2e`) are unchanged, since the HTTP default is `post`.
- [ ] **Step 4: Run** `make gen` (commit only the sqlc store). Then `go build ./...`, `go test ./internal/...` → PASS.
- [ ] **Step 5: Commit** `feat(approval): rules per document action`.

---

### Task 3: Holding a document; completing an action

**Files:**
- Modify: `internal/core/record/{hooks.go,types.go,service.go,handler.go}`
- Modify: `internal/core/approval/{service.go,types.go,handler.go,queries.sql}` (+ store)
- Modify: `internal/app/app.go` only if wiring changes. The gate is already wired; the interface just grows.
- Test: `internal/core/approval/service_test.go`, `internal/core/record/contract_test.go` (or the record test file that exercises the gate with a real approval)

**Interfaces:**
- Produces in `record`:

```go
// hooks.go — ApprovalGate gains:
// Hold opens a request for action on d when a rule requires one; params travel to the run.
Hold(ctx context.Context, d Doc, action Action, params json.RawMessage) (needed bool, err error)
// HeldBy returns the open request holding ref, or nil.
HeldBy(ctx context.Context, ref Ref) (*Hold, error)
// HeldInPeriod lists the held documents of a legal entity dated on or before until.
HeldInPeriod(ctx context.Context, legalEntity int64, until pgtype.Date) ([]HeldDoc, error)

// types.go
type Hold struct {
	Ticket      int64
	Action      Action
	Params      json.RawMessage
	SubmittedBy int64
	Version     int32 // the document's version when requested
}
type HeldDoc struct { DocType string; ID int64; Number string; Date pgtype.Date }
// ModuleAction runs a module's own document action; it re-checks what it needs.
type ModuleAction func(ctx context.Context, d Doc, params json.RawMessage) error
// Type gains (documents only):
ModuleActions map[Action]ModuleAction

var ErrAwaitingApproval = &platform.Error{Status: http.StatusConflict, Code: "awaiting_approval"}

// service.go
// Request asks whether action on ref must wait for approval and, if so, holds ref.
// Modules call it inside their transaction after their own checks, before running the action.
func (s *Service) Request(ctx context.Context, ref Ref, action Action, params json.RawMessage) (held bool, err error)
// LockToDepend takes ref's locks for a write that depends on it; refused while ref is held.
func (s *Service) LockToDepend(ctx context.Context, ref Ref) (Doc, error)
// CompleteAction runs the held action of ref once its request is approved; see Hold.
func (s *Service) CompleteAction(ctx context.Context, ref Ref, ticket int64) error
```

- `Transition` becomes `func (s *Service) Transition(ctx context.Context, ref Ref, version int32, to Status) (held bool, err error)`. From posted to cancelled, after today's checks, it calls `Request(ctx, ref, Cancel, nil)`. When held, it audits `record.cancel_requested` and returns `(true, nil)` without applying. The handler `POST /documents/{type}/{id}/transitions` answers `200 {"held": bool}` instead of 204. Update the Go callers of `Transition` (`grep -rn "\.Transition(" internal`) to take the extra result.
- Held refusals: `Edit`, `Touch`, `Delete`, `Transition` (any target) and `Request` call `s.refuseHeld(ctx, ref)` after the document lock, which uses `gate.HeldBy` and returns `ErrAwaitingApproval`. `Actions(ctx, d, extra...)`, when held, returns only `print`/`export` (if allowed) and drops `extra`.
- `SetPeriodLock`: after `PendingInPeriod`, it also lists `gate.HeldInPeriod` into the same `period_has_pending_documents` error.
- `CompleteAction`:
  1. Lock the document, then `h := gate.HeldBy`.
  2. If `h == nil || h.Ticket != ticket || d.Version != h.Version`, return `ErrApprovalStale`.
  3. `checkPeriod`.
  4. `as := s.d.IAM.AsUser(ctx, h.SubmittedBy)`.
  5. `Cancel`: `require(as, ref, Cancel)`, then status must be `posted` (else `ErrInvalidTransition`), then `apply(as, d, Cancelled)`.
  6. Module action: `t.ModuleActions[h.Action](as, d, h.Params)`; a missing name is an internal error.
- Produces in `approval`:
  - `Hold`: reads the rule for `(d.Type, action)`. With none, `(false, nil)`. Otherwise it resolves and starts the flow like `Submit`, inserting the instance with `action` and `params`. A unique-index violation maps to `record.ErrAwaitingApproval` through `platform.Violates`. When the flow resolves to no steps: `(false, nil)`.
  - `HeldBy`: the open instance of the document with `action <> 'post'`, mapped to `*record.Hold`. `Version` is the instance's `version`.
  - `HeldInPeriod`: open instances with `action <> 'post'`, joined to `record.documents` on legal entity and date.
  - `act` stale check:
    - when `inst.Action == "post"`, as today;
    - otherwise, the instance must be open and `d.Version == inst.Version`; else close it as stale.
  - `Approve` when finished:
    - `post`: as today.
    - Any other action: call `record.CompleteAction(ctx, d.Ref, id)` first, then close the instance `approved`, then notify the submitter.
  - `Reject`: for a non-`post` action, cancel the steps and close the instance; no record call.
  - New: `func (s *Service) Withdraw(ctx context.Context, id int64) error`, through `act`-style locking. Only the submitter may withdraw, and only a non-`post`, open request (`post` is withdrawn through the record transition as today). It cancels the steps, closes the instance `withdrawn` and audits `approval.withdrawn`. Route: `POST /approvals/{id}/withdraw` (204).
  - `Instance` gains `Action string \`json:"action"\`` and `AllowedActions []string` (`withdraw` for the submitter of an open non-`post` request while the product takes writes).

- [ ] **Step 1: Write the failing tests** (approval `service_test.go`, with a `cancel` rule on `test.doc` posted first):
  - `TestCancelWaitsForApproval`: cancel → `held == true`. The document stays `posted` with the same version, `allowed_actions` has no `cancel` and no `edit`, and `Edit`/`Transition` give `awaiting_approval`. Approving the step gives `cancelled`.
  - `TestRejectedCancelKeepsTheDocument`: reject → `posted`, version unchanged, `cancel` back in `allowed_actions`, and the instance `rejected`. The same for withdraw by the submitter; anyone else gets `forbidden`.
  - `TestOneOpenRequestPerDocument`: two cancels in sequence → the second gets `awaiting_approval`. With `pgtest.AwaitLockWait`, as other concurrency tests in record do, two concurrent cancels → exactly one held.
  - `TestCompletionRechecksAsSubmitter`: revoke the submitter's cancel permission, then approve → error. The vote is not kept, the instance stays open, and the document stays `posted`.
  - `TestPeriodLockRefusesHeldDocuments`: a held document dated 2026-03-10; `SetPeriodLock(…, "2026-03-31")` → `period_has_pending_documents` listing it.
  - `TestModuleActionRunsOnce`: `test.doc` registers `ModuleActions{"stamp": func…}` (counts calls; writes nothing) and a `stamp` rule. `Request` holds it; approving runs it exactly once, as the submitter (assert `platform.ActorFrom` inside).
- [ ] **Step 2: Run** `go test ./internal/core/approval ./internal/core/record/...` → FAIL.
- [ ] **Step 3: Implement** per the interfaces. Update `lifecycleActions` so a held document offers no lifecycle action. Run `make gen` and commit the sqlc stores only. `api/openapi.json` and `schema.gen.ts` are committed in Task 5, so the web build stays green in between.
- [ ] **Step 4: Run** `go test ./internal/...` and `make lint` → PASS.
- [ ] **Step 5: Commit** `feat(record): hold a document while an action awaits approval`.

---

### Task 4: Sales `create_order` and the contract suite

**Files:**
- Modify: `internal/modules/sales/service_doc.go`: `CreateOrderFromQuote` splits into the route entry, which checks, locks, refuses held and then calls `Request` or runs, and a run used by both the route and the quote type's `ModuleActions["create_order"]`.
- Modify: `internal/modules/sales/service.go` (or wherever the quote type is registered): add `ModuleActions`.
- Modify: `internal/modules/sales/handler.go`: `POST /sales/quotes/{id}/order` answers `200 {"id": n | null, "held": bool}`, where `id` is null while held. One shape, so the client reads one type.
- Modify: `internal/core/record/recordtest/recordtest.go`: `RequireApproval func(t *testing.T)` becomes `RequireApproval func(t *testing.T, action record.Action)`, plus a new case. Update every harness (`grep -rn "RequireApproval:" internal`).
- Test: `internal/modules/sales/service_test.go`

**Interfaces:**
- Consumes: Task 3's `Request`, `LockToDepend`, `ModuleAction`, `ErrAwaitingApproval`; Task 2's `SaveRule(ctx, docType, action, in)`.
- `CreateOrderFromQuote(ctx, quoteID) (id int64, held bool, err error)`:
  1. Read, WriteGate and the permission, as today.
  2. `record.LockToDepend`, which refuses a held quote.
  3. Return the live order if one exists (today's rule).
  4. Validate posted and unexpired.
  5. `held, err := s.d.Record.Request(ctx, ref, "create_order", nil)`. When held, return `(0, true, nil)`.
  6. Otherwise `s.makeOrder(ctx, quoteID)`.

  The `ModuleActions["create_order"]` entry runs the same checks except `Request`, then `makeOrder`, inside the approver's transaction, as the submitter.
- `recordtest` new case `a document held by a request refuses writes`. It is skipped when `RequireApproval` is nil or the type can't be cancelled. Steps: post the document, `RequireApproval(t, record.Cancel)`, cancel → held; then `Edit`, `Transition` and a second cancel refuse with `awaiting_approval`; approving through `h.Approve` gives `cancelled`.

- [ ] **Step 1: Write the failing tests** (Sales):
  - `TestCreateOrderWaitsForApproval`: rule `create_order` → manager approves. Staff create the order → `held`, and no order exists yet. The manager approves, then exactly one order exists, with the quote's lines, and its `record.created` audit entry names the staff user as actor.
  - `TestHeldQuoteRefusesDependentActions`: a quote held for `cancel` → `CreateOrderFromQuote` gives `awaiting_approval`. A quote held for `create_order` → cancel gives `awaiting_approval`, and `allowed_actions` lacks `create_order` and `cancel`.
- [ ] **Step 2: Run** `go test ./internal/modules/sales` → FAIL.
- [ ] **Step 3: Implement**, update the `recordtest` signature and every harness, add the case.
- [ ] **Step 4: Run** `go test ./internal/...` → PASS. Every document type passes the new `recordtest` case or skips it with its reason in `Gaps`, if a type can't be cancelled.
- [ ] **Step 5: Commit** `feat(sales): creating an order from a quote may need approval`.

---

### Task 5: Frontend: rules per action, requests on the document, e2e

**Files:**
- Regenerate: `api/openapi.json`, `web/src/shared/api/schema.gen.ts`
- Modify: `web/src/shared/document/ApprovalRules.tsx`:
  - the list shows one row per `(doc_type, action)`, labelled "<type name> · <action label>";
  - routes become `:docType` (post) and `:docType/:action`;
  - save and delete send `?action=`.
- Modify: `web/src/shared/document/ApprovalPanel.tsx`:
  - for a non-`post` instance, the panel title names the request (`shared.approval.request.<action>`, or the module's `<product>.action.<name>`, e.g. "Yêu cầu huỷ chứng từ");
  - the submitter gets "Rút yêu cầu" (withdraw, with a confirm) from `instance.allowed_actions`.
- Modify: `web/src/shared/document/DocumentActions.tsx`: after cancel, when the response says `held`, toast `shared.document.sent_for_approval` instead of the cancelled toast, and invalidate as today.
- Modify: `web/src/sales/…` (the quote page's create-order action): on `held`, toast the same and stay on the quote; otherwise navigate as today.
- Modify: i18n vi and en: action labels (`shared.rule.action.post` "Gửi duyệt", `shared.rule.action.cancel` "Huỷ chứng từ", `sales.action.create_order` "Tạo đơn bán"), request titles, withdraw, `shared.error.awaiting_approval` ("Chứng từ đang chờ duyệt một yêu cầu.").
- Create: `web/e2e/approval-actions.spec.ts`

- [ ] **Step 1: Implement** the above. Follow the `mmerp-ui` skill and finish with its checklist. The terms come from `CONTEXT.md`, as Task 1 updated it.
- [ ] **Step 2: e2e** `approval-actions.spec.ts`, in the exclusive project if it changes tenant-wide rules. Snapshot and restore the rules it touches, as `approval-flows.spec.ts` does.
  - `cancelling a posted order waits for the manager`: set a `cancel` rule on `sales.order` through the editor (row "Đơn bán · Huỷ chứng từ"). Staff cancel a posted order: they see the "sent for approval" toast, the order still reads "Đã xác nhận", and the panel shows the request. The manager approves from the inbox, and the order is cancelled.
  - `a cancel request can be withdrawn`: staff withdraw, and Cancel shows again.
  - `creating an order from a quote waits for approval`: a `create_order` rule. Staff click create order: toast, and no order. The manager approves, and the quote links to the new order.
- [ ] **Step 3: Run** `make gen` (no diff on a second run), `make lint`, `pnpm -C web test`, `make e2e` → all pass.
- [ ] **Step 4: Commit** `feat(web): approval requests for cancel and module actions`.
