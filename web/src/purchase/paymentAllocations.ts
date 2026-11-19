import type { Allocation, AllocationInput } from '@/shared/api/purchase'

// allocationsFor starts a payment from one invoice, for all that other payments leave of it.
export function allocationsFor(open: Allocation[], invoice: number | undefined): AllocationInput[] {
  const a = open.find((o) => o.invoice_id === invoice)
  return a && a.remaining > 0 ? [{ invoice_id: a.invoice_id, amount: a.remaining }] : []
}

// withOwnAllocations adds a draft's own invoices to the open ones: the open list leaves out an
// invoice the draft settles in full, and the draft's allocations already say what is left.
export function withOwnAllocations(open: Allocation[], own: Allocation[]): Allocation[] {
  const ids = new Set(open.map((o) => o.invoice_id))
  return [...open, ...own.filter((o) => !ids.has(o.invoice_id))]
}
