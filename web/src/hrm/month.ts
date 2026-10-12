// monthBounds turns a month filter (YYYY-MM) into the first and last day it covers, for the
// API's from/to; an empty month gives no bounds.
export function monthBounds(month: string): { from?: string; to?: string } {
  if (!month) return {}
  const [y, m] = month.split('-').map(Number) as [number, number]
  const last = new Date(Date.UTC(y, m, 0)).getUTCDate()
  return { from: `${month}-01`, to: `${month}-${String(last).padStart(2, '0')}` }
}
