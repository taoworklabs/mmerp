# Approval flows, part 1: the flow engine — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the approval rule's chain of steps with a structured flow (approval steps with approval and rejection counts, choose-one and parallel splits, AND/OR conditions) for the `post` action, keeping every guarantee `approval` gives today.

**Architecture:** A rule stores a flow (a JSON tree). At submit, `approval` evaluates every condition once and *resolves* the flow into a plan: only the steps that apply, numbered 1..n in reading order, composed by sequence and parallel only. The instance stores the plan; each step row keeps its approvers, counts and status, and each person's vote is its own row. After every vote a pure function over the plan says which steps start next and whether the flow is done. Step numbers stay `int32`, so the API keeps its `step` field.

**Tech Stack:** Go, Postgres (sqlc, SQL migrations), Huma, React + TypeScript (openapi-typescript client), Vitest, Playwright.

**Spec:** `docs/superpowers/specs/2026-10-19-approval-flows-design.md` (goals 2, 3, 4 for header fields, 6 for `post`). Parts 2 to 6 (flow editor, line conditions, other actions, delegation, overdue) are later plans; this plan builds nothing of them.

## Global Constraints

- `approval` is core: it imports only `record`, `iam`, `audit`, `notification`, `platform`; it never imports a product.
- Lock order of an approval action is unchanged: read the instance → `record.Lock` → instance `FOR UPDATE` → checks.
- One transaction writes one document; a non-stale error in `CompleteApproval` rolls back the whole decision.
- Never auto-approve; the submitter and the type's `Subjects` never vote; nobody without `view` on the document votes.
- Migrations are SQL only, new file, never edit an existing one. Next number: `0041`.
- All code, identifiers, error codes and test names in English; user text through i18n in `vi` and `en`.
- Comments short, say why; never reference docs, ADRs or milestones in comments.
- Conventional Commits; commits end with `Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`. Stage explicit paths, never `git add -A`.
- Branch `approval-flows` (already created off `main`).

## Review Focus

1. **A condition that can't be evaluated** (missing amount, malformed value) inside an `all`/`any` expression or a choose-one branch: the leaf counts as true, so doubt adds approval. Test in Task 2 (`TestResolveDoubtTakesTheBranch`).
2. **A parallel split where no branch applies, or a branch whose body resolves empty**: it disappears from the plan; a flow that resolves to nothing posts at once. Test in Task 2 (`TestResolveDropsEmptyParts`).
3. **The same person in two open parallel steps**: they vote in each step separately (one vote per step, not per instance). Test in Task 3 (`TestOneVotePerStep`).
4. **Rejecting while a parallel sibling is open**: the instance closes `rejected`, the sibling becomes `cancelled`, its approvers no longer see it in the inbox. Test in Task 3 (`TestRejectCancelsParallelSteps`).
5. **A rule saved before this change, and an instance open across the migration**: the rule becomes the equivalent flow, the open instance keeps its current step open and its earlier votes. Test in Task 3 (`TestMigrationConvertsRulesAndOpenInstances`).

---

### Task 1: ADR-0036 and the glossary

**Files:**
- Create: `docs/adr/0036-approval-flows.md`
- Modify: `CONTEXT.md` (lines 49–50: Approval rule, Approval inbox; add terms after line 51)
- Modify: `documents.md` (section "Shared features", row **Approval**; section "Approvers by module", bullet "A step is approved by anyone in the list")
- Modify: `docs/adr/0015-approval.md` (Status line only: add "· rules and instances superseded by ADR-0036")

Use the `mattpocock-skills:domain-modeling` skill for the ADR and glossary.

