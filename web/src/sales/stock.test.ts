import { describe, expect, it } from 'vitest'
import { isExpired, monthToDate, stockCardPath } from './stock'

describe('stock helpers', () => {
  it('defaults the range to the current month up to today', () => {
    expect(monthToDate('2026-10-30')).toEqual({ from: '2026-10-01', to: '2026-10-30' })
  })
  it('links an on-hand row to its stock card', () => {
    expect(stockCardPath(3, 9, '2026-10-05')).toBe('/sales/stock/card?warehouse_id=3&item_id=9&from=2026-10-01&to=2026-10-05')
  })
  it('expires a lot only after its expiry day', () => {
    expect(isExpired('2026-10-29', '2026-10-30')).toBe(true)
    expect(isExpired('2026-10-30', '2026-10-30')).toBe(false)
    expect(isExpired(undefined, '2026-10-30')).toBe(false)
  })
})
