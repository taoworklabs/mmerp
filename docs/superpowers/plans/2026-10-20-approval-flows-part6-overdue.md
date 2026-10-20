# Approval flows, part 6: overdue notice — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** An approval step may have a due time in hours. When an open step passes it, the step's pending approvers (with their active delegates) and the submitter each get one notification. Nothing else happens.

**Architecture:**
- `StepDef` gains an optional `due_hours`.
- When a step starts, its row records `due_at = now() + due_hours`.
- A system periodic job runs every 15 minutes and at start, so it catches up on whatever is left. It claims overdue open steps one at a time: `UPDATE … SET overdue_notified_at = now() WHERE … AND overdue_notified_at IS NULL RETURNING`. The claim is the once-only guarantee, and each claim and its notifications run in their own transaction.
- No document is written and no document lock is taken.
- A step that started before this change has no `due_at` and is never overdue.

**Tech Stack:** Go, Postgres, River (periodic jobs), Huma; React + TypeScript; Vitest; Playwright.

**Spec:** `docs/superpowers/specs/2026-10-19-approval-flows-design.md` (goal 8; non-goals: no automatic actions, no working-day calendar; "My calls": notified once per step). The decision is in `docs/adr/0036-approval-flows.md` (part 6 bullet). Ticket #96. Parts 1–5 are done on this branch: read `internal/core/approval/{service.go,flow.go,types.go,module.go,queries.sql}` (`startStep`, `requestVotes`), `internal/core/attachment/jobs.go` (the system periodic job pattern), `internal/core/notification/{types.go,service.go,i18n/*.json}`, and `web/src/core/pages/NotificationsPage.tsx`.

## Global Constraints

- **Wall-clock hours, not working days.** `due_hours` is an integer from 1 to 720 and optional. Absent means no due time.
- **Exactly one notice per step.** The atomic claim guarantees it. Re-running the job, two workers, or a restart never notifies twice.
- **Overdue only notifies.** It never approves, rejects or reassigns.
- **Who gets the notice:**
  - the step's approvers who haven't voted, plus their active delegates, filtered by the same checks as `requestVotes` (excluded, `view`);
  - the instance's submitter.

  Each user is notified once, even if they hold several roles.
- **System job:** `platform.JobSpec{System: true}`, kind `approval.overdue`. It runs with no actor, so notifications carry no actor.
- **Migrations** are SQL only, in a new file `0044`.
- English identifiers. Comments short, say why, never reference docs. i18n in `vi` and `en`, including the email texts. Tokens only in the UI.
- Conventional Commits, ending with `Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`. Stage explicit paths. Branch `approval-flows`.

## Review Focus

1. **The job runs twice over the same overdue step** (or two workers race): the step is notified once. Test: `TestOverdueNotifiesOnce` (Task 1).
2. **A step decided before its due time**, or a step of a closed, withdrawn or stale instance, sends nothing. Test: `TestOverdueIgnoresDecidedSteps` (Task 1).
3. **A step whose approvers all voted except one:** only that one, plus the submitter, are told. Test: `TestOverdueTellsPendingVotersAndSubmitter` (Task 1).
4. **A flow saved without `due_hours`** keeps working as before, and its steps never become overdue. Test: covered by every existing test, plus `TestNoDueTimeNeverOverdue` (Task 1).
5. **The editor shows and saves the due time**, and a flow without it round-trips unchanged (no `due_hours: 0` sent). Test: `flowForm.test.ts` (Task 3).

---

### Task 1: Due times and the overdue job

**Files:**
- Create: `migrations/0044_approval_due.sql`
- Create: `internal/core/approval/jobs.go`
- Modify: `internal/core/approval/{flow.go,types.go,service.go,queries.sql,module.go}`, regenerate the store
- Modify: `internal/core/notification/{types.go,service.go,i18n/vi.json,i18n/en.json}` (new kind `approval_overdue`, emailed)
- Modify: `docs/adr/0036-approval-flows.md`, the part-6 bullet: one sentence saying a step's due time is fixed when the step starts, and the job claims each overdue step once.
- Test: `internal/core/approval/overdue_test.go`

