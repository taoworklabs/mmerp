# Approval Rule Diagram Editor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the nested-card approval rule editor with a diagram on the left and a side panel on the right, with no change to the rule model, API or backend.

**Architecture:** The form stays one React Hook Form over `RuleValues` (`flowForm.ts`). A pure whole-flow validator (`flowProblems`) is the form's `resolver`, so validation no longer depends on which panel is mounted. Its errors land on element-level names (`flow.0.issue`, `flow.1.branches.0.issue`, `flow.root`, `settings`), and the diagram box carrying each name has `id={fieldId(name)}`. The shared `Form`/`FormActions` "Còn N … · Đến … tiếp theo" stepper then focuses diagram boxes, and focusing a box selects it. The diagram is HTML/CSS, recursive over the watched form value. The side panel shows the form of the selected element.

**Tech Stack:** React 19, TypeScript, Mantine, React Hook Form, TanStack Query, react-i18next, Vitest (pure functions only; the repo has no component tests), Playwright e2e.

**Spec:** `docs/superpowers/specs/2026-10-20-approval-rule-diagram-editor-design.md`. The throwaway spike `web/src/app/devui/spike/FlowSpike.tsx` shows the approved look and is a source for CSS and markup. Don't copy its hardcoded text or its own validation.

## Global Constraints

- No backend, API or `schema.gen.ts` change; wire format via `fromFlowValues`/`toFlowValues` unchanged.
- All text through i18n in `web/src/shared/i18n/locales/vi.json` and `en.json`; no literal strings in components (`make lint` runs `i18next/no-literal-string` and the i18n key check).
- Colours only from theme tokens / CSS variables (`make lint` colour-literal scan). Build from `shared/ui` primitives and Mantine.
- The editor stays in `web/src/shared/document/` (shared by HRM and Sales areas).
- Server limits kept in the UI: counts 1–20, due 1–720 hours, ≤ 20 users per step, split depth ≤ 5 (the ⊕ menu hides split items at depth 5), condition group depth ≤ 3, `max_levels` 1–10.
- Choose-one semantics text must match the engine: the *first* branch whose condition holds runs (`internal/core/approval/flow.go`).
- `allowed_actions` lacking `save` ⇒ read-only: no ⊕, panel fields read-only, no footer.
- Code, identifiers, test names in English; comments short; Conventional Commits with scope `web` (`docs` for doc-only commits). Stage explicit paths, never `git add -A`.
- Finish UI work with `ui.md`'s checklist and the `mmerp-ui` skill's `references/checklist.md`.

## Review Focus

1. **Searched condition values** (Sales "Mặt hàng có …"): the chip must read `code · name` from the saved id before any search, not the raw id. Test: Task 3, `approval-lines.spec.ts`.
2. **A server `invalid_rule` path pointing inside a split** (`1.branches.1`, `2.branches.1.body.1`) must mark and select that exact element, and the edit after it clears the mark. Test: Task 3, `approval-flows.spec.ts` "a refused save marks the element".
3. **Moving or deleting the selected element:** the panel must never keep editing a stale path. After *Chuyển xuống* the panel still edits the same step, and after deleting, the panel goes back to the empty hint. Test: Task 3, "moving a step keeps it selected".
4. **The unsaved-changes guard after diagram-only edits:** inserting through ⊕ must make the form dirty (footer appears, leaving asks). Test: Task 3, "inserting a step makes the form unsaved".
5. **Read-only rules still read in full:** boxes can be selected and the panel shows values read-only, with no ⊕ and no footer. Test: Task 3, "a disabled product shows the flow read-only".

---

### Task 1: Rule values and whole-flow problems

**Files:**
- Modify: `web/src/shared/document/flowForm.ts` (add `RuleValues`, `toRuleValues`, `toRuleInput`, `selectionKind`)
- Create: `web/src/shared/document/flowProblems.ts`
- Test: `web/src/shared/document/flowForm.test.ts`, `web/src/shared/document/flowProblems.test.ts`
- Modify: `web/src/shared/document/ApprovalRules.tsx` (delete its local `Values`/`toValues`/`toInput`/`roleValue`, import the new ones; behaviour otherwise unchanged in this task)

