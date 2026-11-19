import { useTranslation } from 'react-i18next'
import { DocumentPreview } from '@/shared/document'
import { formatDate, formatNumber } from '@/shared/i18n'
import { useOrder } from '../hooks/usePurchase'
import { orderDocType } from '../keys'

// OrderPreview is the read-only view of a purchase order in the approval inbox. It is loaded
// outside the purchase screens, so its strings live in the meta translations.
export default function OrderPreview({ id }: { id: number }) {
  const { t } = useTranslation()
  return (
    <DocumentPreview
      docType={orderDocType}
      query={useOrder(id)}
      rows={(d) => [
        [t('purchase.preview.supplier'), `${d.supplier.code} · ${d.supplier.name}`],
        [t('purchase.preview.date'), formatDate(d.date)],
        [t('purchase.preview.org_unit'), d.org_unit_name],
        [t('purchase.preview.lines'), String(d.lines.length)],
        [t('purchase.preview.discount_total'), formatNumber(d.discount_total)],
        [t('purchase.preview.total'), formatNumber(d.total)],
      ]}
    />
  )
}
