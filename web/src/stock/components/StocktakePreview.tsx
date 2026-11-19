import { useTranslation } from 'react-i18next'
import { DocumentPreview } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import { useStocktake } from '../hooks/useStock'
import { stocktakeDocType } from '../keys'

// StocktakePreview is a stocktake in the approval inbox, when a rule holds one for approval. It is
// loaded outside the stock screens, so its strings live in the meta translations.
export default function StocktakePreview({ id }: { id: number }) {
  const { t } = useTranslation()
  return (
    <DocumentPreview
      docType={stocktakeDocType}
      query={useStocktake(id)}
      rows={(d) => [
        [t('stock.preview.warehouse'), `${d.warehouse_code} · ${d.warehouse_name}`],
        [t('stock.preview.date'), formatDate(d.date)],
        [t('stock.preview.lines'), String(d.lines.length)],
      ]}
    />
  )
}