- [ ] **Step 1: Write the ADR.** Status `accepted · 2026-10-19 · supersedes the rule and instance model of ADR-0015; ADR-0022 unchanged`. Context: the spec's intent. Decision, covering the whole spec (parts 1–6) at decision level:
  - Flow shape: sequence of elements; element = approval step | choose-one split (ordered branches, first holding branch runs, always an otherwise) | parallel split (every holding branch runs, join waits for all). Nested; no loops; no free arrows.
  - Approval step: approver (`user` with 1–20 `user_ids`, `role`, `module`), `approvals` and `rejections` (1–20, default 1). Votes: one per person per step. A step passes at `approvals` approvals; it rejects at `rejections` rejections or as soon as the remaining voters can no longer reach `approvals`.
  - Approvers fixed when the step starts, filtered as today (submitter, `Subjects`, no `view`). Empty → the rule's fallback role, counts unchanged. Fewer eligible than `approvals` → `no_approver` with param `step`; never lowers the count.
  - Conditions: expression `{all:[…]}` | `{any:[…]}` | leaf `{field, op, value}` (ops unchanged). A leaf that can't be evaluated holds.
  - Resolution at submit: conditions evaluated once; the instance stores the resolved plan (sequence/parallel of step numbers 1..n in reading order); editing the rule never changes an open instance.
  - Rejection of any step rejects the instance, cancels open siblings, and returns the document to draft.
  - Later parts, decided now: rules keyed by (document type, action) with every write action gateable and the document locked while waiting (part 4); line fields declared by the record type (part 3); delegation one hop, one vote per person (part 5); per-step due time in wall-clock hours, notifying approvers and submitter once (part 6).
  - Rejected alternatives: Odoo-Studio levels only (can't express amount tiers without duplication); free BPMN graph (loops, unreachable nodes, unvalidatable by customers); exclusive branching only (combinatorial blow-up for independent reasons).
- [ ] **Step 2: Update `CONTEXT.md`.** Replace **Approval rule** with: "the approval flow of a document type: approval steps arranged in sequence, choose-one splits and parallel splits, plus a fallback role." Add **Approval flow**, **Approval step** (vi: "Bước duyệt"; approvers plus approval and rejection counts), **Choose-one split** (vi: "Rẽ nhánh chọn một"), **Parallel split** (vi: "Rẽ nhánh song song"), **Approval count / rejection count** (vi: "Số người cần duyệt / số người từ chối"). Change **Approval inbox** to "…waiting for the user at an open step they have not voted on."
- [ ] **Step 3: Update `documents.md`.** Approval row: describe flows; "Deliberate limits" becomes "No loops, scripts or free-form BPMN". The "anyone in the list" bullet becomes "A step passes when its approval count is reached."
- [ ] **Step 4: Commit**

```bash
git add docs/adr/0036-approval-flows.md docs/adr/0015-approval.md CONTEXT.md documents.md
git commit -m "docs(approval): decide approval flows"
```

---

### Task 2: Flow model, validation, resolution and progress (pure Go)

**Files:**
- Create: `internal/core/approval/flow.go`
- Create: `internal/core/approval/flow_test.go` (package `approval`, no database)
- Modify: `internal/core/approval/types.go` (add `Approver.UserIDs` only)

This task is additive: the new types live in `flow.go` next to today's `Step`, `Condition` and `RuleInput.Steps`, which Task 3 removes. The package keeps building and today's tests keep passing.

**Interfaces:**
- Produces (exported, in `types.go` / `flow.go`):

```go
type Flow []Element

// Exactly one of Step and Split is set.
type Element struct {
	Step  *StepDef `json:"step,omitempty"`
	Split *Split   `json:"split,omitempty"`
}

type StepDef struct {
	Approver   Approver `json:"approver"`
	Approvals  int32    `json:"approvals" minimum:"1" maximum:"20" default:"1"`
	Rejections int32    `json:"rejections" minimum:"1" maximum:"20" default:"1"`
}

type SplitMode string

const (
	ChooseOne SplitMode = "one"
	Parallel  SplitMode = "all"
)

type Split struct {
	Mode      SplitMode `json:"mode" enum:"one,all"`
	Branches  []Branch  `json:"branches" minItems:"1" maxItems:"10" nullable:"false"`
	Otherwise Flow      `json:"otherwise,omitempty"` // choose-one only
}

type Branch struct {
	Condition *Expr `json:"condition,omitempty"` // nil holds
	Body      Flow  `json:"body" nullable:"false"`
}

// Exactly one of All, Any and the leaf (Field, Op, Value) is set.
type Expr struct {
	All   []Expr `json:"all,omitempty"`
	Any   []Expr `json:"any,omitempty"`
	Field string `json:"field,omitempty"`
	Op    Op     `json:"op,omitempty" enum:"gt,gte,eq,within"`
	Value string `json:"value,omitempty" maxLength:"30"`
}

// Approver gains UserIDs (kind user: 1–20 users). This task only adds the field;
// Task 3 removes UserID and switches every use.
type Approver struct {
	Kind    ApproverKind `json:"kind" enum:"role,user,module"`
	Product string       `json:"product,omitempty"`
	Role    string       `json:"role,omitempty"`
	UserIDs []int64      `json:"user_ids,omitempty" maxItems:"20"`
}

// Plan is a resolved flow: a leaf (Step > 0), a sequence or a parallel group.
type Plan struct {
	Step int32  `json:"step,omitempty"`
	Seq  []Plan `json:"seq,omitempty"`
	Par  []Plan `json:"par,omitempty"`
}
```

- Produces (unexported, used by Task 3):
  - `func (f Flow) normalize() Flow` — sets `Approvals`/`Rejections` of 0 to 1, recursively (Go callers may leave them unset).
  - `func checkFlow(f Flow, fieldKind func(string) (record.FieldKind, bool), roleExists func(product, role string) bool, moduleApprovers bool) (path string, reason string)` — `""` when valid. `path` names the element, e.g. `"2.branches.1.body.1"` (1-based). Reasons: `"empty_element"`, `"two_kinds"` (both Step and Split, or Expr mixing all/any/leaf), `"unknown_role"`, `"missing_user"`, `"no_module_approvers"`, `"invalid_condition"` (unknown field, op not in `opsByKind`, non-number value for number/org-unit kinds), `"approvals_exceed_users"` (kind `user` with `Approvals > len(UserIDs)`), `"otherwise_on_parallel"`, `"too_large"` (more than 50 steps or 50 condition leaves in the whole flow, or nesting deeper than 5 splits).
  - `func resolve(f Flow, holds func(Expr) (bool, error)) (Plan, []StepDef, error)` — evaluates conditions, numbers the applying steps 1..n in reading order, returns the plan and the steps indexed `steps[n-1]`. Choose-one: first holding branch, else `Otherwise`. Parallel: every holding branch. Empty sequences and groups are dropped; a group with one child collapses into it. A flow with no applying step returns `Plan{}` and no steps.
  - `func (p Plan) next(done map[int32]bool, started map[int32]bool) (start []int32, finished bool)` — steps to start now: in a `Seq`, the first child not fully done; in a `Par`, every child not fully done; a leaf not done and not started is returned. `finished` is true when every leaf is done.
  - `func (e Expr) eval(leaf func(Expr) (bool, error)) (bool, error)` — `all`: every child; `any`: some child; else `leaf(e)`.

- [ ] **Step 1: Write the failing tests** in `flow_test.go`:
  - `TestResolveChooseOneTakesTheFirstHoldingBranch`: flow `[split one: (amount>2000 → A, B), (amount>500 → A), otherwise → C]` with `amount=600` gives plan `{Seq:[{Step:1}]}` and steps `[A]`; with `amount=100`, `[C]`; with `amount=3000`, plan `{Seq:[{Step:1},{Step:2}]}`, steps `[A, B]`.
  - `TestResolveParallelRunsEveryHoldingBranch`: flow `[step H, split all: (always → tier step T), (kind=a → P)]`, kind `a` gives plan `{Seq:[{Step:1},{Par:[{Step:2},{Step:3}]}]}`; kind `b` gives `{Seq:[{Step:1},{Step:2}]}` (single-child group collapsed).
  - `TestResolveDropsEmptyParts`: a parallel split where no branch holds and a choose-one whose otherwise is empty and no branch holds both vanish; flow of only those gives `Plan{}`, zero steps.
  - `TestResolveDoubtTakesTheBranch`: `holds` stub returns true for the leaf it can't read (the service's rule); assert `{any:[{field:x,…}]}` and `{all:[a, b]}` evaluate via `eval` as all/any semantics with a stubbed leaf: `all[true,false]=false`, `any[false,true]=true`, `all[]` is rejected by `checkFlow` (`"empty_element"`).
  - `TestNextWalksSequenceAndParallel`: plan `{Seq:[1, {Par:[2,3]}, 4]}`: nothing done → start `[1]`; `done{1}` → `[2,3]`; `done{1,2}`, `started{3}` → `[]`, not finished; `done{1,2,3}` → `[4]`; all done → finished.
  - `TestCheckFlowReasons`: one case per reason above, asserting `path` and `reason` (e.g. `[step, split all with otherwise]` → `"2"`, `"otherwise_on_parallel"`; user step with `UserIDs:[1]`, `Approvals:2` → `"1"`, `"approvals_exceed_users"`).
  - `TestNormalizeDefaultsCounts`: zero counts become 1 inside nested branches.

