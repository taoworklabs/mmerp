import type { Flow, TypeRule } from '@/shared/api/core'
import { describe, expect, it } from 'vitest'
import { flowSummary, formPathOf, fromFlowValues, newSplit, newStep, parentOf, selectionKind, toFlowValues, toRuleInput, toRuleValues, type ElementValues } from './flowForm'

const nested: Flow = [
  { step: { approver: { kind: 'user', user_ids: [3, 4] }, approvals: 2, rejections: 1 } },
  {
    split: {
      mode: 'one',
      branches: [
        {
          condition: {
            any: [
              { field: 'days', op: 'gt', value: '3' },
              { all: [{ field: 'kind', op: 'eq', value: 'a' }, { field: 'org_unit', op: 'within', value: '7' }] },
            ],
          },
          body: [
            {
              split: {
                mode: 'all',
                branches: [
                  { body: [{ step: { approver: { kind: 'module' }, approvals: 1, rejections: 1 } }] },
                  {
                    condition: { field: 'kind', op: 'eq', value: 'b' },
                    body: [{ step: { approver: { kind: 'role', product: 'hrm', role: 'hr' }, approvals: 1, rejections: 1 } }],
                  },
                ],
              },
            },
          ],
        },
      ],
      otherwise: [{ step: { approver: { kind: 'module' }, approvals: 1, rejections: 1 } }],
    },
  },
]

describe('toFlowValues / fromFlowValues', () => {
  it('round-trips nested flows', () => {
    expect(fromFlowValues(toFlowValues(nested))).toEqual(nested)
  })

  it('drops fields of other approver kinds', () => {
    const v: ElementValues = { type: 'step', kind: 'role', role: 'hrm.hr', users: ['5'], approvals: '1', rejections: '1', dueHours: '', branches: [], otherwise: [] }
    expect(fromFlowValues([v])).toEqual([{ step: { approver: { kind: 'role', product: 'hrm', role: 'hr' }, approvals: 1, rejections: 1 } }])
  })

  it('round-trips due hours', () => {
    const flow: Flow = [{ step: { approver: { kind: 'module' }, approvals: 1, rejections: 1, due_hours: 24 } }]
    expect(toFlowValues(flow)[0]?.dueHours).toBe('24')
    expect(fromFlowValues(toFlowValues(flow))).toEqual(flow)
  })

  it('no due hours stays absent', () => {
    expect(fromFlowValues([newStep(true)])).toEqual([{ step: { approver: { kind: 'module' }, approvals: 1, rejections: 1 } }])
  })

  it('parallel drops otherwise', () => {
    const v: ElementValues = {
      type: 'all',
      kind: null,
      role: null,
      users: [],
      approvals: '1',
      rejections: '1',
      dueHours: '',
      branches: [{ cond: { mode: 'all', rows: [] }, body: [newStep(true)] }],
      otherwise: [newStep(true)],
    }
    expect(fromFlowValues([v])).toEqual([{ split: { mode: 'all', branches: [{ body: [{ step: { approver: { kind: 'module' }, approvals: 1, rejections: 1 } }] }] } }])
  })

  it('a branch without rows has no condition', () => {
    const v = newSplit('one', true)
    expect(fromFlowValues([v])).toEqual([{ split: { mode: 'one', branches: [{ body: [{ step: { approver: { kind: 'module' }, approvals: 1, rejections: 1 } }] }] } }])
  })

  it('drops a nested group left with no rows', () => {
    const v: ElementValues = {
      type: 'one',
      kind: null,
      role: null,
      users: [],
      approvals: '1',
      rejections: '1',
      dueHours: '',
      branches: [
        {
          cond: { mode: 'all', rows: [{ group: null, field: 'days', op: 'gt', value: '3' }, { group: { mode: 'all', rows: [] }, field: null, op: null, value: '' }] },
          body: [newStep(true)],
        },
      ],
      otherwise: [],
    }
    const out = fromFlowValues([v])[0]
    expect(out?.split?.branches[0]?.condition).toEqual({ field: 'days', op: 'gt', value: '3' })
  })

  it('a branch with only an empty nested group has no condition', () => {
    const v: ElementValues = {
      type: 'one',
      kind: null,
      role: null,
      users: [],
      approvals: '1',
      rejections: '1',
      dueHours: '',
      branches: [{ cond: { mode: 'all', rows: [{ group: { mode: 'all', rows: [] }, field: null, op: null, value: '' }] }, body: [newStep(true)] }],
      otherwise: [],
    }
    const out = fromFlowValues([v])[0]
    expect(out?.split?.branches[0]?.condition).toBeUndefined()
  })

  it('one leaf is written bare', () => {
    const v: ElementValues = {
      type: 'one',
      kind: null,
      role: null,
      users: [],
      approvals: '1',
      rejections: '1',
      dueHours: '',
      branches: [{ cond: { mode: 'any', rows: [{ group: null, field: 'days', op: 'gt', value: '3' }] }, body: [newStep(true)] }],
      otherwise: [],
    }
    const out = fromFlowValues([v])[0]
    expect(out?.split?.branches[0]?.condition).toEqual({ field: 'days', op: 'gt', value: '3' })
  })
})

describe('formPathOf', () => {
  it('maps api paths to form paths', () => {
    expect(formPathOf('1')).toBe('flow.0')
    expect(formPathOf('2.branches.1.body.1')).toBe('flow.1.branches.0.body.0')
    expect(formPathOf('3.otherwise.2')).toBe('flow.2.otherwise.1')
    expect(formPathOf('')).toBe('flow')
  })
})

describe('flowSummary', () => {
  it('summarises flows', () => {
    expect(flowSummary(nested)).toEqual({ steps: 4, branched: true })
    expect(flowSummary([{ step: { approver: { kind: 'module' }, approvals: 1, rejections: 1 } }])).toEqual({ steps: 1, branched: false })
  })
})

const typeRule: TypeRule = {
  action: 'post',
  allowed_actions: ['save'],
  doc_type: 'hrm.leave_request',
  fields: [],
  module_approvers: true,
  product: 'hrm',
  rule: { flow: nested, max_levels: 3, fallback_product: 'hrm', fallback_role: 'hr' },
}

describe('rule values', () => {
  it('a type without a rule opens with an empty flow', () => {
    expect(toRuleValues({ ...typeRule, rule: undefined })).toEqual({ flow: [], max_levels: '3', fallback: null })
  })

  it('round-trips a rule through values', () => {
    expect(toRuleInput(toRuleValues(typeRule))).toEqual(typeRule.rule)
  })
})

describe('selectionKind', () => {
  it('names what a selection points at', () => {
    const v = toRuleValues(typeRule)
    expect(selectionKind('flow.0', v)).toBe('step')
    expect(selectionKind('flow.1', v)).toBe('split')
    expect(selectionKind('flow.1.branches.0', v)).toBe('branch')
    expect(selectionKind('flow.1.otherwise', v)).toBe('otherwise')
    expect(selectionKind('flow.1.branches.0.body.0.branches.1.body.0', v)).toBe('step')
    expect(selectionKind('settings', v)).toBe('settings')
    expect(selectionKind('flow.9', v)).toBeNull()
    expect(selectionKind('flow.1.branches.4', v)).toBeNull()
  })

  it('splits a path into its array and index', () => {
    expect(parentOf('flow.1.branches.0')).toEqual(['flow.1.branches', 0])
    expect(parentOf('flow.2')).toEqual(['flow', 2])
  })
})
