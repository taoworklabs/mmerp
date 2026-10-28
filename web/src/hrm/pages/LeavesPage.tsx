import { Button } from '@mmerp/ui'
import { IconPlus } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { api, unwrap, type LeaveListItem, type LeaveSort } from '@/shared/api/hrm'
import { DocumentStatus, DocumentStatusFilter, type DocStatus } from '@/shared/document'
import { formatDate, formatDecimal } from '@/shared/i18n'
import { PersonName } from '@/shared/ui/avatar'
import { type Column } from '@/shared/ui/DataTable'
import { MonthFilter, OrgUnitFilter, SearchInput } from '@/shared/ui/form'
import { useListView } from '@/shared/ui/ListView'
import { ListPage } from '@/shared/ui/page'
import { SidePanel, usePeek } from '@/shared/ui/SidePanel'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import LeavePreview from '../components/LeavePreview'
import { useLeaveActions } from '../hooks/useLeave'
import { hrmKeys, leaveDocType } from '../keys'
import { monthBounds } from '../month'

const sorts = ['-start_date', 'start_date', '-number', 'number'] as const satisfies readonly LeaveSort[]
const defaults = { filters: { q: '', status: '', org_unit: '', month: '' }, sort: '-start_date', sorts: [...sorts], pageSize: 50 }

// LeavesPage lists the leave requests the user may see; a row opens its preview beside the list.
// team: in the personal space, only the requests of the user's reports.
export function LeavesPage({ team = false }: { team?: boolean }) {
  const { t } = useTranslation()
  const actions = useLeaveActions().data
  const canCreate = !team && (actions?.includes('create') ?? false)
  const texts = team
    ? { title: t('hrm.me.team.leaves.title'), description: t('hrm.me.team.leaves.description'), empty: t('hrm.me.team.leaves.empty'), emptyDescription: t('hrm.me.team.leaves.empty_description') }
    : { title: t('hrm.leaves.title'), description: t('hrm.leaves.description'), empty: t('hrm.leaves.empty'), emptyDescription: t('hrm.leaves.empty_description') }
  const [params, set, filterBar] = useListParams(defaults)
  const peek = usePeek()
  const { q, status, org_unit, month } = params.filters as typeof defaults.filters
  const query = { team, q, status, orgUnit: org_unit, month, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: hrmKeys.leaves.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/hrm/leaves', {
          params: {
            query: {
              reports: team || undefined,
              q: q || undefined,
              status: status ? [status as DocStatus] : undefined,
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
  const view = useListView(team ? 'hrm.me.team.leaves' : 'hrm.leaves', columns)
  const href = (id: number) => (team ? `/me/requests/leave/${id}` : `/hrm/leaves/${id}`)
  const open = list.data?.items.find((l) => String(l.id) === peek.id)

  return (
    <ListPage
      title={texts.title}
      description={texts.description}
      action={
        canCreate && (
          <Button render={<Link to="/hrm/leaves/new" />} leftSection={<IconPlus {...icon.button} />}>
            {t('hrm.leaves.create')}
          </Button>
        )
      }
      onClearFilters={filterBar.active ? filterBar.clear : undefined}
      filters={
        <>
          <SearchInput label={t('hrm.leaves.search')} placeholder={t('hrm.common.search_request_hint')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
          <DocumentStatusFilter docType={leaveDocType} {...filterBar.bind('status')} />
          {!team && <OrgUnitFilter product="hrm" permission="hrm.leave.view" label={t('hrm.common.org_unit')} {...filterBar.bind('org_unit')} />}
          <MonthFilter label={t('hrm.common.month')} {...filterBar.bind('month')} />
        </>
      }
      tools={view.tools}
      list={{
        query: list,
        table: {
          label: texts.title,
          columns: view.columns,
          onResize: view.onResize,
          rowKey: (l) => l.id,
          href: (l) => href(l.id),
          onRowClick: (l) => peek.open(l.id),
          sort: params.sort,
          onSort: (sort) => set({ sort }),
        },
        empty: { title: texts.empty, description: texts.emptyDescription },
        noResults: t('hrm.leaves.no_results'),
        paging: { page: params.page, pageSize: params.pageSize, onPage: (page) => set({ page }), onPageSize: (pageSize) => set({ pageSize }) },
      }}
    >
      <SidePanel title={open?.number ?? ''} href={open ? href(open.id) : ''} opened={!!open} onClose={peek.close}>
        {open && <LeavePreview id={open.id} />}
      </SidePanel>
    </ListPage>
  )
}
