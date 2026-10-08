import { Stack, Text, Timeline } from '@mantine/core'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type HistoryEntry } from '@/shared/api/core'
import { useMe } from '@/shared/auth/me'
import { errorText, formatDate, formatDateTime } from '@/shared/i18n'
import { ContentSkeleton, ErrorState } from '@/shared/ui/states'
import { useStatusLabel } from './DocumentStatus'
import { documentKeys } from './keys'

type Change = { old: unknown; new: unknown; sensitive?: boolean }

// runs merges back-to-back entries of the same person doing the same thing without changing a
// field (e.g. viewing the pay lines again), so they read as one line with a count.
export function runs(entries: HistoryEntry[]): { entry: HistoryEntry; count: number }[] {
  const out: { entry: HistoryEntry; count: number }[] = []
  for (const e of entries) {
    const last = out[out.length - 1]
    const same = last && !e.data?.changes && last.entry.action === e.action && last.entry.actor_name === e.actor_name && JSON.stringify(last.entry.data) === JSON.stringify(e.data)
    if (same) last.count++
    else out.push({ entry: e, count: 1 })
  }
  return out
}

// DocumentHistory is the timeline of a document from the audit log; sensitive values stay masked.
export function DocumentHistory({ docType, id }: { docType: string; id: number }) {
  const { t } = useTranslation()
  const me = useMe()
  const status = useStatusLabel()
  const q = useQuery({
    queryKey: documentKeys.history(docType, id),
    queryFn: () => unwrap(api.GET('/documents/{type}/{id}/history', { params: { path: { type: docType, id } } })),
  })
  if (q.isPending) return <ContentSkeleton />
  if (q.isError) return <ErrorState message={errorText(t, q.error)} onRetry={() => void q.refetch()} />

  const what = (e: HistoryEntry) => {
    const d = e.data ?? {}
    if (e.action === 'record.transitioned') return t('shared.history.transitioned', { to: status(docType, String(d.to)) })
    return t(`shared.history.${e.action}`, { ...d, defaultValue: t(`${e.action}`, { defaultValue: e.action }) })
  }
  const value = (v: unknown, sensitive?: boolean) => {
    if (sensitive) return '••••••'
    if (v === null || v === undefined || v === '' || v === 0) return '—'
    return typeof v === 'string' && /^\d{4}-\d{2}-\d{2}$/.test(v) ? formatDate(v) : String(v)
  }

  return (
    <Timeline bulletSize={12} lineWidth={1} active={-1}>
      {runs([...(q.data ?? [])].reverse()).map(({ entry: e, count }) => {
        const changes = Object.entries((e.data?.changes ?? {}) as Record<string, Change>)
        return (
          <Timeline.Item key={e.id} title={<Text size="sm" fw={500}>{what(e)}</Text>}>
            <Stack gap={2}>
              <Text size="xs" c="dimmed">
                {[e.actor_name, formatDateTime(e.at, me.timezone), count > 1 && t('shared.history.times', { count })].filter(Boolean).join(' · ')}
              </Text>
              {changes.map(([field, c]) => (
                <Text key={field} size="xs">
                  {t('shared.history.change', {
                    field: t(`${docType}.field.${field}`, { defaultValue: field }),
                    old: value(c.old, c.sensitive),
                    new: value(c.new, c.sensitive),
                  })}
                </Text>
              ))}
            </Stack>
          </Timeline.Item>
        )
      })}
    </Timeline>
  )
}
