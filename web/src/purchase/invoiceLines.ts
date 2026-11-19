import type { Order, InvoiceLineInput, InvoiceOptions, PaymentState } from '@/shared/api/purchase'
import type { StatusTone } from '@/shared/ui/StatusBadge'
import { addQuantities } from './receiptLines'

const nothing = /^0+(\.0+)?$/

// invoiceLinesFromOptions suggests an invoice's lines: every order line billed for what remains,
// at the order line's price, discount and VAT rate until the supplier's paper says otherwise.
export const invoiceLinesFromOptions = (options: InvoiceOptions): InvoiceLineInput[] =>
  options.lines
    .filter((l) => !nothing.test(l.remaining))
    .map((l) => ({ order_position: l.order_position, quantity: l.remaining, unit_price: l.unit_price, discount_percent: l.discount_percent, vat_rate: l.vat_rate }))

// withOwnInvoiceLines gives the options as a draft sees them: they count its own lines as taken.
export function withOwnInvoiceLines(options: InvoiceOptions, own: InvoiceLineInput[]): InvoiceOptions {
  const taken = new Map<number, string>()
  for (const l of own) taken.set(l.order_position, addQuantities(taken.get(l.order_position) ?? '0', l.quantity))
  return { ...options, lines: options.lines.map((l) => ({ ...l, remaining: addQuantities(l.remaining, taken.get(l.order_position) ?? '0') })) }
}

// invoiceStateBadge: an order's billing progress worth a badge; nothing billed shows none.
export function invoiceStateBadge(state: Order['invoice_state']): { tone: StatusTone; label: string } | null {
  if (state === 'partial') return { tone: 'info', label: 'purchase.order.invoice_state.partial' }
  if (state === 'full') return { tone: 'success', label: 'purchase.order.invoice_state.full' }
  return null
}

// paymentStateBadge: how much of an invoice or order has been paid.
export function paymentStateBadge(state: PaymentState | Order['payment_state'] | undefined): { tone: StatusTone; label: string } | null {
  if (state === 'unpaid') return { tone: 'neutral', label: 'purchase.payment_state.unpaid' }
  if (state === 'partial') return { tone: 'info', label: 'purchase.payment_state.partial' }
  if (state === 'paid') return { tone: 'success', label: 'purchase.payment_state.paid' }
  return null
}
