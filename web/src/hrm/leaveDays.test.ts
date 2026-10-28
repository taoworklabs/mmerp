import { describe, expect, it } from 'vitest'
import { leaveDays } from './leaveDays'

describe('leaveDays', () => {
  it('takes half a day off for each half day', () => {
    expect(leaveDays(5, false, false)).toBe('5')
    expect(leaveDays(5, true, false)).toBe('4.5')
    expect(leaveDays(5, true, true)).toBe('4')
    expect(leaveDays(1, true, false)).toBe('0.5')
  })
  it('has no count without a working day, or with both halves of one day', () => {
    expect(leaveDays(0, false, false)).toBeNull()
    expect(leaveDays(1, true, true)).toBeNull()
  })
})
