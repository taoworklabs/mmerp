import { describe, expect, it } from 'vitest'
import { tenure } from './tenure'

describe('tenure', () => {
  it('counts whole years and months', () => {
    expect(tenure('2022-06-06', '2026-10-15')).toEqual({ years: 4, months: 4 })
    expect(tenure('2018-01-31', '2026-10-15')).toEqual({ years: 8, months: 8 })
  })
  it('counts a month only once its day is reached', () => {
    expect(tenure('2026-09-16', '2026-10-15')).toEqual({ years: 0, months: 0 })
    expect(tenure('2026-09-15', '2026-10-15')).toEqual({ years: 0, months: 1 })
  })
  it('is zero before the start', () => {
    expect(tenure('2026-11-01', '2026-10-15')).toEqual({ years: 0, months: 0 })
  })
})
