import { describe, expect, it } from 'vitest'
import { addDays, addMonths, monthGrid } from './calendarGrid'

describe('monthGrid', () => {
  it('starts on the chosen weekday and spans 42 days', () => {
    const g = monthGrid(2026, 10, 1) // 1 Oct 2026 is a Thursday
    expect(g).toHaveLength(42)
    expect(g[0]).toBe('2026-09-28')
    expect(g[3]).toBe('2026-10-01')
    expect(g[41]).toBe('2026-11-08')
  })
  it('starts on the 1st when it falls on the first weekday', () => {
    expect(monthGrid(2026, 2, 0)[0]).toBe('2026-02-01') // a Sunday
  })
})

describe('date arithmetic', () => {
  it('crosses month and year ends', () => {
    expect(addDays('2026-12-31', 1)).toBe('2027-01-01')
    expect(addMonths('2026-01-31', 1)).toBe('2026-02-28')
  })
})
