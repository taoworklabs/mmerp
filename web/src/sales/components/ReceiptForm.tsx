import { Text } from '@mmerp/ui'
import { IconCash, IconListDetails } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type AllocationInput, type Receipt } from '@/shared/api/sales'
import { useMe } from '@/shared/auth/me'
import { useDocumentMutation } from '@/shared/document'
import { formatDate, formatNumber, today } from '@/shared/i18n'
import { DateField, Form, FormActions, FormRow, FormSection, LinesTable, LinesTotals, MoneyField, OrgUnitField, SelectField, TextField, type ErrorFields, type LineColumn } from '@/shared/ui/form'
import { openInvoices } from '../hooks/useSales'
import { receiptDocType } from '../keys'
import { withOwnAllocations } from '../receiptAllocations'
import { CustomerField } from './pickers'

type AllocationValues = { invoice_id: string | null; amount: number | null }

type Values = {
  date: string | null
  org_unit_id: string | null
  customer_id: string | null
  method: string | null
  note: string
  allocations: AllocationValues[]
}

const emptyAllocation: AllocationValues = { invoice_id: null, amount: null }
const idOf = (id: number | undefined) => (id ? String(id) : null)
const toValues = (a: AllocationInput): AllocationValues => ({ invoice_id: String(a.invoice_id), amount: a.amount })

const errorFields: ErrorFields<Values> = {
  customer_inactive: 'customer_id',
  sales_receipt_invoice: 'allocations.*.invoice_id',
  sales_receipt_invoice_org_unit: 'allocations.*.invoice_id',
  sales_receipt_duplicate_invoice: 'allocations.*.invoice_id',
  sales_receipt_before_invoice: 'date',
  sales_receipt_over_invoice: 'allocations.*.amount',
}

// Where a new receipt starts: from an invoice, its customer, org unit and what is left of it.
export type ReceiptStart = { customer: { id: number; code: string; name: string }; orgUnit: number; allocations: AllocationInput[] }

type Props = {
  doc?: Receipt
  start?: ReceiptStart
  cancelTo: string
  // A new receipt: called with its id once created.
  onCreated?: (id: number) => Promise<void>
}

