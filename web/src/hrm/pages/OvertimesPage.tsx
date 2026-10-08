import { Button, Select } from '@mantine/core'
import { IconCircleDot, IconPlus } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { api, unwrap, type OvertimeListItem, type OvertimeSort } from '@/shared/api/hrm'
import { DocumentStatus, useStatusLabel } from '@/shared/document'
import { errorText, formatDate, formatDecimal } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { ListPage } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import { useOvertimeActions } from '../hooks/useOvertime'
import { hrmKeys, overtimeDocType } from '../keys'

const sorts = ['-date', 'date', '-number', 'number'] as const satisfies readonly OvertimeSort[]
const statuses = ['draft', 'pending_approval', 'posted', 'cancelled'] as const
const defaults = { filters: { status: '' }, sort: '-date', sorts: [...sorts], pageSize: 50 }

export function OvertimesPage() {
  const { t } = useTranslation()
  const statusLabel = useStatusLabel()
  const canCreate = useOvertimeActions().data?.allowed_actions.includes('create') ?? false
  const [params, set] = useListParams(defaults)
  const { status } = params.filters as typeof defaults.filters
  const query = { status, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: hrmKeys.overtimes.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/hrm/overtimes', {
          params: {
            query: {
              status: (status || undefined) as (typeof statuses)[number] | undefined,
              sort: params.sort as OvertimeSort,
              page: params.page,
              page_size: params.pageSize as 50,
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  })

  const columns: Column<OvertimeListItem>[] = [
    { key: 'number', header: t('hrm.overtime.number'), role: 'title', sortable: true, render: (o) => o.number },
    { key: 'employee', header: t('hrm.overtime.employee'), role: 'meta', render: (o) => o.employee_name },
    { key: 'date', header: t('hrm.overtime.date'), role: 'meta', sortable: true, render: (o) => formatDate(o.date) },
    { key: 'day_kind', header: t('hrm.overtime.day_kind'), role: 'hidden', render: (o) => t(`hrm.overtime.day_kind.${o.day_kind}`) },
    { key: 'day_hours', header: t('hrm.overtime.day_hours'), role: 'hidden', numeric: true, render: (o) => formatDecimal(o.day_hours) },
    { key: 'night_hours', header: t('hrm.overtime.night_hours'), role: 'hidden', numeric: true, render: (o) => formatDecimal(o.night_hours) },
    { key: 'status', header: t('hrm.overtime.status'), role: 'status', render: (o) => <DocumentStatus docType={overtimeDocType} status={o.status} /> },
  ]

  let body
  if (list.isPending) body = <ContentSkeleton />
  else if (list.isError) body = <ErrorState message={errorText(t, list.error)} onRetry={() => void list.refetch()} />
  else if (list.data.items.length === 0)
    body =
      status !== '' ? (
        <EmptyState
          title={t('hrm.overtimes.no_results')}
          action={
            <Button variant="default" onClick={() => set({ filters: defaults.filters })}>
              {t('hrm.common.clear_filters')}
            </Button>
          }
        />
      ) : (
        <EmptyState title={t('hrm.overtimes.empty')} description={canCreate ? t('hrm.overtimes.empty_description') : undefined} />
      )
  else
    body = (
      <DataTable
        label={t('hrm.overtimes.title')}
        columns={columns}
        rows={list.data.items}
        rowKey={(l) => l.id}
        href={(l) => `/hrm/overtimes/${l.id}`}
        sort={params.sort}
        onSort={(sort) => set({ sort })}
      />
    )

  return (
    <ListPage
      title={t('hrm.overtimes.title')}
      description={t('hrm.overtimes.description')}
      action={
        canCreate && (
          <Button component={Link} to="/hrm/overtimes/new" leftSection={<IconPlus {...icon.button} />}>
            {t('hrm.overtimes.create')}
          </Button>
        )
      }
      filters={
        <Select
          aria-label={t('hrm.overtime.status')}
          leftSection={<IconCircleDot {...icon.text} />}
          w={{ base: '100%', md: 200 }}
          data={statuses.map((s) => ({ value: s, label: statusLabel(overtimeDocType, s) }))}
          value={status || null}
          onChange={(v) => set({ filters: { status: v ?? '' } })}
          placeholder={t('hrm.overtime.status')}
          clearable
        />
      }
      paging={
        list.data && {
          page: params.page,
          pageSize: params.pageSize,
          total: list.data.total,
          onPage: (page) => set({ page }),
          onPageSize: (pageSize) => set({ pageSize }),
        }
      }
    >
      {body}
    </ListPage>
  )
}
