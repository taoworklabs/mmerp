import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router'
import { CreatePage } from '@/shared/document'
import { loaded } from '@/shared/ui/loaded'
import { NotFoundPage } from '@/shared/ui/states'
import { InvoiceForm } from '../components/InvoiceForm'
import { useDoc } from '../hooks/useSales'
import { invoiceDocType } from '../keys'

const orders = '/sales/orders'

// NewInvoicePage bills the order in ?order=, when the order allows it.
export function NewInvoicePage() {
  const { t } = useTranslation()
  const order = Number(useSearchParams()[0].get('order'))
  if (!Number.isInteger(order) || order <= 0) return <NotFoundPage back={{ to: orders, label: t('sales.orders.back') }} />
  return <NewInvoice order={order} />
}

function NewInvoice({ order: id }: { order: number }) {
  const { t } = useTranslation()
  const state = loaded(useDoc('order', id), t, { to: orders, label: t('sales.orders.back') })
  if (!state.ok) return state.fallback
  const order = state.data
  return (
    <CreatePage
      docType={invoiceDocType}
      actions={order.allowed_actions}
      action="create_invoice"
      title={t('sales.invoices.create')}
      breadcrumbs={[
        { label: t('sales.orders.title'), to: orders },
        { label: order.number, to: `${orders}/${order.id}` },
      ]}
      description={t('sales.invoices.create_description', { number: order.number, customer: order.customer.name })}
      created={t('sales.invoice.created')}
    >
      {(onCreated) => <InvoiceForm order={order.id} onCreated={onCreated} />}
    </CreatePage>
  )
}
