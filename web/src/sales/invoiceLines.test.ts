import { describe, expect, it } from 'vitest'
import type { InvoiceOptions } from '@/shared/api/sales'
import { invoiceLinesFromOptions, invoiceStateBadge, paymentStateBadge, withOwnInvoiceLines } from './invoiceLines'

type Line = InvoiceOptions['lines'][number]
const line = (l: Partial<Line>): Line => ({
  order_position: 1,
  item_id: 1,
  item_code: 'A',
  description: 'A',
  unit: 'cái',
  unit_price: 1000,
  discount_percent: '0',
  vat_rate: '8',
  ordered: '5.000',
  remaining: '5.000',
  ...l,
})

describe('invoiceLinesFromOptions', () => {
  it('bills what remains of each line, services included, and skips lines fully billed', () => {
    const options = { lines: [line({ remaining: '3.500' }), line({ order_position: 2, remaining: '0.000' }), line({ order_position: 3, remaining: '2.000' })] }
    expect(invoiceLinesFromOptions(options)).toEqual([
      { order_position: 1, quantity: '3.500' },
      { order_position: 3, quantity: '2.000' },
    ])
  })
})

describe('withOwnInvoiceLines', () => {
  it('gives a draft back what its own lines take', () => {
    const options = { lines: [line({ remaining: '1.000' })] }
    expect(withOwnInvoiceLines(options, [{ order_position: 1, quantity: '2.5' }]).lines[0]?.remaining).toBe('3.500')
  })
})

describe('badges', () => {
  it('shows invoice progress only once something is billed', () => {
    expect(invoiceStateBadge('none')).toBeNull()
    expect(invoiceStateBadge('partial')?.tone).toBe('info')
    expect(invoiceStateBadge('full')?.tone).toBe('success')
  })
  it('shows payment progress', () => {
    expect(paymentStateBadge('unpaid')?.tone).toBe('neutral')
    expect(paymentStateBadge('partial')?.tone).toBe('info')
    expect(paymentStateBadge('paid')?.tone).toBe('success')
    expect(paymentStateBadge(undefined)).toBeNull()
  })
})
