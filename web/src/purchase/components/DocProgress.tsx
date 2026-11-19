import { useTranslation } from 'react-i18next'
import type { Order } from '@/shared/api/purchase'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { goodsReceiptStateBadge } from '../receiptLines'
import { invoiceStateBadge, paymentStateBadge } from '../invoiceLines'

// DocProgress is a posted order's derived progress: received, invoiced and paid in part or in full.
export function DocProgress({ doc }: { doc: Partial<Pick<Order, 'receipt_state' | 'invoice_state' | 'payment_state'>> }) {
  const { t } = useTranslation()
  const goodsReceipt = goodsReceiptStateBadge(doc.receipt_state)
  const invoice = invoiceStateBadge(doc.invoice_state)
  // Payment shows once something is invoiced.
  const payment = invoice ? paymentStateBadge(doc.payment_state) : null
  return (
    <>
      {goodsReceipt && <StatusBadge tone={goodsReceipt.tone}>{t(goodsReceipt.label)}</StatusBadge>}
      {invoice && <StatusBadge tone={invoice.tone}>{t(invoice.label)}</StatusBadge>}
      {payment && <StatusBadge tone={payment.tone}>{t(payment.label)}</StatusBadge>}
    </>
  )
}
