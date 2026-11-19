import { Text } from '@mmerp/ui'
import { IconCash, IconListDetails } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type AllocationInput, type Payment } from '@/shared/api/purchase'
import { useMe } from '@/shared/auth/me'
import { useDocumentMutation } from '@/shared/document'
import { formatDate, formatNumber, today } from '@/shared/i18n'
import { DateField, Form, FormActions, FormRow, FormSection, LinesTable, LinesTotals, MoneyField, OrgUnitField, SelectField, TextField, type ErrorFields, type LineColumn } from '@/shared/ui/form'
import { openInvoices } from '../hooks/usePurchase'
import { paymentDocType } from '../keys'
import { withOwnAllocations } from '../paymentAllocations'
import { SupplierField } from './pickers'

type AllocationValues = { invoice_id: string | null; amount: number | null }

type Values = {
  date: string | null
  org_unit_id: string | null
  supplier_id: string | null
  method: string | null
  note: string
  allocations: AllocationValues[]
}

const emptyAllocation: AllocationValues = { invoice_id: null, amount: null }
const idOf = (id: number | undefined) => (id ? String(id) : null)
const toValues = (a: AllocationInput): AllocationValues => ({ invoice_id: String(a.invoice_id), amount: a.amount })

const errorFields: ErrorFields<Values> = {
  purchase_payment_invoice: 'allocations.*.invoice_id',
  purchase_payment_invoice_org_unit: 'allocations.*.invoice_id',
  purchase_payment_duplicate_invoice: 'allocations.*.invoice_id',
  purchase_payment_before_invoice: 'date',
  purchase_payment_over_invoice: 'allocations.*.amount',
}

// Where a new payment starts: from an invoice, its supplier, org unit and what is left of it.
export type PaymentStart = { supplier: { id: number; code: string; name: string }; orgUnit: number; allocations: AllocationInput[] }

type Props = {
  doc?: Payment
  start?: PaymentStart
  cancelTo: string
  // A new payment: called with its id once created.
  onCreated?: (id: number) => Promise<void>
}

