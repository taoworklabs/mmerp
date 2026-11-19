import { IconReceipt } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type GoodsReceiptListItem, type GoodsReceiptSort } from '@/shared/api/purchase'
import { DocumentStatus, DocumentStatusFilter, type DocStatus } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import type { Column } from '@/shared/ui/DataTable'
import { FilterChip, SearchInput } from '@/shared/ui/form'
import { useListView } from '@/shared/ui/ListView'
import { ListPage } from '@/shared/ui/page'
import { useListParams } from '@/shared/url/listParams'
import { useOrder } from '../hooks/usePurchase'
import { goodsReceiptDocType, purchaseKeys } from '../keys'

const sorts = ['-date', 'date', '-number', 'number'] as const satisfies readonly GoodsReceiptSort[]
const defaults = { filters: { q: '', status: '', order_id: '' }, sort: '-date', sorts: [...sorts], pageSize: 50 }
const base = '/purchase/goods-receipts'

// GoodsReceiptsPage lists the goodsReceipts the user may see; opened from an order, only that order's.
// GoodsReceipts are made from an order, so the list has no create button.
export function GoodsReceiptsPage() {
  const { t } = useTranslation()
  const [params, set, filterBar] = useListParams(defaults)
  const { q, status, order_id } = params.filters as typeof defaults.filters
  const order = Number(order_id)
  const byOrder = Number.isInteger(order) && order > 0
  const orderNumber = useOrder(order, byOrder).data?.number
  const query = { q, status, order: order_id, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: purchaseKeys.goodsReceipts.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/purchase/goods-receipts', {
          params: {
            query: {
              q: q || undefined,
              status: (status || undefined) as DocStatus | undefined,
              order_id: byOrder ? order : undefined,
              sort: params.sort as GoodsReceiptSort,
              page: params.page,
              page_size: params.pageSize as 50,
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  })

  const columns: Column<GoodsReceiptListItem>[] = [
    { key: 'number', header: t('purchase.doc.number'), role: 'title', kind: 'code', sortable: true, render: (d) => d.number },
    { key: 'date', header: t('purchase.goods_receipt.date'), role: 'meta', kind: 'date', sortable: true, render: (d) => formatDate(d.date) },
    { key: 'order', header: t('purchase.goods_receipt.order'), role: 'meta', kind: 'code', render: (d) => d.order_number },
    { key: 'warehouse', header: t('purchase.goods_receipt.warehouse'), role: 'meta', render: (d) => d.warehouse_name },
    { key: 'status', header: t('purchase.doc.status'), role: 'status', kind: 'status', render: (d) => <DocumentStatus docType={goodsReceiptDocType} status={d.status} /> },
  ]
  const view = useListView('purchase.goods_receipts', columns)

  return (
    <ListPage
      title={t('purchase.goods_receipts.title')}
      description={t('purchase.goods_receipts.description')}
      onClearFilters={filterBar.active ? filterBar.clear : undefined}
      filters={
        <>
          <SearchInput label={t('purchase.goods_receipts.search')} placeholder={t('purchase.goods_receipts.search_hint')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
          <DocumentStatusFilter docType={goodsReceiptDocType} {...filterBar.bind('status')} />
          {/* Only set from an order's page: the chip shows it and takes it off. */}
          {byOrder && (
            <FilterChip label={t('purchase.goods_receipt.order')} icon={IconReceipt} data={[{ value: order_id, label: orderNumber ?? order_id }]} {...filterBar.bind('order_id')} />
          )}
        </>
      }
      tools={view.tools}
      list={{
        query: list,
        table: {
          label: t('purchase.goods_receipts.title'),
          columns: view.columns,
          onResize: view.onResize,
          rowKey: (d) => d.id,
          href: (d) => `${base}/${d.id}`,
          sort: params.sort,
          onSort: (sort) => set({ sort }),
        },
        empty: { title: t('purchase.goods_receipts.empty'), description: t('purchase.goods_receipts.empty_description') },
        noResults: t('purchase.common.no_results'),
        paging: { page: params.page, pageSize: params.pageSize, onPage: (page) => set({ page }), onPageSize: (pageSize) => set({ pageSize }) },
      }}
    />
  )
}
