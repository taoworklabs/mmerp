import { useCallback, useMemo } from 'react'
import { useSearchParams } from 'react-router'

// ListParams is what a list screen keeps on the URL: filters, sort, page and page size.
export type ListParams = { filters: Record<string, string>; sort: string; page: number; pageSize: number }

// ListDefaults names every filter with its default and every sort the list offers;
// only these are read from the URL.
export type ListDefaults = { filters: Record<string, string>; sort: string; sorts: string[]; pageSize: number }

export const pageSizes = [20, 50, 100]

// parseList reads the URL, falling back to defaults for missing or invalid values.
export function parseList(search: URLSearchParams, defaults: ListDefaults): ListParams {
  const filters = Object.fromEntries(Object.entries(defaults.filters).map(([k, v]) => [k, search.get(k) ?? v]))
  const sort = search.get('sort') ?? ''
  const page = Number(search.get('page'))
  const pageSize = Number(search.get('page_size'))
  return {
    filters,
    sort: defaults.sorts.includes(sort) ? sort : defaults.sort,
    page: Number.isInteger(page) && page > 1 ? page : 1,
    pageSize: pageSizes.includes(pageSize) ? pageSize : defaults.pageSize,
  }
}

// serializeList writes only what differs from the defaults, so a bare URL is the default view.
export function serializeList(p: ListParams, defaults: ListDefaults): URLSearchParams {
  const out = new URLSearchParams()
  for (const [k, v] of Object.entries(p.filters)) if (v !== defaults.filters[k]) out.set(k, v)
  if (p.sort !== defaults.sort) out.set('sort', p.sort)
  if (p.page !== 1) out.set('page', String(p.page))
  if (p.pageSize !== defaults.pageSize) out.set('page_size', String(p.pageSize))
  out.sort()
  return out
}

// useListParams keeps list state on the URL. Each change is a history entry, so Back
// returns to the previous filter, sort or page; changing anything but the page goes to page 1.
export function useListParams(defaults: ListDefaults) {
  const [search, setSearch] = useSearchParams()
  // Defaults are usually a literal at the call site; keying on their content keeps them stable.
  const key = JSON.stringify(defaults)
  const stable = useMemo<ListDefaults>(() => JSON.parse(key), [key])
  const params = useMemo(() => parseList(search, stable), [search, stable])
  const set = useCallback(
    (patch: Partial<ListParams>) => {
      const next = { ...params, ...patch, filters: { ...params.filters, ...patch.filters } }
      if (patch.page === undefined) next.page = 1
      setSearch(serializeList(next, stable))
    },
    [params, setSearch, stable],
  )
  return [params, set] as const
}
