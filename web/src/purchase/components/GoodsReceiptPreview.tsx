import { useTranslation } from 'react-i18next'
import { DocumentPreview } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import { useGoodsReceipt } from '../hooks/usePurchase'
import { goodsReceiptDocType } from '../keys'

// GoodsReceiptPreview is a goodsReceipt in the approval inbox, when a rule holds one for approval. It is
// loaded outside the purchase screens, so its strings live in the meta translations.
export default function GoodsReceiptPreview({ id }: { id: number }) {
  const { t } = useTranslation()
  return (
    <DocumentPreview
      docType={goodsReceiptDocType}
      query={useGoodsReceipt(id)}
      rows={(d) => [
        [t('purchase.preview.order'), d.order_number],
        [t('purchase.preview.warehouse'), `${d.warehouse_code} · ${d.warehouse_name}`],
        [t('purchase.preview.date'), formatDate(d.date)],
        [t('purchase.preview.lines'), String(d.lines.length)],
      ]}
    />
  )
}
