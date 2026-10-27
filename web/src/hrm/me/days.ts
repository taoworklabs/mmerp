const DAY = 86_400_000
const at = (iso: string) => new Date(`${iso}T00:00:00Z`).getTime()
const iso = (ms: number) => new Date(ms).toISOString().slice(0, 10)

// shiftDate moves a business date (YYYY-MM-DD) by whole days.
export const shiftDate = (date: string, days: number) => iso(at(date) + days * DAY)

// weekOf lists the seven dates of the week holding `date`, Monday to Sunday.
export function weekOf(date: string): string[] {
  const monday = shiftDate(date, -((new Date(`${date}T00:00:00Z`).getUTCDay() + 6) % 7))
  return Array.from({ length: 7 }, (_, i) => shiftDate(monday, i))
}

// monthsOf lists the distinct months (YYYY-MM) of some dates, in order of appearance.
export const monthsOf = (dates: string[]) => [...new Set(dates.map((d) => d.slice(0, 7)))]

// monthGrid lays a month (YYYY-MM) out in whole Monday-first weeks; days outside it are null.
export function monthGrid(month: string): (string | null)[] {
  const first = `${month}-01`
  const lead = (new Date(`${first}T00:00:00Z`).getUTCDay() + 6) % 7
  const [y, m] = month.split('-').map(Number) as [number, number]
  const last = new Date(Date.UTC(y, m, 0)).getUTCDate()
  const cells: (string | null)[] = [...Array<null>(lead).fill(null), ...Array.from({ length: last }, (_, i) => shiftDate(first, i))]
  while (cells.length % 7) cells.push(null)
  return cells
}

// shiftMonth moves a month (YYYY-MM) by whole months.
export function shiftMonth(month: string, n: number): string {
  const [y, m] = month.split('-').map(Number) as [number, number]
  return new Date(Date.UTC(y, m - 1 + n, 1)).toISOString().slice(0, 7)
}