- [ ] **Step 2: Run to see them fail**

Run: `go test ./internal/core/approval -run 'TestResolve|TestNext|TestCheckFlow|TestNormalize'`
Expected: FAIL to compile (`undefined: resolve`, …).

- [ ] **Step 3: Implement** `flow.go` with the signatures above. `checkFlow` reuses `opsByKind` and `record.ParseNumber` as `checkStep` does today.
- [ ] **Step 4: Run to see them pass**

Run: `go test ./internal/core/approval -run 'TestResolve|TestNext|TestCheckFlow|TestNormalize'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/core/approval/flow.go internal/core/approval/flow_test.go internal/core/approval/types.go
git commit -m "feat(approval): resolve flows of steps and splits"
```

---

### Task 3: Storage, submission and voting

**Files:**
- Create: `migrations/0041_approval_flows.sql`
- Modify: `internal/core/approval/queries.sql`, regenerate `internal/core/approval/internal/store` (`make gen`)
- Modify: `internal/core/approval/service.go`, `internal/core/approval/types.go`, `internal/core/approval/handler.go`
- Test: `internal/core/approval/service_test.go`, `internal/core/approval/migration_test.go` (new)

**Interfaces:**
- Consumes: Task 2's `Flow`, `Plan`, `resolve`, `next`, `checkFlow`, `normalize`, `Expr.eval`.
- Removes: `Step`, `Condition`, `Approver.UserID`, `RuleInput.Steps`, `checkStep`. `RuleInput` becomes:

