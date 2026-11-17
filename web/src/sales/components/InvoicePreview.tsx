import { useTranslation } from 'react-i18next'
import { DocumentPreview } from '@/shared/document'
import { formatDate, formatNumber } from '@/shared/i18n'
import { useInvoice } from '../hooks/useSales'
import { invoiceDocType } from '../keys'

// InvoicePreview is an invoice in the approval inbox, when a rule holds one for approval. It is
// loaded outside the sales screens, so its strings live in the meta translations.
export default function InvoicePreview({ id }: { id: number }) {
  const { t } = useTranslation()
  return (
    <DocumentPreview
      docType={invoiceDocType}
      query={useInvoice(id)}
      rows={(d) => [
        [t('sales.preview.customer'), d.customer.name],
        [t('sales.preview.order'), d.order_number],
        [t('sales.preview.date'), formatDate(d.date)],
        [t('sales.preview.lines'), String(d.lines.length)],
        [t('sales.preview.total'), formatNumber(d.totals.total)],
      ]}
    />
  )
}
