import { Anchor, Button, Checkbox, Select, Text } from '@mantine/core'
import { IconRefresh } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router'
import { api, unwrap } from '@/shared/api/core'
import { can, useMe } from '@/shared/auth/me'
import { finished, jobError, jobKeys, JobOutcome, type Job } from '@/shared/jobs'
import { errorText, formatDateTime, formatNumber } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { FormModal, ListPage } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { StatusBadge, type StatusTone } from '@/shared/ui/StatusBadge'
import { icon } from '@/shared/ui/theme'

const tone: Record<Job['state'], StatusTone> = { queued: 'neutral', running: 'info', retrying: 'warning', completed: 'positive', failed: 'negative' }

// The user's own background jobs (imports, exports) of the last 7 days, newest first; an
// administrator may list the system jobs instead, and only failing ones.
// Its strings are in meta, like the inbox: it lives at the root, outside the admin screens.
export default function JobsPage() {
  const { t } = useTranslation()
  const me = useMe()
  const [failed, setFailed] = useState<Job | null>(null)
  const monitor = can(me, 'core.job.monitor')
  const [search, setSearch] = useSearchParams()
  const system = monitor && search.get('scope') === 'system'
  const onlyFailed = monitor && search.get('failed') === '1'
  const filter = (key: string, value: string | null) => {
    const next = new URLSearchParams(search)
    if (value) next.set(key, value)
    else next.delete(key)
    setSearch(next, { replace: true })
  }
  const query = { system, failed: onlyFailed }
  const jobs = useQuery({
    queryKey: [...jobKeys.list(), query],
    queryFn: () => unwrap(api.GET('/jobs', { params: { query } })),
    // While something runs the list keeps itself current; "Refresh" is there for the impatient.
    refetchInterval: (q) => (q.state.data?.some((j) => !finished(j)) ? 10_000 : false),
  })

  const result = (j: Job) => {
    if (j.state === 'completed' && j.file_id)
      return (
        <Anchor href={`/api/files/${j.file_id}`} size="sm">
          {t('core.jobs.download')}
        </Anchor>
      )
    if (j.state === 'completed') return t('core.jobs.imported', { rows: formatNumber(j.rows ?? 0) })
    if (j.state === 'retrying')
      return (
        <Text size="sm" c="dimmed">
          {jobError(t, j)}
        </Text>
      )
    if (j.state !== 'failed') return null
    // The reason can be long; it opens in a dialog so the table stays readable.
    return (
      <Anchor component="button" type="button" size="sm" onClick={() => setFailed(j)}>
        {j.row_errors.length > 0 ? t('core.jobs.row_errors', { count: j.row_errors.length }) : t('core.jobs.reason')}
      </Anchor>
    )
  }

  const columns: Column<Job>[] = [
    { key: 'kind', header: t('core.jobs.kind'), role: 'title', render: (j) => t(`core.jobs.kind.${j.kind}`, { defaultValue: j.kind }) },
    { key: 'target', header: t('core.jobs.target'), role: 'meta', render: (j) => (j.target ? t(`${j.target}.name`, { defaultValue: j.target }) : '') },
    { key: 'created_at', header: t('core.jobs.created_at'), role: 'meta', render: (j) => formatDateTime(j.created_at, me.timezone) },
    { key: 'state', header: t('core.jobs.state'), role: 'status', render: (j) => <StatusBadge tone={tone[j.state]}>{t(`core.jobs.state.${j.state}`)}</StatusBadge> },
    ...(system ? [{ key: 'attempts', header: t('core.jobs.attempts'), role: 'meta' as const, numeric: true, render: (j: Job) => formatNumber(j.attempts) }] : []),
    { key: 'result', header: t('core.jobs.result'), role: 'meta', render: result },
  ]

  let body
  if (jobs.isPending) body = <ContentSkeleton />
  else if (jobs.isError) body = <ErrorState message={errorText(t, jobs.error)} onRetry={() => void jobs.refetch()} />
  else if (jobs.data.length === 0) body = <EmptyState title={t('core.jobs.empty')} description={t('core.jobs.empty_description')} />
  else body = <DataTable label={t('core.jobs.title')} columns={columns} rows={jobs.data} rowKey={(j) => j.id} />

  return (
    <ListPage
      title={t('core.jobs.title')}
      description={t('core.jobs.description')}
      filters={
        monitor && (
          <>
            <Select
              label={t('core.jobs.scope')}
              data={[
                { value: 'mine', label: t('core.jobs.scope.mine') },
                { value: 'system', label: t('core.jobs.scope.system') },
              ]}
              value={system ? 'system' : 'mine'}
              onChange={(v) => filter('scope', v === 'system' ? 'system' : null)}
              allowDeselect={false}
            />
            <Checkbox label={t('core.jobs.only_failed')} checked={onlyFailed} onChange={(e) => filter('failed', e.currentTarget.checked ? '1' : null)} />
          </>
        )
      }
      action={
        <Button leftSection={<IconRefresh {...icon.button} />} loading={jobs.isFetching && !jobs.isPending} onClick={() => void jobs.refetch()}>
          {t('core.jobs.refresh')}
        </Button>
      }
    >
      {body}
      {failed && (
        <FormModal opened title={t('core.jobs.failed_title')} onClose={() => setFailed(null)}>
          <JobOutcome job={failed} />
        </FormModal>
      )}
    </ListPage>
  )
}
