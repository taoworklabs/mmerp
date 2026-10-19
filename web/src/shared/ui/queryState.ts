import { ApiError } from '@/shared/api/error'

type Query<T> = { isPending: boolean; isError: boolean; error: unknown; data?: T }

const isEmpty = (v: unknown) => v == null || (Array.isArray(v) && v.length === 0)

// queryState picks what a region fed by one query shows. Only the states the caller offers a
// view for are taken: without `notFound` a 404 is an error, without `empty` nothing is empty, and
// without `noResults` a filtered-out result is shown as empty.
export function queryState<T, U>(
  query: Query<T>,
  select: ((data: T) => U) | undefined,
  offers: { empty: boolean; noResults: boolean; notFound: boolean },
): { state: 'loading' | 'not_found' | 'error' | 'empty' | 'no_results' } | { state: 'data'; data: U } {
  if (query.isPending) return { state: 'loading' }
  if (offers.notFound && query.error instanceof ApiError && query.error.status === 404) return { state: 'not_found' }
  if (query.isError) return { state: 'error' }
  const data = select ? select(query.data as T) : (query.data as unknown as U)
  if (offers.empty && isEmpty(data)) return { state: offers.noResults && !isEmpty(query.data) ? 'no_results' : 'empty' }
  return { state: 'data', data }
}
