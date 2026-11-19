import type { Order, ReceiptLineInput, ReceiptOptions } from '@/shared/api/purchase'
import type { StatusTone } from '@/shared/ui/StatusBadge'

// Quantities are decimal strings with up to 3 decimals; these count them in thousandths so no
// float ever touches them.
const milli = (s: string) => {
  const [int = '0', frac = ''] = s.trim().split('.')
  return BigInt(int + frac.padEnd(3, '0').slice(0, 3))
}
const fromMilli = (n: bigint) => `${n / 1000n}.${String(n % 1000n).padStart(3, '0')}`

// addQuantities adds two non-negative quantities.
export const addQuantities = (a: string, b: string) => fromMilli(milli(a) + milli(b))

// goodsReceiptStateBadge: an order's receiving progress worth a badge; nothing yet or nothing to
// receive shows none.
export function goodsReceiptStateBadge(state: Order['receipt_state']): { tone: StatusTone; label: string } | null {
  if (state === 'partial') return { tone: 'info', label: 'purchase.order.receipt_state.partial' }
  if (state === 'full') return { tone: 'success', label: 'purchase.order.receipt_state.full' }
  return null
}

// ponytail: one line per serial, capped so a mistyped order quantity never floods the form
const maxSerialLines = 200

// linesFromOptions suggests a receipt's lines from what is left on the order: a quantity or lot
// line takes all that remains (the lot code left for the user to type), a serial line becomes one
// line per whole unit left, each waiting for its serial.
export function linesFromOptions(options: ReceiptOptions): ReceiptLineInput[] {
  return options.lines.flatMap((l): ReceiptLineInput[] => {
    const remaining = milli(l.remaining)
    if (remaining <= 0n) return []
    if (l.tracking === 'serial') {
      const units = Math.min(Number(remaining / 1000n), maxSerialLines)
      return Array.from({ length: units }, () => ({ order_position: l.order_position, quantity: '1' }))
    }
    return [{ order_position: l.order_position, quantity: fromMilli(remaining) }]
  })
}

// withOwnLines gives the options as a draft sees them: they count the draft's own lines as taken,
// though saving it again may take them anew.
export function withOwnLines(options: ReceiptOptions, own: Pick<ReceiptLineInput, 'order_position' | 'quantity'>[]): ReceiptOptions {
  const taken = new Map<number, string>()
  for (const l of own) taken.set(l.order_position, addQuantities(taken.get(l.order_position) ?? '0', l.quantity))
  return { ...options, lines: options.lines.map((l) => ({ ...l, remaining: addQuantities(l.remaining, taken.get(l.order_position) ?? '0') })) }
}