// ReceiptForm records money from a customer, settling posted invoices of that customer at the
// receipt's org unit; a draft the user may edit is edited in place, otherwise it shows read-only.
export function ReceiptForm({ doc, start, cancelTo, onCreated }: Props) {
  const { t } = useTranslation()
  const me = useMe()
  const ro = doc ? !doc.allowed_actions.includes('edit') : false
  const mutation = useDocumentMutation(receiptDocType, doc?.id ?? 0, doc?.version ?? 0)
  // One id per opened form: sending it twice returns the receipt made the first time.
  const [requestId] = useState(() => crypto.randomUUID())
  const initial = doc ? doc.allocations.map(toValues) : (start?.allocations ?? []).map(toValues)
  const form = useForm<Values>({
    mode: 'onBlur',
    defaultValues: {
      date: doc?.date ?? today(me.timezone),
      org_unit_id: idOf(doc?.org_unit_id ?? start?.orgUnit),
      customer_id: idOf(doc?.customer_id ?? start?.customer.id),
      method: doc?.method ?? 'bank',
      note: doc?.note ?? '',
      allocations: initial.length ? initial : [{ ...emptyAllocation }],
    },
  })
  const customer = form.watch('customer_id')
  const unit = form.watch('org_unit_id')
  const allocations = form.watch('allocations')
  const open = useQuery({ ...openInvoices(Number(customer), Number(unit), doc?.id ?? 0), enabled: !ro && !!customer && !!unit })
  const invoices = withOwnAllocations(open.data ?? [], doc?.allocations ?? [])
  const byId = new Map(invoices.map((a) => [String(a.invoice_id), a]))

  async function save(v: Values) {
    const body = {
      date: v.date ?? '',
      org_unit_id: Number(v.org_unit_id),
      customer_id: Number(v.customer_id),
      method: v.method as Receipt['method'],
      note: v.note.trim() || null,
      allocations: v.allocations.map((a): AllocationInput => ({ invoice_id: Number(a.invoice_id), amount: a.amount ?? 0 })),
    }
    if (doc) {
      await mutation.run((version) => unwrap(api.PUT('/sales/receipts/{id}', { params: { path: { id: doc.id } }, body: { version, ...body } })))
      return
    }
    const { id } = await unwrap(api.POST('/sales/receipts', { body: { request_id: requestId, ...body } }))
    await onCreated?.(id)
  }

  type Line = Receipt['allocations'][number]
  const invoiceLabel = (a: { invoice_number: string; invoice_date: string }) => `${a.invoice_number} · ${formatDate(a.invoice_date)}`
  const columns: LineColumn<Line>[] = [
    {
      key: 'invoice',
      header: t('sales.receipt.allocation.invoice_col'),
      required: true,
      value: invoiceLabel,
      cell: (i, bare) => (
        <SelectField
          name={`allocations.${i}.invoice_id`}
          label={t('sales.receipt.allocation.invoice', { n: i + 1 })}
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
      header: t('sales.receipt.allocation.invoice_total'),
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
      header: t('sales.receipt.allocation.remaining'),
      numeric: true,
      width: 144,
      cell: (i) => {
        const a = byId.get(allocations[i]?.invoice_id ?? '')
        return a && <Text size="sm" tone="dimmed">{formatNumber(a.remaining)}</Text>
      },
    },
    {
      key: 'amount',
      header: t('sales.receipt.allocation.amount_col'),
      numeric: true,
      required: true,
      width: 160,
      value: (a) => formatNumber(a.amount),
      cell: (i, bare) => <MoneyField name={`allocations.${i}.amount`} label={t('sales.receipt.allocation.amount', { n: i + 1 })} bare={bare} required />,
    },
  ]
  const totals = doc && (
    <LinesTotals note={ro ? undefined : t('sales.doc.totals_saved')} rows={[[t('sales.receipt.amount'), formatNumber(doc.amount)]]} />
  )

  return (
    <Form form={form} onSubmit={save} errorFields={errorFields}>
      <FormSection title={t('sales.receipt.section.general')} description={t('sales.receipt.section.general_hint')} icon={IconCash}>
        <DateField name="date" label={t('sales.receipt.date')} required readOnly={ro} />
        <OrgUnitField name="org_unit_id" label={t('sales.doc.org_unit')} required product="sales" permission={`sales.order.${ro ? 'view' : 'edit'}`} readOnly={ro} />
        <CustomerField
          name="customer_id"
          label={t('sales.doc.customer')}
          readOnly={ro}
          current={doc ? { id: doc.customer_id, code: doc.customer_code, name: doc.customer_name } : start?.customer}
        />
        <SelectField
          name="method"
          label={t('sales.receipt.method')}
          required
          readOnly={ro}
          data={(['cash', 'bank'] as const).map((m) => ({ value: m, label: t(`sales.receipt.method_${m}`) }))}
        />
        <TextField name="note" label={t('sales.receipt.note')} multiline maxLength={2000} readOnly={ro} />
      </FormSection>
      <FormSection
        title={t('sales.receipt.section.allocations')}
        description={t(ro ? 'sales.receipt.section.allocations_hint_ro' : 'sales.receipt.section.allocations_hint')}
        icon={IconListDetails}
        wide
      >
        <FormRow>
          {ro && doc ? (
            <LinesTable readOnly label={t('sales.receipt.section.allocations')} columns={columns} lines={doc.allocations} lineLabel={(n) => t('sales.line.title', { n })} footer={totals} />
          ) : (
            <LinesTable
              name="allocations"
              label={t('sales.receipt.section.allocations')}
              columns={columns}
              newLine={() => ({ ...emptyAllocation })}
              addLabel={t('sales.receipt.allocation.add')}
              lineLabel={(n) => t('sales.line.title', { n })}
              removeLabel={(n) => t('sales.line.remove', { n })}
              minLines={1}
              footer={totals}
            />
          )}
        </FormRow>
      </FormSection>
      {!ro && <FormActions submitLabel={t(doc ? 'sales.common.save' : 'sales.common.create_draft')} cancelTo={doc ? undefined : cancelTo} />}
      {mutation.dialog}
    </Form>
  )
}
