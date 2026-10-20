import type { TFunction } from 'i18next'
import type { FieldErrors, Resolver } from 'react-hook-form'
import type { CondValues, ElementValues, FlowValues, RuleValues } from './flowForm'

export type ProblemCode =
  | 'missing_role'
  | 'missing_user'
  | 'approvals_exceed_users'
  | 'count_range'
  | 'invalid_due'
  | 'invalid_condition'
  | 'flow_required'
  | 'fallback_required'
  | 'max_levels_range'

// at is the form error name the problem is shown at: "<element>.issue" for a step, "<branch>.issue"
// for a branch's condition, "flow.root" for an empty flow, "settings" for the fallback and levels.
// field, when there is one, is the input that fixes it, where the panel shows it too.
export type Problem = { at: string; code: ProblemCode; field?: string }

// The server's limits.
const maxCount = 20
const maxDueHours = 720
const maxLevels = 10

// Number fields hold a number, or null once cleared, although the form types them as strings.
const blank = (v: unknown) => v === null || v === undefined || v === ''
const inRange = (v: unknown, min: number, max: number) => !blank(v) && /^\d+$/.test(String(v)) && Number(v) >= min && Number(v) <= max

function stepProblem(el: ElementValues): [ProblemCode, keyof ElementValues] | null {
  if (el.kind === 'role' && !el.role) return ['missing_role', 'role']
  if (el.kind === 'user' && el.users.length === 0) return ['missing_user', 'users']
  if (el.kind === 'user' && Number(el.approvals) > el.users.length) return ['approvals_exceed_users', 'approvals']
  if (!inRange(el.approvals, 1, maxCount)) return ['count_range', 'approvals']
  if (!inRange(el.rejections, 1, maxCount)) return ['count_range', 'rejections']
  if (!blank(el.dueHours) && !inRange(el.dueHours, 1, maxDueHours)) return ['invalid_due', 'dueHours']
  return null
}

// unfinished names the first input a condition still lacks (a leaf's field, comparison or value),
// or null when every leaf is whole; no rows means always.
function unfinished(c: CondValues, path: string): string | null {
  for (const [i, r] of c.rows.entries()) {
    const p = `${path}.rows.${i}`
    if (r.group) {
      const inner = unfinished(r.group, `${p}.group`)
      if (inner) return inner
    } else if (!r.field) return `${p}.field`
    else if (!r.op) return `${p}.op`
    else if (blank(r.value)) return `${p}.value`
  }
  return null
}

function walk(flow: FlowValues, path: string, out: Problem[]) {
  flow.forEach((el, i) => {
    const p = `${path}.${i}`
    if (el.type === 'step') {
      const found = stepProblem(el)
      if (found) out.push({ at: `${p}.issue`, code: found[0], field: `${p}.${found[1]}` })
      return
    }
    el.branches.forEach((b, j) => {
      const field = unfinished(b.cond, `${p}.branches.${j}.cond`)
      if (field) out.push({ at: `${p}.branches.${j}.issue`, code: 'invalid_condition', field })
      walk(b.body, `${p}.branches.${j}.body`, out)
    })
    walk(el.otherwise, `${p}.otherwise`, out)
  })
}

// flowProblems checks the whole rule at once, in reading order, so the editor need not have
// every element's form on screen to know what is left to fix.
export function flowProblems(v: RuleValues): Problem[] {
  const out: Problem[] = []
  if (v.flow.length === 0) out.push({ at: 'flow.root', code: 'flow_required' })
  walk(v.flow, 'flow', out)
  if (!v.fallback) out.push({ at: 'settings', code: 'fallback_required', field: 'fallback' })
  else if (!inRange(v.max_levels, 1, maxLevels)) out.push({ at: 'settings', code: 'max_levels_range', field: 'max_levels' })
  return out
}

// setAt writes an error at a dotted name, creating the objects on the way.
function setAt(errors: Record<string, unknown>, name: string, value: unknown) {
  const parts = name.split('.')
  let node = errors
  for (const part of parts.slice(0, -1)) node = (node[part] ??= {}) as Record<string, unknown>
  node[parts[parts.length - 1]!] = value
}

// problemsResolver makes flowProblems the form's validation: each problem becomes an error at its name.
export function problemsResolver(t: TFunction): Resolver<RuleValues> {
  return (values) => {
    const problems = flowProblems(values)
    if (problems.length === 0) return { values, errors: {} }
    const errors: Record<string, unknown> = {}
    for (const p of problems) setAt(errors, p.at, { type: 'problem', message: t(`shared.rule.reason.${p.code}`) })
    return { values: {}, errors: errors as FieldErrors<RuleValues> }
  }
}