```go
type RuleInput struct {
	Flow            Flow   `json:"flow" minItems:"1" nullable:"false"`
	MaxLevels       int32  `json:"max_levels" minimum:"1" maximum:"10" default:"3"`
	FallbackProduct string `json:"fallback_product" minLength:"1"`
	FallbackRole    string `json:"fallback_role" minLength:"1"`
}
```

- Produces (service, signatures unchanged unless listed):
  - `Submit`, `Withdrawn`, `Approve(ctx, id int64, step int32) error`, `Reject(ctx, id int64, step int32, reason string) error`, `Inbox`, `DocumentInstance`, `Rules`, `SaveRule(ctx, docType string, in RuleInput) error`, `DeleteRule`, `Users`.
  - Changed: `Reassign(ctx context.Context, id int64, step int32, from int64, login string) error` — replaces voter `from` (who has not voted) with the user `login`; counts unchanged. Handler body gains `"from": int64`.
  - API types:

```go
type StepVoter struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Decision  *string `json:"decision" enum:"approved,rejected"`
	Reason    *string `json:"reason"`
	DecidedAt *string `json:"decided_at" format:"date-time"`
}

type StepView struct {
	Position   int32       `json:"position"`
	Status     string      `json:"status" enum:"waiting,open,approved,rejected,cancelled"`
	Approvals  int32       `json:"approvals"`
	Rejections int32       `json:"rejections"`
	Fallback   bool        `json:"fallback"`
	Voters     []StepVoter `json:"voters" nullable:"false"` // empty while waiting
	// approve, reject (the actor may vote), reassign (fallback role holder).
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

type Instance struct {
	ID              int64      `json:"id"`
	Status          string     `json:"status" enum:"open,approved,rejected,withdrawn,stale"`
	SubmittedByName string     `json:"submitted_by_name"`
	SubmittedAt     string     `json:"submitted_at" format:"date-time"`
	Plan            Plan       `json:"plan"`
	Steps           []StepView `json:"steps" nullable:"false"` // by position
}
```

  `Instance.CurrentStep` and `Instance.AllowedActions` are removed; `InboxItem.Step` stays (the open step waiting for the actor).
  - Errors: `ErrNoApprover` gains `Params: {"step": n}` (build with a helper `errNoApprover(step int32) error`); `errInvalidRule(path, reason string)` with params `{"path": …, "reason": …}`; new `ErrAlreadyVoted = &platform.Error{Status: 409, Code: "approval_already_voted"}`.

- [ ] **Step 1: Write the migration** `0041_approval_flows.sql`:

