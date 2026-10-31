import { Anchor } from '@mmerp/ui'
import { useTranslation } from 'react-i18next'
import { Link, useParams } from 'react-router'
import { DocumentScreen } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import { loaded } from '@/shared/ui/loaded'
import { DeliveryForm } from '../components/DeliveryForm'
import { useDelivery } from '../hooks/useSales'
import { deliveryDocType } from '../keys'

const list = '/sales/deliveries'

// DeliveryPage shows a delivery; a draft the user may edit is edited in place.
export function DeliveryPage() {
  const { t } = useTranslation()
  const id = Number(useParams().id)
  const state = loaded(useDelivery(id), t, { to: list, label: t('sales.deliveries.back') })
  if (!state.ok) return state.fallback
  const d = state.data
  return (
    <DocumentScreen
      docType={deliveryDocType}
      doc={d}
      list={{ label: t('sales.deliveries.title'), to: list }}
      facts={[
        <Anchor key="order" render={<Link to={`/sales/orders/${d.order_id}`} />} size="sm">
          {t('sales.delivery.order_link', { number: d.order_number })}
        </Anchor>,
        `${d.warehouse_code} · ${d.warehouse_name}`,
        formatDate(d.date),
      ]}
      attachments
      // The editable lines table needs the whole width; approval and history become tabs.
      fullWidth
    >
      {/* Keyed by version: a reload after a conflict starts from the saved data. */}
      <DeliveryForm key={d.version} order={d.order_id} doc={d} />
    </DocumentScreen>
  )
}
