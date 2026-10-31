import { describe, expect, it } from 'vitest'
import { adjustmentLineErrors } from './AdjustmentForm'

const line = { lot_code: '', serial: '', quantity: '5' }

describe('adjustmentLineErrors', () => {
  it('asks nothing more of a quantity item', () => {
    expect(adjustmentLineErrors(line, 'quantity')).toEqual([])
    expect(adjustmentLineErrors({ ...line, quantity: '-2.5' }, 'quantity')).toEqual([])
  })
  it('refuses an item not tracked in stock', () => {
    expect(adjustmentLineErrors(line, 'none')).toEqual(['stock_untracked'])
  })
  it('needs a lot for a lot item, coming in or going out', () => {
    expect(adjustmentLineErrors(line, 'lot')).toEqual(['stock_lot_required'])
    expect(adjustmentLineErrors({ ...line, quantity: '-3' }, 'lot')).toEqual(['stock_lot_required'])
    expect(adjustmentLineErrors({ ...line, lot_code: '  ' }, 'lot')).toEqual(['stock_lot_required'])
    expect(adjustmentLineErrors({ ...line, lot_code: 'L01' }, 'lot')).toEqual([])
  })
  it('needs a serial and a quantity of one unit, in or out, for a serial item', () => {
    expect(adjustmentLineErrors({ ...line, quantity: '1' }, 'serial')).toEqual(['stock_serial_required'])
    expect(adjustmentLineErrors({ ...line, serial: 'SN1', quantity: '2' }, 'serial')).toEqual(['stock_serial_quantity'])
    expect(adjustmentLineErrors({ ...line, serial: 'SN1', quantity: '1.000' }, 'serial')).toEqual([])
    expect(adjustmentLineErrors({ ...line, serial: 'SN1', quantity: '-1' }, 'serial')).toEqual([])
  })
})
