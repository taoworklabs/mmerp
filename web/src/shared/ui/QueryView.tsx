import type { UseQueryResult } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { errorText } from '@/shared/i18n'
import { queryState } from './queryState'
import { ContentSkeleton, ErrorState } from './states'

// QueryView renders a region fed by one query: a skeleton while it loads, the error with Retry,
// `empty` when there is nothing, else the children with the data. `select` shapes the data first
// (sort, flatten pages, filter); when a filter empties a non-empty result, `noResults` shows
// instead of `empty`. `notFound` answers a 404, which some endpoints give for "not yours to see".
export function QueryView<T, U = T>({
  query,
  select,
  empty,
  noResults,
  notFound,
  children,
}: {
  query: UseQueryResult<T>
  select?: (data: T) => U
  empty?: ReactNode
  noResults?: ReactNode
  notFound?: ReactNode
  children: (data: U) => ReactNode
}) {
  const { t } = useTranslation()
  const s = queryState(query, select, { empty: empty !== undefined, noResults: noResults !== undefined, notFound: notFound !== undefined })
  switch (s.state) {
    case 'loading':
      return <ContentSkeleton />
    case 'not_found':
      return notFound
    case 'error':
      return <ErrorState message={errorText(t, query.error)} onRetry={() => void query.refetch()} />
    case 'empty':
      return empty
    case 'no_results':
      return noResults
    case 'data':
      return children(s.data)
  }
}
