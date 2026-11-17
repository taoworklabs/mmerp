import { useTranslation } from 'react-i18next'
import type { Doc } from '@/shared/api/sales'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { deliveryStateBadge } from '../deliveryLines'
import { invoiceStateBadge, paymentStateBadge } from '../invoiceLines'

// DocProgress is a document's derived progress: a quotation ordered, or expired while still
// posted; an order delivered, invoiced and paid in part or in full.
export function DocProgress({ doc }: { doc: Partial<Pick<Doc, 'ordered' | 'expired' | 'delivery_state' | 'invoice_state' | 'payment_state'>> }) {
  const { t } = useTranslation()
  if (doc.ordered) return <StatusBadge tone="info">{t('sales.quote.progress.ordered')}</StatusBadge>
  if (doc.expired) return <StatusBadge tone="neutral">{t('sales.quote.progress.expired')}</StatusBadge>
  const delivery = deliveryStateBadge(doc.delivery_state)
  const invoice = invoiceStateBadge(doc.invoice_state)
  // Payment shows once something is invoiced.
  const payment = invoice ? paymentStateBadge(doc.payment_state) : null
  return (
    <>
      {delivery && <StatusBadge tone={delivery.tone}>{t(delivery.label)}</StatusBadge>}
      {invoice && <StatusBadge tone={invoice.tone}>{t(invoice.label)}</StatusBadge>}
      {payment && <StatusBadge tone={payment.tone}>{t(payment.label)}</StatusBadge>}
    </>
  )
}
