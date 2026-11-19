import { Stack, Text } from '@mmerp/ui'
import { IconFileInvoice, IconListDetails } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, ApiError, unwrap, type Invoice, type InvoiceLineInput, type InvoiceOptions } from '@/shared/api/purchase'
import { useMe } from '@/shared/auth/me'
import { useDocumentMutation } from '@/shared/document'
import { errorText, formatDecimal, formatNumber, today } from '@/shared/i18n'
import { DateField, DecimalField, Form, FormActions, FormRow, FormSection, LinesTable, LinesTotals, MoneyField, SelectField, TextField, type ErrorFields, type LineColumn } from '@/shared/ui/form'
import { Notice } from '@/shared/ui/Notice'
import { ContentSkeleton, ErrorState, ForbiddenPage } from '@/shared/ui/states'
import { invoiceLinesFromOptions, withOwnInvoiceLines } from '../invoiceLines'
import { invoiceOptions } from '../hooks/usePurchase'
import { invoiceDocType } from '../keys'
import { useVatOptions } from './vat'

type LineValues = { order_position: string | null; quantity: string; unit_price: number | null; discount_percent: string; vat_rate: string | null }

type Values = {
  date: string | null
  supplier_invoice_symbol: string
  supplier_invoice_no: string
  supplier_invoice_date: string | null
  note: string
  lines: LineValues[]
}

const emptyLine: LineValues = { order_position: null, quantity: '', unit_price: null, discount_percent: '0', vat_rate: null }
const toValues = (l: InvoiceLineInput): LineValues => ({
  order_position: String(l.order_position),
  quantity: l.quantity,
  unit_price: l.unit_price,
  discount_percent: l.discount_percent,
  vat_rate: l.vat_rate,
})

const errorFields: ErrorFields<Values> = {
  purchase_invoice_before_order: 'date',
  purchase_invoice_supplier_ref_required: 'supplier_invoice_no',
  purchase_invoice_duplicate: 'supplier_invoice_no',
  purchase_invoice_position: 'lines.*.order_position',
  purchase_invoice_over_order: 'lines.*.quantity',
  invalid_quantity: 'lines.*.quantity',
  line_amount_too_large: 'lines.*.quantity',
  invalid_discount: 'lines.*.discount_percent',
  invalid_vat_rate: 'lines.*.vat_rate',
}

type Props = {
  order: number
  doc?: Invoice
  // A new invoice: called with its id once created.
  onCreated?: (id: number) => Promise<void>
}

// InvoiceForm books the supplier's bill for a posted order, or edits a draft invoice whose
// allowed_actions has edit; otherwise it shows the invoice read-only. Lines start from what is left
// to bill at the order's prices, which the user corrects to the supplier's paper; amounts are the server's.
export function InvoiceForm(props: Props) {
  const { t } = useTranslation()
  const ro = props.doc ? !props.doc.allowed_actions.includes('edit') : false
  const options = useQuery({ ...invoiceOptions(props.order), enabled: !ro })
  if (!ro) {
    if (options.isPending) return <ContentSkeleton />
    if (options.isError && props.doc)
      return (
        <Stack gap="md">
          <Notice tone="danger">{errorText(t, options.error)}</Notice>
          <Fields {...props} ro />
        </Stack>
      )
    if (options.error instanceof ApiError && options.error.status === 403) return <ForbiddenPage />
    if (options.isError) return <ErrorState queries={[options]} />
  }
  const data = options.data && props.doc ? withOwnInvoiceLines(options.data, props.doc.lines) : options.data
  return <Fields {...props} ro={ro} options={data} />
}

