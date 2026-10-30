import type { DeliveryLineInput, DeliveryOptions } from '@/shared/api/sales'

// Quantities are decimal strings with up to 3 decimals; these count them in thousandths so no
// float ever touches them.
const milli = (s: string) => {
  const [int = '0', frac = ''] = s.trim().split('.')
  return BigInt(int + frac.padEnd(3, '0').slice(0, 3))
}
const fromMilli = (n: bigint) => `${n / 1000n}.${String(n % 1000n).padStart(3, '0')}`

// addQuantities adds two non-negative quantities.
export const addQuantities = (a: string, b: string) => fromMilli(milli(a) + milli(b))

// linesFromOptions suggests a delivery's lines from what is left on the order: a quantity line
// takes all that remains, a lot line one line per suggested lot with the shortfall left without a
// lot for the user to pick, and a serial line nothing, as each serial is picked by hand.
export function linesFromOptions(options: DeliveryOptions): DeliveryLineInput[] {
  return options.lines.flatMap((l): DeliveryLineInput[] => {
    const remaining = milli(l.remaining)
    if (remaining <= 0n || l.tracking === 'serial') return []
    if (l.tracking === 'quantity') return [{ order_position: l.order_position, quantity: l.remaining }]
    const picked = l.lots.reduce((sum, lot) => sum + milli(lot.quantity), 0n)
    const lines = l.lots.map((lot) => ({ order_position: l.order_position, lot_code: lot.code, quantity: lot.quantity }))
    return picked < remaining ? [...lines, { order_position: l.order_position, quantity: fromMilli(remaining - picked) }] : lines
  })
}
