import { Anchor } from '@mmerp/ui'
import { useQueryClient } from '@tanstack/react-query'
import { createContext, lazy, Suspense, useCallback, useContext, useMemo, type ComponentType, type LazyExoticComponent, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { documentKeys } from '@/shared/document/keys'
import { ContentSkeleton } from '@/shared/ui/states'
import { useAreasEntitled } from './access'
import type { AreaManifest, RecordTypeEntry } from './index'

// link is the record's page in the first area the user may enter that declares the type, or null.
type Entry = RecordTypeEntry & { link: ((id: number) => string) | null }

const RecordTypes = createContext<Record<string, Entry>>({})

// RecordTypesProvider gathers every shown area's record types, so core can open and
// refresh an area's records knowing only their doc_type. Previews and refreshes come from the
// first area declaring a type, whether the user may enter it or not.
export function RecordTypesProvider({ areas, children }: { areas: AreaManifest[]; children: ReactNode }) {
  const entitled = useAreasEntitled(areas)
  const key = entitled.join()
  const table = useMemo(() => {
    const out: Record<string, Entry> = {}
    areas.forEach((a, i) => {
      for (const [type, e] of Object.entries(a.recordTypes ?? {})) {
        const link = entitled[i] ? (id: number) => `${a.basePath.replace(/\/$/, '')}/${e.path(id)}` : null
        if (!out[type]) out[type] = { ...e, link }
        else if (!out[type].link) out[type] = { ...out[type], link }
      }
    })
    return out
    // entitled is a new array each render; key is its values.
  }, [areas, key])
  return <RecordTypes value={table}>{children}</RecordTypes>
}

// useRecordLink is the detail page of a record, or null when the user may enter no area declaring it.
export function useRecordLink(docType: string, id: number): string | null {
  return useRecordLinks()(docType, id)
}

// useRecordLinks is useRecordLink for a list of records of any type.
export function useRecordLinks(): (docType: string, id: number) => string | null {
  const table = useContext(RecordTypes)
  return useCallback(
    (docType: string, id: number) => {
      return table[docType]?.link?.(id) ?? null
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
      <Anchor render={<Link to={link} />}>
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
