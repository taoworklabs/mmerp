import { Button } from '@mmerp/ui'
import { IconPlus } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { api, unwrap, type AdjustmentListItem, type AdjustmentSort } from '@/shared/api/core'
import { DocumentStatus, DocumentStatusFilter, type DocStatus } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import type { Column } from '@/shared/ui/DataTable'
import { SearchInput } from '@/shared/ui/form'
import { useListView } from '@/shared/ui/ListView'
import { ListPage } from '@/shared/ui/page'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import { useAdjustmentActions } from '../hooks/useStock'
import { adjustmentDocType, stockKeys } from '../keys'

const sorts = ['-date', 'date', '-number', 'number'] as const satisfies readonly AdjustmentSort[]
const defaults = { filters: { q: '', status: '' }, sort: '-date', sorts: [...sorts], pageSize: 50 }
const base = '/stock/adjustments'

// AdjustmentsPage lists the stock adjustments the user may see.
export function AdjustmentsPage() {
  const { t } = useTranslation()
  const canCreate = useAdjustmentActions().data?.includes('create') ?? false
  const [params, set, filterBar] = useListParams(defaults)
  const { q, status } = params.filters as typeof defaults.filters
  const query = { q, status, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: stockKeys.adjustments.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/stock/adjustments', {
          params: {
            query: {
              q: q || undefined,
              status: (status || undefined) as DocStatus | undefined,
              sort: params.sort as AdjustmentSort,
              page: params.page,
              page_size: params.pageSize as 50,
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  })

  const columns: Column<AdjustmentListItem>[] = [
    { key: 'number', header: t('stock.doc.number'), role: 'title', kind: 'code', sortable: true, render: (d) => d.number },
    { key: 'date', header: t('stock.adjustment.date'), role: 'meta', kind: 'date', sortable: true, render: (d) => formatDate(d.date) },
    { key: 'warehouse', header: t('stock.adjustment.warehouse'), role: 'meta', render: (d) => `${d.warehouse_code} · ${d.warehouse_name}` },
    { key: 'reason', header: t('stock.adjustment.reason'), role: 'meta', render: (d) => d.reason },
    { key: 'status', header: t('stock.doc.status'), role: 'status', kind: 'status', render: (d) => <DocumentStatus docType={adjustmentDocType} status={d.status} /> },
  ]
  const view = useListView('stock.adjustments', columns)

  return (
    <ListPage
      title={t('stock.adjustments.title')}
      description={t('stock.adjustments.description')}
      action={
        canCreate && (
          <Button render={<Link to={`${base}/new`} />} leftSection={<IconPlus {...icon.button} />}>
            {t('stock.adjustments.create')}
          </Button>
        )
      }
      onClearFilters={filterBar.active ? filterBar.clear : undefined}
      filters={
        <>
          <SearchInput label={t('stock.adjustments.search')} placeholder={t('stock.adjustments.search_hint')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
          <DocumentStatusFilter docType={adjustmentDocType} {...filterBar.bind('status')} />
        </>
      }
      tools={view.tools}
      list={{
        query: list,
        table: {
          label: t('stock.adjustments.title'),
          columns: view.columns,
          onResize: view.onResize,
          rowKey: (d) => d.id,
          href: (d) => `${base}/${d.id}`,
          sort: params.sort,
          onSort: (sort) => set({ sort }),
        },
        empty: { title: t('stock.adjustments.empty'), description: t('stock.adjustments.empty_description') },
        noResults: t('stock.common.no_results'),
        paging: { page: params.page, pageSize: params.pageSize, onPage: (page) => set({ page }), onPageSize: (pageSize) => set({ pageSize }) },
      }}
    />
  )
}
