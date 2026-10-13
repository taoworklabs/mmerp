import { Button } from '@mantine/core'
import { IconCircleDot, IconPlus } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { api, unwrap, type LeaveListItem, type LeaveSort } from '@/shared/api/hrm'
import { DocumentStatus, useStatusLabel } from '@/shared/document'
import { errorText, formatDate, formatDecimal } from '@/shared/i18n'
import { PersonName } from '@/shared/ui/avatar'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { FilterChip, MonthFilter, OrgUnitFilter, SearchInput } from '@/shared/ui/form'
import { useListView } from '@/shared/ui/ListView'
import { ListPage } from '@/shared/ui/page'
import { SidePanel, usePeek } from '@/shared/ui/SidePanel'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import LeavePreview from '../components/LeavePreview'
import { useLeaveActions } from '../hooks/useLeave'
import { hrmKeys, leaveDocType } from '../keys'
import { monthBounds } from '../month'

const sorts = ['-start_date', 'start_date', '-number', 'number'] as const satisfies readonly LeaveSort[]
const statuses = ['draft', 'pending_approval', 'posted', 'cancelled'] as const
const defaults = { filters: { q: '', status: '', org_unit: '', month: '' }, sort: '-start_date', sorts: [...sorts], pageSize: 50 }

// LeavesPage lists the leave requests the user may see; a row opens its preview beside the list.
export function LeavesPage() {
  const { t } = useTranslation()
  const statusLabel = useStatusLabel()
  const canCreate = useLeaveActions().data?.allowed_actions.includes('create') ?? false
  const [params, set, filterBar] = useListParams(defaults)
  const peek = usePeek()
  const { q, status, org_unit, month } = params.filters as typeof defaults.filters
  const query = { q, status, orgUnit: org_unit, month, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: hrmKeys.leaves.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/hrm/leaves', {
          params: {
            query: {
              q: q || undefined,
              status: (status || undefined) as (typeof statuses)[number] | undefined,
              org_unit_id: org_unit ? Number(org_unit) : undefined,
              ...monthBounds(month),
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
    { key: 'number', header: t('hrm.leave.number'), role: 'title', kind: 'code', sortable: true, render: (l) => l.number },
    { key: 'employee', header: t('hrm.leave.employee'), role: 'meta', kind: 'person', render: (l) => <PersonName name={l.employee_name} /> },
    { key: 'type', header: t('hrm.leave.type'), role: 'hidden', render: (l) => l.leave_type_name },
    { key: 'start_date', header: t('hrm.leave.period'), role: 'meta', kind: 'date', sortable: true, render: (l) => `${formatDate(l.start_date)} – ${formatDate(l.end_date)}` },
    { key: 'days', header: t('hrm.leave.days'), role: 'hidden', kind: 'number', render: (l) => formatDecimal(l.days) },
    { key: 'status', header: t('hrm.leave.status'), role: 'status', kind: 'status', render: (l) => <DocumentStatus docType={leaveDocType} status={l.status} /> },
  ]
  const view = useListView('hrm.leaves', columns)
  const open = list.data?.items.find((l) => String(l.id) === peek.id)

  let body
  if (list.isPending) body = <ContentSkeleton />
  else if (list.isError) body = <ErrorState message={errorText(t, list.error)} onRetry={() => void list.refetch()} />
  else if (list.data.items.length === 0)
    body =
      filterBar.active ? (
        <EmptyState
          title={t('hrm.leaves.no_results')}
          action={
            <Button variant="default" onClick={filterBar.clear}>
              {t('hrm.common.clear_filters')}
            </Button>
          }
        />
      ) : (
        <EmptyState title={t('hrm.leaves.empty')} description={t('hrm.leaves.empty_description')} />
      )
  else
    body = (
      <DataTable
        label={t('hrm.leaves.title')}
        columns={view.columns}
        onResize={view.onResize}
        rows={list.data.items}
        rowKey={(l) => l.id}
        href={(l) => `/hrm/leaves/${l.id}`}
        onRowClick={(l) => peek.open(l.id)}
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
      onClearFilters={filterBar.active ? filterBar.clear : undefined}
      filters={
        <>
          <SearchInput label={t('hrm.leaves.search')} placeholder={t('hrm.common.search_request_hint')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
          <FilterChip
            label={t('hrm.leave.status')}
            icon={IconCircleDot}
            data={statuses.map((s) => ({ value: s, label: statusLabel(leaveDocType, s) }))}
            {...filterBar.bind('status')}
          />
          <OrgUnitFilter
            product="hrm"
            permission="hrm.leave.view"
            label={t('hrm.common.org_unit')}
            {...filterBar.bind('org_unit')}
          />
          <MonthFilter label={t('hrm.common.month')} {...filterBar.bind('month')} />
        </>
      }
      tools={view.tools}
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
      <SidePanel title={open?.number ?? ''} href={`/hrm/leaves/${open?.id ?? ''}`} opened={!!open} onClose={peek.close}>
        {open && <LeavePreview id={open.id} />}
      </SidePanel>
    </ListPage>
  )
}
