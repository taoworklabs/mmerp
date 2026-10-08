import { describe, expect, it } from 'vitest'
import type { Me } from '@/shared/api/core'
import { can } from './me'

const me = { permissions: { hrm: ['hrm.employee.view'] } } as unknown as Me

describe('can', () => {
  it('takes any one of a list', () => {
    expect(can(me, 'hrm.employee.view')).toBe(true)
    expect(can(me, 'core.approval.manage')).toBe(false)
    expect(can(me, ['core.approval.manage', 'hrm.employee.view'])).toBe(true)
    expect(can(me, ['core.approval.manage', 'hrm.approval.manage'])).toBe(false)
    expect(can(me, undefined)).toBe(true)
  })
})
