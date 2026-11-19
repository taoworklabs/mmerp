import { describe, expect, it } from 'vitest'
import { transferBody, transferLineErrors } from './TransferForm'

const line = { item_id: '7', tracking: 'quantity' as const, lot_code: '', serial: '', quantity: '4' }

describe('transferLineErrors', () => {
  it('asks nothing more of a quantity item', () => {
    expect(transferLineErrors(line)).toEqual([])
  })
  it('needs a lot for a lot item', () => {
    expect(transferLineErrors({ ...line, tracking: 'lot', lot_code: ' ' })).toEqual(['stock_lot_required'])
  })
  it('moves a serial one unit at a time', () => {
    expect(transferLineErrors({ ...line, tracking: 'serial', serial: '' })).toEqual(['stock_serial_required'])
    expect(transferLineErrors({ ...line, tracking: 'serial', serial: 'S1', quantity: '2' })).toEqual(['stock_serial_quantity'])
    expect(transferLineErrors({ ...line, tracking: 'serial', serial: 'S1', quantity: '1' })).toEqual([])
  })
})

describe('transferBody', () => {
  it('sends the lot or serial its item asks for, trimmed', () => {
    const body = transferBody({
      date: '2026-11-02',
      from_warehouse_id: '1',
      to_warehouse_id: '2',
      note: ' gấp ',
      lines: [line, { ...line, tracking: 'lot', lot_code: ' L1 ', serial: 'stale' }, { ...line, tracking: 'serial', lot_code: 'stale', serial: ' S1 ', quantity: '1' }],
    })
    expect(body).toEqual({
      date: '2026-11-02',
      from_warehouse_id: 1,
      to_warehouse_id: 2,
      note: 'gấp',
      lines: [
        { item_id: 7, quantity: '4', lot: undefined, serial: undefined },
        { item_id: 7, quantity: '4', lot: 'L1', serial: undefined },
        { item_id: 7, quantity: '1', lot: undefined, serial: 'S1' },
      ],
    })
  })
})
