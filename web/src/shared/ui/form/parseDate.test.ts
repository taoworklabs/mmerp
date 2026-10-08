import { describe, expect, it } from 'vitest'
import { parseDate } from './fields'

describe('parseDate', () => {
  it('reads day first', () => {
    expect(parseDate('05/01/2026')).toBe('2026-01-05')
    expect(parseDate('5.1.2026')).toBe('2026-01-05')
  })
  it('rejects impossible dates and other shapes', () => {
    expect(parseDate('31/02/2026')).toBeNull()
    expect(parseDate('2026-01-05')).toBeNull()
    expect(parseDate('')).toBeNull()
  })
})
