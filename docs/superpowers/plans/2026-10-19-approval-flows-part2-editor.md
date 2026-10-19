# Approval flows, part 2: the flow editor — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Customers edit any approval flow the API accepts: steps with approver and counts, choose-one and parallel splits, nested branches and all/any conditions. The approval panel shows the plan's parallel structure and lets an actor pick which open step to decide.

**Architecture:** The rule page keeps React Hook Form. The flow lives in the form as a tree of plain values. Nested components take a string path and build field names from it. Pure functions in `flowForm.ts` convert between the API `Flow` and the form values, map an `invalid_rule` error path to a form path, and summarise a flow for the rule list. A new shared `MultiSelectField` picks several users. The backend adds one thing: the header `org_unit` field joins the fields a condition may test.

**Tech Stack:** React 19 + TypeScript, Mantine, React Hook Form, TanStack Query, openapi-typescript client, Vitest, Playwright; Go (one backend change).

**Spec:** `docs/superpowers/specs/2026-10-19-approval-flows-design.md` (goal 2 as configured by customers, goal 4 header conditions). Ticket #92. Part 1 (engine, API) is done on this branch; read `internal/core/approval/flow.go` and `types.go` for the API shape.

## Global Constraints

- Areas never import each other. Shared components live in `shared/ui` (generic) or `shared/document` (approval screens) and are built from `shared/ui` primitives. Page layout comes from a `shared/ui/page` template.
- Areas carry no styling: no `style`, no CSS files and no colour codes in an area. Tokens only: a pixel or colour value is written only in `theme.ts` or a `shared/ui` CSS module. `make lint` enforces this.
- Every user-facing string goes through i18n in both `vi` and `en` (`web/src/shared/i18n/locales`), and must follow `content.md` of the `mmerp-ui` skill.
- Buttons come from `allowed_actions`. The frontend never derives permissions.
- A new shared component covers every state and is shown on `/dev/ui` (`web/src/app/devui`).
- Finish every UI task with the `mmerp-ui` checklist (`.claude/skills/mmerp-ui/references/checklist.md`).
- English identifiers. Comments are short, say why, and never reference docs, ADRs or milestones.
- Conventional Commits, ending with `Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`. Stage explicit paths only, never `git add -A`.
- Branch `approval-flows`.

## Design (decided here)

```
Page (centered, title = document type, breadcrumbs → rules list, [Delete] in header when allowed)
├ FormSection wide "Quy trình duyệt"  (description: steps run top to bottom)
│  FlowBlock path="flow"
│  ┌ Panel "Bước duyệt" ─────────────────────────────── […] ┐   step card
│  │ Người duyệt [Vai trò ▾]  Vai trò [Quản lý ▾]            │
│  │ Số người cần duyệt [1]   Số người từ chối để loại [1]   │
│  └─────────────────────────────────────────────────────────┘
│  ┌ Panel "Rẽ nhánh chọn một" ──────────────────────── […] ┐   split card
│  │ ┌ Nhánh 1 ────────────────────────────────────── […] ┐ │
│  │ │ Điều kiện  (Tất cả ◉ / Một trong ○)                 │ │   ConditionEditor
│  │ │  [Số ngày ▾] [lớn hơn ▾] [3]                    [x] │ │
│  │ │  [+ Điều kiện] [+ Nhóm điều kiện]                   │ │
│  │ │ FlowBlock path=".branches.0.body"  (nested cards)   │ │
│  │ └─────────────────────────────────────────────────────┘ │
│  │ [+ Nhánh]                                               │
│  │ ┌ Còn lại ───────────────────────────────────────────┐ │   choose-one only
│  │ │ FlowBlock path=".otherwise"  (may be empty)         │ │
│  │ └─────────────────────────────────────────────────────┘ │
│  └─────────────────────────────────────────────────────────┘
│  [+ Thêm ▾]  → Bước duyệt · Rẽ nhánh chọn một · Rẽ nhánh song song
├ FormSection "Người duyệt dự phòng" (fallback role, max levels; unchanged)
└ FormActions (sticky)
```

