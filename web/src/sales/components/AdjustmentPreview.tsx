import { useTranslation } from 'react-i18next'
import { DocumentPreview } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import { useAdjustment } from '../hooks/useStock'
import { adjustmentDocType } from '../keys'

// AdjustmentPreview is a stock adjustment in the approval inbox, when a rule holds one for
// approval. It is loaded outside the sales screens, so its strings live in the meta translations.
export default function AdjustmentPreview({ id }: { id: number }) {
  const { t } = useTranslation()
  return (
    <DocumentPreview
      docType={adjustmentDocType}
      query={useAdjustment(id)}
      rows={(d) => [
        [t('sales.preview.warehouse'), `${d.warehouse_code} · ${d.warehouse_name}`],
        [t('sales.preview.date'), formatDate(d.date)],
        [t('sales.preview.reason'), d.reason],
        [t('sales.preview.lines'), String(d.lines.length)],
      ]}
    />
  )
}