```sql
-- Rules: steps (a chain) become a flow; a step with a condition becomes a choose-one
-- split with an empty otherwise, which skips it exactly as before.
ALTER TABLE approval.rules ADD COLUMN flow jsonb;
UPDATE approval.rules r SET flow = (
    SELECT jsonb_agg(CASE WHEN s->'condition' IS NULL OR s->'condition' = 'null'::jsonb THEN jsonb_build_object('step', n.def)
        ELSE jsonb_build_object('split', jsonb_build_object('mode', 'one', 'otherwise', '[]'::jsonb,
            'branches', jsonb_build_array(jsonb_build_object('condition', s->'condition', 'body', jsonb_build_array(jsonb_build_object('step', n.def))))))
        END ORDER BY i)
    FROM jsonb_array_elements(r.steps) WITH ORDINALITY e(s, i)
    CROSS JOIN LATERAL (SELECT jsonb_build_object('approvals', 1, 'rejections', 1, 'approver',
        CASE WHEN s->'approver'->>'kind' = 'user' THEN jsonb_build_object('kind', 'user', 'user_ids', jsonb_build_array(s->'approver'->'user_id'))
             ELSE s->'approver' END) AS def) n);
ALTER TABLE approval.rules ALTER COLUMN flow SET NOT NULL, DROP COLUMN steps;

-- Instances: the resolved plan replaces current_step; old instances were a plain sequence.
ALTER TABLE approval.instances ADD COLUMN plan jsonb;
UPDATE approval.instances i SET plan = jsonb_build_object('seq',
    (SELECT jsonb_agg(jsonb_build_object('step', s.position) ORDER BY s.position) FROM approval.instance_steps s WHERE s.instance_id = i.id));

-- Steps: counts and a status; each vote moves to its own row.
ALTER TABLE approval.instance_steps
    ADD COLUMN approvals int NOT NULL DEFAULT 1 CHECK (approvals BETWEEN 1 AND 20),
    ADD COLUMN rejections int NOT NULL DEFAULT 1 CHECK (rejections BETWEEN 1 AND 20),
    ADD COLUMN status text;
UPDATE approval.instance_steps s SET
    status = CASE WHEN s.decision IS NOT NULL THEN s.decision
                  WHEN i.status = 'open' AND s.position = i.current_step THEN 'open'
                  WHEN i.status = 'open' THEN 'waiting'
                  ELSE 'cancelled' END,
    approver = CASE WHEN s.approver->>'kind' = 'user'
                    THEN jsonb_build_object('kind', 'user', 'user_ids', jsonb_build_array(s.approver->'user_id')) ELSE s.approver END
FROM approval.instances i WHERE i.id = s.instance_id;
ALTER TABLE approval.instance_steps ALTER COLUMN status SET NOT NULL,
    ADD CONSTRAINT instance_steps_status_check CHECK (status IN ('waiting', 'open', 'approved', 'rejected', 'cancelled'));

CREATE TABLE approval.step_votes (
    instance_id bigint NOT NULL,
    position    int NOT NULL,
    user_id     bigint NOT NULL REFERENCES iam.users,
    decision    text NOT NULL CHECK (decision IN ('approved', 'rejected')),
    reason      text,
    decided_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (instance_id, position, user_id),
    FOREIGN KEY (instance_id, position) REFERENCES approval.instance_steps
);
INSERT INTO approval.step_votes (instance_id, position, user_id, decision, reason, decided_at)
SELECT instance_id, position, decided_by, decision, reason, decided_at FROM approval.instance_steps WHERE decided_by IS NOT NULL;

DROP INDEX approval.instance_steps_approvers_idx;
ALTER TABLE approval.instance_steps DROP COLUMN decided_by, DROP COLUMN decision, DROP COLUMN reason, DROP COLUMN decided_at;
ALTER TABLE approval.instances ALTER COLUMN plan SET NOT NULL, DROP COLUMN current_step;
CREATE INDEX instance_steps_open_idx ON approval.instance_steps USING gin (approvers) WHERE status = 'open';
```

