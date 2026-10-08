import type { UseQueryResult } from '@tanstack/react-query'
import type { TFunction } from 'i18next'
import type { ReactNode } from 'react'
import { ApiError } from '@/shared/api/hrm'
import { errorText } from '@/shared/i18n'
import { ErrorState, NotFoundPage, PageSkeleton } from '@/shared/ui/states'

type Loaded<T> = { ok: true; data: T } | { ok: false; fallback: ReactNode }

// loaded gives a record page its record, or what to show instead: a skeleton while loading,
// not found for a 404 (also the answer for a record the user may not view), or the error.
export function loaded<T>(q: UseQueryResult<T>, t: TFunction, back: { to: string; label: string }): Loaded<T> {
  if (q.isPending) return { ok: false, fallback: <PageSkeleton /> }
  if (q.error instanceof ApiError && q.error.status === 404) return { ok: false, fallback: <NotFoundPage back={back} /> }
  if (q.isError) return { ok: false, fallback: <ErrorState message={errorText(t, q.error)} onRetry={() => void q.refetch()} /> }
  return { ok: true, data: q.data }
}
