import { useTranslation } from 'react-i18next'
import { CreatePage } from '@/shared/document'
import { DocForm } from '../components/DocForm'
import { useOrderActions } from '../hooks/usePurchase'
import { orderDocType } from '../keys'

export function NewOrderPage() {
  const { t } = useTranslation()
  const actions = useOrderActions()
  return (
    <CreatePage
      docType={orderDocType}
      actions={actions}
      title={t('purchase.orders.create')}
      breadcrumbs={[{ label: t('purchase.orders.title'), to: '/purchase/orders' }]}
      description={t('purchase.orders.create_description')}
      created={t('purchase.order.created')}
    >
      {(onCreated) => <DocForm onCreated={onCreated} />}
    </CreatePage>
  )
}