- [ ] **Step 2: Rewrite `queries.sql`** for the new columns: `SaveRule`/`GetRule`/`ListRules` on `flow`; `InsertInstance` takes `plan`; `InsertStep(instance_id, position, approver, approvals, rejections)` with `status 'waiting'`; `StartStep` sets `approvers`, `fallback`, `status = 'open'`; `SetStepStatus(instance_id, position, status)`; `CancelOpenSteps(instance_id)` (`status = 'cancelled' WHERE status IN ('open','waiting')`); `InsertVote`; `StepVotes(instance_id, position)` → counts by decision; `Steps` returns step rows plus voters (`jsonb_agg` of `{id, name, decision, reason, decided_at}` over `approvers` left-joined to `step_votes`); `Inbox` filters `s.status = 'open' AND s.approvers @> ARRAY[@actor] AND NOT EXISTS (vote by actor)`, returning `s.position AS step`. Drop `SetCurrentStep`, `StepCount`, `Decide`. Run `make gen`.
- [ ] **Step 3: Write the failing tests** in `service_test.go`. Change the fixture helper to `f.rule(flow ...approval.Element)` (wraps them into `RuleInput{Flow: …, MaxLevels: 3, FallbackProduct: "test", FallbackRole: "hr"}`), add users `c1`, `c2`, `c3` (role `test.hr` at the unit, so they can view), and convert the existing tests to flows with unchanged assertions except: `inst.AllowedActions` → `inst.Steps[i].AllowedActions`; `ApproverNames` → names from `Voters`; `CurrentStep` → the open step's `Position`. New tests:
  - `TestGroupNeedsItsApprovalCount`: step `user [c1,c2,c3]`, approvals 2. `c1` approves → document still `pending_approval`, step 1 `open`; `c2` approves → `posted`.
  - `TestRejectionCountRejects`: approvals 1, rejections 2, voters `[c1,c2,c3]`: `c1` rejects → still open; `c2` rejects → document `draft`, instance `rejected`.
  - `TestStepRejectsWhenItCanNoLongerPass`: approvals 3, rejections 2, voters `[c1,c2,c3]`: `c1` rejects → document `draft` at once.
  - `TestOneVotePerStep`: `c1` approves step 1 twice → second returns `approval_already_voted`; a parallel flow with `c1` in both branches: `c1` votes in step 2 and step 3 separately, both succeed.
  - `TestParallelStepsOpenTogether`: flow `[split all: (→ user boss), (kind=a → user big_boss)]` with kind `a`: both steps `open`, both in their inboxes; boss approves → still pending; big_boss approves → `posted`.
  - `TestRejectCancelsParallelSteps`: same flow, boss rejects → instance `rejected`, step 2 `cancelled`, big_boss's inbox empty, `Approve(big_boss, …, 2)` → `approval_closed`.
  - `TestChooseOneByAmountTiers`: flow `[split one: (days>5 → big_boss), (days>2 → boss), otherwise → []]`: days 6 → one step for big_boss; days 3 → boss; days 1 → posts at once with no instance.
  - `TestAllAnyConditions`: flow `[split one: ({any:[{all:[days>2, kind=a]}, days>5]} → boss), otherwise → []]`: days 3 kind a → one step; days 3 kind b → posts at once; days 6 kind b → one step.
  - `TestTooFewEligibleRefuses`: approvals 3, voters `[c1, c2, submitter]` → sending returns `no_approver` with `Params["step"] == 1`, document stays `draft`.
  - `TestReassignReplacesOneVoter`: voters `[c1,c2]`, approvals 2: hr reassigns `c2` → `big_boss`; `c2` approving is `forbidden`; `c1` and `big_boss` approving posts. Reassigning a voter who already voted → `approval_already_voted`.
  - `TestMigrationConvertsRulesAndOpenInstances` in `migration_test.go`: `pool := pgtest.Empty(t)`; `platform.Migrate(ctx, pool, before)` where `before` is an `fstest.MapFS` copy of `migrations.FS` holding only files named below `0041`; insert with raw SQL a user, a rule `[{"approver":{"kind":"user","user_id":<u>},"condition":{"field":"days","op":"gt","value":"3"}},{"approver":{"kind":"module"}}]`, and an open instance at `current_step 2` with step 1 decided `approved` by the user and step 2 started with approvers `{<u>}`; then `platform.Migrate(ctx, pool, migrations.FS)`. Assert: the rule's `flow` equals `[{"split":{"mode":"one","otherwise":[],"branches":[{"condition":{"field":"days","op":"gt","value":"3"},"body":[{"step":{"approver":{"kind":"user","user_ids":[<u>]},"approvals":1,"rejections":1}}]}]}},{"step":{"approver":{"kind":"module"},"approvals":1,"rejections":1}}]` (compare as decoded JSON); the instance `plan` is `{"seq":[{"step":1},{"step":2}]}`; step statuses are `approved`, `open`; `approval.step_votes` holds one `approved` row for step 1.
