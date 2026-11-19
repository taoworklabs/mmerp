import { useTranslation } from 'react-i18next'
import { DocumentPreview } from '@/shared/document'
import { formatDate, formatNumber } from '@/shared/i18n'
import { usePayment } from '../hooks/usePurchase'
import { paymentDocType } from '../keys'

// PaymentPreview is a payment in the approval inbox, when a rule holds one for approval. It is
// loaded outside the purchase screens, so its strings live in the meta translations.
export default function PaymentPreview({ id }: { id: number }) {
  const { t } = useTranslation()
  return (
    <DocumentPreview
      docType={paymentDocType}
      query={usePayment(id)}
      rows={(d) => [
        [t('purchase.preview.supplier'), d.supplier_name],
        [t('purchase.preview.date'), formatDate(d.date)],
        [t('purchase.preview.invoices'), d.allocations.map((a) => a.invoice_number).join(', ')],
        [t('purchase.preview.amount'), formatNumber(d.amount)],
      ]}
    />
  )
}
