import { Button } from '@mmerp/ui'
import { IconCash, IconPlus } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { api, unwrap, type PaymentListItem, type PaymentSort } from '@/shared/api/purchase'
import { DocumentStatus, DocumentStatusFilter, type DocStatus } from '@/shared/document'
import { formatDate, formatNumber } from '@/shared/i18n'
import type { Column } from '@/shared/ui/DataTable'
import { FilterChip, SearchInput } from '@/shared/ui/form'
import { useListView } from '@/shared/ui/ListView'
import { ListPage } from '@/shared/ui/page'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import { usePaymentActions } from '../hooks/usePurchase'
import { paymentDocType, purchaseKeys } from '../keys'

const sorts = ['-date', 'date', '-number', 'number', '-amount', 'amount'] as const satisfies readonly PaymentSort[]
const defaults = { filters: { q: '', status: '', method: '' }, sort: '-date', sorts: [...sorts], pageSize: 50 }
const base = '/purchase/payments'
const methods = ['cash', 'bank'] as const

// PaymentsPage lists the supplier payments the user may see.
export function PaymentsPage() {
  const { t } = useTranslation()
  const [params, set, filterBar] = useListParams(defaults)
  const { q, status, method } = params.filters as typeof defaults.filters
  const canCreate = usePaymentActions().data?.includes('create')
  const query = { q, status, method, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: purchaseKeys.payments.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/purchase/payments', {
          params: {
            query: {
              q: q || undefined,
              status: (status || undefined) as DocStatus | undefined,
              method: (method || undefined) as (typeof methods)[number] | undefined,
              sort: params.sort as PaymentSort,
              page: params.page,
              page_size: params.pageSize as 50,
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  })

  const columns: Column<PaymentListItem>[] = [
    { key: 'number', header: t('purchase.doc.number'), role: 'title', kind: 'code', sortable: true, render: (d) => d.number },
    { key: 'date', header: t('purchase.payment.date'), role: 'meta', kind: 'date', sortable: true, render: (d) => formatDate(d.date) },
    { key: 'supplier', header: t('purchase.doc.supplier'), role: 'meta', render: (d) => d.supplier_name },
    { key: 'method', header: t('purchase.payment.method'), role: 'meta', render: (d) => t(`purchase.payment.method_${d.method}`) },
    { key: 'amount', header: t('purchase.payment.amount_vnd'), role: 'meta', kind: 'number', sortable: true, render: (d) => formatNumber(d.amount) },
    { key: 'status', header: t('purchase.doc.status'), role: 'status', kind: 'status', render: (d) => <DocumentStatus docType={paymentDocType} status={d.status} /> },
  ]
  const view = useListView('purchase.payments', columns)

  return (
    <ListPage
      title={t('purchase.payments.title')}
      description={t('purchase.payments.description')}
      action={
        canCreate && (
          <Button render={<Link to={`${base}/new`} />} leftSection={<IconPlus {...icon.button} />}>
            {t('purchase.payments.create')}
          </Button>
        )
      }
      onClearFilters={filterBar.active ? filterBar.clear : undefined}
      filters={
        <>
          <SearchInput label={t('purchase.payments.search')} placeholder={t('purchase.payments.search_hint')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
          <DocumentStatusFilter docType={paymentDocType} {...filterBar.bind('status')} />
          <FilterChip
            label={t('purchase.payment.method')}
            icon={IconCash}
            data={methods.map((m) => ({ value: m, label: t(`purchase.payment.method_${m}`) }))}
            {...filterBar.bind('method')}
          />
        </>
      }
      tools={view.tools}
      list={{
        query: list,
        table: {
          label: t('purchase.payments.title'),
          columns: view.columns,
          onResize: view.onResize,
          rowKey: (d) => d.id,
          href: (d) => `${base}/${d.id}`,
          sort: params.sort,
          onSort: (sort) => set({ sort }),
        },
        empty: { title: t('purchase.payments.empty'), description: t('purchase.payments.empty_description') },
        noResults: t('purchase.common.no_results'),
        paging: { page: params.page, pageSize: params.pageSize, onPage: (page) => set({ page }), onPageSize: (pageSize) => set({ pageSize }) },
      }}
    />
  )
}
