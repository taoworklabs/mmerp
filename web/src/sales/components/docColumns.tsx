import { Group } from '@mmerp/ui'
import { useTranslation } from 'react-i18next'
import type { DocListItem } from '@/shared/api/sales'
import { DocumentStatus } from '@/shared/document'
import { formatDate, formatNumber } from '@/shared/i18n'
import type { Column } from '@/shared/ui/DataTable'
import { docTypeOf, type Kind } from '../keys'
import { DocProgress } from './DocProgress'

// useDocColumns: the columns of a list of quotations or orders; a customer's own list leaves the customer out.
export function useDocColumns(kind: Kind, { customer = true } = {}): Column<DocListItem>[] {
  const { t } = useTranslation()
  const docType = docTypeOf(kind)
  return [
    { key: 'number', header: t('sales.doc.number'), role: 'title', kind: 'code', sortable: true, render: (d) => d.number },
    ...(customer ? [{ key: 'customer', header: t('sales.doc.customer'), role: 'meta' as const, render: (d: DocListItem) => d.customer_name }] : []),
    { key: 'date', header: t(`sales.${kind}.date`), role: 'meta', kind: 'date', sortable: true, render: (d) => formatDate(d.date) },
    ...(kind === 'quote'
      ? [{ key: 'valid_until', header: t('sales.quote.valid_until'), role: 'hidden' as const, kind: 'date' as const, render: (d: DocListItem) => formatDate(d.valid_until) }]
      : []),
    { key: 'org_unit', header: t('sales.doc.org_unit'), role: 'hidden', render: (d) => d.org_unit_name },
    { key: 'total', header: t('sales.doc.total_vnd'), role: 'meta', kind: 'number', sortable: true, render: (d) => formatNumber(d.total) },
    {
      key: 'status',
      header: t('sales.doc.status'),
      role: 'status',
      kind: 'status',
      render: (d) => (
        <Group gap="xs" wrap={false}>
          <DocumentStatus docType={docType} status={d.status} />
          <DocProgress doc={d} />
        </Group>
      ),
    },
  ]
}
