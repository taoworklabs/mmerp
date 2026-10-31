import { useTranslation } from 'react-i18next'
import { DocumentPreview } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import { useDelivery } from '../hooks/useSales'
import { deliveryDocType } from '../keys'

// DeliveryPreview is a delivery in the approval inbox, when a rule holds one for approval. It is
// loaded outside the sales screens, so its strings live in the meta translations.
export default function DeliveryPreview({ id }: { id: number }) {
  const { t } = useTranslation()
  return (
    <DocumentPreview
      docType={deliveryDocType}
      query={useDelivery(id)}
      rows={(d) => [
        [t('sales.preview.order'), d.order_number],
        [t('sales.preview.warehouse'), `${d.warehouse_code} · ${d.warehouse_name}`],
        [t('sales.preview.date'), formatDate(d.date)],
        [t('sales.preview.lines'), String(d.lines.length)],
      ]}
    />
  )
}
