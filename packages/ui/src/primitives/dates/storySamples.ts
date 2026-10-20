import { addDays, type CalendarLocale } from './calendarGrid.ts'

// Samples for the stories: the app passes its own locale, parse and format.
export const viLocale: CalendarLocale = {
  weekdays: ['CN', 'T2', 'T3', 'T4', 'T5', 'T6', 'T7'],
  months: Array.from({ length: 12 }, (_, i) => `Tháng ${i + 1}`),
  firstDayOfWeek: 1,
}

export const monthLabels = { previousMonth: 'Tháng trước', nextMonth: 'Tháng sau' }
export const yearLabels = { previousYear: 'Năm trước', nextYear: 'Năm sau' }

// dd/mm/yyyy; a day that does not exist (31/02) fails the round trip and parses to null.
export function parseDate(text: string): string | null {
  const match = /^(\d{1,2})\/(\d{1,2})\/(\d{4})$/.exec(text)
  if (!match) return null
  const [, d = '', m = '', y = ''] = match
  const iso = `${y}-${m.padStart(2, '0')}-${d.padStart(2, '0')}`
  return addDays(iso, 0) === iso ? iso : null
}

export function formatDate(iso: string): string {
  const [y, m, d] = iso.split('-')
  return `${d}/${m}/${y}`
}

export function formatMonth(month: string): string {
  const [y, m] = month.split('-')
  return `${m}/${y}`
}
