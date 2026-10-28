import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type TerminationListItem, type TerminationSort } from '@/shared/api/hrm'
import { DocumentStatus, DocumentStatusFilter, type DocStatus } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import { type Column } from '@/shared/ui/DataTable'
import { OrgUnitFilter, SearchInput } from '@/shared/ui/form'
import { useListView } from '@/shared/ui/ListView'
import { ListPage } from '@/shared/ui/page'
import { SidePanel, usePeek } from '@/shared/ui/SidePanel'
import { useListParams } from '@/shared/url/listParams'
import TerminationPreview from '../components/TerminationPreview'
import { hrmKeys, terminationDocType } from '../keys'

// Every sort the API offers; `satisfies` fails tsc if the API drops one.
const sorts = ['-date', 'date', '-number', 'number'] as const satisfies readonly TerminationSort[]
const defaults = { filters: { q: '', status: '', org_unit_id: '' }, sort: '-date', sorts: [...sorts], pageSize: 50 }

// TerminationsPage: every termination in the user's scope, without amounts. New ones start from the employee.
export function TerminationsPage() {
  const { t } = useTranslation()
  const [params, set, filterBar] = useListParams(defaults)
  const peek = usePeek()
  const filters = params.filters as typeof defaults.filters
  const { q, status, org_unit_id } = filters
  const query = { ...filters, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: hrmKeys.terminations.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/hrm/terminations', {
          params: {
            query: {
              q: q || undefined,
              status: (status || undefined) as DocStatus | undefined,
              org_unit_id: org_unit_id ? Number(org_unit_id) : undefined,
              sort: params.sort as TerminationSort,
              page: params.page,
              page_size: params.pageSize as 50,
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  })

  const columns: Column<TerminationListItem>[] = [
    { key: 'number', header: t('hrm.termination.number'), role: 'title', kind: 'code', sortable: true, render: (x) => x.number },
    { key: 'employee', header: t('hrm.termination.employee'), role: 'meta', render: (x) => `${x.employee_code} · ${x.employee_name}` },
    { key: 'org_unit', header: t('hrm.termination.org_unit'), role: 'hidden', render: (x) => x.org_unit_name },
    { key: 'date', header: t('hrm.termination.date'), role: 'meta', kind: 'date', sortable: true, render: (x) => formatDate(x.termination_date) },
    { key: 'reason', header: t('hrm.termination.reason'), role: 'hidden', render: (x) => t(`hrm.termination.reason.${x.reason}`) },
    { key: 'status', header: t('hrm.termination.status'), role: 'status', kind: 'status', render: (x) => <DocumentStatus docType={terminationDocType} status={x.status} /> },
  ]
  const view = useListView('hrm.terminations', columns)
  const open = list.data?.items.find((x) => String(x.id) === peek.id)

  return (
    <ListPage
      title={t('hrm.terminations.title')}
      description={t('hrm.terminations.description')}
      onClearFilters={filterBar.active ? filterBar.clear : undefined}
      filters={
        <>
          <SearchInput label={t('hrm.terminations.search')} placeholder={t('hrm.common.search_request_hint')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
          <DocumentStatusFilter docType={terminationDocType} {...filterBar.bind('status')} />
          <OrgUnitFilter label={t('hrm.termination.org_unit')} product="hrm" permission="hrm.termination.view" {...filterBar.bind('org_unit_id')} />
        </>
      }
      tools={view.tools}
      list={{
        query: list,
        table: {
          label: t('hrm.terminations.title'),
          columns: view.columns,
          onResize: view.onResize,
          rowKey: (x) => x.id,
          href: (x) => `/hrm/terminations/${x.id}`,
          onRowClick: (x) => peek.open(x.id),
          sort: params.sort,
          onSort: (sort) => set({ sort }),
        },
        empty: { title: t('hrm.terminations.empty'), description: t('hrm.terminations.empty_description') },
        noResults: t('hrm.terminations.no_results'),
        paging: { page: params.page, pageSize: params.pageSize, onPage: (page) => set({ page }), onPageSize: (pageSize) => set({ pageSize }) },
      }}
    >
      <SidePanel title={open?.number ?? ''} href={`/hrm/terminations/${open?.id ?? ''}`} opened={!!open} onClose={peek.close}>
        {open && <TerminationPreview id={open.id} />}
      </SidePanel>
    </ListPage>
  )
}