function Fields({ order, doc, onCreated, ro, options }: Props & { ro: boolean; options?: InvoiceOptions }) {
  const { t } = useTranslation()
  const me = useMe()
  const vat = useVatOptions()
  const mutation = useDocumentMutation(invoiceDocType, doc?.id ?? 0, doc?.version ?? 0)
  // One id per opened form: sending it twice returns the invoice made the first time.
  const [requestId] = useState(() => crypto.randomUUID())
  const start = doc ? doc.lines.map(toValues) : options ? invoiceLinesFromOptions(options).map(toValues) : []
  const form = useForm<Values>({
    mode: 'onBlur',
    defaultValues: {
      date: doc?.date ?? today(me.timezone),
      supplier_invoice_symbol: doc?.supplier_invoice_symbol ?? '',
      supplier_invoice_no: doc?.supplier_invoice_no ?? '',
      supplier_invoice_date: doc?.supplier_invoice_date ?? null,
      note: doc?.note ?? '',
      lines: start.length ? start : [{ ...emptyLine }],
    },
  })
  const lines = form.watch('lines')
  const orderLines = new Map((options?.lines ?? []).map((l) => [String(l.order_position), l]))
  const lineOf = (i: number) => orderLines.get(lines[i]?.order_position ?? '')

  async function save(v: Values) {
    const body = {
      date: v.date ?? '',
      supplier_invoice_symbol: v.supplier_invoice_symbol.trim() || null,
      supplier_invoice_no: v.supplier_invoice_no.trim() || null,
      supplier_invoice_date: v.supplier_invoice_date,
      note: v.note.trim() || null,
      lines: v.lines.map(
        (l): InvoiceLineInput => ({
          order_position: Number(l.order_position),
          quantity: l.quantity,
          unit_price: l.unit_price ?? 0,
          discount_percent: l.discount_percent || '0',
          vat_rate: (l.vat_rate ?? 'none') as InvoiceLineInput['vat_rate'],
        }),
      ),
    }
    if (doc) {
      await mutation.run((version) => unwrap(api.PUT('/purchase/invoices/{id}', { params: { path: { id: doc.id } }, body: { version, ...body } })))
      return
    }
    const { id } = await unwrap(api.POST('/purchase/invoices', { body: { request_id: requestId, order_id: order, ...body } }))
    await onCreated?.(id)
  }

  type Line = Invoice['lines'][number]
  const lineColumns: LineColumn<Line>[] = [
    {
      key: 'order_line',
      header: t('purchase.goods_receipt.line.order_line_col'),
      required: true,
      value: (l) => `${l.item_code} · ${l.description}`,
      cell: (i, bare) => (
        <SelectField
          name={`lines.${i}.order_position`}
          label={t('purchase.goods_receipt.line.order_line', { n: i + 1 })}
          bare={bare}
          required
          data={[...orderLines.values()].map((l) => ({ value: String(l.order_position), label: `${l.item_code} · ${l.description}` }))}
          onChange={(v) => {
            // The order line's price starts the line; the supplier's paper may change it.
            const l = orderLines.get(v ?? '')
            if (!l) return
            form.setValue(`lines.${i}.unit_price`, l.unit_price, { shouldDirty: true })
            form.setValue(`lines.${i}.discount_percent`, l.discount_percent, { shouldDirty: true })
            form.setValue(`lines.${i}.vat_rate`, l.vat_rate, { shouldDirty: true })
          }}
        />
      ),
    },
    {
      key: 'remaining',
      header: t('purchase.invoice.line.remaining'),
      numeric: true,
      width: 128,
      cell: (i, bare) => {
        const l = lineOf(i)
        if (!l) return null
        const quantity = `${formatDecimal(l.remaining, 3)} ${l.unit}`
        return (
          <Text size="sm" tone="dimmed">
            {bare ? quantity : t('purchase.invoice.line.remaining_hint', { quantity })}
          </Text>
        )
      },
    },
    {
      key: 'quantity',
      header: t('purchase.line.quantity_col'),
      numeric: true,
      required: true,
      width: 112,
      value: (l) => `${formatDecimal(l.quantity, 3)} ${l.unit}`,
      cell: (i, bare) => <DecimalField name={`lines.${i}.quantity`} label={t('purchase.line.quantity', { n: i + 1 })} bare={bare} required scale={3} />,
    },
    {
      key: 'unit_price',
      header: t('purchase.line.unit_price_col'),
      numeric: true,
      required: true,
      width: 128,
      value: (l) => formatNumber(l.unit_price),
      cell: (i, bare) => <MoneyField name={`lines.${i}.unit_price`} label={t('purchase.line.unit_price', { n: i + 1 })} bare={bare} required />,
    },
    {
      key: 'discount',
      header: t('purchase.line.discount_percent_col'),
      numeric: true,
      width: 88,
      value: (l) => formatDecimal(l.discount_percent, 2),
      cell: (i, bare) => <DecimalField name={`lines.${i}.discount_percent`} label={t('purchase.line.discount', { n: i + 1 })} bare={bare} scale={2} min={0} />,
    },
    {
      key: 'vat_rate',
      header: t('purchase.line.vat_rate_col'),
      required: true,
      width: 112,
      value: (l) => vat.label(l.vat_rate),
      cell: (i, bare) => <SelectField name={`lines.${i}.vat_rate`} label={t('purchase.line.vat_rate', { n: i + 1 })} bare={bare} required data={vat.options} />,
    },
    ...(ro ? [{ key: 'net', header: t('purchase.invoice.line.net'), numeric: true, width: 144, value: (l: Line) => formatNumber(l.amount - l.discount) }] : []),
  ]
  const totals = doc && (
    <LinesTotals
      note={ro ? undefined : t('purchase.doc.totals_saved')}
      rows={[
        [t('purchase.doc.subtotal'), formatNumber(doc.totals.subtotal)],
        [t('purchase.doc.discount_total'), formatNumber(doc.totals.discount_total)],
        [t('purchase.doc.vat_total'), formatNumber(doc.totals.vat_total)],
        [t('purchase.doc.total'), formatNumber(doc.totals.total)],
      ]}
    />
  )

  return (
    <Form form={form} onSubmit={save} errorFields={errorFields}>
      <FormSection title={t('purchase.invoice.section.general')} description={t('purchase.invoice.section.general_hint')} icon={IconFileInvoice}>
        <DateField name="date" label={t('purchase.invoice.date')} required readOnly={ro} />
        <TextField
          name="supplier_invoice_symbol"
          label={t('purchase.invoice.supplier_invoice_symbol')}
          description={t('purchase.invoice.supplier_ref_hint')}
          maxLength={20}
          readOnly={ro}
        />
        <TextField name="supplier_invoice_no" label={t('purchase.invoice.supplier_invoice_no')} maxLength={20} readOnly={ro} />
        <DateField name="supplier_invoice_date" label={t('purchase.invoice.supplier_invoice_date')} clearable readOnly={ro} />
        <TextField name="note" label={t('purchase.invoice.note')} multiline maxLength={2000} readOnly={ro} />
      </FormSection>
      <FormSection
        title={t('purchase.doc.section.lines')}
        description={t(ro ? 'purchase.invoice.section.lines_hint_ro' : 'purchase.invoice.section.lines_hint')}
        icon={IconListDetails}
        wide
      >
        <FormRow>
          {ro && doc ? (
            <LinesTable readOnly label={t('purchase.doc.section.lines')} columns={lineColumns} lines={doc.lines} lineLabel={(n) => t('purchase.line.title', { n })} footer={totals} />
          ) : (
            <LinesTable
              name="lines"
              label={t('purchase.doc.section.lines')}
              columns={lineColumns}
              newLine={() => ({ ...emptyLine })}
              addLabel={t('purchase.line.add')}
              lineLabel={(n) => t('purchase.line.title', { n })}
              removeLabel={(n) => t('purchase.line.remove', { n })}
              minLines={1}
              footer={totals}
            />
          )}
        </FormRow>
      </FormSection>
      {!ro && <FormActions submitLabel={t(doc ? 'purchase.common.save' : 'purchase.common.create_draft')} cancelTo={doc ? undefined : `/purchase/orders/${order}`} />}
      {mutation.dialog}
    </Form>
  )
}
