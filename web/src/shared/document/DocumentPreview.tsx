import { Stack } from '@mantine/core'
import type { UseQueryResult } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { errorText } from '@/shared/i18n'
import { FieldList } from '@/shared/ui/FieldList'
import { ContentSkeleton, ErrorState } from '@/shared/ui/states'
import { DocumentStatus, type DocStatus } from './DocumentStatus'

// A label–value row, or a falsy value for one the document leaves out.
export type PreviewRow = [string, string] | false | null | undefined | ''

export const previewRows = (rows: PreviewRow[]): [string, string][] => rows.filter((r): r is [string, string] => !!r)

type Props<T> = {
  docType: string
  query: UseQueryResult<T>
  rows: (doc: T) => PreviewRow[]
  // Shown between the status and the rows (a warning).
  notice?: (doc: T) => ReactNode
}

// DocumentPreview is the read-only view of a document in the approval inbox and side panels:
// its status, then its rows. An area's preview loads outside its screens, so its strings live
// in the area's meta translations.
export function DocumentPreview<T extends { status: DocStatus }>({ docType, query, rows, notice }: Props<T>) {
  const { t } = useTranslation()
  if (query.isPending) return <ContentSkeleton />
  if (query.isError) return <ErrorState message={errorText(t, query.error)} onRetry={() => void query.refetch()} />
  const d = query.data
  return (
    <Stack gap="sm">
      <DocumentStatus docType={docType} status={d.status} />
      {notice?.(d)}
      <FieldList rows={previewRows(rows(d))} />
    </Stack>
  )
}
