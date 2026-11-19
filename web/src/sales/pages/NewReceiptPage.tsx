import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router'
import { CreatePage } from '@/shared/document'
import { loaded } from '@/shared/ui/loaded'
import { ContentSkeleton, ErrorState } from '@/shared/ui/states'
import { ReceiptForm } from '../components/ReceiptForm'
import { openInvoices, useInvoice, useReceiptActions } from '../hooks/useSales'
import { receiptDocType } from '../keys'
import { allocationsFor } from '../receiptAllocations'

const list = '/sales/receipts'
const invoices = '/sales/invoices'

// NewReceiptPage records money from a customer; opened from an invoice (?invoice=), it starts
// with that invoice's customer, org unit and what is left of it.
export function NewReceiptPage() {
  const invoice = Number(useSearchParams()[0].get('invoice'))
  return Number.isInteger(invoice) && invoice > 0 ? <FromInvoice id={invoice} /> : <Direct />
}

function Direct() {
  const { t } = useTranslation()
  const actions = useReceiptActions()
  if (actions.isPending) return <ContentSkeleton />
  if (actions.isError) return <ErrorState queries={[actions]} />
  return (
    <CreatePage
      docType={receiptDocType}
      actions={actions.data}
      action="create"
      title={t('sales.receipts.create')}
      breadcrumbs={[{ label: t('sales.receipts.title'), to: list }]}
      description={t('sales.receipts.create_description')}
      created={t('sales.receipt.created')}
    >
      {(onCreated) => <ReceiptForm cancelTo={list} onCreated={onCreated} />}
    </CreatePage>
  )
}

function FromInvoice({ id }: { id: number }) {
  const { t } = useTranslation()
  const state = loaded(useInvoice(id), t, { to: invoices, label: t('sales.invoices.back') })
  const inv = state.ok ? state.data : undefined
  const open = useQuery({ ...openInvoices(inv?.customer.id ?? 0, inv?.org_unit_id ?? 0, 0), enabled: !!inv })
  if (!state.ok) return state.fallback
  if (open.isPending) return <ContentSkeleton />
  if (open.isError) return <ErrorState queries={[open]} />
  const d = state.data
  return (
    <CreatePage
      docType={receiptDocType}
      actions={d.allowed_actions}
      action="create_receipt"
      title={t('sales.receipts.create')}
      breadcrumbs={[
        { label: t('sales.invoices.title'), to: invoices },
        { label: d.number, to: `${invoices}/${d.id}` },
      ]}
      description={t('sales.receipts.create_from_invoice', { number: d.number, customer: d.customer.name })}
      created={t('sales.receipt.created')}
    >
      {(onCreated) => (
        <ReceiptForm
          cancelTo={`${invoices}/${d.id}`}
          start={{ customer: { id: d.customer.id, code: d.customer.code, name: d.customer.name }, orgUnit: d.org_unit_id, allocations: allocationsFor(open.data, d.id) }}
          onCreated={onCreated}
        />
      )}
    </CreatePage>
  )
}