// PaymentForm records money from a supplier, settling posted invoices of that supplier at the
// payment's org unit; a draft the user may edit is edited in place, otherwise it shows read-only.
export function PaymentForm({ doc, start, cancelTo, onCreated }: Props) {
  const { t } = useTranslation()
  const me = useMe()
  const ro = doc ? !doc.allowed_actions.includes('edit') : false
  const mutation = useDocumentMutation(paymentDocType, doc?.id ?? 0, doc?.version ?? 0)
  // One id per opened form: sending it twice returns the payment made the first time.
  const [requestId] = useState(() => crypto.randomUUID())
  const initial = doc ? doc.allocations.map(toValues) : (start?.allocations ?? []).map(toValues)
  const form = useForm<Values>({
    mode: 'onBlur',
    defaultValues: {
      date: doc?.date ?? today(me.timezone),
      org_unit_id: idOf(doc?.org_unit_id ?? start?.orgUnit),
      supplier_id: idOf(doc?.supplier_id ?? start?.supplier.id),
      method: doc?.method ?? 'bank',
      note: doc?.note ?? '',
      allocations: initial.length ? initial : [{ ...emptyAllocation }],
    },
  })
  const supplier = form.watch('supplier_id')
  const unit = form.watch('org_unit_id')
  const allocations = form.watch('allocations')
  const open = useQuery({ ...openInvoices(Number(supplier), Number(unit), doc?.id ?? 0), enabled: !ro && !!supplier && !!unit })
  const invoices = withOwnAllocations(open.data ?? [], doc?.allocations ?? [])
  const byId = new Map(invoices.map((a) => [String(a.invoice_id), a]))

  async function save(v: Values) {
    const body = {
      date: v.date ?? '',
      org_unit_id: Number(v.org_unit_id),
      supplier_id: Number(v.supplier_id),
      method: v.method as Payment['method'],
      note: v.note.trim() || null,
      allocations: v.allocations.map((a): AllocationInput => ({ invoice_id: Number(a.invoice_id), amount: a.amount ?? 0 })),
    }
    if (doc) {
      await mutation.run((version) => unwrap(api.PUT('/purchase/payments/{id}', { params: { path: { id: doc.id } }, body: { version, ...body } })))
      return
    }
    const { id } = await unwrap(api.POST('/purchase/payments', { body: { request_id: requestId, ...body } }))
    await onCreated?.(id)
  }

  type Line = Payment['allocations'][number]
  const invoiceLabel = (a: { invoice_number: string; invoice_date: string }) => `${a.invoice_number} · ${formatDate(a.invoice_date)}`
  const columns: LineColumn<Line>[] = [
    {
      key: 'invoice',
      header: t('purchase.payment.allocation.invoice_col'),
      required: true,
      value: invoiceLabel,
      cell: (i, bare) => (
        <SelectField
          name={`allocations.${i}.invoice_id`}
          label={t('purchase.payment.allocation.invoice', { n: i + 1 })}
          bare={bare}
          required
          data={invoices.map((a) => ({ value: String(a.invoice_id), label: invoiceLabel(a) }))}
          onChange={(v) => {
            const a = byId.get(v ?? '')
            if (a && !form.getValues(`allocations.${i}.amount`)) form.setValue(`allocations.${i}.amount`, a.remaining, { shouldDirty: true })
          }}
        />
      ),
    },
    {
      key: 'total',
      header: t('purchase.payment.allocation.invoice_total'),
      numeric: true,
      width: 144,
      value: (a) => formatNumber(a.invoice_total),
      cell: (i) => {
        const a = byId.get(allocations[i]?.invoice_id ?? '')
        return a && <Text size="sm" tone="dimmed">{formatNumber(a.invoice_total)}</Text>
      },
    },
    {
      key: 'remaining',
      header: t('purchase.payment.allocation.remaining'),
      numeric: true,
      width: 144,
      cell: (i) => {
        const a = byId.get(allocations[i]?.invoice_id ?? '')
        return a && <Text size="sm" tone="dimmed">{formatNumber(a.remaining)}</Text>
      },
    },
    {
      key: 'amount',
      header: t('purchase.payment.allocation.amount_col'),
      numeric: true,
      required: true,
      width: 160,
      value: (a) => formatNumber(a.amount),
      cell: (i, bare) => <MoneyField name={`allocations.${i}.amount`} label={t('purchase.payment.allocation.amount', { n: i + 1 })} bare={bare} required />,
    },
  ]
  const totals = doc && (
    <LinesTotals note={ro ? undefined : t('purchase.doc.totals_saved')} rows={[[t('purchase.payment.amount'), formatNumber(doc.amount)]]} />
  )

  return (
    <Form form={form} onSubmit={save} errorFields={errorFields}>
      <FormSection title={t('purchase.payment.section.general')} description={t('purchase.payment.section.general_hint')} icon={IconCash}>
        <DateField name="date" label={t('purchase.payment.date')} required readOnly={ro} />
        <OrgUnitField name="org_unit_id" label={t('purchase.doc.org_unit')} required product="purchase" permission={`purchase.order.${ro ? 'view' : 'edit'}`} readOnly={ro} />
        <SupplierField
          name="supplier_id"
          label={t('purchase.doc.supplier')}
          readOnly={ro}
          current={doc ? { id: doc.supplier_id, code: doc.supplier_code, name: doc.supplier_name } : start?.supplier}
        />
        <SelectField
          name="method"
          label={t('purchase.payment.method')}
          required
          readOnly={ro}
          data={(['cash', 'bank'] as const).map((m) => ({ value: m, label: t(`purchase.payment.method_${m}`) }))}
        />
        <TextField name="note" label={t('purchase.payment.note')} multiline maxLength={2000} readOnly={ro} />
      </FormSection>
      <FormSection
        title={t('purchase.payment.section.allocations')}
        description={t(ro ? 'purchase.payment.section.allocations_hint_ro' : 'purchase.payment.section.allocations_hint')}
        icon={IconListDetails}
        wide
      >
        <FormRow>
          {ro && doc ? (
            <LinesTable readOnly label={t('purchase.payment.section.allocations')} columns={columns} lines={doc.allocations} lineLabel={(n) => t('purchase.line.title', { n })} footer={totals} />
          ) : (
            <LinesTable
              name="allocations"
              label={t('purchase.payment.section.allocations')}
              columns={columns}
              newLine={() => ({ ...emptyAllocation })}
              addLabel={t('purchase.payment.allocation.add')}
              lineLabel={(n) => t('purchase.line.title', { n })}
              removeLabel={(n) => t('purchase.line.remove', { n })}
              minLines={1}
              footer={totals}
            />
          )}
        </FormRow>
      </FormSection>
      {!ro && <FormActions submitLabel={t(doc ? 'purchase.common.save' : 'purchase.common.create_draft')} cancelTo={doc ? undefined : cancelTo} />}
      {mutation.dialog}
    </Form>
  )
}