**Interfaces:**

```sql
ALTER TABLE approval.instance_steps
    ADD COLUMN due_hours int CHECK (due_hours BETWEEN 1 AND 720),
    ADD COLUMN due_at timestamptz,
    ADD COLUMN overdue_notified_at timestamptz;
-- The overdue job scans only open steps that have a due time and were not yet told.
CREATE INDEX instance_steps_due_idx ON approval.instance_steps (due_at) WHERE status = 'open' AND overdue_notified_at IS NULL;
```

```go
// StepDef gains:
DueHours *int32 `json:"due_hours,omitempty" minimum:"1" maximum:"720" required:"false" doc:"Hours after the step opens before approvers and submitter are told it is overdue"`

// notification
ApprovalOverdue Kind = "approval_overdue" // registered as emailed

// approval
type overdueArgs struct{} // Kind() "approval.overdue"; Spec() platform.JobSpec{System: true}
func (s *Service) NotifyOverdue(ctx context.Context) (int, error) // claims and notifies every overdue step, each in its own transaction; returns how many
```

- `checkFlow`: a `due_hours` outside 1–720 gives `invalid_count` at the step's path. Reuse the existing count reason if there is one; otherwise add `invalid_due`.
- `InsertStep` stores `due_hours`. `StartStep` sets `due_at = CASE WHEN due_hours IS NULL THEN NULL ELSE now() + due_hours * interval '1 hour' END`.
- `ClaimOverdue :one`: the claim is one row at a time, so each runs in its own transaction.

  ```sql
  UPDATE approval.instance_steps SET overdue_notified_at = now()
  WHERE (instance_id, position) = (
      SELECT s.instance_id, s.position FROM approval.instance_steps s JOIN approval.instances i ON i.id = s.instance_id
      WHERE s.status = 'open' AND i.status = 'open' AND s.overdue_notified_at IS NULL AND s.due_at < now()
      ORDER BY s.due_at LIMIT 1 FOR UPDATE OF s SKIP LOCKED)
  RETURNING instance_id, position, approvers;
  ```

  `NotifyOverdue` loops `InTx(claim → notify)` until the claim returns no row. To notify:
  1. load the instance and the document (`record.Get`; no lock, since nothing is written to it);
  2. compute the pending approvers, who haven't voted (`CastVotes`), and the `excluded` set;
  3. call `requestVotes`, generalised to take the `notification.Kind`, so delegates are included and filtered, for the pending approvers;
  4. send `ApprovalOverdue` to the submitter, unless the submitter is already among the users told.
- `StepView` gains `DueAt *string \`json:"due_at" format:"date-time"\`` and `Overdue bool \`json:"overdue"\``, the latter true when the step is open and `due_at < now`.
- `module.go`:
  - `Workers` registers the overdue worker; it needs the `*Service`, so use a closure.
  - `Periodic` holds `river.NewPeriodicJob(river.PeriodicInterval(15*time.Minute), …, &river.PeriodicJobOpts{RunOnStart: true})`.
- Notification email texts: `notification.email.approval_overdue.subject` "Một chứng từ đã quá hạn duyệt" / "A document is overdue for approval"; body "Một bước duyệt đã quá thời hạn. Mở mmerp để xem." / "An approval step is past its due time. Open mmerp to see it." Follow the neighbouring texts' tone.