**Interfaces:**
- Produces (flowForm.ts):
  - `type RuleValues = { flow: FlowValues; max_levels: string; fallback: string | null }`
  - `toRuleValues(r: TypeRule): RuleValues`. With no saved rule it returns `{ flow: [], max_levels: '3', fallback: null }`: **empty flow**, no seeded step.
  - `toRuleInput(v: RuleValues): RuleInput`, the body of today's `toInput`.
  - `type Selection = string | null`. It holds an element path (`flow.0`, `flow.1.branches.0.body.0`), a branch path (`flow.1.branches.0`), an otherwise path (`flow.1.otherwise`) or `'settings'`.
  - `selectionKind(sel: string, v: RuleValues): 'settings' | 'branch' | 'otherwise' | 'step' | 'split' | null`. Returns null when the path no longer exists.
  - `parentOf(path: string): [arrayPath: string, index: number]`
- Produces (flowProblems.ts):
  - `type ProblemCode = 'missing_role' | 'missing_user' | 'approvals_exceed_users' | 'count_range' | 'invalid_due' | 'invalid_condition' | 'flow_required' | 'fallback_required' | 'max_levels_range'`
  - `type Problem = { at: string; code: ProblemCode }`. `at` is an RHF error name. It is `${elementPath}.issue` for a step, `${branchPath}.issue` for a branch condition, `'flow.root'` for an empty top-level flow, and `'settings'` for the fallback role or max levels. One problem per `at`: the first found, checked in the order the codes are listed.
  - `flowProblems(v: RuleValues): Problem[]`. Results come in reading order: settings last, elements depth-first, a split's branches left to right, then its otherwise.
  - `problemsResolver(t: TFunction): Resolver<RuleValues>`. Returns `{ values, errors: {} }` when there are no problems. Otherwise it returns errors built with `set(errors, at, { type: 'problem', message: t(\`shared.rule.reason.${code}\`) })`.

- [ ] **Step 1: Write the failing tests**

`flowForm.test.ts` adds:
```ts
it('a type without a rule opens with an empty flow', () => {
  expect(toRuleValues({ ...typeRule, rule: undefined }).flow).toEqual([])
})
it('round-trips a rule through values', () => {
  expect(toRuleInput(toRuleValues(typeRule))).toEqual(typeRule.rule)
})
it('names what a selection points at', () => {
  const v = toRuleValues({ ...typeRule, rule: { ...typeRule.rule!, flow: nested } })
  expect(selectionKind('flow.0', v)).toBe('step')
  expect(selectionKind('flow.1', v)).toBe('split')
  expect(selectionKind('flow.1.branches.0', v)).toBe('branch')
  expect(selectionKind('flow.1.otherwise', v)).toBe('otherwise')
  expect(selectionKind('flow.1.branches.0.body.0.branches.1.body.0', v)).toBe('step')
  expect(selectionKind('settings', v)).toBe('settings')
  expect(selectionKind('flow.9', v)).toBeNull()
})
it('splits a path into its array and index', () => {
  expect(parentOf('flow.1.branches.0')).toEqual(['flow.1.branches', 0])
})
```
(`typeRule` is a `TypeRule` fixture in the test with a fallback `hrm.hr`, `max_levels: 3` and `nested` as its flow.)

