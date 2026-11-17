import { Stack, Text } from '@mmerp/ui'
import { IconFileInvoice, IconListDetails } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, ApiError, unwrap, type Invoice, type InvoiceLineInput, type InvoiceOptions } from '@/shared/api/sales'
import { useMe } from '@/shared/auth/me'
import { useDocumentMutation } from '@/shared/document'
import { errorText, formatDecimal, formatNumber, today } from '@/shared/i18n'
import { DateField, DecimalField, Form, FormActions, FormRow, FormSection, LinesTable, LinesTotals, SelectField, TextField, type ErrorFields, type LineColumn } from '@/shared/ui/form'
import { Notice } from '@/shared/ui/Notice'
import { ContentSkeleton, ErrorState, ForbiddenPage } from '@/shared/ui/states'
import { invoiceLinesFromOptions, withOwnInvoiceLines } from '../invoiceLines'
import { invoiceOptions } from '../hooks/useSales'
import { invoiceDocType } from '../keys'
import { useVatOptions } from './vat'

type LineValues = { order_position: string | null; quantity: string }

type Values = {
  date: string | null
  note: string
  lines: LineValues[]
}

const emptyLine: LineValues = { order_position: null, quantity: '' }
const toValues = (l: InvoiceLineInput): LineValues => ({ order_position: String(l.order_position), quantity: l.quantity })

const errorFields: ErrorFields<Values> = {
  sales_invoice_before_order: 'date',
  sales_invoice_position: 'lines.*.order_position',
  sales_invoice_over_ordered: 'lines.*.quantity',
  invalid_quantity: 'lines.*.quantity',
}

type Props = {
  order: number
  doc?: Invoice
  // A new invoice: called with its id once created.
  onCreated?: (id: number) => Promise<void>
}

// InvoiceForm bills a posted order, or edits a draft invoice whose allowed_actions has edit;
// otherwise it shows the invoice read-only. Lines start from what is left to bill; prices,
// discounts and VAT rates are the order's and amounts are the server's.
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
    defaultValues: { date: doc?.date ?? today(me.timezone), note: doc?.note ?? '', lines: start.length ? start : [{ ...emptyLine }] },
  })
  const lines = form.watch('lines')
  const orderLines = new Map((options?.lines ?? []).map((l) => [String(l.order_position), l]))
  const lineOf = (i: number) => orderLines.get(lines[i]?.order_position ?? '')

  async function save(v: Values) {
    const body = {
      date: v.date ?? '',
      note: v.note.trim() || null,
      lines: v.lines.map((l): InvoiceLineInput => ({ order_position: Number(l.order_position), quantity: l.quantity })),
    }
    if (doc) {
      await mutation.run((version) => unwrap(api.PUT('/sales/invoices/{id}', { params: { path: { id: doc.id } }, body: { version, ...body } })))
      return
    }
    const { id } = await unwrap(api.POST('/sales/invoices', { body: { request_id: requestId, order_id: order, ...body } }))
    await onCreated?.(id)
  }

  type Line = Invoice['lines'][number]
  const priceOf = (i: number) => {
    const l = lineOf(i)
    return l && `${formatNumber(l.unit_price)}${l.discount_percent !== '0' ? ` −${formatDecimal(l.discount_percent, 2)}%` : ''} · ${vat.label(l.vat_rate)}`
  }
  const lineColumns: LineColumn<Line>[] = [
    {
      key: 'order_line',
      header: t('sales.delivery.line.order_line_col'),
      required: true,
      value: (l) => `${l.item_code} · ${l.description}`,
      cell: (i, bare) => (
        <SelectField
          name={`lines.${i}.order_position`}
          label={t('sales.delivery.line.order_line', { n: i + 1 })}
          bare={bare}
          required
          data={[...orderLines.values()].map((l) => ({ value: String(l.order_position), label: `${l.item_code} · ${l.description}` }))}
        />
      ),
    },
    {
      key: 'remaining',
      header: t('sales.invoice.line.remaining'),
      numeric: true,
      width: 128,
      cell: (i, bare) => {
        const l = lineOf(i)
        if (!l) return null
        const quantity = `${formatDecimal(l.remaining, 3)} ${l.unit}`
        return (
          <Text size="sm" tone="dimmed">
            {bare ? quantity : t('sales.invoice.line.remaining_hint', { quantity })}
          </Text>
        )
      },
    },
    {
      key: 'price',
      header: t('sales.invoice.line.price'),
      numeric: true,
      width: 176,
      value: (l) => `${formatNumber(l.unit_price)} · ${vat.label(l.vat_rate)}`,
      cell: (i) => (
        <Text size="sm" tone="dimmed">
          {priceOf(i)}
        </Text>
      ),
    },
    {
      key: 'quantity',
      header: t('sales.line.quantity_col'),
      numeric: true,
      required: true,
      width: 112,
      value: (l) => `${formatDecimal(l.quantity, 3)} ${l.unit}`,
      cell: (i, bare) => <DecimalField name={`lines.${i}.quantity`} label={t('sales.line.quantity', { n: i + 1 })} bare={bare} required scale={3} />,
    },
    ...(ro ? [{ key: 'net', header: t('sales.invoice.line.net'), numeric: true, width: 144, value: (l: Line) => formatNumber(l.amount - l.discount) }] : []),
  ]
  const totals = doc && (
    <LinesTotals
      note={ro ? undefined : t('sales.doc.totals_saved')}
      rows={[
        [t('sales.doc.subtotal'), formatNumber(doc.totals.subtotal)],
        [t('sales.doc.discount_total'), formatNumber(doc.totals.discount_total)],
        [t('sales.doc.vat_total'), formatNumber(doc.totals.vat_total)],
        [t('sales.doc.total'), formatNumber(doc.totals.total)],
      ]}
    />
  )

  return (
    <Form form={form} onSubmit={save} errorFields={errorFields}>
      <FormSection title={t('sales.invoice.section.general')} description={t('sales.invoice.section.general_hint')} icon={IconFileInvoice}>
        <DateField name="date" label={t('sales.invoice.date')} required readOnly={ro} />
        <TextField name="note" label={t('sales.invoice.note')} multiline maxLength={2000} readOnly={ro} />
      </FormSection>
      <FormSection
        title={t('sales.doc.section.lines')}
        description={t(ro ? 'sales.invoice.section.lines_hint_ro' : 'sales.invoice.section.lines_hint')}
        icon={IconListDetails}
        wide
      >
        <FormRow>
          {ro && doc ? (
            <LinesTable readOnly label={t('sales.doc.section.lines')} columns={lineColumns} lines={doc.lines} lineLabel={(n) => t('sales.line.title', { n })} footer={totals} />
          ) : (
            <LinesTable
              name="lines"
              label={t('sales.doc.section.lines')}
              columns={lineColumns}
              newLine={() => ({ ...emptyLine })}
              addLabel={t('sales.line.add')}
              lineLabel={(n) => t('sales.line.title', { n })}
              removeLabel={(n) => t('sales.line.remove', { n })}
              minLines={1}
              footer={totals}
            />
          )}
        </FormRow>
      </FormSection>
      {!ro && <FormActions submitLabel={t(doc ? 'sales.common.save' : 'sales.common.create_draft')} cancelTo={doc ? undefined : `/sales/orders/${order}`} />}
      {mutation.dialog}
    </Form>
  )
}
