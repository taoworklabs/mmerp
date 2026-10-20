# Approval flows, part 5: delegation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A user declares "from date A to date B, user X approves for me". An administrator can declare it for anyone. While the delegation is active, X may vote in any open step where the user holds a seat. The history shows "X for Y".

**Architecture:**
- A delegation is a row in `approval.delegations`.
- Delegation is resolved when a vote is cast, not when a step starts. So a delegation declared while a document already waits still lets the delegate act, and approver lists stay fixed as today.
- A vote keeps its seat (`user_id`, the approver) and gains `cast_by` (the delegate, null when the approver votes in person).
- A unique index on `(instance_id, position, coalesce(cast_by, user_id))` keeps one vote per person per step. The existing primary key keeps one vote per seat.
- Delegation is one hop: a delegate's own delegation is never followed.
- The delegate must pass the same checks as any voter: not the submitter, not a subject of the document, and able to view it.

**Tech Stack:** Go, Postgres (sqlc, SQL migrations), Huma; React + TypeScript; Vitest; Playwright.

**Spec:** `docs/superpowers/specs/2026-10-19-approval-flows-design.md` (goal 7; "My calls" on one hop, one vote, business dates, every product). The decision is recorded in `docs/adr/0036-approval-flows.md` (part 5 bullet). Ticket #95. Parts 1–4 are done on this branch: read `internal/core/approval/{service.go,types.go,handler.go,queries.sql}`, especially `act`, `vote`, `Inbox`, `DocumentInstance`, `startStep` and `excluded`.

## Global Constraints

- **Never self-approval.** Neither the submitter nor a subject of the document votes, in person or as a delegate. A delegate without `view` on the document never votes.
- **One person, one vote per step.** This holds whether they vote as a member, as a delegate, or both.
- **One hop.** If X has delegated to Z, Z gets nothing of Y's seats through X.
- **Dates are business dates in the tenant time zone.** A delegation is active on day D when `starts_on <= D <= ends_on`, where D is `setting.Today`.
- **Lock order unchanged.** A vote reads the instance, then `record.Lock`, then the instance `FOR UPDATE`. Writing a delegation locks nothing of any document.
- **Who manages delegations:**
  - a user manages their own;
  - holders of `core.approval.manage` manage anyone's;
  - no other permission is added.
- **Migrations** are SQL only, in a new file `0043`.
- English identifiers. Comments short, say why, never reference docs. i18n in `vi` and `en`. Tokens only in the UI. Buttons come from `allowed_actions`.
- Conventional Commits, ending with `Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`. Stage explicit paths. Branch `approval-flows`.

## Review Focus

1. **A user is both a member of a step and the delegate of another member.** They vote once; a second attempt gives `approval_already_voted`. Test: `TestDelegateVotesOncePerStep` (Task 3).
2. **The delegate is the submitter, or a subject of the document.** They get no actions and their vote is refused (`self_approval`); the step still waits for the absent approver or a reassign. Test: `TestDelegateWhoSubmittedCannotVote` (Task 3).
3. **A delegation that ended yesterday, or starts tomorrow,** gives nothing. Test: `TestDelegationOnlyWhileActive` (Task 3).
4. **Chained delegations.** Y delegates to X and X delegates to Z: Z can't vote Y's seat. Test: `TestDelegationIsOneHop` (Task 3).
5. **Overlapping delegations of one user** (two rows covering the same day) are refused when created, under a lock, so concurrent creates can't both pass. Test: `TestDelegationsDoNotOverlap` (Task 2).

---

### Task 1: Docs and glossary

**Files:**
- Modify: `CONTEXT.md`: add **Delegation** (vi: "Ủy quyền duyệt"): a user's dated declaration that another user votes for them in approval steps; one hop; one vote per person per step.
- Modify: `documents.md`: in "Shared features", the Approval row, describe delegation in one or two sentences.
- Modify: `docs/adr/0036-approval-flows.md`: only add, to the part-5 bullet, that delegation is resolved at vote time and not at step start, and why: a delegation declared while a document already waits still works, and approver lists stay fixed.

- [ ] **Step 1:** Write them.
- [ ] **Step 2: Commit** `docs(approval): delegation`.

---

### Task 2: Delegations: storage and management

**Files:**
- Create: `migrations/0043_approval_delegations.sql`
- Modify: `internal/core/approval/{queries.sql,service.go (or a new service_delegation.go),types.go,handler.go,deps.go}`. Add `Setting *setting.Service` to `Deps` for `Today`, and wire it in `internal/app/app.go` and `internal/core/coretest`. Check first that `setting` doesn't import `approval`.
- Test: `internal/core/approval/delegation_test.go`

**Interfaces:**

