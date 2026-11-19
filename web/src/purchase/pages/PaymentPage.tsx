import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router'
import { DocumentScreen } from '@/shared/document'
import { formatDate, formatNumber } from '@/shared/i18n'
import { loaded } from '@/shared/ui/loaded'
import { PaymentForm } from '../components/PaymentForm'
import { usePayment } from '../hooks/usePurchase'
import { paymentDocType } from '../keys'

const list = '/purchase/payments'

// PaymentPage shows a payment; a draft the user may edit is edited in place.
export function PaymentPage() {
  const { t } = useTranslation()
  const id = Number(useParams().id)
  const state = loaded(usePayment(id), t, { to: list, label: t('purchase.payments.back') })
  if (!state.ok) return state.fallback
  const d = state.data
  return (
    <DocumentScreen
      docType={paymentDocType}
      doc={d}
      list={{ label: t('purchase.payments.title'), to: list }}
      facts={[d.supplier_name, d.org_unit_name, formatDate(d.date), t(`purchase.payment.method_${d.method}`), `${formatNumber(d.amount)} đ`]}
      attachments
      fullWidth
      printLabel={t('purchase.payment.print')}
    >
      {/* Keyed by version: a reload after a conflict starts from the saved data. */}
      <PaymentForm key={d.version} doc={d} cancelTo={list} />
    </DocumentScreen>
  )
}
