import { Button, Select } from '@mantine/core'
import { IconCircleDot, IconPlus } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { api, unwrap, type LeaveListItem, type LeaveSort } from '@/shared/api/hrm'
import { DocumentStatus, useStatusLabel } from '@/shared/document'
import { errorText, formatDate, formatDecimal } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { ListPage } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import { useLeaveActions } from '../hooks/useLeave'
import { hrmKeys, leaveDocType } from '../keys'

const sorts = ['-start_date', 'start_date', '-number', 'number'] as const satisfies readonly LeaveSort[]
const statuses = ['draft', 'pending_approval', 'posted', 'cancelled'] as const
const defaults = { filters: { status: '' }, sort: '-start_date', sorts: [...sorts], pageSize: 50 }

export function LeavesPage() {
  const { t } = useTranslation()
  const statusLabel = useStatusLabel()
  const canCreate = useLeaveActions().data?.allowed_actions.includes('create') ?? false
  const [params, set] = useListParams(defaults)
  const { status } = params.filters as typeof defaults.filters
  const query = { status, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: hrmKeys.leaves.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/hrm/leaves', {
          params: {
            query: {
              status: (status || undefined) as (typeof statuses)[number] | undefined,
              sort: params.sort as LeaveSort,
              page: params.page,
              page_size: params.pageSize as 50,
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  })

  const columns: Column<LeaveListItem>[] = [
    { key: 'number', header: t('hrm.leave.number'), role: 'title', sortable: true, render: (l) => l.number },
    { key: 'employee', header: t('hrm.leave.employee'), role: 'meta', render: (l) => l.employee_name },
    { key: 'type', header: t('hrm.leave.type'), role: 'hidden', render: (l) => l.leave_type_name },
    { key: 'start_date', header: t('hrm.leave.period'), role: 'meta', sortable: true, render: (l) => `${formatDate(l.start_date)} – ${formatDate(l.end_date)}` },
    { key: 'days', header: t('hrm.leave.days'), role: 'hidden', numeric: true, render: (l) => formatDecimal(l.days) },
    { key: 'status', header: t('hrm.leave.status'), role: 'status', render: (l) => <DocumentStatus docType={leaveDocType} status={l.status} /> },
  ]

  let body
  if (list.isPending) body = <ContentSkeleton />
  else if (list.isError) body = <ErrorState message={errorText(t, list.error)} onRetry={() => void list.refetch()} />
  else if (list.data.items.length === 0)
    body =
      status !== '' ? (
        <EmptyState
          title={t('hrm.leaves.no_results')}
          action={
            <Button variant="default" onClick={() => set({ filters: defaults.filters })}>
              {t('hrm.common.clear_filters')}
            </Button>
          }
        />
      ) : (
        <EmptyState title={t('hrm.leaves.empty')} description={canCreate ? t('hrm.leaves.empty_description') : undefined} />
      )
  else
    body = (
      <DataTable
        label={t('hrm.leaves.title')}
        columns={columns}
        rows={list.data.items}
        rowKey={(l) => l.id}
        href={(l) => `/hrm/leaves/${l.id}`}
        sort={params.sort}
        onSort={(sort) => set({ sort })}
      />
    )

  return (
    <ListPage
      title={t('hrm.leaves.title')}
      description={t('hrm.leaves.description')}
      action={
        canCreate && (
          <Button component={Link} to="/hrm/leaves/new" leftSection={<IconPlus {...icon.button} />}>
            {t('hrm.leaves.create')}
          </Button>
        )
      }
      filters={
        <Select
          aria-label={t('hrm.leave.status')}
          leftSection={<IconCircleDot {...icon.text} />}
          w={{ base: '100%', md: 200 }}
          data={statuses.map((s) => ({ value: s, label: statusLabel(leaveDocType, s) }))}
          value={status || null}
          onChange={(v) => set({ filters: { status: v ?? '' } })}
          placeholder={t('hrm.leave.status')}
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
