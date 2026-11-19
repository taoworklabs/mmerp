import { Button } from '@mmerp/ui'
import { IconCash, IconFileInvoice, IconPlus, IconPackageImport } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { api, unwrap, type OrderSort, type PaymentState } from '@/shared/api/purchase'
import { DocumentStatusFilter, type DocStatus } from '@/shared/document'
import { FilterChip, SearchInput } from '@/shared/ui/form'
import { ListPage } from '@/shared/ui/page'
import { useListView } from '@/shared/ui/ListView'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import { useOrderColumns } from '../components/docColumns'
import { useOrderActions } from '../hooks/usePurchase'
import { orderDocType, purchaseKeys } from '../keys'

const sorts = ['-date', 'date', '-number', 'number', '-total', 'total'] as const satisfies readonly OrderSort[]
const goodsReceiptStates = ['none', 'partial', 'full', 'not_applicable'] as const
type GoodsReceiptState = (typeof goodsReceiptStates)[number]
const paymentStates: PaymentState[] = ['unpaid', 'partial', 'paid']
const invoiceStates = ['none', 'partial', 'full'] as const
const defaults = { filters: { q: '', status: '', receipt_state: '', invoice_state: '', payment_state: '' }, sort: '-date', sorts: [...sorts], pageSize: 50 }

const base = '/purchase/orders'

// OrdersPage lists the purchase orders the user may see.
export function OrdersPage() {
  const { t } = useTranslation()
  const canCreate = useOrderActions().data?.includes('create') ?? false
  const [params, set, filterBar] = useListParams(defaults)
  const { q, status, receipt_state, invoice_state, payment_state } = params.filters as typeof defaults.filters
  const query = { q, status, receipt_state, invoice_state, payment_state, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: purchaseKeys.orders.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/purchase/orders', {
          params: {
            query: {
              q: q || undefined,
              status: (status || undefined) as DocStatus | undefined,
              sort: params.sort as OrderSort,
              page: params.page,
              page_size: params.pageSize as 50,
              receipt_state: (receipt_state || undefined) as GoodsReceiptState | undefined,
              invoice_state: (invoice_state || undefined) as (typeof invoiceStates)[number] | undefined,
              payment_state: (payment_state || undefined) as PaymentState | undefined,
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  })

  const columns = useOrderColumns()
  const view = useListView('purchase.orders', columns)

  return (
    <ListPage
      title={t('purchase.orders.title')}
      description={t('purchase.orders.description')}
      action={
        canCreate && (
          <Button render={<Link to={`${base}/new`} />} leftSection={<IconPlus {...icon.button} />}>
            {t('purchase.orders.create')}
          </Button>
        )
      }
      onClearFilters={filterBar.active ? filterBar.clear : undefined}
      filters={
        <>
          <SearchInput label={t('purchase.doc.search')} placeholder={t('purchase.doc.search_hint')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
          <DocumentStatusFilter docType={orderDocType} {...filterBar.bind('status')} />
          <FilterChip
            label={t('purchase.order.receipt_state.title')}
            icon={IconPackageImport}
            data={goodsReceiptStates.map((s) => ({ value: s, label: t(`purchase.order.receipt_state.${s}`) }))}
            {...filterBar.bind('receipt_state')}
          />
          <FilterChip
            label={t('purchase.order.invoice_state.title')}
            icon={IconFileInvoice}
            data={invoiceStates.map((s) => ({ value: s, label: t(`purchase.order.invoice_state.${s}`) }))}
            {...filterBar.bind('invoice_state')}
          />
          <FilterChip
            label={t('purchase.payment_state.title')}
            icon={IconCash}
            data={paymentStates.map((s) => ({ value: s, label: t(`purchase.payment_state.${s}`) }))}
            {...filterBar.bind('payment_state')}
          />
        </>
      }
      tools={view.tools}
      list={{
        query: list,
        table: {
          label: t('purchase.orders.title'),
          columns: view.columns,
          onResize: view.onResize,
          rowKey: (d) => d.id,
          href: (d) => `${base}/${d.id}`,
          sort: params.sort,
          onSort: (sort) => set({ sort }),
        },
        empty: { title: t('purchase.orders.empty'), description: t('purchase.orders.empty_description') },
        noResults: t('purchase.common.no_results'),
        paging: { page: params.page, pageSize: params.pageSize, onPage: (page) => set({ page }), onPageSize: (pageSize) => set({ pageSize }) },
      }}
    />
  )
}