```sql
CREATE TABLE approval.delegations (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     bigint NOT NULL REFERENCES iam.users,
    delegate_id bigint NOT NULL REFERENCES iam.users,
    starts_on   date NOT NULL,
    ends_on     date NOT NULL,
    created_by  bigint NOT NULL REFERENCES iam.users,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CHECK (delegate_id <> user_id),
    CHECK (ends_on >= starts_on)
);
CREATE INDEX delegations_delegate_idx ON approval.delegations (delegate_id, starts_on, ends_on);
CREATE INDEX delegations_user_idx ON approval.delegations (user_id, starts_on, ends_on);

ALTER TABLE approval.step_votes ADD COLUMN cast_by bigint REFERENCES iam.users;
-- One vote per person per step, whether in person or for someone else.
CREATE UNIQUE INDEX step_votes_one_per_person_idx ON approval.step_votes (instance_id, position, (coalesce(cast_by, user_id)));
```

```go
type Delegation struct {
	ID           int64  `json:"id"`
	UserID       int64  `json:"user_id"`
	UserName     string `json:"user_name"`
	DelegateID   int64  `json:"delegate_id"`
	DelegateName string `json:"delegate_name"`
	StartsOn     string `json:"starts_on" format:"date"`
	EndsOn       string `json:"ends_on" format:"date"`
	Active       bool   `json:"active"` // today falls within it
	// delete, for its user and for core.approval.manage holders.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

type DelegationInput struct {
	UserID     *int64 `json:"user_id,omitempty" doc:"Whose seats; omitted means the actor's own"`
	DelegateID int64  `json:"delegate_id"`
	StartsOn   string `json:"starts_on" format:"date"`
	EndsOn     string `json:"ends_on" format:"date"`
}

func (s *Service) Delegations(ctx context.Context) ([]Delegation, error)              // the actor's own, plus everyone's for core.approval.manage; ends_on >= today, by starts_on
func (s *Service) CreateDelegation(ctx context.Context, in DelegationInput) (int64, error)
func (s *Service) DeleteDelegation(ctx context.Context, id int64) error
func (s *Service) CanManageAllDelegations(ctx context.Context) (bool, error)        // for the page's "for whom" field
```

- **Validation, in order:**
  1. A `user_id` other than the actor's needs `core.approval.manage` tenant-wide (`iam.RequireCore`), else `forbidden`.
  2. `delegate_id == user_id` gives `invalid_delegation` with `{reason: "self"}`.
  3. An unknown user or delegate gives `not_found`.
  4. `ends_on < starts_on` gives `{reason: "dates"}`.
  5. `ends_on < today` gives `{reason: "past"}`.
  6. An overlap with an existing delegation of the same `user_id` gives `{reason: "overlap", id: <existing>}`.

  The overlap check runs after `pg_advisory_xact_lock(hashtextextended('approval.delegation', user_id))`, inside `InTx`. All errors use `ErrInvalidDelegation(reason)`. Audit `approval.delegation_created` / `approval.delegation_deleted`.
- **Delete:** allowed for the delegation's user or a `core.approval.manage` holder; anyone else gets `not_found`. It deletes the row. Votes already cast keep their `cast_by`.
- **Routes:**
  - `GET /approval-delegations`;
  - `POST /approval-delegations` (201 `{id}`);
  - `DELETE /approval-delegations/{id}` (204);
  - `GET /approval-delegations/meta` → `{"manage_all": bool}`.

  Every signed-in user may call them. They are core routes and need no product gate.