- [ ] **Step 4: Run to see them fail**

Run: `make db && go test ./internal/core/approval`
Expected: FAIL (compile errors, then the new behaviours).

- [ ] **Step 5: Implement in `service.go`:**
  - `Submit`: read the rule, `json.Unmarshal` the flow, `resolve(flow, holdsFor(d))` where `holdsFor` wraps today's `holds` for a leaf; nothing → `(0, false, nil)`; insert the instance with the plan, one step row per resolved step, then `startSteps(ctx, inst, d, plan.next(nil, nil))`.
  - `startStep(ctx, inst, d, pos)`: as today (role/user/module with level climbing; filter `others`; empty → fallback role), then `if len(users) < int(st.Approvals) { return errNoApprover(pos) }`; `StartStep`; notify the step's users.
  - `act(ctx, id, step, fn)`: replace `inst.CurrentStep != step` with: the step row exists and its `status == "open"`, else `ErrClosed`.
  - `vote(ctx, inst, d, step, decision, reason)`: actor in `approvers` (else `ErrForbidden`), not excluded (else `ErrSelfApproval`), `InsertVote` (unique violation → `ErrAlreadyVoted`, via `platform.Violates`), audit `approval.<decision>` as today. Then count votes: `approved >= approvals` → step `approved`; `rejected >= rejections || approved + (len(approvers) - votes) < approvals` → step `rejected`; else return.
  - `Approve`: `vote`; on step approved, load all step statuses, `plan.next(done, started)`: start the returned steps; when `finished`, close `approved`, `CompleteApproval`, notify the submitter (today's code).
  - `Reject`: `vote` with the reason; on step rejected: `CancelOpenSteps`, close `rejected`, `record.Reject`, notify the submitter. A rejection vote that doesn't reject the step yet just records the vote.
  - `Reassign(ctx, id, step, from, login)`: fallback-role check as today; `from` in approvers and has no vote (else `ErrAlreadyVoted`); the new user passes the excluded/view checks and isn't already a voter; replace `from` with them in `approvers`; notify and audit `{"from": from, "to": user}`.
  - `DocumentInstance`: build `StepView`s from the `Steps` query, `AllowedActions` per step: `approve`,`reject` when open, actor in voters, no vote yet; `reassign` when open and actor holds the fallback role; all empty when the product gate refuses writes.
  - `Rules`/`SaveRule`: `RuleInput.Flow`; `SaveRule` normalizes, then `checkFlow(…)` → `errInvalidRule(path, reason)`; fallback role check as today.
  - `handler.go`: `reassignInput.Body` gains `From int64 \`json:"from"\``.
- [ ] **Step 6: Run to see them pass**

Run: `go test ./internal/core/approval`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add migrations/0041_approval_flows.sql internal/core/approval
git commit -m "feat(approval): run flows with approval counts and parallel steps"
```

---

### Task 4: Callers: product tests, record contract suite, app tests, e2e and seed

**Files (each: rule literals from `Steps: []approval.Step{{X}}` to `Flow: approval.Flow{{Step: &approval.StepDef{X}}}`, `kind:user` from `user_id: n` to `user_ids: [n]`, JSON `steps` to `flow`):**
- Modify: `internal/core/record/contract_test.go:98`
- Modify: `internal/modules/hrm/{timesheet,payroll,overtime,contract,leave}_test.go` (lines listed by `grep -rn "approval.RuleInput" internal/modules`)
- Modify: `internal/modules/sales/service_test.go:129` (the step with a condition becomes `{Split: &approval.Split{Mode: approval.ChooseOne, Branches: []approval.Branch{{Condition: &approval.Expr{Field: "max_discount", Op: "gt", Value: "10"}, Body: approval.Flow{{Step: …}}}}}}`)
- Modify: `internal/app/{discussion,leave,jobs,notification,payroll}_test.go` (JSON bodies `{"flow":[{"step":{"approver":{"kind":"module"}}}],…}`)
- Modify: `web/e2e/{m5,m6,m12,review-probe}.spec.ts`, `web/e2e/sales.seed.ts:296` (and `demo.seed.ts` if it calls `rule(…)` with steps)

**Interfaces:**
- Consumes: Task 2's `Flow`, `Element`, `StepDef`, `Split`, `Branch`, `Expr`; Task 3's unchanged `Approve(ctx, id, 1)` / `Reject(ctx, id, 1, reason)` (step 1 is the first step in reading order, so these calls stay).

- [ ] **Step 1: Update the literals** listed above.
- [ ] **Step 2: Run the Go suites**

Run: `go test ./internal/...`
Expected: PASS (recordtest runs inside the product suites).

- [ ] **Step 3: Commit**

```bash
git add internal/core/record/contract_test.go internal/modules/hrm internal/modules/sales/service_test.go internal/app web/e2e
git commit -m "test: configure approval as flows"
```

---

### Task 5: Frontend: approval panel and the chain editor on flows

**Files:**
- Regenerate: `api/openapi.json`, `web/src/shared/api/schema.gen.ts` (`make gen`)
- Modify: `web/src/shared/api/core.ts` (exported type aliases: drop `RuleCondition`, add `Flow`, `FlowElement`, `ApprovalStepView`, `StepVoter` if the file aliases schema types)
- Modify: `web/src/shared/document/ApprovalPanel.tsx`
- Modify: `web/src/shared/document/ApprovalRules.tsx` (lines ~80–110: `fromRule`, `toInput`)
- Modify: i18n files holding `shared.approval.*` (both `vi` and `en`)
- Test: `web/src/shared/document/ruleFlow.test.ts` (new), `web/src/shared/document/ruleFlow.ts` (new)

**Interfaces:**
- Consumes: Task 3's API types (`Instance.steps[].status`, `.voters`, `.allowed_actions`, `.approvals`; reassign body `{step, from, login}`; `RuleInput.flow`).
- Produces: `ruleFlow.ts`: `toFlow(steps: StepValues[]): Flow` and `fromFlow(flow: Flow): StepValues[] | null` (null when the flow isn't a chain of steps and single-branch choose-one splits with empty otherwise — part 2's editor replaces this; until then every saved flow is such a chain).

- [ ] **Step 1: Write the failing test** `ruleFlow.test.ts`: `toFlow` of `[{kind:'user', user:'5', field:'days', op:'gt', value:'3'}, {kind:'module', …no condition}]` equals `[{split:{mode:'one', branches:[{condition:{field:'days',op:'gt',value:'3'}, body:[{step:{approver:{kind:'user',user_ids:[5]}, approvals:1, rejections:1}}]}], otherwise:[]}}, {step:{approver:{kind:'module'}, approvals:1, rejections:1}}]`; `fromFlow(toFlow(x))` deep-equals `x`; `fromFlow` of a parallel split returns `null`.
- [ ] **Step 2: Run to see it fail**

Run: `cd web && pnpm vitest run src/shared/document/ruleFlow.test.ts`
Expected: FAIL (module not found).

- [ ] **Step 3: Implement** `ruleFlow.ts`; in `ApprovalRules.tsx` replace the step mapping with `fromFlow`/`toFlow` (a `null` from `fromFlow` shows an `Alert` with `shared.rule.not_a_chain` and no form). In `ApprovalPanel.tsx`: the open steps are `steps.filter(s => s.status === 'open')`; `ApprovalDecision` and `useDecide` act on the open step whose `allowed_actions` includes `approve` (send its `position` as `step`); the timeline shows each step with its status colour (`approved` success, `rejected` danger, `open` warning, `waiting`/`cancelled` grey), a "`n` of `m` approvals" line when `approvals > 1` (`shared.approval.count`), and each voter with their vote; reassign picks the voter to replace (`from`) then the login. Follow the `mmerp-ui` skill and finish with the `ui.md` checklist.
- [ ] **Step 4: Run checks**

Run: `make gen && make lint && cd web && pnpm vitest run`
Expected: no diff after gen in a second run, lint clean, tests PASS.

- [ ] **Step 5: Run e2e**

Run: `make e2e`
Expected: PASS (the approval specs m3, m4, m5, m6, m8, m12, m14 cover sending, approving and rejecting through the UI).

- [ ] **Step 6: Commit**

```bash
git add api/openapi.json web/src/shared/api web/src/shared/document web/src/shared/i18n web/src/core/i18n
git commit -m "feat(web): approval panel shows step counts and parallel steps"
```

(Stage only the i18n files actually touched.)