`flowProblems.test.ts`:
```ts
const ok: RuleValues = { flow: [newStep(true)], max_levels: '3', fallback: 'hrm.hr' }
it('a valid rule has no problems', () => expect(flowProblems(ok)).toEqual([]))
it('an empty flow and no fallback', () =>
  expect(flowProblems({ flow: [], max_levels: '3', fallback: null })).toEqual([
    { at: 'flow.root', code: 'flow_required' },
    { at: 'settings', code: 'fallback_required' },
  ]))
it('max levels outside 1–10', () =>
  expect(flowProblems({ ...ok, max_levels: '11' })).toEqual([{ at: 'settings', code: 'max_levels_range' }]))
it('step problems, one per step', () => {
  const role = { ...newStep(false), role: null }
  const users = { ...newStep(false), kind: 'user', users: ['1'], approvals: '2' }
  const due = { ...newStep(true), dueHours: '721' }
  const count = { ...newStep(true), rejections: '0' }
  expect(flowProblems({ ...ok, flow: [role, users, due, count] })).toEqual([
    { at: 'flow.0.issue', code: 'missing_role' },
    { at: 'flow.1.issue', code: 'approvals_exceed_users' },
    { at: 'flow.2.issue', code: 'invalid_due' },
    { at: 'flow.3.issue', code: 'count_range' },
  ])
})
it('a user step with nobody chosen', () =>
  expect(flowProblems({ ...ok, flow: [{ ...newStep(false), kind: 'user', users: [] }] })).toEqual([{ at: 'flow.0.issue', code: 'missing_user' }]))
it('an unfinished condition, nested, marks its branch', () => {
  const split = newSplit('one', true)
  split.branches[0]!.cond = { mode: 'all', rows: [{ group: { mode: 'any', rows: [{ group: null, field: 'days', op: 'gt', value: '' }] }, field: null, op: null, value: '' }] }
  split.otherwise = [{ ...newStep(false), role: null }]
  expect(flowProblems({ ...ok, flow: [split] })).toEqual([
    { at: 'flow.0.branches.0.issue', code: 'invalid_condition' },
    { at: 'flow.0.otherwise.0.issue', code: 'missing_role' },
  ])
})
it('the resolver puts translated messages at each name', async () => {
  const r = await problemsResolver(((k: string) => k) as TFunction)({ flow: [], max_levels: '3', fallback: 'hrm.hr' }, undefined, {} as never)
  expect(r.errors).toEqual({ flow: { root: { type: 'problem', message: 'shared.rule.reason.flow_required' } } })
})
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `pnpm -C web exec vitest run src/shared/document/flowForm.test.ts src/shared/document/flowProblems.test.ts`
Expected: FAIL (missing exports).

- [ ] **Step 3: Implement the interfaces above**

A condition leaf is unfinished when `field`, `op` or `value` is empty. A branch with no rows is fine ("always"). Counts are checked as integers in 1–20. Due is either empty or an integer in 1–720. Use `set` from `react-hook-form`'s utilities or a small local path setter for the resolver. Move `roleValue`'s inverse (`splitRole`) usage into `toRuleInput`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `pnpm -C web exec vitest run src/shared/document` then `pnpm -C web exec tsc --noEmit -p .`
Expected: PASS, no type errors.

- [ ] **Step 5: Commit**

```bash
git add web/src/shared/document/flowForm.ts web/src/shared/document/flowForm.test.ts web/src/shared/document/flowProblems.ts web/src/shared/document/flowProblems.test.ts web/src/shared/document/ApprovalRules.tsx
git commit -m "feat(web): check a whole approval flow at once"
```

---

### Task 2: Reading the flow back in words

**Files:**
- Create: `web/src/shared/document/flowText.ts`
- Test: `web/src/shared/document/flowText.test.ts`

**Interfaces:**
- Consumes: `ElementValues`, `CondValues` (flowForm.ts); `TypeRule['fields']` (`Field`), `Option` (`@/shared/ui/form`).
- Produces:
  - `approverText(t: TFunction, el: ElementValues, roles: Option[], users: Option[]): string` returns:
    - `module` → `t('shared.rule.box.module')` ("Quản lý trực tiếp")
    - `role` → the role option's label, or `t('shared.rule.box.pick_role')` ("Chọn vai trò…")
    - `user` → up to two names (the option label without the trailing ` (login)`), then ` +n`, or `t('shared.rule.box.pick_user')` ("Chọn người duyệt…")
  - `stepDetail(t, el): string` → parts joined by ` · `:
    - `t('shared.rule.box.approvals', { approvals, of })` ("2/3 người đồng ý") only when kind is `user` with more than one user
    - `t('shared.rule.box.approvals_any', { approvals })` ("Cần 2 người đồng ý") for role/module when approvals > 1
    - `t('shared.rule.box.due', { hours })` ("hạn 24 giờ") when a due time is set
    - empty string otherwise
  - `type ValueLabel = (field: Field, value: string) => string | undefined`
  - `condText(t, cond: CondValues, fields: Field[], valueLabel: ValueLabel): string` reads each leaf as `${fieldLabel} ${op} ${value}`, where:
    - `fieldLabel` is `t(field.label, { defaultValue: field.key })`
    - `op` is `t('shared.rule.op_word.<op>')`: gt `>`, gte `≥`, eq `là`, within `thuộc`, has `có`
    - `value` is `field.options` label for `choice` fields, else `valueLabel(...)`, else the raw value
    - an unfinished leaf is `…`, and a group is `(…)`
    - rows are joined with `t('shared.rule.and')` / `t('shared.rule.or')` (" và " / " hoặc "); no rows gives `''`

- [ ] **Step 1: Write the failing tests** (t stub returns the key plus params, e.g. `(k, p) => p ? \`${k}:${JSON.stringify(p)}\` : k`; one test with the real vi strings via `i18n` init is optional)

