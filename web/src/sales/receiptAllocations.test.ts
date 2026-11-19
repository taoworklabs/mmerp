import { describe, expect, it } from 'vitest'
import type { Allocation } from '@/shared/api/sales'
import { allocationsFor, withOwnAllocations } from './receiptAllocations'

const open = (id: number, remaining: number): Allocation => ({
  invoice_id: id,
  amount: remaining,
  invoice_number: `HB-${id}`,
  invoice_date: '2026-03-12',
  invoice_total: 10_000,
  remaining,
})

describe('allocationsFor', () => {
  it('starts from one invoice with what is left of it', () => {
    expect(allocationsFor([open(1, 300), open(2, 500)], 2)).toEqual([{ invoice_id: 2, amount: 500 }])
  })
  it('starts empty when the invoice has nothing left or is unknown', () => {
    expect(allocationsFor([open(1, 300)], 9)).toEqual([])
    expect(allocationsFor([open(1, 300)], undefined)).toEqual([])
  })
})

describe('withOwnAllocations', () => {
  it("keeps a draft's own invoices selectable, with its amounts given back", () => {
    const own = [{ ...open(3, 0), amount: 700, remaining: 900 }]
    const list = withOwnAllocations([open(1, 300)], own)
    expect(list.map((a) => [a.invoice_id, a.remaining])).toEqual([
      [1, 300],
      [3, 900],
    ])
  })
})
