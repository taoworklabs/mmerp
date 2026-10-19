import { describe, expect, it } from 'vitest'
import { fromFlow, toFlow, type StepValues } from './ruleFlow'

const conditional: StepValues = { kind: 'user', role: null, user: '5', field: 'days', op: 'gt', value: '3' }
const always: StepValues = { kind: 'module', role: null, user: null, field: null, op: null, value: '' }
const byRole: StepValues = { kind: 'role', role: 'hrm.manager', user: null, field: null, op: null, value: '' }

describe('toFlow', () => {
  it('wraps a conditional step in a choose-one split and leaves the rest as steps', () => {
    expect(toFlow([conditional, always])).toEqual([
      {
        split: {
          mode: 'one',
          branches: [{ condition: { field: 'days', op: 'gt', value: '3' }, body: [{ step: { approver: { kind: 'user', user_ids: [5] }, approvals: 1, rejections: 1 } }] }],
          otherwise: [],
        },
      },
      { step: { approver: { kind: 'module' }, approvals: 1, rejections: 1 } },
    ])
  })
})

describe('fromFlow', () => {
  it('reads back what toFlow wrote', () => {
    const steps = [conditional, always, byRole]
    expect(fromFlow(toFlow(steps))).toEqual(steps)
  })

  it('reads a split whose otherwise the server left out', () => {
    expect(fromFlow([{ split: { mode: 'one', branches: [{ condition: { field: 'days', op: 'gt', value: '3' }, body: [{ step: { approver: { kind: 'user', user_ids: [5] }, approvals: 1, rejections: 1 } }] }] } }])).toEqual([conditional])
  })

  it('gives up on a parallel split', () => {
    expect(fromFlow([{ split: { mode: 'all', branches: [{ body: [{ step: { approver: { kind: 'module' }, approvals: 1, rejections: 1 } }] }] } }])).toBeNull()
  })

  it('gives up on a step that needs more than one approval', () => {
    expect(fromFlow([{ step: { approver: { kind: 'module' }, approvals: 2, rejections: 1 } }])).toBeNull()
  })
})
