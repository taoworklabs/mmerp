import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router'
import { DocumentScreen } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import { loaded } from '@/shared/ui/loaded'
import { AdjustmentForm } from '../components/AdjustmentForm'
import { useAdjustment } from '../hooks/useStock'
import { adjustmentDocType } from '../keys'

const list = '/sales/stock/adjustments'

// AdjustmentPage shows a stock adjustment; a draft the user may edit is edited in place.
export function AdjustmentPage() {
  const { t } = useTranslation()
  const id = Number(useParams().id)
  const state = loaded(useAdjustment(id), t, { to: list, label: t('sales.adjustments.back') })
  if (!state.ok) return state.fallback
  const d = state.data
  return (
    <DocumentScreen
      docType={adjustmentDocType}
      doc={d}
      list={{ label: t('sales.adjustments.title'), to: list }}
      facts={[`${d.warehouse_code} · ${d.warehouse_name}`, formatDate(d.date), d.reason]}
      attachments
      // The editable lines table needs the whole width; approval and history become tabs.
      fullWidth
      printLabel={t('sales.adjustment.print')}
    >
      {/* Keyed by version: a reload after a conflict starts from the saved data. */}
      <AdjustmentForm key={d.version} doc={d} />
    </DocumentScreen>
  )
}
