import { describe, expect, it } from 'vitest'
import type { DeliveryOptions } from '@/shared/api/sales'
import { addQuantities, linesFromOptions, withOwnLines } from './deliveryLines'

type Line = DeliveryOptions['lines'][number]
const line = (l: Partial<Line>): Line => ({
  order_position: 1,
  item_id: 1,
  item_code: 'A',
  description: 'A',
  unit: 'cái',
  tracking: 'quantity',
  ordered: '5.000',
  remaining: '5.000',
  lots: [],
  serials: [],
  ...l,
})
const options = (lines: Line[]): DeliveryOptions => ({ warehouse_id: 1, warehouses: [], lines })

describe('linesFromOptions', () => {
  it('takes what remains of a quantity line', () => {
    expect(linesFromOptions(options([line({ remaining: '3.500' })]))).toEqual([{ order_position: 1, quantity: '3.500' }])
  })
  it('splits a lot line over the suggested lots and leaves the shortfall without a lot', () => {
    const lots = [
      { code: 'L1', expiry: '2027-01-01', quantity: '2.000' },
      { code: 'L2', expiry: null, quantity: '2.000' },
    ]
    expect(linesFromOptions(options([line({ order_position: 2, tracking: 'lot', lots })]))).toEqual([
      { order_position: 2, lot_code: 'L1', quantity: '2.000' },
      { order_position: 2, lot_code: 'L2', quantity: '2.000' },
      { order_position: 2, quantity: '1.000' },
    ])
  })
  it('suggests nothing for a serial line nor for a line already delivered', () => {
    expect(linesFromOptions(options([line({ tracking: 'serial', serials: ['S1'] }), line({ order_position: 2, remaining: '0.000' })]))).toEqual([])
  })
  it('leaves a lot line with no suggested lots without a lot', () => {
    expect(linesFromOptions(options([line({ tracking: 'lot', remaining: '2.000' })]))).toEqual([{ order_position: 1, quantity: '2.000' }])
  })
})

describe('addQuantities', () => {
  it('adds decimal strings without floating point', () => {
    expect(addQuantities('0.1', '0.2')).toBe('0.300')
    expect(addQuantities('12', '0.005')).toBe('12.005')
  })
})

describe('withOwnLines', () => {
  it("gives a draft back what its own lines take, so its lines rebuild", () => {
    const opts = options([line({ remaining: '0.000' }), line({ order_position: 2, remaining: '1.000' })])
    const own = [
      { order_position: 1, quantity: '2' },
      { order_position: 1, quantity: '0.5' },
    ]
    expect(withOwnLines(opts, own).lines.map((l) => l.remaining)).toEqual(['2.500', '1.000'])
    expect(linesFromOptions(withOwnLines(opts, own))).toEqual([
      { order_position: 1, quantity: '2.500' },
      { order_position: 2, quantity: '1.000' },
    ])
  })
})
