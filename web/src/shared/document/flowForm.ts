import type { Flow, FlowCondition, FlowElement, RuleInput, TypeRule } from '@/shared/api/core'

// One row of a condition group: a nested group, or a leaf (field/op/value).
export type CondValues = { mode: 'all' | 'any'; rows: CondRow[] }
export type CondRow = { group: CondValues | null; field: string | null; op: string | null; value: string }

// One element of the flow editor: a step or a split, holding only the fields its type/kind use.
export type ElementValues = {
  type: 'step' | 'one' | 'all'
  kind: string | null
  role: string | null
  users: string[]
  approvals: string
  rejections: string
  dueHours: string // empty for no due time
  branches: BranchValues[]
  otherwise: ElementValues[]
}
export type BranchValues = { cond: CondValues; body: ElementValues[] }
export type FlowValues = ElementValues[]

type StepDef = NonNullable<FlowElement['step']>
type SplitDef = NonNullable<FlowElement['split']>
type BranchDef = SplitDef['branches'][number]

// The whole rule form: the flow, plus the fallback role ("product.role") and the level limit.
export type RuleValues = { flow: FlowValues; max_levels: string; fallback: string | null }

// splitRole parses a "product.role" select value.
export const splitRole = (v: string) => {
  const [product = '', ...rest] = v.split('.')
  return { product, role: rest.join('.') }
}

export function newStep(moduleApprovers: boolean): ElementValues {
  return { type: 'step', kind: moduleApprovers ? 'module' : 'role', role: null, users: [], approvals: '1', rejections: '1', dueHours: '', branches: [], otherwise: [] }
}

export function newSplit(type: 'one' | 'all', moduleApprovers: boolean): ElementValues {
  return {
    type,
    kind: null,
    role: null,
    users: [],
    approvals: '1',
    rejections: '1',
    dueHours: '',
    branches: [{ cond: { mode: 'all', rows: [] }, body: [newStep(moduleApprovers)] }],
    otherwise: [],
  }
}

// exprToRow reads one child of a condition tree: a nested group, or a leaf.
function exprToRow(e: FlowCondition): CondRow {
  if (e.all?.length || e.any?.length) return { group: exprToCond(e), field: null, op: null, value: '' }
  return { group: null, field: e.field ?? null, op: e.op ?? null, value: e.value ?? '' }
}

// exprToCond reads a condition tree into a group; a bare leaf becomes a one-row "all" group.
function exprToCond(e: FlowCondition): CondValues {
  if (e.all?.length) return { mode: 'all', rows: e.all.map(exprToRow) }
  if (e.any?.length) return { mode: 'any', rows: e.any.map(exprToRow) }
  return { mode: 'all', rows: [exprToRow(e)] }
}

// rowToExpr reads one row; undefined means the row holds no constraint (an empty nested group),
// so the caller must drop it rather than send the server an empty "all"/"any".
function rowToExpr(row: CondRow): FlowCondition | undefined {
  if (row.group) return condToExpr(row.group)
  return { field: row.field ?? undefined, op: (row.op as FlowCondition['op']) ?? undefined, value: row.value }
}

// condToExpr writes a group back: a single row (leaf or nested group) is written bare, several rows
// get the mode wrapper; a group left with no constraints (defence: the UI never leaves one nested) is
// undefined, for the caller to drop.
function condToExpr(c: CondValues): FlowCondition | undefined {
  const rows = c.rows.map(rowToExpr).filter((r): r is FlowCondition => r !== undefined)
  if (rows.length === 0) return undefined
  if (rows.length === 1) return rows[0]
  return { [c.mode]: rows }
}

function approverOf(v: ElementValues): StepDef['approver'] {
  if (v.kind === 'role') return { kind: 'role', ...splitRole(v.role ?? '') }
  if (v.kind === 'user') return { kind: 'user', user_ids: v.users.map(Number) }
  return { kind: 'module' }
}

function elementToValues(e: FlowElement): ElementValues {
  if (e.step) {
    const a = e.step.approver
    return {
      type: 'step',
      kind: a.kind,
      role: a.kind === 'role' ? `${a.product}.${a.role}` : null,
      users: a.kind === 'user' ? (a.user_ids ?? []).map(String) : [],
      approvals: String(e.step.approvals || 1),
      rejections: String(e.step.rejections || 1),
      dueHours: e.step.due_hours ? String(e.step.due_hours) : '',
      branches: [],
      otherwise: [],
    }
  }
  const sp = e.split as SplitDef
  return {
    type: sp.mode,
    kind: null,
    role: null,
    users: [],
    approvals: '1',
    rejections: '1',
    dueHours: '',
    branches: sp.branches.map(branchToValues),
    otherwise: (sp.otherwise ?? []).map(elementToValues),
  }
}

