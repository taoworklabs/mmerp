import { Group } from '@mmerp/ui'
import { useTranslation } from 'react-i18next'
import type { OrderListItem } from '@/shared/api/purchase'
import { DocumentStatus } from '@/shared/document'
import { formatDate, formatNumber } from '@/shared/i18n'
import type { Column } from '@/shared/ui/DataTable'
import { orderDocType } from '../keys'
import { DocProgress } from './DocProgress'

// useOrderColumns: the columns of a list of purchase orders; a supplier's own list leaves the supplier out.
export function useOrderColumns({ supplier = true } = {}): Column<OrderListItem>[] {
  const { t } = useTranslation()
  return [
    { key: 'number', header: t('purchase.doc.number'), role: 'title', kind: 'code', sortable: true, render: (d) => d.number },
    ...(supplier ? [{ key: 'supplier', header: t('purchase.doc.supplier'), role: 'meta' as const, render: (d: OrderListItem) => d.supplier_name }] : []),
    { key: 'date', header: t('purchase.order.date'), role: 'meta', kind: 'date', sortable: true, render: (d) => formatDate(d.date) },
    {
      key: 'receipt_state',
      header: t('purchase.order.receipt_state.title'),
      role: 'hidden',
      render: (d) => d.receipt_state && t(`purchase.order.receipt_state.${d.receipt_state}`),
    },
    { key: 'org_unit', header: t('purchase.doc.org_unit'), role: 'hidden', render: (d) => d.org_unit_name },
    { key: 'total', header: t('purchase.doc.total_vnd'), role: 'meta', kind: 'number', sortable: true, render: (d) => formatNumber(d.total) },
    {
      key: 'status',
      header: t('purchase.doc.status'),
      role: 'status',
      kind: 'status',
      render: (d) => (
        <Group gap="xs" wrap={false}>
          <DocumentStatus docType={orderDocType} status={d.status} />
          {d.status === 'posted' && <DocProgress doc={d} />}
        </Group>
      ),
    },
  ]
}