- [ ] **Step 1: Write the failing tests** (`delegation_test.go`, using the approval fixture's users):
  - `TestCreateOwnDelegation`: `boss` delegates to `big_boss` from today to +7 days. `Delegations(boss)` lists it with `Active` true and `allowed_actions` `["delete"]`. `Delegations(big_boss)` lists nothing, because a delegate doesn't manage it.
  - `TestDelegationRules`: one case per reason above, plus `hr` creating one for `boss` → `forbidden` and admin creating one for `boss` → OK.
  - `TestDelegationsDoNotOverlap`: a second delegation of `boss` that overlaps by one day → `overlap`. One that is adjacent (`starts_on` = the previous `ends_on` + 1) → OK. Two concurrent creates of overlapping rows (use `pgtest.AwaitLockWait` as other tests do) → exactly one succeeds.
  - `TestDeleteDelegation`: the owner deletes it → gone; another user → `not_found`.
- [ ] **Step 2: Run** `go test ./internal/core/approval -run Delegation` → FAIL.
- [ ] **Step 3: Implement.** Run `make gen` and commit the sqlc store only.
- [ ] **Step 4: Run** `go build ./...`, `go test ./internal/...` and `make lint` → PASS.
- [ ] **Step 5: Commit** `feat(approval): delegations`.

---

### Task 3: Voting through a delegation

**Files:**
- Modify: `internal/core/approval/{service.go,queries.sql,types.go}`
- Test: `internal/core/approval/delegation_test.go`

**Interfaces:**
- `seatFor(ctx, inst, step, actor) (seat int64, onBehalfOf *int64, err error)` picks the seat the actor votes, in this order:
  1. the actor's own seat, if the actor is a voter who hasn't voted;
  2. otherwise the first pending seat, in approver order, of a voter with an active delegation to the actor (one hop: only rows with `delegate_id = actor`);
  3. otherwise `ErrForbidden`.

  When the actor has already cast a vote in this step (as anyone), it returns `ErrAlreadyVoted`.
- `vote` uses `seatFor`, then:
  - the excluded and view checks apply to the actor (the delegate);
  - `InsertVote(instance, position, user_id = seat, cast_by = actor if delegating, …)`;
  - a violation of `step_votes_one_per_person_idx` maps to `ErrAlreadyVoted`;
  - audit adds `"on_behalf_of": seat` when delegating.
- `Inbox`: an open step also counts for the actor when one of its pending seats has an active delegation to the actor. `InboxItem` gains `OnBehalfOfName *string \`json:"on_behalf_of_name"\`` (the first such seat's name, when the actor has no own pending seat).
- `DocumentInstance`: a step's `allowed_actions` include `approve`/`reject` when `seatFor` would succeed and the actor passes the excluded and view checks. `StepVoter` gains `CastByName *string \`json:"cast_by_name"\``.
- `startStep` also notifies, with `approval_requested`, the active delegates of the step's approvers, filtered by the excluded and view checks.

- [ ] **Step 1: Write the failing tests:**
  - `TestDelegateApproves`: a step for `boss`; `boss` delegates to `big_boss` today. `big_boss` sees it in their inbox, with `on_behalf_of_name` "boss", and in their `allowed_actions`. They approve: posted; the voter row shows `boss` with `cast_by_name` "big_boss".
  - `TestDelegationDeclaredWhileWaiting`: send first, then create the delegation, then the delegate approves.
  - `TestDelegateVotesOncePerStep`: a step with voters `[c1, c2]` and approvals 2; `c2` delegates to `c1`. `c1` approves (own seat) and approves again → `approval_already_voted`; the step stays open.
  - `TestDelegateWhoSubmittedCannotVote`: `boss` delegates to `submitter`. The submitter has no actions on the step, and `Approve` → `self_approval`.
  - `TestDelegateWhoCannotViewCannotVote`: delegate `outsider` → no actions; `Approve` → `forbidden`.
  - `TestDelegationOnlyWhileActive`: a delegation that ended yesterday, and one that starts tomorrow → `forbidden`.
  - `TestDelegationIsOneHop`: `boss`→`big_boss` and `big_boss`→`hr`; `hr` can't vote `boss`'s seat.
  - `TestDelegateIsNotified`: when a step starts for `boss` while `boss`→`big_boss` is active, `big_boss` gets an `approval_requested` notification.
- [ ] **Step 2: Run** → FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run** `go test ./internal/...` and `make lint` → PASS.
- [ ] **Step 5: Commit** `feat(approval): a delegate votes for an absent approver`.

---

### Task 4: Frontend; e2e

**Files:**
- Regenerate: `api/openapi.json`, `web/src/shared/api/schema.gen.ts`
- Create: `web/src/core/pages/DelegationsPage.tsx`, an app-level route `/delegations` next to `/inbox`. Find where `/inbox` is routed (`web/src/app`) and follow it. The page:
  - is a `ListPage` "Ủy quyền duyệt" with "Tạo ủy quyền" in the header;
  - has a `DataTable` with columns Người ủy quyền (shown only when `manage_all`), Người được ủy quyền, Từ ngày, Đến ngày, and a status badge (Đang hiệu lực / Sắp tới);
  - offers Delete in each row's menu from `allowed_actions`, with a confirm;
  - creates through a `FormModal`: user (`SelectField`, only when `manage_all`), delegate (`SelectField` of users), from (`DateField`), to (`DateField`).

  `invalid_delegation` reasons map to fields: `self` → delegate, `dates`/`past` → to, `overlap` → from. Use `errorFields` or a banner, as `forms.md` says.
- Modify: `web/src/core/pages/InboxPage.tsx`: a header link or button "Ủy quyền duyệt" → `/delegations`; a row's description adds "thay {{name}}" when `on_behalf_of_name` is set.
- Modify: `web/src/shared/document/ApprovalPanel.tsx`: a voter with `cast_by_name` reads "{{cast_by}} thay {{name}}".
- Modify: i18n vi and en for all the new text, following `content.md`; also `shared.error.invalid_delegation`.
- Create: `web/e2e/approval-delegation.spec.ts`, in the exclusive project if it touches tenant-wide rules, with snapshot and restore.

- [ ] **Step 1: Implement.** Use the `mmerp-ui` skill and finish with its checklist. The terms come from `CONTEXT.md`.
- [ ] **Step 2: e2e:**
  - `an absent manager's delegate approves a leave request`:
    1. The manager opens `/delegations` and delegates to a colleague from today to +3 days.
    2. An employee sends a leave request.
    3. The colleague sees it in the inbox, marked "thay <manager>", and approves it.
    4. The document's panel shows "<colleague> thay <manager>".
  - `the delegation form refuses delegating to oneself` (the field error).
  - `an administrator creates a delegation for another user` (the "for whom" field is visible only to them).
- [ ] **Step 3: Run** `make gen` (a second run shows no diff), `make lint`, `pnpm -C web test`, then the full `make e2e` → pass.
- [ ] **Step 4: Commit** `feat(web): approval delegations`.
