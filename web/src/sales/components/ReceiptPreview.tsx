import { useTranslation } from 'react-i18next'
import { DocumentPreview } from '@/shared/document'
import { formatDate, formatNumber } from '@/shared/i18n'
import { useReceipt } from '../hooks/useSales'
import { receiptDocType } from '../keys'

// ReceiptPreview is a receipt in the approval inbox, when a rule holds one for approval. It is
// loaded outside the sales screens, so its strings live in the meta translations.
export default function ReceiptPreview({ id }: { id: number }) {
  const { t } = useTranslation()
  return (
    <DocumentPreview
      docType={receiptDocType}
      query={useReceipt(id)}
      rows={(d) => [
        [t('sales.preview.customer'), d.customer_name],
        [t('sales.preview.date'), formatDate(d.date)],
        [t('sales.preview.invoices'), d.allocations.map((a) => a.invoice_number).join(', ')],
        [t('sales.preview.amount'), formatNumber(d.amount)],
      ]}
    />
  )
}
