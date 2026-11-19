import { Button } from '@mmerp/ui'
import { IconBuildingWarehouse, IconPlus } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { api, unwrap, type StocktakeListItem, type StocktakeSort } from '@/shared/api/core'
import { DocumentStatus, DocumentStatusFilter, type DocStatus } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import type { Column } from '@/shared/ui/DataTable'
import { FilterChip, SearchInput } from '@/shared/ui/form'
import { useListView } from '@/shared/ui/ListView'
import { ListPage } from '@/shared/ui/page'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import { useStocktakeActions, useWarehouseOptions } from '../hooks/useStock'
import { stockKeys, stocktakeDocType } from '../keys'

const sorts = ['-date', 'date', '-number', 'number'] as const satisfies readonly StocktakeSort[]
const defaults = { filters: { q: '', status: '', warehouse: '' }, sort: '-date', sorts: [...sorts], pageSize: 50 }
const base = '/stock/stocktakes'

// StocktakesPage lists the stocktakes the user may see.
export function StocktakesPage() {
  const { t } = useTranslation()
  const canCreate = useStocktakeActions().data?.includes('create') ?? false
  const warehouses = useWarehouseOptions()
  const [params, set, filterBar] = useListParams(defaults)
  const { q, status, warehouse } = params.filters as typeof defaults.filters
  const query = { q, status, warehouse, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: stockKeys.stocktakes.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/stock/stocktakes', {
          params: {
            query: {
              q: q || undefined,
              status: (status || undefined) as DocStatus | undefined,
              warehouse_id: warehouse ? Number(warehouse) : undefined,
              sort: params.sort as StocktakeSort,
              page: params.page,
              page_size: params.pageSize as 50,
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  })

  const columns: Column<StocktakeListItem>[] = [
    { key: 'number', header: t('stock.doc.number'), role: 'title', kind: 'code', sortable: true, render: (d) => d.number },
    { key: 'date', header: t('stock.stocktake.date'), role: 'meta', kind: 'date', sortable: true, render: (d) => formatDate(d.date) },
    { key: 'warehouse', header: t('stock.stocktake.warehouse'), role: 'meta', render: (d) => `${d.warehouse_code} · ${d.warehouse_name}` },
    { key: 'note', header: t('stock.stocktake.note'), role: 'meta', render: (d) => d.note },
    { key: 'status', header: t('stock.doc.status'), role: 'status', kind: 'status', render: (d) => <DocumentStatus docType={stocktakeDocType} status={d.status} /> },
  ]
  const view = useListView('stock.stocktakes', columns)

  return (
    <ListPage
      title={t('stock.stocktakes.title')}
      description={t('stock.stocktakes.description')}
      action={
        canCreate && (
          <Button render={<Link to={`${base}/new`} />} leftSection={<IconPlus {...icon.button} />}>
            {t('stock.stocktakes.create')}
          </Button>
        )
      }
      onClearFilters={filterBar.active ? filterBar.clear : undefined}
      filters={
        <>
          <SearchInput label={t('stock.stocktakes.search')} placeholder={t('stock.stocktakes.search_hint')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
          <FilterChip label={t('stock.stocktake.warehouse')} icon={IconBuildingWarehouse} data={warehouses} searchable {...filterBar.bind('warehouse')} />
          <DocumentStatusFilter docType={stocktakeDocType} {...filterBar.bind('status')} />
        </>
      }
      tools={view.tools}
      list={{
        query: list,
        table: {
          label: t('stock.stocktakes.title'),
          columns: view.columns,
          onResize: view.onResize,
          rowKey: (d) => d.id,
          href: (d) => `${base}/${d.id}`,
          sort: params.sort,
          onSort: (sort) => set({ sort }),
        },
        empty: { title: t('stock.stocktakes.empty'), description: t('stock.stocktakes.empty_description') },
        noResults: t('stock.common.no_results'),
        paging: { page: params.page, pageSize: params.pageSize, onPage: (page) => set({ page }), onPageSize: (pageSize) => set({ pageSize }) },
      }}
    />
  )
}
