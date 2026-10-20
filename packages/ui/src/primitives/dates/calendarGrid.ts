export type FirstDayOfWeek = 0 | 1 | 2 | 3 | 4 | 5 | 6

export interface CalendarLocale {
  // Short names, Sunday first.
  weekdays: string[]
  months: string[]
  firstDayOfWeek?: FirstDayOfWeek
}

function toDate(iso: string): Date {
  const [y = 0, m = 1, d = 1] = iso.split('-').map(Number)
  return new Date(Date.UTC(y, m - 1, d))
}

function toIso(date: Date): string {
  return date.toISOString().slice(0, 10)
}

export function addDays(iso: string, n: number): string {
  const date = toDate(iso)
  date.setUTCDate(date.getUTCDate() + n)
  return toIso(date)
}

export function addMonths(iso: string, n: number): string {
  const [y = 0, m = 1, d = 1] = iso.split('-').map(Number)
  const lastDay = new Date(Date.UTC(y, m + n, 0)).getUTCDate()
  return toIso(new Date(Date.UTC(y, m - 1 + n, Math.min(d, lastDay))))
}

// Days since the start of the week, 0–6.
export function weekdayIndex(iso: string, firstDayOfWeek: FirstDayOfWeek): number {
  return (toDate(iso).getUTCDay() - firstDayOfWeek + 7) % 7
}

export function monthGrid(year: number, month: number, firstDayOfWeek: FirstDayOfWeek): string[] {
  const first = toIso(new Date(Date.UTC(year, month - 1, 1)))
  const start = addDays(first, -weekdayIndex(first, firstDayOfWeek))
  return Array.from({ length: 42 }, (_, i) => addDays(start, i))
}

// ISO strings sort chronologically, so plain string comparison clamps them.
export function clamp(iso: string, min?: string, max?: string): string {
  if (min !== undefined && iso < min) return min
  if (max !== undefined && iso > max) return max
  return iso
}

export function localToday(): string {
  const now = new Date()
  return toIso(new Date(Date.UTC(now.getFullYear(), now.getMonth(), now.getDate())))
}
