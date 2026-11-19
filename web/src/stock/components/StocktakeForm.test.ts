import { describe, expect, it } from 'vitest'
import { draftDifference, stocktakeBody } from './StocktakeForm'

describe('draftDifference', () => {
  it('is counted less on hand, exactly, to three decimals', () => {
    expect(draftDifference('10', '7')).toBe('-3')
    expect(draftDifference('0.1', '0.3')).toBe('0.2')
    expect(draftDifference('2.5', '2.5')).toBe('0')
    expect(draftDifference('-1.25', '0')).toBe('1.25')
  })
  it('says nothing for a count not entered yet', () => {
    expect(draftDifference('4', '')).toBeNull()
    expect(draftDifference(undefined, '4')).toBeNull()
  })
  it('says nothing for a number it cannot read exactly', () => {
    expect(draftDifference('0', '1e+21')).toBeNull()
  })
})

describe('stocktakeBody', () => {
  it('sends the lot or serial its item asks for, trimmed', () => {
    const body = stocktakeBody({
      date: '2026-11-02',
      warehouse_id: '3',
      note: '',
      lines: [
        { item_id: '7', tracking: 'quantity', lot_code: 'stale', expiry: null, serial: '', counted: '5' },
        { item_id: '8', tracking: 'lot', lot_code: ' L1 ', expiry: '2027-01-31', serial: '', counted: '0' },
        { item_id: '9', tracking: 'serial', lot_code: '', expiry: null, serial: ' S1 ', counted: '1' },
      ],
    })
    expect(body.warehouse_id).toBe(3)
    expect(body.lines).toEqual([
      { item_id: 7, counted: '5', lot: undefined, serial: undefined },
      { item_id: 8, counted: '0', lot: { code: 'L1', expiry: '2027-01-31' }, serial: undefined },
      { item_id: 9, counted: '1', lot: undefined, serial: 'S1' },
    ])
  })
})
