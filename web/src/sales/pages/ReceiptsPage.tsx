import { Button } from '@mmerp/ui'
import { IconCash, IconPlus } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { api, unwrap, type ReceiptListItem, type ReceiptSort } from '@/shared/api/sales'
import { DocumentStatus, DocumentStatusFilter, type DocStatus } from '@/shared/document'
import { formatDate, formatNumber } from '@/shared/i18n'
import type { Column } from '@/shared/ui/DataTable'
import { FilterChip, SearchInput } from '@/shared/ui/form'
import { useListView } from '@/shared/ui/ListView'
import { ListPage } from '@/shared/ui/page'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import { useReceiptActions } from '../hooks/useSales'
import { receiptDocType, salesKeys } from '../keys'

const sorts = ['-date', 'date', '-number', 'number', '-amount', 'amount'] as const satisfies readonly ReceiptSort[]
const defaults = { filters: { q: '', status: '', method: '' }, sort: '-date', sorts: [...sorts], pageSize: 50 }
const base = '/sales/receipts'
const methods = ['cash', 'bank'] as const

// ReceiptsPage lists the customer receipts the user may see.
export function ReceiptsPage() {
  const { t } = useTranslation()
  const [params, set, filterBar] = useListParams(defaults)
  const { q, status, method } = params.filters as typeof defaults.filters
  const canCreate = useReceiptActions().data?.includes('create')
  const query = { q, status, method, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: salesKeys.receipts.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/sales/receipts', {
          params: {
            query: {
              q: q || undefined,
              status: (status || undefined) as DocStatus | undefined,
              method: (method || undefined) as (typeof methods)[number] | undefined,
              sort: params.sort as ReceiptSort,
              page: params.page,
              page_size: params.pageSize as 50,
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  })

  const columns: Column<ReceiptListItem>[] = [
    { key: 'number', header: t('sales.doc.number'), role: 'title', kind: 'code', sortable: true, render: (d) => d.number },
    { key: 'date', header: t('sales.receipt.date'), role: 'meta', kind: 'date', sortable: true, render: (d) => formatDate(d.date) },
    { key: 'customer', header: t('sales.doc.customer'), role: 'meta', render: (d) => d.customer_name },
    { key: 'method', header: t('sales.receipt.method'), role: 'meta', render: (d) => t(`sales.receipt.method_${d.method}`) },
    { key: 'amount', header: t('sales.receipt.amount_vnd'), role: 'meta', kind: 'number', sortable: true, render: (d) => formatNumber(d.amount) },
    { key: 'status', header: t('sales.doc.status'), role: 'status', kind: 'status', render: (d) => <DocumentStatus docType={receiptDocType} status={d.status} /> },
  ]
  const view = useListView('sales.receipts', columns)

  return (
    <ListPage
      title={t('sales.receipts.title')}
      description={t('sales.receipts.description')}
      action={
        canCreate && (
          <Button render={<Link to={`${base}/new`} />} leftSection={<IconPlus {...icon.button} />}>
            {t('sales.receipts.create')}
          </Button>
        )
      }
      onClearFilters={filterBar.active ? filterBar.clear : undefined}
      filters={
        <>
          <SearchInput label={t('sales.receipts.search')} placeholder={t('sales.receipts.search_hint')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
          <DocumentStatusFilter docType={receiptDocType} {...filterBar.bind('status')} />
          <FilterChip
            label={t('sales.receipt.method')}
            icon={IconCash}
            data={methods.map((m) => ({ value: m, label: t(`sales.receipt.method_${m}`) }))}
            {...filterBar.bind('method')}
          />
        </>
      }
      tools={view.tools}
      list={{
        query: list,
        table: {
          label: t('sales.receipts.title'),
          columns: view.columns,
          onResize: view.onResize,
          rowKey: (d) => d.id,
          href: (d) => `${base}/${d.id}`,
          sort: params.sort,
          onSort: (sort) => set({ sort }),
        },
        empty: { title: t('sales.receipts.empty'), description: t('sales.receipts.empty_description') },
        noResults: t('sales.common.no_results'),
        paging: { page: params.page, pageSize: params.pageSize, onPage: (page) => set({ page }), onPageSize: (pageSize) => set({ pageSize }) },
      }}
    />
  )
}
