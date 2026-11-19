import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router'
import { CreatePage } from '@/shared/document'
import { loaded } from '@/shared/ui/loaded'
import { ContentSkeleton, ErrorState } from '@/shared/ui/states'
import { PaymentForm } from '../components/PaymentForm'
import { openInvoices, useInvoice, usePaymentActions } from '../hooks/usePurchase'
import { paymentDocType } from '../keys'
import { allocationsFor } from '../paymentAllocations'

const list = '/purchase/payments'
const invoices = '/purchase/invoices'

// NewPaymentPage records money from a supplier; opened from an invoice (?invoice=), it starts
// with that invoice's supplier, org unit and what is left of it.
export function NewPaymentPage() {
  const invoice = Number(useSearchParams()[0].get('invoice'))
  return Number.isInteger(invoice) && invoice > 0 ? <FromInvoice id={invoice} /> : <Direct />
}

function Direct() {
  const { t } = useTranslation()
  const actions = usePaymentActions()
  if (actions.isPending) return <ContentSkeleton />
  if (actions.isError) return <ErrorState queries={[actions]} />
  return (
    <CreatePage
      docType={paymentDocType}
      actions={actions.data}
      action="create"
      title={t('purchase.payments.create')}
      breadcrumbs={[{ label: t('purchase.payments.title'), to: list }]}
      description={t('purchase.payments.create_description')}
      created={t('purchase.payment.created')}
    >
      {(onCreated) => <PaymentForm cancelTo={list} onCreated={onCreated} />}
    </CreatePage>
  )
}

function FromInvoice({ id }: { id: number }) {
  const { t } = useTranslation()
  const state = loaded(useInvoice(id), t, { to: invoices, label: t('purchase.invoices.back') })
  const inv = state.ok ? state.data : undefined
  const open = useQuery({ ...openInvoices(inv?.supplier.id ?? 0, inv?.org_unit_id ?? 0, 0), enabled: !!inv })
  if (!state.ok) return state.fallback
  if (open.isPending) return <ContentSkeleton />
  if (open.isError) return <ErrorState queries={[open]} />
  const d = state.data
  return (
    <CreatePage
      docType={paymentDocType}
      actions={d.allowed_actions}
      action="create_payment"
      title={t('purchase.payments.create')}
      breadcrumbs={[
        { label: t('purchase.invoices.title'), to: invoices },
        { label: d.number, to: `${invoices}/${d.id}` },
      ]}
      description={t('purchase.payments.create_from_invoice', { number: d.number, supplier: d.supplier.name })}
      created={t('purchase.payment.created')}
    >
      {(onCreated) => (
        <PaymentForm
          cancelTo={`${invoices}/${d.id}`}
          start={{ supplier: { id: d.supplier.id, code: d.supplier.code, name: d.supplier.name }, orgUnit: d.org_unit_id, allocations: allocationsFor(open.data, d.id) }}
          onCreated={onCreated}
        />
      )}
    </CreatePage>
  )
}
