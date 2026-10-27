import { describe, expect, it } from 'vitest'
import { monthGrid, monthsOf, shiftDate, shiftMonth, weekOf } from './days'

describe('weekOf', () => {
  it('runs Monday to Sunday across a month boundary', () => {
    expect(weekOf('2026-10-01')).toEqual(['2026-09-28', '2026-09-29', '2026-09-30', '2026-10-01', '2026-10-02', '2026-10-03', '2026-10-04'])
  })
  it('keeps a Sunday in the week it ends', () => {
    expect(weekOf('2026-10-04')[0]).toBe('2026-09-28')
    expect(weekOf('2026-10-05')[0]).toBe('2026-10-05')
  })
})

describe('monthsOf', () => {
  it('lists the distinct months in order', () => {
    expect(monthsOf(weekOf('2026-10-01'))).toEqual(['2026-09', '2026-10'])
    expect(monthsOf(weekOf('2026-10-14'))).toEqual(['2026-10'])
  })
})

describe('shiftDate', () => {
  it('moves by days across months', () => {
    expect(shiftDate('2026-10-01', -7)).toBe('2026-09-24')
    expect(shiftDate('2026-12-30', 3)).toBe('2027-01-02')
  })
})

describe('monthGrid', () => {
  it('pads to whole Monday-first weeks', () => {
    const g = monthGrid('2026-10')
    expect(g).toHaveLength(35)
    expect(g.slice(0, 4)).toEqual([null, null, null, '2026-10-01'])
    expect(g.at(-1)).toBe(null)
    expect(g.filter(Boolean)).toHaveLength(31)
  })
})

describe('shiftMonth', () => {
  it('crosses year ends', () => {
    expect(shiftMonth('2026-01', -1)).toBe('2025-12')
    expect(shiftMonth('2026-12', 1)).toBe('2027-01')
  })
})
