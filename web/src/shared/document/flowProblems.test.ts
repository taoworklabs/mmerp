import type { TFunction } from 'i18next'
import { describe, expect, it } from 'vitest'
import { formPathOf, newSplit, newStep, type RuleValues } from './flowForm'
import { flowProblems, problemsResolver } from './flowProblems'

const ok: RuleValues = { flow: [newStep(true)], max_levels: '3', fallback: 'hrm.hr' }

describe('flowProblems', () => {
  it('a valid rule has no problems', () => {
    expect(flowProblems(ok)).toEqual([])
  })

  it('an empty flow and no fallback', () => {
    expect(flowProblems({ flow: [], max_levels: '3', fallback: null })).toEqual([
      { at: 'flow.root', code: 'flow_required' },
      { at: 'settings', code: 'fallback_required', field: 'fallback' },
    ])
  })

  it('max levels outside 1–10', () => {
    expect(flowProblems({ ...ok, max_levels: '11' })).toEqual([{ at: 'settings', code: 'max_levels_range', field: 'max_levels' }])
    expect(flowProblems({ ...ok, max_levels: '' })).toEqual([{ at: 'settings', code: 'max_levels_range', field: 'max_levels' }])
  })

  it('step problems, one per step', () => {
    const role = { ...newStep(false), role: null }
    const users = { ...newStep(false), kind: 'user', users: ['1'], approvals: '2' }
    const due = { ...newStep(true), dueHours: '721' }
    const count = { ...newStep(true), rejections: '0' }
    expect(flowProblems({ ...ok, flow: [role, users, due, count] })).toEqual([
      { at: 'flow.0.issue', code: 'missing_role', field: 'flow.0.role' },
      { at: 'flow.1.issue', code: 'approvals_exceed_users', field: 'flow.1.approvals' },
      { at: 'flow.2.issue', code: 'invalid_due', field: 'flow.2.dueHours' },
      { at: 'flow.3.issue', code: 'count_range', field: 'flow.3.rejections' },
    ])
  })

  it('a cleared due time or count, left as null by its field, reads as empty or wrong', () => {
    const cleared = { ...newStep(true), dueHours: null as unknown as string }
    expect(flowProblems({ ...ok, flow: [cleared] })).toEqual([])
    const noCount = { ...newStep(true), approvals: null as unknown as string }
    expect(flowProblems({ ...ok, flow: [noCount] })).toEqual([{ at: 'flow.0.issue', code: 'count_range', field: 'flow.0.approvals' }])
  })

  it('a user step with nobody chosen', () => {
    expect(flowProblems({ ...ok, flow: [{ ...newStep(false), kind: 'user', users: [] }] })).toEqual([{ at: 'flow.0.issue', code: 'missing_user', field: 'flow.0.users' }])
  })

  it('an unfinished condition, nested, marks its branch', () => {
    const split = newSplit('one', true)
    split.branches[0]!.cond = {
      mode: 'all',
      rows: [{ group: { mode: 'any', rows: [{ group: null, field: 'days', op: 'gt', value: '' }] }, field: null, op: null, value: '' }],
    }
    split.otherwise = [{ ...newStep(false), role: null }]
    expect(flowProblems({ ...ok, flow: [split] })).toEqual([
      { at: 'flow.0.branches.0.issue', code: 'invalid_condition', field: 'flow.0.branches.0.cond.rows.0.group.rows.0.value' },
      { at: 'flow.0.otherwise.0.issue', code: 'missing_role', field: 'flow.0.otherwise.0.role' },
    ])
  })

  it('looks inside branch bodies', () => {
    const split = newSplit('all', true)
    split.branches[0]!.body = [newStep(true), { ...newStep(false), kind: 'user', users: [] }]
    expect(flowProblems({ ...ok, flow: [split] })).toEqual([{ at: 'flow.0.branches.0.body.1.issue', code: 'missing_user', field: 'flow.0.branches.0.body.1.users' }])
  })
})

describe('server paths', () => {
  // The server names an element by a 1-based path; the editor marks the box whose error name
  // formPathOf gives. The client check must use the very same names, or Save and the server
  // would point at different boxes.
  it('name the same boxes as the client check, inside nested splits', () => {
    const split = newSplit('one', true)
    split.branches.push({ cond: { mode: 'all', rows: [{ group: null, field: 'days', op: null, value: '' }] }, body: [newStep(true), { ...newStep(false), role: null }] })
    const ats = flowProblems({ ...ok, flow: [newStep(true), split] }).map((p) => p.at)
    expect(ats).toEqual([`${formPathOf('2.branches.2')}.issue`, `${formPathOf('2.branches.2.body.2')}.issue`])
  })
})

describe('problemsResolver', () => {
  const t = ((k: string) => k) as unknown as TFunction

  it('puts translated messages at each name', async () => {
    const r = await problemsResolver(t)({ flow: [], max_levels: '3', fallback: 'hrm.hr' }, undefined, {} as never)
    expect(r.errors).toEqual({ flow: { root: { type: 'problem', message: 'shared.rule.reason.flow_required' } } })
  })

  it('passes the values through when there are none', async () => {
    const r = await problemsResolver(t)(ok, undefined, {} as never)
    expect(r).toEqual({ values: ok, errors: {} })
  })
})
