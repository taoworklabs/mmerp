import { IconReceipt } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type DeliveryListItem, type DeliverySort } from '@/shared/api/sales'
import { DocumentStatus, DocumentStatusFilter, type DocStatus } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import type { Column } from '@/shared/ui/DataTable'
import { FilterChip, SearchInput } from '@/shared/ui/form'
import { useListView } from '@/shared/ui/ListView'
import { ListPage } from '@/shared/ui/page'
import { useListParams } from '@/shared/url/listParams'
import { useDoc } from '../hooks/useSales'
import { deliveryDocType, salesKeys } from '../keys'

const sorts = ['-date', 'date', '-number', 'number'] as const satisfies readonly DeliverySort[]
const defaults = { filters: { q: '', status: '', order_id: '' }, sort: '-date', sorts: [...sorts], pageSize: 50 }
const base = '/sales/deliveries'

// DeliveriesPage lists the deliveries the user may see; opened from an order, only that order's.
// Deliveries are made from an order, so the list has no create button.
export function DeliveriesPage() {
  const { t } = useTranslation()
  const [params, set, filterBar] = useListParams(defaults)
  const { q, status, order_id } = params.filters as typeof defaults.filters
  const order = Number(order_id)
  const byOrder = Number.isInteger(order) && order > 0
  const orderNumber = useDoc('order', order, byOrder).data?.number
  const query = { q, status, order: order_id, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: salesKeys.deliveries.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/sales/deliveries', {
          params: {
            query: {
              q: q || undefined,
              status: (status || undefined) as DocStatus | undefined,
              order_id: byOrder ? order : undefined,
              sort: params.sort as DeliverySort,
              page: params.page,
              page_size: params.pageSize as 50,
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  })

  const columns: Column<DeliveryListItem>[] = [
    { key: 'number', header: t('sales.doc.number'), role: 'title', kind: 'code', sortable: true, render: (d) => d.number },
    { key: 'date', header: t('sales.delivery.date'), role: 'meta', kind: 'date', sortable: true, render: (d) => formatDate(d.date) },
    { key: 'order', header: t('sales.delivery.order'), role: 'meta', kind: 'code', render: (d) => d.order_number },
    { key: 'warehouse', header: t('sales.delivery.warehouse'), role: 'meta', render: (d) => d.warehouse_name },
    { key: 'status', header: t('sales.doc.status'), role: 'status', kind: 'status', render: (d) => <DocumentStatus docType={deliveryDocType} status={d.status} /> },
  ]
  const view = useListView('sales.deliveries', columns)

  return (
    <ListPage
      title={t('sales.deliveries.title')}
      description={t('sales.deliveries.description')}
      onClearFilters={filterBar.active ? filterBar.clear : undefined}
      filters={
        <>
          <SearchInput label={t('sales.deliveries.search')} placeholder={t('sales.deliveries.search_hint')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
          <DocumentStatusFilter docType={deliveryDocType} {...filterBar.bind('status')} />
          {/* Only set from an order's page: the chip shows it and takes it off. */}
          {byOrder && (
            <FilterChip label={t('sales.delivery.order')} icon={IconReceipt} data={[{ value: order_id, label: orderNumber ?? order_id }]} {...filterBar.bind('order_id')} />
          )}
        </>
      }
      tools={view.tools}
      list={{
        query: list,
        table: {
          label: t('sales.deliveries.title'),
          columns: view.columns,
          onResize: view.onResize,
          rowKey: (d) => d.id,
          href: (d) => `${base}/${d.id}`,
          sort: params.sort,
          onSort: (sort) => set({ sort }),
        },
        empty: { title: t('sales.deliveries.empty'), description: t('sales.deliveries.empty_description') },
        noResults: t('sales.common.no_results'),
        paging: { page: params.page, pageSize: params.pageSize, onPage: (page) => set({ page }), onPageSize: (pageSize) => set({ pageSize }) },
      }}
    />
  )
}
