import { describe, expect, it } from 'vitest'
import { monthBounds } from './month'

describe('monthBounds', () => {
  it('covers the whole month, leap February included', () => {
    expect(monthBounds('2026-04')).toEqual({ from: '2026-04-01', to: '2026-04-30' })
    expect(monthBounds('2028-02')).toEqual({ from: '2028-02-01', to: '2028-02-29' })
    expect(monthBounds('2026-12')).toEqual({ from: '2026-12-01', to: '2026-12-31' })
  })
  it('has no bounds without a month', () => {
    expect(monthBounds('')).toEqual({})
  })
})
