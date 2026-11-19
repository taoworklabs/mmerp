import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router'
import { DocumentScreen } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import { loaded } from '@/shared/ui/loaded'
import { StocktakeForm } from '../components/StocktakeForm'
import { useStocktake } from '../hooks/useStock'
import { stocktakeDocType } from '../keys'

const list = '/stock/stocktakes'

// StocktakePage shows a stocktake; a draft the user may edit is edited in place.
export function StocktakePage() {
  const { t } = useTranslation()
  const id = Number(useParams().id)
  const state = loaded(useStocktake(id), t, { to: list, label: t('stock.stocktakes.back') })
  if (!state.ok) return state.fallback
  const d = state.data
  return (
    <DocumentScreen
      docType={stocktakeDocType}
      doc={d}
      list={{ label: t('stock.stocktakes.title'), to: list }}
      facts={[`${d.warehouse_code} · ${d.warehouse_name}`, formatDate(d.date)]}
      attachments
      // The editable lines table needs the whole width; approval and history become tabs.
      fullWidth
      printLabel={t('stock.stocktake.print')}
    >
      {/* Keyed by version: a reload after a conflict starts from the saved data. */}
      <StocktakeForm key={d.version} doc={d} />
    </DocumentScreen>
  )
}
