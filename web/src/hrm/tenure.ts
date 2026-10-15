// tenure is how long from one date (YYYY-MM-DD) to another, in whole years and months: from
// 2022-06-06 to 2026-10-15 is 4 years 4 months. A month counts once its day is reached.
export function tenure(from: string, to: string): { years: number; months: number } {
  const [fy, fm, fd] = from.split('-').map(Number) as [number, number, number]
  const [ty, tm, td] = to.split('-').map(Number) as [number, number, number]
  const total = Math.max(0, (ty - fy) * 12 + (tm - fm) - (td < fd ? 1 : 0))
  return { years: Math.floor(total / 12), months: total % 12 }
}