```ts
it('names the approver', () => {
  expect(approverText(t, { ...newStep(false), kind: 'role', role: 'hrm.hr' }, [{ value: 'hrm.hr', label: 'Nhân sự' }], [])).toBe('Nhân sự')
  expect(approverText(t, { ...newStep(false), kind: 'user', users: ['1', '2', '3'] }, [], users)).toBe('An, Bình +1')
  expect(approverText(t, { ...newStep(false), kind: 'user', users: [] }, [], users)).toBe('shared.rule.box.pick_user')
})
it('details only what differs from the defaults', () => {
  expect(stepDetail(t, newStep(true))).toBe('')
  expect(stepDetail(t, { ...newStep(true), dueHours: '24' })).toBe('shared.rule.box.due:{"hours":"24"}')
})
it('reads a nested condition', () => {
  const cond: CondValues = { mode: 'any', rows: [
    { group: null, field: 'days', op: 'gt', value: '3' },
    { group: { mode: 'all', rows: [{ group: null, field: 'kind', op: 'eq', value: 'unpaid' }, { group: null, field: 'org_unit', op: 'within', value: '' }] }, field: null, op: null, value: '' },
  ] }
  expect(condText(t, cond, fields, () => undefined)).toBe('days shared.rule.op_word.gt 3shared.rule.or(kind shared.rule.op_word.eq Không lươngshared.rule.and…)')
})
it('uses the value label for searched and org unit fields', () =>
  expect(condText(t, { mode: 'all', rows: [{ group: null, field: 'org_unit', op: 'within', value: '7' }] }, fields, () => 'Kế toán')).toBe('org_unit shared.rule.op_word.within Kế toán'))
```

- [ ] **Step 2: Run to verify it fails**: `pnpm -C web exec vitest run src/shared/document/flowText.test.ts`. Expected: FAIL.
- [ ] **Step 3: Implement** in `flowText.ts`. Add the `shared.rule.box.*`, `shared.rule.op_word.*`, `shared.rule.and`, `shared.rule.or` keys to `vi.json` and `en.json`.
- [ ] **Step 4: Run to verify it passes**, plus `pnpm -C web exec tsc --noEmit -p .`
- [ ] **Step 5: Commit**: `git add` the two files and both locale files; message `feat(web): read approval flow elements back in words`.

---

### Task 3: The diagram editor

**Files:**
- Create: `web/src/shared/document/flowEditor/FlowDiagram.tsx`, `FlowDiagram.module.css`. These hold `Lane`, `Edge` (⊕ menu), `StepBox`, `SplitBox`, `BranchChip`, start/end boxes and the empty-state line. The CSS starts from the spike's `FlowSpike.module.css`, with the nested join selectors scoped `.branch:first-child > .join`.
- Create: `web/src/shared/document/flowEditor/ElementPanel.tsx`. It holds `StepPanel`, `SplitPanel`, `BranchPanel`, `OtherwisePanel`, `SettingsPanel` and `EmptyPanel`.
- Create: `web/src/shared/document/flowEditor/editor.tsx`. It holds `FlowEditorContext`, `useFlowEditor()`, the `FlowEditor` provider (selection state, `rev`, edits), `EditorCtx` (moved from `FlowBlock.tsx`) and `IssueAlert` (moved). `IssueAlert` **loses its `id`**, because the diagram box now owns `fieldId(`${path}.issue`)` and ids must stay unique.
- Modify: `web/src/shared/document/flowEditor/ConditionEditor.tsx`. Import `EditorCtx` from `./editor`. `Leaf` stacks: field on its own row, comparison and value on the next (two columns from `xs`). Drop `rules`/`required` validation props that the resolver now covers; keep `required` for the asterisk only if the field components need it for display.
- Delete: `flowEditor/FlowBlock.tsx`, `flowEditor/StepCard.tsx`, `flowEditor/SplitCard.tsx`.
- Modify: `web/src/shared/document/ApprovalRules.tsx`, `RuleForm`. Details:
  - `useForm<RuleValues>({ mode: 'onBlur', defaultValues: toRuleValues(rule), resolver: problemsResolver(t) })`.
  - `Page fullWidth`. Header actions: *Cài đặt chung* (default button, `IconSettings`, `id={fieldId('settings')}`, selects `'settings'` on click and on focus) and *Xóa quy tắc* as today.
  - Body: `FlowEditor` with the diagram frame and the panel.
  - Footer: `FormActions submitLabel={t('shared.rule.save')}` when not read-only.
  - Server errors: an `invalid_rule` with a `path` → `form.setError(\`${formPathOf(path)}.issue\`)` and `select(formPathOf(path))`. Without a path → `form.setError('settings')` and `select('settings')`.
  - After a failed submit (`submitCount > 0`), every value change calls `form.trigger()` with no names, so the count and markers follow the edits. This replaces the `issue` ref hack.
