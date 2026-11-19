import { Button } from '@mmerp/ui'
import { IconBuildingWarehouse, IconPlus } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { api, unwrap, type TransferListItem, type TransferSort } from '@/shared/api/core'
import { DocumentStatus, DocumentStatusFilter, type DocStatus } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import type { Column } from '@/shared/ui/DataTable'
import { FilterChip, SearchInput } from '@/shared/ui/form'
import { useListView } from '@/shared/ui/ListView'
import { ListPage } from '@/shared/ui/page'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import { useTransferActions, useWarehouseOptions } from '../hooks/useStock'
import { stockKeys, transferDocType } from '../keys'

const sorts = ['-date', 'date', '-number', 'number'] as const satisfies readonly TransferSort[]
const defaults = { filters: { q: '', status: '', warehouse: '' }, sort: '-date', sorts: [...sorts], pageSize: 50 }
const base = '/stock/transfers'

// TransfersPage lists the transfers the user may see; the warehouse filter matches either side.
export function TransfersPage() {
  const { t } = useTranslation()
  const canCreate = useTransferActions().data?.includes('create') ?? false
  const warehouses = useWarehouseOptions()
  const [params, set, filterBar] = useListParams(defaults)
  const { q, status, warehouse } = params.filters as typeof defaults.filters
  const query = { q, status, warehouse, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: stockKeys.transfers.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/stock/transfers', {
          params: {
            query: {
              q: q || undefined,
              status: (status || undefined) as DocStatus | undefined,
              warehouse_id: warehouse ? Number(warehouse) : undefined,
              sort: params.sort as TransferSort,
              page: params.page,
              page_size: params.pageSize as 50,
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  })

  const columns: Column<TransferListItem>[] = [
    { key: 'number', header: t('stock.doc.number'), role: 'title', kind: 'code', sortable: true, render: (d) => d.number },
    { key: 'date', header: t('stock.transfer.date'), role: 'meta', kind: 'date', sortable: true, render: (d) => formatDate(d.date) },
    { key: 'from', header: t('stock.transfer.from_warehouse'), role: 'meta', render: (d) => `${d.from_warehouse_code} · ${d.from_warehouse_name}` },
    { key: 'to', header: t('stock.transfer.to_warehouse'), role: 'meta', render: (d) => `${d.to_warehouse_code} · ${d.to_warehouse_name}` },
    { key: 'note', header: t('stock.transfer.note'), role: 'meta', render: (d) => d.note },
    { key: 'status', header: t('stock.doc.status'), role: 'status', kind: 'status', render: (d) => <DocumentStatus docType={transferDocType} status={d.status} /> },
  ]
  const view = useListView('stock.transfers', columns)

  return (
    <ListPage
      title={t('stock.transfers.title')}
      description={t('stock.transfers.description')}
      action={
        canCreate && (
          <Button render={<Link to={`${base}/new`} />} leftSection={<IconPlus {...icon.button} />}>
            {t('stock.transfers.create')}
          </Button>
        )
      }
      onClearFilters={filterBar.active ? filterBar.clear : undefined}
      filters={
        <>
          <SearchInput label={t('stock.transfers.search')} placeholder={t('stock.transfers.search_hint')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
          <FilterChip label={t('stock.transfer.warehouse')} icon={IconBuildingWarehouse} data={warehouses} searchable {...filterBar.bind('warehouse')} />
          <DocumentStatusFilter docType={transferDocType} {...filterBar.bind('status')} />
        </>
      }
      tools={view.tools}
      list={{
        query: list,
        table: {
          label: t('stock.transfers.title'),
          columns: view.columns,
          onResize: view.onResize,
          rowKey: (d) => d.id,
          href: (d) => `${base}/${d.id}`,
          sort: params.sort,
          onSort: (sort) => set({ sort }),
        },
        empty: { title: t('stock.transfers.empty'), description: t('stock.transfers.empty_description') },
        noResults: t('stock.common.no_results'),
        paging: { page: params.page, pageSize: params.pageSize, onPage: (page) => set({ page }), onPageSize: (pageSize) => set({ pageSize }) },
      }}
    />
  )
}