- Cards use `Panel`, the shared bordered card; see `components.md` for its exact export. Nested cards indent by sitting inside their parent's body; no new CSS is needed beyond what `shared/ui` already has.
- Each card's "…" menu (`MoreActions`) offers: Move up, Move down, Delete. Moving stays within the same block.
- Approver fields depend on the kind:
  - `module`: no extra field.
  - `role`: a role select.
  - `user`: a `MultiSelectField` (1–20 users).
- The two count fields are `IntegerField`s with `min=1`, `max=20` and default `1`. They show for every kind. Their client-side checks:
  - kind `user`: approvals ≤ the number of chosen users;
  - every kind: rejections ≤ approvals + rejections, and both must be in 1–20.
- A condition is a group: an all/any `SegmentedControl` over a list of rows. A row is either a leaf or a nested group (recursive).
  - A leaf has three inputs: field select, op select (filtered by the field's `ops`), and value.
  - The value input depends on the field kind: `SelectField` for `choice` (with the field's `options`), `DecimalField` for `number`, `OrgUnitField` for `org_unit`.
  - A branch with no condition rows has no condition, so it always holds.
  - Labels per field: `shared.rule.cond_all` = "Tất cả điều kiện đúng" and `shared.rule.cond_any` = "Một trong các điều kiện đúng".
- A choose-one card shows its branches and then "Còn lại". A parallel card shows only its branches; each branch's condition is optional ("Luôn chạy" when empty).
- An empty block shows a short `dimmed` line, `shared.rule.empty_block` = "Không có bước nào: đi tiếp.", plus the Add menu. The top-level flow must keep at least one element: saving an empty top-level flow is a form error on the section.
- Server error `invalid_rule` with params `{path, reason}`: the page catches it in `save`, converts `path` to the form path of that element, and sets an error on `<elementPath>.issue`. The card shows it as an `Alert` at its top and scrolls into view. The text is `shared.rule.reason.<reason>`, one key per reason that `checkFlow` returns. Any other error shows the `Form` banner as today.
- The rule list's column shows `flowSummary`: `shared.rule.summary`, "{{count}} bước", plus ", có rẽ nhánh" when the flow has a split (`shared.rule.summary_branched`).
- Approval panel:
  - The timeline renders `plan`. A `par` group shows its children under a small "Song song" label (`shared.approval.parallel`), indented with the theme's spacing. A `seq` renders in order.
  - When the actor can vote on exactly one open step, `ApprovalDecision` (page header) works as today.
  - When the actor can vote on several, the header shows nothing. Each such step in the panel shows its own Approve and Reject (`size="xs"`), acting on that step's `position`.

## Review Focus

1. **Loading a deeply nested flow and saving it unchanged** sends back the same flow, compared as JSON. Test: `flowForm.test.ts` `round-trips nested flows`.
2. **Switching a step's approver kind from `user` to `role`** must not send the stale `user_ids` or role. Test: `flowForm.test.ts` `drops fields of other approver kinds`.
3. **Switching a split from choose-one to parallel** drops its otherwise on save. If the otherwise had elements, the user confirms first. Test: `flowForm.test.ts` `parallel drops otherwise`; the confirm is checked by hand in Task 4.
4. **A condition row with a field picked but no value**: the client refuses it as a field error ("Chưa nhập."), and it is never sent as an empty value. Test: covered by `required` on the value field; asserted in the e2e of Task 5.
5. **The disabled product** shows everything read-only: no Add, Move or Delete, and no FormActions. Test: `ApprovalRules` read-only state, e2e in Task 5 (`PRODUCTS` without the product on port 8091).

---

### Task 1: The header `org_unit` field is offered for conditions

**Files:**
- Modify: `internal/core/approval/service.go` (`Rules`: build `tr.Fields`)
- Test: `internal/core/approval/service_test.go`

**Interfaces:**
- Produces: every `TypeRule.Fields` starts with `{Key: "org_unit", Kind: "org_unit", Label: "shared.rule.field.org_unit", Ops: ["within"]}`, followed by the type's declared fields as today.

- [ ] **Step 1: Write the failing test** `TestRulesOfferTheOrgUnit`: `f.appr.Rules(f.admin, "test")` → `rules[0].Fields[0]` equals the field above, and `Fields[1:]` keys are `["days", "kind"]`.
- [ ] **Step 2: Run** `go test ./internal/core/approval -run TestRulesOfferTheOrgUnit` → FAIL.
- [ ] **Step 3: Implement**: prepend the field in `Rules`. Use the constants `FieldOrgUnit` and `OpWithin`.
- [ ] **Step 4: Run** `go test ./internal/core/approval/...` → PASS.
- [ ] **Step 5: Commit** `feat(approval): conditions may test the org unit` (stage `internal/core/approval/service.go internal/core/approval/service_test.go`).

---

### Task 2: Form values for flows (pure TypeScript)

**Files:**
- Create: `web/src/shared/document/flowForm.ts`, `web/src/shared/document/flowForm.test.ts`

This task is additive: `ruleFlow.ts` stays until Task 4 replaces its only user and deletes it. `flowForm.ts` carries its own `splitRole`.

**Interfaces:**
- Produces:

```ts
export type CondValues = { mode: 'all' | 'any'; rows: CondRow[] }
export type CondRow = { group: CondValues | null; field: string | null; op: string | null; value: string } // group set → nested group, else a leaf
export type ElementValues = {
  type: 'step' | 'one' | 'all'
  kind: string | null          // approver kind: module | role | user
  role: string | null          // "product.role"
  users: string[]              // user ids as strings
  approvals: string            // integer as string, default "1"
  rejections: string
  branches: BranchValues[]     // splits only
  otherwise: ElementValues[]   // choose-one only
}
export type BranchValues = { cond: CondValues; body: ElementValues[] }
export type FlowValues = ElementValues[]

export const splitRole: (v: string) => { product: string; role: string }
export function newStep(moduleApprovers: boolean): ElementValues   // kind module when the type has module approvers, else role
export function newSplit(type: 'one' | 'all', moduleApprovers: boolean): ElementValues // one branch holding one new step
export function toFlowValues(flow: Flow): FlowValues
export function fromFlowValues(v: FlowValues): Flow
export function formPathOf(apiPath: string): string               // "2.branches.1.body.1" → "flow.1.branches.0.body.0"; "2.otherwise.1" → "flow.1.otherwise.0"
export function flowSummary(flow: Flow): { steps: number; branched: boolean }
```

- `fromFlowValues` writes fields by type and kind:
  - only the fields of the element's `type`;
  - only the approver fields of its `kind`: `role` → `{kind, product, role}`, `user` → `{kind, user_ids: users.map(Number)}`, `module` → `{kind}`;
  - `approvals` and `rejections` as numbers.

  A branch whose `cond.rows` is empty gets no `condition`. A group with exactly one leaf row is written as that leaf. A group with several rows is written as `{all: [...]}` or `{any: [...]}`, and nested groups recurse. A parallel split gets no `otherwise`.
- `toFlowValues` is the inverse. A leaf condition becomes a group `{mode: 'all', rows: [leaf]}`. A missing `approvals` or `rejections` becomes `"1"`.

- [ ] **Step 1: Write the failing tests** (`flowForm.test.ts`):
  - `round-trips nested flows`: a flow `[step(user [3,4], approvals 2, rejections 1), split one: [branch {any:[{field:days,op:gt,value:'3'},{all:[{field:kind,op:eq,value:'a'},{field:org_unit,op:within,value:'7'}]}]} → [split all: [branch (no condition) → [step module], branch {field:kind,op:eq,value:'b'} → [step role hrm.hr]]]], otherwise [step module]]`. `fromFlowValues(toFlowValues(f))` deep-equals `f`.
  - `drops fields of other approver kinds`: an element with `kind: 'role'`, `role: 'hrm.hr'` and stale `users: ['5']` gives approver `{kind:'role', product:'hrm', role:'hr'}`, with no `user_ids`.
  - `parallel drops otherwise`: an element with `type 'all'` and a non-empty `otherwise` writes no `otherwise`.
  - `a branch without rows has no condition`; `one leaf is written bare`.
  - `maps api paths to form paths`: `formPathOf('1')` → `'flow.0'`, `formPathOf('2.branches.1.body.1')` → `'flow.1.branches.0.body.0'`, `formPathOf('3.otherwise.2')` → `'flow.2.otherwise.1'`, `formPathOf('')` → `'flow'`.
  - `summarises flows`: the round-trip flow → `{steps: 4, branched: true}`; `[step]` → `{steps: 1, branched: false}`.
- [ ] **Step 2: Run** `pnpm -C web test -- src/shared/document/flowForm.test.ts` → FAIL (module not found).
- [ ] **Step 3: Implement** `flowForm.ts`. Don't touch `ApprovalRules.tsx` or `ruleFlow.ts`.
- [ ] **Step 4: Run** `pnpm -C web test` → PASS.
- [ ] **Step 5: Commit** `feat(web): form values for approval flows` (stage the two new files).

---

### Task 3: `MultiSelectField`

**Files:**
- Modify: `web/src/shared/ui/form/fields.tsx` (add next to `SelectField`), `web/src/shared/ui/form/index.ts` (export)
- Modify: `web/src/app/devui/input.tsx` (sample in `Fields`)
- Modify: `.claude/skills/mmerp-ui/references/components.md` and `forms.md` ("Fields by data": several values from a list → `MultiSelectField`)

**Interfaces:**
- Produces: `export function MultiSelectField(props: Base & { data: Option[]; max?: number; searchable?: boolean }): JSX.Element`. It is wired by `name` to a `string[]` value. It follows `SelectField` for the label, `required` (at least one), `description`, `readOnly` (a grey fill showing the chosen labels, with `data-readonly` on the wrapper as the other fields do), `bare`, and the error under the field. `max` caps the selection (Mantine `maxValues`).

- [ ] **Step 1:** Read `SelectField`, `Base` and the read-only convention in `fields.tsx`. Implement `MultiSelectField` on Mantine `MultiSelect` with the same wrapper props.
- [ ] **Step 2:** Add a `/dev/ui` sample covering every state: empty, chosen, error, read-only and disabled.
- [ ] **Step 3: Run** `make lint` and `pnpm -C web test` → clean.
- [ ] **Step 4: Commit** `feat(web): MultiSelectField` (stage the files above).

---

### Task 4: The flow editor

**Files:**
- Create: `web/src/shared/document/flowEditor/FlowBlock.tsx` (a block: its elements and the Add menu), `StepCard.tsx`, `SplitCard.tsx`, `ConditionEditor.tsx`
- Modify: `web/src/shared/document/ApprovalRules.tsx` (use the editor; drop the chain-only alert; list summary; `invalid_rule` mapping)
- Delete: `web/src/shared/document/ruleFlow.ts`, `ruleFlow.test.ts`
- Modify: `web/src/shared/i18n/locales/vi.json`, `en.json` (`shared.rule.*`; remove the keys no longer used: `not_a_chain`, `step`, `add_step`, `remove_step`, `condition_hint` if unused)

**Interfaces:**
- Consumes: Task 2's `flowForm.ts`, Task 3's `MultiSelectField`, Task 1's `org_unit` field.
- Produces (props):
  - `FlowBlock({ path, ctx, depth })`
  - `StepCard({ path, ctx, onMove, onRemove, first, last })`
  - `SplitCard({ path, ctx, depth, onMove, onRemove, first, last })`
  - `ConditionEditor({ path, ctx, depth })`

  `path` is a form path string such as `"flow.1.branches.0.body"`. `ctx` is `{ rule: TypeRule; roleOptions: Option[]; userOptions: Option[]; readOnly: boolean }`. The components use `useFormContext()` and `useFieldArray({ name: path as never })`. The tree is recursive, so nested names are plain strings; only `RuleForm` knows `Values = { flow: FlowValues; max_levels: string; fallback: string | null }`.

- [ ] **Step 1: Write the editor** per the Design section. Notes:
  - Split cards stop offering nested splits in their Add menu at depth 5, because the server limit is 5 nested splits.
  - Changing a split's type from choose-one to parallel, when its `otherwise` holds elements, asks first through `useConfirm`, with `shared.rule.drop_otherwise`.
  - The value input resets when the field changes.
  - `save` catches `ApiError` with code `invalid_rule`. It calls `form.setError(\`${formPathOf(path)}.issue\`, { message: t(\`shared.rule.reason.${reason}\`) })` and scrolls the card into view. The card has id `fieldId(path)`; check `fieldId` in `Form.tsx`. Any other error is rethrown so the `Form` banner shows it.
- [ ] **Step 2: Rule list** shows `flowSummary`. Delete stays in the header, now for every flow.
- [ ] **Step 3: Run** `make lint` and `pnpm -C web test` → clean. Then run `make dev` (or `make ui-dev` if the rule page has a `/dev/ui` sample) and build by hand the Review Focus flow at 1280px and 375px: a top step, a choose-one with two branches and an otherwise, and a parallel split inside a branch. Save it, reload, and check that it's unchanged. Switch a split to parallel and get the confirm. Go through the `mmerp-ui` checklist.
- [ ] **Step 4: Commit** `feat(web): approval flow editor` (stage the new and changed files, and the deletions with `git rm`).

---

### Task 5: The approval panel shows the plan; e2e

**Files:**
- Modify: `web/src/shared/document/ApprovalPanel.tsx` (render `plan`; per-step decide when several votable steps)
- Modify: `web/src/app/devui/*` sample for the approval panel (the file that holds it: `grep -rn ApprovalPanel web/src/app/devui`) with a plan `{seq:[1,{par:[2,3]}]}`
- Create: `web/e2e/approval-flows.spec.ts`
- Modify: i18n `shared.approval.parallel`

**Interfaces:**
- Consumes: the instance `plan` (`{step?, seq?, par?}`), `steps[].allowed_actions`.

- [ ] **Step 1: Panel**: render the plan as described in the Design section. `useDecide` takes the step position from the caller. `ApprovalDecision` renders only when exactly one step allows `approve`.
- [ ] **Step 2: e2e** `approval-flows.spec.ts`, following `m8.spec.ts` / `m4.spec.ts` helpers (`as`, `signedIn`, `heading`, `status`):
  - `an administrator builds a flow with a choose-one and a parallel split, and it routes a leave request`:
    1. On `/hrm/approval-rules/hrm.leave_request`, build: a choose-one split with branch `days > 3`, whose body is a parallel split of [step: user A, step: user B], and an otherwise of [step: module].
    2. Save, and reload: the page shows the same structure.
    3. Through the API helpers, file a 5-day leave request as an employee and send it.
    4. User A and user B each see it in their inbox, and the panel shows "Song song". A approves; it is still pending. B approves; it is posted.
  - `a condition row without a value is refused before saving`: pick a field and leave the value empty → "Chưa nhập." under the value, and no request is sent.
  - `a disabled product shows the flow read-only`: use the server on port 8091 (no products enabled; same database), as `m9.spec.ts` does. The rule page for `hrm.leave_request` shows the flow with no Add button and no Save.
- [ ] **Step 3: Run** `make lint`, `pnpm -C web test`, then `make e2e` → all pass.
- [ ] **Step 4: Commit** `feat(web): approval panel shows parallel steps; e2e for flows` (stage the files above).