- Modify: `web/src/shared/i18n/locales/vi.json`, `en.json` (texts below).
- Modify: `web/e2e/approval-flows.spec.ts`, `approval-lines.spec.ts`, `approval-overdue.spec.ts`, `approval-actions.spec.ts`, `m4.spec.ts`, `m8.spec.ts`.

**Interfaces:**
- Consumes: Task 1 (`RuleValues`, `toRuleValues`, `toRuleInput`, `Selection`, `selectionKind`, `parentOf`, `problemsResolver`, `flowProblems`) and Task 2 (`approverText`, `stepDetail`, `condText`, `ValueLabel`).
- Produces:
  - `type EditorCtx = { rule: TypeRule; roleOptions: Option[]; userOptions: Option[]; readOnly: boolean }` (unchanged shape)
  - `useFlowEditor(): { ctx: EditorCtx; selected: Selection; select(s: Selection): void; rev: number; insert(arrayPath: string, at: number, el: ElementValues | BranchValues): void; remove(path: string): void; move(path: string, by: -1 | 1): void }`
  - The edits use `getValues`/`setValue(path, list, { shouldDirty: true })` on the whole array, then bump `rev`. `insert` selects the new item. `remove` clears the selection. `move` keeps the moved item selected.
  - The panel is keyed by `` `${selected}:${rev}` `` so field arrays inside (`ConditionEditor`) remount on fresh paths.

