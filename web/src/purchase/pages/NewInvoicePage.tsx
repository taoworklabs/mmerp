import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router'
import { CreatePage } from '@/shared/document'
import { loaded } from '@/shared/ui/loaded'
import { NotFoundPage } from '@/shared/ui/states'
import { InvoiceForm } from '../components/InvoiceForm'
import { useOrder } from '../hooks/usePurchase'
import { invoiceDocType } from '../keys'

const orders = '/purchase/orders'

// NewInvoicePage bills the order in ?order=, when the order allows it.
export function NewInvoicePage() {
  const { t } = useTranslation()
  const order = Number(useSearchParams()[0].get('order'))
  if (!Number.isInteger(order) || order <= 0) return <NotFoundPage back={{ to: orders, label: t('purchase.orders.back') }} />
  return <NewInvoice order={order} />
}

function NewInvoice({ order: id }: { order: number }) {
  const { t } = useTranslation()
  const state = loaded(useOrder(id), t, { to: orders, label: t('purchase.orders.back') })
  if (!state.ok) return state.fallback
  const order = state.data
  return (
    <CreatePage
      docType={invoiceDocType}
      actions={order.allowed_actions}
      action="create_invoice"
      title={t('purchase.invoices.create')}
      breadcrumbs={[
        { label: t('purchase.orders.title'), to: orders },
        { label: order.number, to: `${orders}/${order.id}` },
      ]}
      description={t('purchase.invoices.create_description', { number: order.number, supplier: order.supplier.name })}
      created={t('purchase.invoice.created')}
    >
      {(onCreated) => <InvoiceForm order={order.id} onCreated={onCreated} />}
    </CreatePage>
  )
}
