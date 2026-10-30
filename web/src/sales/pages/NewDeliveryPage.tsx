import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router'
import { CreatePage } from '@/shared/document'
import { loaded } from '@/shared/ui/loaded'
import { NotFoundPage } from '@/shared/ui/states'
import { DeliveryForm } from '../components/DeliveryForm'
import { useDoc } from '../hooks/useSales'
import { deliveryDocType } from '../keys'

const orders = '/sales/orders'

// NewDeliveryPage makes a delivery from the order in ?order=, when the order allows it.
export function NewDeliveryPage() {
  const { t } = useTranslation()
  const order = Number(useSearchParams()[0].get('order'))
  if (!Number.isInteger(order) || order <= 0) return <NotFoundPage back={{ to: orders, label: t('sales.orders.back') }} />
  return <NewDelivery order={order} />
}

function NewDelivery({ order: id }: { order: number }) {
  const { t } = useTranslation()
  const state = loaded(useDoc('order', id), t, { to: orders, label: t('sales.orders.back') })
  if (!state.ok) return state.fallback
  const order = state.data
  return (
    <CreatePage
      docType={deliveryDocType}
      actions={order.allowed_actions}
      action="create_delivery"
      title={t('sales.deliveries.create')}
      breadcrumbs={[
        { label: t('sales.orders.title'), to: orders },
        { label: order.number, to: `${orders}/${order.id}` },
      ]}
      description={t('sales.deliveries.create_description', { number: order.number, customer: order.customer.name })}
      created={t('sales.delivery.created')}
    >
      {(onCreated) => <DeliveryForm order={order.id} onCreated={onCreated} />}
    </CreatePage>
  )
}