**Behaviour the code must have** (spec goals 1–7 and "Calls I made"):
- Every box and chip is a `button`:
  - id: `fieldId(\`${path}.issue\`)`
  - `aria-pressed={selected === path}`
  - accessible name = its visible text, prefixed by kind (`t('shared.rule.box.step_label', { text })` → "Bước duyệt: …"; branch chips "Trường hợp 1: …" / "Nhánh 1: …"; split pill as its kind name)
  - it selects its element on click **and on focus** (so the shared stepper's focus selects it).
- The top-level empty ⊕ has `id={fieldId('flow.root')}`.
- A box is red (`data-error`) when the live `flowProblems(watch())` has its `at`, or RHF has an error at that name (server). Its detail line then shows the message.
- ⊕ is a `button` named `t('shared.rule.add_here')` ("Thêm vào đây"). Its menu items are *Bước duyệt* / *Tùy trường hợp* / *Cùng lúc*, each with a hint. At depth 5 only *Bước duyệt*. In read-only, no ⊕.
- Empty branch body: dashed "Không cần duyệt thêm" box. Under a parallel split: "Đi tiếp khi mọi nhánh đã xong". Empty top flow: end box "✓ Có hiệu lực ngay" and the hint line.
- Start box *Gửi duyệt*, end box *✓ Có hiệu lực*.
- Branch chip value labels:
  - org units via `useOrgUnits` (name by id)
  - searchable fields: fetch saved ids with `api.GET('/approval-rules/{doc_type}/fields/{key}/options', { query: { ids } })` under `documentKeys.approvalFieldOptions(docType, key, { ids })`, labelled as `RemoteSelectField` shows them (`code · name`)
  - `…` while loading
- The diagram frame:
  - scrolls in both directions
  - its height is the window minus header and footer (`max-height: calc(100dvh - …)`)
  - it has the sunken dotted background
  - clicking its empty background clears the selection.
- The panel is an `<aside aria-label={panel title}>` (e2e finds it by role `complementary`). From `md` up it sits beside the frame, sticky, `width: var(--mmerp-aside-width)`.
- Panel below `md` (`useMediaQuery` from `@mantine/hooks` with the theme's `md`): a Mantine `Drawer` (`position="bottom"`, `size="85%"`) open while `selected !== null`, titled with the panel title. Closing it clears the selection.
- `StepPanel` (Panel title "Bước duyệt"), in order:
  - `IssueAlert` for `${path}.issue`
  - *Ai duyệt* select (kinds, `module` only if `rule.module_approvers`); changing it clears role and users
  - *Vai trò* select or *Người duyệt* multiselect (max 20)
  - *Cần bao nhiêu người đồng ý*: shown for role/module always, and for user kind only with more than one user
  - *Hạn duyệt (giờ)* with its hint
  - *Nâng cao* toggle (open at mount when `rejections !== '1'`) holding *Bao nhiêu người từ chối thì trả đơn* with its hint
  - then *Chuyển lên* / *Chuyển xuống* (hidden at the ends) and *Xóa bước* (subtle danger, no confirm).
- `SplitPanel`:
  - a segmented control *Tùy trường hợp* / *Cùng lúc*. Switching to parallel with otherwise steps asks first (`shared.rule.to_parallel` / `drop_otherwise`, as `SplitCard.changeType` does today, including the restore on cancel); switching drops the otherwise list
  - the kind's hint text
  - *Thêm trường hợp* / *Thêm nhánh* (light, `IconPlus`), which appends `{ cond: { mode: 'all', rows: [] }, body: [newStep(module_approvers)] }` and selects it
  - *Xóa* (subtle danger).
- `BranchPanel`:
  - title *Trường hợp {n}* (choose-one) or *Nhánh {n}* (parallel)
  - the hint, then `IssueAlert` and `ConditionEditor path={\`${path}.cond\`} depth={0}`
  - *Sang trái* / *Sang phải* (disabled at the ends)
  - *Xóa trường hợp* / *Xóa nhánh*, only when more than one branch.
- `OtherwisePanel`: title *Còn lại* and its hint.
- `SettingsPanel` (title *Cài đặt chung*):
  - `IssueAlert` for `settings`
  - fallback `SelectField` "Không tìm được người duyệt thì chuyển cho" with its hint
  - `DecimalField` max levels "Khi quản lý trực tiếp lại chính là người gửi, đi lên tối đa (cấp)" (`scale={0} min={1} max={10}`).
- `EmptyPanel`: the how-to line.
- In read-only every panel field gets `readOnly` and the move/delete/add buttons are hidden.

**UI text** (vi; write matching en). Change the values of existing keys, add new keys under `shared.rule.*`, and delete keys nothing uses any more (the i18n key check fails on orphans):
- `split.one` "Tùy trường hợp", `split.all` "Cùng lúc"; hints "Chỉ đi một nhánh, theo điều kiện", "Nhiều nhánh duyệt song song", step "Một người hoặc nhóm người duyệt"
- `approver` "Ai duyệt"
- `kind.module` "Quản lý trực tiếp (theo chứng từ)", `kind.role` "Người có vai trò, tại đơn vị của chứng từ", `kind.user` "Những người cụ thể"
- `approvals` "Cần bao nhiêu người đồng ý"
- `rejections` "Bao nhiêu người từ chối thì trả đơn" + hint "Thường là 1: một người từ chối là đủ."
- `due_hours` "Hạn duyệt (giờ)" + hint "Để trống nếu không có hạn. Quá hạn thì nhắc người duyệt và người gửi."
- `advanced` "Nâng cao"
- `settings` "Cài đặt chung", `fallback` "Không tìm được người duyệt thì chuyển cho" (keep `fallback_hint`), `max_levels` "Khi quản lý trực tiếp lại chính là người gửi, đi lên tối đa (cấp)"
- branch titles "Trường hợp {{n}}" / "Nhánh {{n}}"; chips "Nếu {{text}}", "Khi {{text}}", "Luôn đúng", "Luôn chạy", "Còn lại"
- split hints: "Xét các trường hợp từ trái sang phải; đi theo trường hợp đầu tiên đúng. Không trường hợp nào đúng thì đi nhánh \"Còn lại\"." and "Mọi nhánh có điều kiện đúng cùng chạy một lúc; đi tiếp khi tất cả đã duyệt xong."
- `otherwise_hint` "Đi nhánh này khi không trường hợp nào đúng. Để trống thì không cần duyệt thêm."
- `parallel_join` "Đi tiếp khi mọi nhánh đã xong", `skip` "Không cần duyệt thêm"
- `start` "Gửi duyệt", `end` "✓ Có hiệu lực", `end_now` "✓ Có hiệu lực ngay", `empty_hint` "Chưa có ai duyệt. Bấm dấu + để thêm người duyệt đầu tiên."
- `panel_hint` "Bấm vào một ô trên sơ đồ để sửa. Bấm dấu + trên đường nối để thêm bước hoặc tách nhánh ngay tại đó."
- `add_here` "Thêm vào đây", `move_left` "Sang trái", `move_right` "Sang phải", `remove` "Xóa"
- reasons: `reason.missing_role` "Chưa chọn vai trò.", `reason.count_range` "Số người phải từ 1 đến 20.", `reason.flow_required` "Thêm ít nhất một bước.", `reason.fallback_required` "Chưa chọn người nhận khi không tìm được người duyệt.", `reason.max_levels_range` "Số cấp phải từ 1 đến 10."

- [ ] **Step 1: Rewrite the UI-driven e2e specs to the diagram (failing)**

Shared helpers in each spec that needs them:
```ts
const box = (page: Page, name: RegExp | string) => page.getByRole('main').getByRole('button', { name })
async function insertAt(page: Page, nth: number, element: 'Bước duyệt' | 'Tùy trường hợp' | 'Cùng lúc') {
  await page.getByRole('button', { name: 'Thêm vào đây' }).nth(nth).click()
  await page.getByRole('menuitem', { name: new RegExp(`^${element}`) }).click()
}
const panel = (page: Page) => page.getByRole('complementary') // the side panel is an <aside>
```
`approval-flows.spec.ts`:
- "an administrator builds a flow…": starts from no rule, so the flow is empty.
  - Build the same `built()` flow using `insertAt` and the panel: choose-one; branch 1 condition days > 3; inside it parallel with two user branches A and B; otherwise holds one manager step. Set the fallback *Nhân sự* through *Cài đặt chung*.
  - Save, assert the saved flow equals `built()`.
  - Reload and assert the chips/boxes read `Nếu Số ngày > 3`, `${login.a}`, `${login.b}`, `Quản lý trực tiếp`.
  - Routing part unchanged.
- "an unfinished condition is refused before saving": the branch chip is red after Save, no PUT is sent, and `page.getByRole('alert')` shows "Còn 1 trường cần sửa". Clicking "Đến trường tiếp theo" selects the chip (`aria-pressed=true`).
- "removing a nested group down to no rows drops the group itself": same, inside the branch panel.
- "a refused save marks the element": the route returns `path: '1.otherwise.1'` (the otherwise step of `built()`), `reason: 'unknown_role'`. The otherwise step box gets `aria-pressed=true` and its panel's alert says "Vai trò này không còn tồn tại. Chọn vai trò khác.". After the next edit the box is no longer red, and *Bỏ thay đổi* stays visible.
- "moving a step keeps it selected": add a second top-level step, select it, *Chuyển lên*. The panel still shows that step (its *Ai duyệt* value) and the box at the new position is pressed. *Xóa bước* brings back the panel hint.
- "inserting a step makes the form unsaved": ⊕ → *Bước duyệt* shows "Có thay đổi chưa lưu"; navigating to `/hrm/overview` opens the shared leave confirmation (`shared.form.leave_title`).
- "a disabled product shows the flow read-only": `Cùng lúc` visible, no `Thêm vào đây`, no `Lưu`. Selecting a step box shows its *Ai duyệt* read-only.

Other specs:
- `approval-lines.spec.ts`: start empty; insert *Tùy trường hợp* via ⊕; the branch condition *Mặt hàng có* special item; save. After reload, assert the chip reads `Nếu Mặt hàng có ${special.code} · ${special.name}` (Review Focus 1). Then select the chip and assert the value combobox shows the same.
- `approval-overdue.spec.ts`: select the step box, then use the `/^Hạn duyệt \(giờ\)/` textbox.
- `approval-actions.spec.ts`: the cancel rule starts empty. ⊕ → *Bước duyệt*, *Ai duyệt* → *Người có vai trò, tại đơn vị của chứng từ*, *Vai trò* → *Quản lý bán hàng*; *Cài đặt chung* → fallback *Quản lý bán hàng*.
- `m4.spec.ts`: assert a chip containing `Ngày lễ` is visible.
- `m8.spec.ts`: *Cài đặt chung*, then the max levels textbox.

- [ ] **Step 2: Run them to verify they fail**

Run: `make e2e` (or `pnpm -C web exec playwright test approval-flows approval-lines approval-overdue approval-actions m4 m8` with the e2e servers up via `scripts/e2e-server.sh`)
Expected: FAIL on the new selectors.

- [ ] **Step 3: Build `editor.tsx`, `FlowDiagram.tsx` + CSS, `ElementPanel.tsx`; reshape `ConditionEditor`'s `Leaf`; rewire `RuleForm`; delete the three card files; update both locale files**

Follow the behaviour list above.

- [ ] **Step 4: Run checks**

Run: `make lint`, `pnpm -C web test`, then the e2e specs from Step 2.
Expected: all PASS. Then look at it in the running app (`make dev`, `/hrm/approval-rules/hrm.leave_request`) at 1440px and 375px.

- [ ] **Step 5: Commit**

```bash
git add web/src/shared/document/flowEditor web/src/shared/document/ApprovalRules.tsx web/src/shared/i18n/locales/vi.json web/src/shared/i18n/locales/en.json web/e2e/approval-flows.spec.ts web/e2e/approval-lines.spec.ts web/e2e/approval-overdue.spec.ts web/e2e/approval-actions.spec.ts web/e2e/m4.spec.ts web/e2e/m8.spec.ts
git commit -m "feat(web): edit approval rules on a diagram with a side panel"
```
(`git add` of the `flowEditor` directory also stages the deletions; check `git status` first that nothing else is in it.)

---

### Task 4: Docs, glossary and removing the spike

**Files:**
- Modify: `CONTEXT.md`. Change the `vi:` names of **Choose-one split** to "Tùy trường hợp" and **Parallel split** to "Cùng lúc"; the English terms stay.
- Modify: `ui.md`, line 153's width rule. Add "the approval rule editor (diagram and side panel)" to the screens that take the whole width.
- Modify: `.claude/skills/mmerp-ui/references/layout.md` (full-width row, line 45), `references/checklist.md` (line 32), `references/components.md` (line 17: "Add step (approval rule)" becomes the diagram's ⊕; light buttons stay for *Thêm trường hợp*/*Thêm nhánh*), and `references/forms.md` (line 12, if it describes the editor's shape).
- Modify: `frontend.md` line 244, only if the `<ApprovalRules>` description no longer holds. It should still hold, so check it.
- Delete: `web/src/app/devui/spike/` and revert its route and import in `web/src/app/devui/index.tsx`.

- [ ] **Step 1: Delete the spike and verify nothing references it**: `grep -rn "spike\|FlowSpike" web/src`. Expected: no output.
- [ ] **Step 2: Edit the docs above.**
- [ ] **Step 3: Run** `make lint` and `pnpm -C web test`. Expected: PASS.
- [ ] **Step 4: Go through `ui.md`'s checklist and `mmerp-ui`'s `references/checklist.md`** for the editor page. Fix anything it finds, in the files of Task 3, and record what you fixed in the commit body.
- [ ] **Step 5: Commit**

```bash
git add CONTEXT.md ui.md .claude/skills/mmerp-ui/references frontend.md web/src/app/devui/index.tsx web/src/app/devui/spike
git commit -m "docs(web): approval rule editor is a full-width diagram; drop the spike"
```
