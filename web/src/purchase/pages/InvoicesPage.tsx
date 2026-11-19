import { IconCash, IconReceipt } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type InvoiceListItem, type InvoiceSort, type PaymentState } from '@/shared/api/purchase'
import { DocumentStatus, DocumentStatusFilter, type DocStatus } from '@/shared/document'
import { formatDate, formatNumber } from '@/shared/i18n'
import type { Column } from '@/shared/ui/DataTable'
import { FilterChip, SearchInput } from '@/shared/ui/form'
import { useListView } from '@/shared/ui/ListView'
import { ListPage } from '@/shared/ui/page'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { useListParams } from '@/shared/url/listParams'
import { useOrder } from '../hooks/usePurchase'
import { paymentStateBadge } from '../invoiceLines'
import { invoiceDocType, purchaseKeys } from '../keys'

const sorts = ['-date', 'date', '-number', 'number', '-total', 'total'] as const satisfies readonly InvoiceSort[]
const defaults = { filters: { q: '', status: '', order_id: '', payment_state: '' }, sort: '-date', sorts: [...sorts], pageSize: 50 }
const base = '/purchase/invoices'
const paymentStates: PaymentState[] = ['unpaid', 'partial', 'paid']

// InvoicesPage lists the invoices the user may see; opened from an order, only that order's.
// Invoices are made from an order, so the list has no create button.
export function InvoicesPage() {
  const { t } = useTranslation()
  const [params, set, filterBar] = useListParams(defaults)
  const { q, status, order_id, payment_state } = params.filters as typeof defaults.filters
  const order = Number(order_id)
  const byOrder = Number.isInteger(order) && order > 0
  const orderNumber = useOrder(order, byOrder).data?.number
  const query = { q, status, order: order_id, payment_state, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: purchaseKeys.invoices.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/purchase/invoices', {
          params: {
            query: {
              q: q || undefined,
              status: (status || undefined) as DocStatus | undefined,
              order_id: byOrder ? order : undefined,
              payment_state: (payment_state || undefined) as PaymentState | undefined,
              sort: params.sort as InvoiceSort,
              page: params.page,
              page_size: params.pageSize as 50,
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  })

  const columns: Column<InvoiceListItem>[] = [
    { key: 'number', header: t('purchase.doc.number'), role: 'title', kind: 'code', sortable: true, render: (d) => d.number },
    { key: 'date', header: t('purchase.invoice.date'), role: 'meta', kind: 'date', sortable: true, render: (d) => formatDate(d.date) },
    { key: 'supplier', header: t('purchase.doc.supplier'), role: 'meta', render: (d) => d.supplier_name },
    { key: 'order', header: t('purchase.goods_receipt.order'), role: 'meta', kind: 'code', render: (d) => d.order_number },
    { key: 'total', header: t('purchase.doc.total_vnd'), role: 'meta', kind: 'number', sortable: true, render: (d) => formatNumber(d.total) },
    { key: 'status', header: t('purchase.doc.status'), role: 'status', kind: 'status', render: (d) => <DocumentStatus docType={invoiceDocType} status={d.status} /> },
    {
      key: 'payment_state',
      header: t('purchase.payment_state.title'),
      role: 'status',
      kind: 'status',
      render: (d) => {
        const b = d.status === 'posted' ? paymentStateBadge(d.payment_state) : null
        return b && <StatusBadge tone={b.tone}>{t(b.label)}</StatusBadge>
      },
    },
  ]
  const view = useListView('purchase.invoices', columns)

  return (
    <ListPage
      title={t('purchase.invoices.title')}
      description={t('purchase.invoices.description')}
      onClearFilters={filterBar.active ? filterBar.clear : undefined}
      filters={
        <>
          <SearchInput label={t('purchase.invoices.search')} placeholder={t('purchase.invoices.search_hint')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
          <DocumentStatusFilter docType={invoiceDocType} {...filterBar.bind('status')} />
          <FilterChip
            label={t('purchase.payment_state.title')}
            icon={IconCash}
            data={paymentStates.map((s) => ({ value: s, label: t(`purchase.payment_state.${s}`) }))}
            {...filterBar.bind('payment_state')}
          />
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
          label: t('purchase.invoices.title'),
          columns: view.columns,
          onResize: view.onResize,
          rowKey: (d) => d.id,
          href: (d) => `${base}/${d.id}`,
          sort: params.sort,
          onSort: (sort) => set({ sort }),
        },
        empty: { title: t('purchase.invoices.empty'), description: t('purchase.invoices.empty_description') },
        noResults: t('purchase.common.no_results'),
        paging: { page: params.page, pageSize: params.pageSize, onPage: (page) => set({ page }), onPageSize: (pageSize) => set({ pageSize }) },
      }}
    />
  )
}