- [ ] **Step 1: Write the failing tests** (`overdue_test.go`). Make the deadline pass by setting `due_at` in the past with an `UPDATE` through `platform.DBFrom`.
  - `TestOverdueTellsPendingVotersAndSubmitter`: step `user [c1, c2]`, approvals 2, `due_hours` 1. `c1` approves. Push `due_at` into the past and call `NotifyOverdue`. It returns 1. `c2` and `submitter` each have one `approval_overdue` notification; `c1` has none.
  - `TestOverdueNotifiesOnce`: call `NotifyOverdue` twice → the second returns 0, and the notification count is unchanged. Also run two concurrent `NotifyOverdue` calls → their total is 1.
  - `TestOverdueIgnoresDecidedSteps`: a step approved before its due time, a withdrawn instance and a rejected one, all pushed past due → 0.
  - `TestNoDueTimeNeverOverdue`: a step without `due_hours` → `due_at` is null, and `NotifyOverdue` returns 0.
  - `TestOverdueTellsDelegates`: `c2` delegates to `big_boss` (active), and `big_boss` is told too.
  - `TestStepViewShowsOverdue`: the panel step has `due_at` set and `overdue` true after the push.
- [ ] **Step 2: Run** `go test ./internal/core/approval -run Overdue` → FAIL.
- [ ] **Step 3: Implement.** Run `make gen` and commit the sqlc store only; Task 3 commits `api/openapi.json` and `schema.gen.ts`.
- [ ] **Step 4: Run** `go build ./...`, `go test ./internal/...` and `make lint` → PASS.
- [ ] **Step 5: Commit** `feat(approval): steps may be due, and overdue steps are announced once`.

---

### Task 2: Notification texts in the app

**Files:**
- Modify: `web/src/core/pages/NotificationsPage.tsx`: icon and colour for `approval_overdue`, `IconClockExclamation` (or the closest Tabler icon that exists) with `danger`.
- Modify: `web/src/core/i18n/{vi,en}/meta.json`: `core.notifications.kind.approval_overdue`, "Quá hạn duyệt" / "Overdue for approval".
- Modify: `web/src/core/notificationKeys.ts` if kinds are listed there.

- [ ] **Step 1:** Implement, following the existing kinds.
- [ ] **Step 2: Run** `make lint` and `pnpm -C web test` → clean.
- [ ] **Step 3: Commit** `feat(web): overdue approval notifications`.

---

### Task 3: The editor sets due times; the panel shows them; e2e

**Files:**
- Regenerate: `api/openapi.json`, `web/src/shared/api/schema.gen.ts`
- Modify: `web/src/shared/document/flowForm.ts` (+ test):
  - `ElementValues` gains `dueHours: string`; `""` means none.
  - `toFlowValues` maps `due_hours` to its string.
  - `fromFlowValues` writes `due_hours` only when the field is non-empty.
- Modify: `web/src/shared/document/flowEditor/StepCard.tsx`: an optional `IntegerField` "Thời hạn (giờ)" with `min 1`, `max 720`, and description "Quá hạn thì báo cho người duyệt và người gửi." / "When it passes, approvers and the submitter are told.", following `content.md`.
- Modify: `web/src/shared/document/ApprovalPanel.tsx`: an open step with `overdue` shows a `StatusBadge` "Quá hạn" in danger, and a step with `due_at` shows "Hạn: <date time>" in `dimmed`. Use the app's date-time formatter.
- Modify: i18n vi and en.
- Create: `web/e2e/approval-overdue.spec.ts`: an administrator sets "Thời hạn (giờ)" = 24 on a leave rule step, saves and reloads, and the value shows. Then, via the API, a leave request is sent and its panel shows "Hạn:". Test the overdue badge only if the spec can push `due_at` without database access; otherwise rely on the Go test. Use the exclusive project with snapshot and restore of the rule.

- [ ] **Step 1: Write the failing `flowForm.test.ts` cases:**
  - `round-trips due hours`: `{step: {…, due_hours: 24}}` survives the round trip.
  - `no due hours stays absent`: an element with `dueHours ""` writes no `due_hours` key.
- [ ] **Step 2: Run** `pnpm -C web test -- src/shared/document/flowForm.test.ts` → FAIL.
- [ ] **Step 3: Implement** the rest. Follow the `mmerp-ui` skill and finish with its checklist.
- [ ] **Step 4: Run** `make gen` (a second run shows no diff), `make lint`, `pnpm -C web test`, then the full `make e2e` → pass.
- [ ] **Step 5: Commit** `feat(web): approval steps show and take due times`.
