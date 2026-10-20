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
export type Problem = { at: string; code: ProblemCode }

// The server's limits.
const maxCount = 20
const maxDueHours = 720
const maxLevels = 10

const inRange = (v: string, min: number, max: number) => /^\d+$/.test(v) && Number(v) >= min && Number(v) <= max

function stepProblem(el: ElementValues): ProblemCode | null {
  if (el.kind === 'role' && !el.role) return 'missing_role'
  if (el.kind === 'user' && el.users.length === 0) return 'missing_user'
  if (el.kind === 'user' && Number(el.approvals) > el.users.length) return 'approvals_exceed_users'
  if (!inRange(el.approvals, 1, maxCount) || !inRange(el.rejections, 1, maxCount)) return 'count_range'
  if (el.dueHours !== '' && !inRange(el.dueHours, 1, maxDueHours)) return 'invalid_due'
  return null
}

// A condition is finished when every leaf has its field, comparison and value; no rows means always.
function condFinished(c: CondValues): boolean {
  return c.rows.every((r) => (r.group ? condFinished(r.group) : !!r.field && !!r.op && r.value !== ''))
}

function walk(flow: FlowValues, path: string, out: Problem[]) {
  flow.forEach((el, i) => {
    const p = `${path}.${i}`
    if (el.type === 'step') {
      const code = stepProblem(el)
      if (code) out.push({ at: `${p}.issue`, code })
      return
    }
    el.branches.forEach((b, j) => {
      if (!condFinished(b.cond)) out.push({ at: `${p}.branches.${j}.issue`, code: 'invalid_condition' })
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
  if (!v.fallback) out.push({ at: 'settings', code: 'fallback_required' })
  else if (!inRange(v.max_levels, 1, maxLevels)) out.push({ at: 'settings', code: 'max_levels_range' })
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
