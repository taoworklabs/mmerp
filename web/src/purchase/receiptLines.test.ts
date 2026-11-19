import { describe, expect, it } from 'vitest'
import type { ReceiptOptions } from '@/shared/api/purchase'
import { linesFromOptions, withOwnLines } from './receiptLines'

type OptionLine = ReceiptOptions['lines'][number]
const line = (o: Partial<OptionLine>): OptionLine => ({
  order_position: 1,
  item_id: 1,
  item_code: 'X',
  description: 'X',
  unit: 'cái',
  tracking: 'quantity',
  ordered: '10.000',
  remaining: '10.000',
  ...o,
})
const options = (...lines: OptionLine[]): ReceiptOptions => ({ warehouses: [], lines })

describe('linesFromOptions', () => {
  it('takes what remains of quantity and lot lines, one line per serial unit', () => {
    const got = linesFromOptions(
      options(
        line({ order_position: 1, remaining: '2.500' }),
        line({ order_position: 2, tracking: 'lot', remaining: '4.000' }),
        line({ order_position: 3, tracking: 'serial', remaining: '2.000' }),
        line({ order_position: 4, remaining: '0.000' }),
      ),
    )
    expect(got).toEqual([
      { order_position: 1, quantity: '2.500' },
      { order_position: 2, quantity: '4.000' },
      { order_position: 3, quantity: '1' },
      { order_position: 3, quantity: '1' },
    ])
  })
})

describe('withOwnLines', () => {
  it('gives a draft back its own quantities', () => {
    const got = withOwnLines(options(line({ remaining: '1.000' })), [{ order_position: 1, quantity: '3' }])
    expect(got.lines[0]?.remaining).toBe('4.000')
  })
})
