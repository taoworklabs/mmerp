import { Anchor } from '@mantine/core'
import { useQueryClient } from '@tanstack/react-query'
import { createContext, lazy, Suspense, useCallback, useContext, useMemo, type ComponentType, type LazyExoticComponent, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { documentKeys } from '@/shared/document/keys'
import { ContentSkeleton } from '@/shared/ui/states'
import type { AreaManifest, RecordTypeEntry } from './index'

type Entry = RecordTypeEntry & { basePath: string }

const RecordTypes = createContext<Record<string, Entry>>({})

// RecordTypesProvider gathers every shown area's record types, so core can open and
// refresh an area's records knowing only their doc_type.
export function RecordTypesProvider({ areas, children }: { areas: AreaManifest[]; children: ReactNode }) {
  const table = useMemo(
    () => Object.fromEntries(areas.flatMap((a) => Object.entries(a.recordTypes ?? {}).map(([type, e]) => [type, { ...e, basePath: a.basePath }]))),
    [areas],
  )
  return <RecordTypes value={table}>{children}</RecordTypes>
}

// useRecordLink is the detail page of a record, or null when its area is not shown.
export function useRecordLink(docType: string, id: number): string | null {
  return useRecordLinks()(docType, id)
}

// useRecordLinks is useRecordLink for a list of records of any type.
export function useRecordLinks(): (docType: string, id: number) => string | null {
  const table = useContext(RecordTypes)
  return useCallback(
    (docType: string, id: number) => {
      const e = table[docType]
      return e ? `${e.basePath.replace(/\/$/, '')}/${e.path(id)}` : null
    },
    [table],
  )
}

// useInvalidateRecord refreshes what the owning area declared for a record, plus core's
// inbox, history and approval of it. Every write to a document calls it.
export function useInvalidateRecord() {
  const qc = useQueryClient()
  const table = useContext(RecordTypes)
  return useCallback(
    async (docType: string, id: number) => {
      const keys = [...(table[docType]?.invalidate(id) ?? []), documentKeys.inbox(), documentKeys.document(docType, id)]
      await Promise.all(keys.map((queryKey) => qc.invalidateQueries({ queryKey })))
    },
    [qc, table],
  )
}

const previews = new Map<string, LazyExoticComponent<ComponentType<{ id: number }>>>()

// RecordPreview shows the area's read-only preview of a record, or a link when it has none.
export function RecordPreview({ docType, id }: { docType: string; id: number }) {
  const { t } = useTranslation()
  const e = useContext(RecordTypes)[docType]
  const link = useRecordLink(docType, id)
  if (!e?.preview) {
    return link ? (
      <Anchor component={Link} to={link}>
        {t('shared.document.open')}
      </Anchor>
    ) : null
  }
  let Preview = previews.get(docType)
  if (!Preview) {
    Preview = lazy(e.preview)
    previews.set(docType, Preview)
  }
  return (
    <Suspense fallback={<ContentSkeleton />}>
      <Preview id={id} />
    </Suspense>
  )
}
