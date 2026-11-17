import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router'
import { DocumentScreen } from '@/shared/document'
import { formatDate, formatNumber } from '@/shared/i18n'
import { loaded } from '@/shared/ui/loaded'
import { ReceiptForm } from '../components/ReceiptForm'
import { useReceipt } from '../hooks/useSales'
import { receiptDocType } from '../keys'

const list = '/sales/receipts'

// ReceiptPage shows a receipt; a draft the user may edit is edited in place.
export function ReceiptPage() {
  const { t } = useTranslation()
  const id = Number(useParams().id)
  const state = loaded(useReceipt(id), t, { to: list, label: t('sales.receipts.back') })
  if (!state.ok) return state.fallback
  const d = state.data
  return (
    <DocumentScreen
      docType={receiptDocType}
      doc={d}
      list={{ label: t('sales.receipts.title'), to: list }}
      facts={[d.customer_name, d.org_unit_name, formatDate(d.date), t(`sales.receipt.method_${d.method}`), `${formatNumber(d.amount)} đ`]}
      attachments
      fullWidth
      printLabel={t('sales.receipt.print')}
    >
      {/* Keyed by version: a reload after a conflict starts from the saved data. */}
      <ReceiptForm key={d.version} doc={d} cancelTo={list} />
    </DocumentScreen>
  )
}
