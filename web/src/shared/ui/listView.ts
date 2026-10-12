import type { ListSettings } from '@/shared/api/core'
import type { Column } from './DataTable'

export const listViewKeys = { one: (list: string) => ['core', 'me', 'lists', list] as const }

// Page and the side panel's record are where the user is, not what they look at; a view leaves them out.
const transient = ['page', 'peek']

// viewQuery is the part of a list's URL a saved view keeps, in a stable order so equal views compare equal.
export function viewQuery(search: URLSearchParams): string {
  const out = new URLSearchParams([...search].filter(([k]) => !transient.includes(k)))
  out.sort()
  return out.toString()
}

// shownColumns applies the user's choices to the designed columns: hidden ones go, widths apply.
// The first column names the row and always shows; choices for unknown keys are ignored.
export function shownColumns<T>(columns: Column<T>[], settings: ListSettings | undefined): Column<T>[] {
  const of = new Map((settings?.columns ?? []).map((c) => [c.key, c]))
  return columns.filter((c, i) => i === 0 || !of.get(c.key)?.hidden).map((c) => ({ ...c, width: of.get(c.key)?.width ?? c.width }))
}

// withColumn returns settings with one column's choice changed, keeping the rest.
export function withColumn(settings: ListSettings, key: string, patch: { hidden?: boolean; width?: number | null }): ListSettings {
  const columns = settings.columns.some((c) => c.key === key) ? settings.columns : [...settings.columns, { key, hidden: false, width: null }]
  return { ...settings, columns: columns.map((c) => (c.key === key ? { ...c, ...patch } : c)) }
}