function branchToValues(b: BranchDef): BranchValues {
  return { cond: b.condition ? exprToCond(b.condition) : { mode: 'all', rows: [] }, body: b.body.map(elementToValues) }
}

function valuesToElement(v: ElementValues): FlowElement {
  if (v.type === 'step') {
    const step: StepDef = { approver: approverOf(v), approvals: Number(v.approvals), rejections: Number(v.rejections) }
    if (v.dueHours) step.due_hours = Number(v.dueHours)
    return { step }
  }
  const split: SplitDef = { mode: v.type, branches: v.branches.map(branchToElement) }
  if (v.type === 'one' && v.otherwise.length) split.otherwise = v.otherwise.map(valuesToElement)
  return { split }
}

function branchToElement(b: BranchValues): BranchDef {
  const body = b.body.map(valuesToElement)
  const cond = condToExpr(b.cond)
  return cond ? { condition: cond, body } : { body }
}

export function toFlowValues(flow: Flow): FlowValues {
  return flow.map(elementToValues)
}

export function fromFlowValues(v: FlowValues): Flow {
  return v.map(valuesToElement)
}

// formPathOf maps a backend flow error path (1-based: "2.branches.1.body.1") to the matching
// react-hook-form field path (0-based, under "flow"): "flow.1.branches.0.body.0". An empty
// path (the whole flow) maps to "flow".
export function formPathOf(apiPath: string): string {
  if (!apiPath) return 'flow'
  const parts = apiPath.split('.').map((p) => (/^\d+$/.test(p) ? String(Number(p) - 1) : p))
  return ['flow', ...parts].join('.')
}

// flowSummary counts every step leaf and reports whether the flow branches at all.
export function flowSummary(flow: Flow): { steps: number; branched: boolean } {
  let steps = 0
  let branched = false
  const walk = (f: Flow) => {
    for (const e of f) {
      if (e.step) steps++
      if (e.split) {
        branched = true
        for (const b of e.split.branches) walk(b.body)
        if (e.split.otherwise) walk(e.split.otherwise)
      }
    }
  }
  walk(flow)
  return { steps, branched }
}

// toRuleValues opens a type's rule in the form; a type without one starts from an empty flow.
export function toRuleValues(r: TypeRule): RuleValues {
  if (!r.rule) return { flow: [], max_levels: '3', fallback: null }
  return { flow: toFlowValues(r.rule.flow), max_levels: String(r.rule.max_levels), fallback: `${r.rule.fallback_product}.${r.rule.fallback_role}` }
}

export function toRuleInput(v: RuleValues): RuleInput {
  const fallback = splitRole(v.fallback ?? '')
  return { flow: fromFlowValues(v.flow), max_levels: Number(v.max_levels), fallback_product: fallback.product, fallback_role: fallback.role }
}

// A selection in the editor: an element's path ("flow.1.branches.0.body.0"), a branch's
// ("flow.1.branches.0"), an otherwise list's ("flow.1.otherwise"), or "settings".
export type Selection = string | null

// parentOf splits an item's path into its list's path and its index: "flow.2" → ["flow", 2].
export function parentOf(path: string): [string, number] {
  const at = path.lastIndexOf('.')
  return [path.slice(0, at), Number(path.slice(at + 1))]
}

// selectionKind names what a selection points at, or null once it no longer exists.
export function selectionKind(sel: string, v: RuleValues): 'settings' | 'branch' | 'otherwise' | 'step' | 'split' | null {
  if (sel === 'settings') return 'settings'
  let node: unknown = v
  for (const part of sel.split('.')) {
    if (node === null || typeof node !== 'object') return null
    node = (node as Record<string, unknown>)[part]
  }
  return kindAt(sel, node)
}

// kindAt names what a selection points at, given the value at its path (undefined once it is gone).
export function kindAt(sel: string, value: unknown): 'settings' | 'branch' | 'otherwise' | 'step' | 'split' | null {
  if (sel === 'settings') return 'settings'
  if (value === undefined || value === null) return null
  if (sel.endsWith('.otherwise')) return 'otherwise'
  if (/\.branches\.\d+$/.test(sel)) return 'branch'
  return (value as ElementValues).type === 'step' ? 'step' : 'split'
}
