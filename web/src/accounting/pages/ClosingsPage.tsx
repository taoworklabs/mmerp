import { Button } from '@mmerp/ui'
import { IconBuildingSkyscraper, IconPlus } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { api, unwrap, type ClosingListItem, type ClosingSort } from '@/shared/api/accounting'
import { DocumentStatus, DocumentStatusFilter, type DocStatus } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import type { Column } from '@/shared/ui/DataTable'
import { FilterChip, SearchInput, useLegalEntities } from '@/shared/ui/form'
import { useListView } from '@/shared/ui/ListView'
import { ListPage } from '@/shared/ui/page'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import { useClosingActions } from '../hooks/useAccounting'
import { accountingKeys, closingDocType } from '../keys'

const sorts = ['-date', 'date', '-number', 'number'] as const satisfies readonly ClosingSort[]
const defaults = { filters: { q: '', status: '', legal_entity: '' }, sort: '-date', sorts: [...sorts], pageSize: 50 }
const base = '/accounting/closings'

// ClosingsPage lists the closing entries the user may see.
export function ClosingsPage() {
  const { t } = useTranslation()
  const canCreate = useClosingActions().data?.includes('create') ?? false
  const entities = useLegalEntities({ product: 'accounting', permission: 'accounting.view' }).options
  const [params, set, filterBar] = useListParams(defaults)
  const { q, status, legal_entity } = params.filters as typeof defaults.filters
  const query = { q, status, legal_entity, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: accountingKeys.closings.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/accounting/closings', {
          params: {
            query: {
              q: q || undefined,
              status: (status || undefined) as DocStatus | undefined,
              legal_entity_id: Number(legal_entity) || undefined,
              sort: params.sort as ClosingSort,
              page: params.page,
              page_size: params.pageSize as 50,
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  })

  const columns: Column<ClosingListItem>[] = [
    { key: 'number', header: t('accounting.entry.number'), role: 'title', kind: 'code', sortable: true, render: (d) => d.number },
    { key: 'date', header: t('accounting.closing.date'), role: 'meta', kind: 'date', sortable: true, render: (d) => formatDate(d.date) },
    { key: 'description', header: t('accounting.entry.description'), role: 'meta', render: (d) => d.description },
    { key: 'legal_entity', header: t('accounting.common.legal_entity'), role: 'hidden', render: (d) => entities.find((o) => o.value === String(d.legal_entity_id))?.label ?? '' },
    { key: 'status', header: t('accounting.entry.status'), role: 'status', kind: 'status', render: (d) => <DocumentStatus docType={closingDocType} status={d.status} /> },
  ]
  const view = useListView('accounting.closings', columns)

  return (
    <ListPage
      title={t('accounting.closings.title')}
      description={t('accounting.closings.description')}
      action={
        canCreate && (
          <Button render={<Link to={`${base}/new`} />} leftSection={<IconPlus {...icon.button} />}>
            {t('accounting.closings.create')}
          </Button>
        )
      }
      onClearFilters={filterBar.active ? filterBar.clear : undefined}
      filters={
        <>
          <SearchInput label={t('accounting.closings.search')} placeholder={t('accounting.entries.search_hint')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
          <DocumentStatusFilter docType={closingDocType} {...filterBar.bind('status')} />
          {entities.length > 1 && <FilterChip label={t('accounting.common.legal_entity')} icon={IconBuildingSkyscraper} data={entities} {...filterBar.bind('legal_entity')} />}
        </>
      }
      tools={view.tools}
      list={{
        query: list,
        table: {
          label: t('accounting.closings.title'),
          columns: view.columns,
          onResize: view.onResize,
          rowKey: (d) => d.id,
          href: (d) => `${base}/${d.id}`,
          sort: params.sort,
          onSort: (sort) => set({ sort }),
        },
        empty: { title: t('accounting.closings.empty'), description: t('accounting.closings.empty_description') },
        noResults: t('accounting.common.no_results'),
        paging: { page: params.page, pageSize: params.pageSize, onPage: (page) => set({ page }), onPageSize: (pageSize) => set({ pageSize }) },
      }}
    />
  )
}
