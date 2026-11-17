import type { Doc, InvoiceLineInput, InvoiceOptions, PaymentState } from '@/shared/api/sales'
import type { StatusTone } from '@/shared/ui/StatusBadge'
import { addQuantities } from './deliveryLines'

const nothing = /^0+(\.0+)?$/

// invoiceLinesFromOptions suggests an invoice's lines: every order line billed for what remains.
export const invoiceLinesFromOptions = (options: InvoiceOptions): InvoiceLineInput[] =>
  options.lines.filter((l) => !nothing.test(l.remaining)).map((l) => ({ order_position: l.order_position, quantity: l.remaining }))

// withOwnInvoiceLines gives the options as a draft sees them: they count its own lines as taken.
export function withOwnInvoiceLines(options: InvoiceOptions, own: InvoiceLineInput[]): InvoiceOptions {
  const taken = new Map<number, string>()
  for (const l of own) taken.set(l.order_position, addQuantities(taken.get(l.order_position) ?? '0', l.quantity))
  return { ...options, lines: options.lines.map((l) => ({ ...l, remaining: addQuantities(l.remaining, taken.get(l.order_position) ?? '0') })) }
}

// invoiceStateBadge: an order's billing progress worth a badge; nothing billed shows none.
export function invoiceStateBadge(state: Doc['invoice_state']): { tone: StatusTone; label: string } | null {
  if (state === 'partial') return { tone: 'info', label: 'sales.order.invoice_state.partial' }
  if (state === 'full') return { tone: 'success', label: 'sales.order.invoice_state.full' }
  return null
}

// paymentStateBadge: how much of an invoice or order has been received.
export function paymentStateBadge(state: PaymentState | Doc['payment_state'] | undefined): { tone: StatusTone; label: string } | null {
  if (state === 'unpaid') return { tone: 'neutral', label: 'sales.payment_state.unpaid' }
  if (state === 'partial') return { tone: 'info', label: 'sales.payment_state.partial' }
  if (state === 'paid') return { tone: 'success', label: 'sales.payment_state.paid' }
  return null
}
