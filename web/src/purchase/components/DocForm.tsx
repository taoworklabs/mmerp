import { IconFileDollar, IconListDetails, IconNotes } from '@tabler/icons-react'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Order, type OrderFields, type LineInput } from '@/shared/api/purchase'
import { useDocumentMutation } from '@/shared/document'
import { formatDecimal, formatNumber } from '@/shared/i18n'
import { DateField, DecimalField, Form, FormActions, FormRow, FormSection, LinesTable, LinesTotals, MoneyField, OrgUnitField, SelectField, TextField, type ErrorFields, type LineColumn } from '@/shared/ui/form'
import { orderDocType } from '../keys'
import { SupplierField, ItemField } from './pickers'
import { useVatOptions } from './vat'

type LineValues = {
  item_id: string | null
  item_code: string
  description: string
  quantity: string
  unit_price: number | null
  discount_percent: string
  vat_rate: string | null
}

type Values = {
  date: string | null
  expected_date: string | null
  org_unit_id: string | null
  supplier_id: string | null
  payment_terms: string
  note: string
  lines: LineValues[]
}

const emptyLine: LineValues = { item_id: null, item_code: '', description: '', quantity: '1', unit_price: null, discount_percent: '0', vat_rate: null }

function defaults(doc?: Order): Values {
  return {
    date: doc?.date ?? null,
    expected_date: doc?.expected_date ?? null,
    org_unit_id: doc ? String(doc.org_unit_id) : null,
    supplier_id: doc ? String(doc.supplier.id) : null,
    payment_terms: doc?.payment_terms ?? '',
    note: doc?.note ?? '',
    lines: doc
      ? doc.lines.map((l) => ({
          item_id: String(l.item_id),
          item_code: l.item_code,
          description: l.description,
          quantity: l.quantity,
          unit_price: l.unit_price,
          discount_percent: l.discount_percent,
          vat_rate: l.vat_rate,
        }))
      : [{ ...emptyLine }],
  }
}

const orNull = (s: string) => (s.trim() === '' ? null : s.trim())

const errorFields: ErrorFields<Values> = {
  supplier_inactive: 'supplier_id',
  purchase_order_expected_before_date: 'expected_date',
  invalid_quantity: 'lines.*.quantity',
  line_amount_too_large: 'lines.*.quantity',
  invalid_discount: 'lines.*.discount_percent',
  invalid_vat_rate: 'lines.*.vat_rate',
  item_inactive: 'lines.*.item_id',
}

type Line = Order['lines'][number]

type Props = {
  doc?: Order
  // A new document: called with its id once created.
  onCreated?: (id: number) => Promise<void>
}

// DocForm creates a purchase order, or edits a draft whose allowed_actions has edit; otherwise it
// shows the order read-only. Amounts come from the server only.
export function DocForm({ doc, onCreated }: Props) {
  const { t } = useTranslation()
  const vat = useVatOptions()
  const mutation = useDocumentMutation(orderDocType, doc?.id ?? 0, doc?.version ?? 0)
  // One id per opened form: sending it twice returns the document made the first time.
  const [requestId] = useState(() => crypto.randomUUID())
  const form = useForm<Values>({ mode: 'onBlur', defaultValues: defaults(doc) })
  const editable = doc ? doc.allowed_actions.includes('edit') : true
  const ro = !editable

  async function save(v: Values) {
    const body: OrderFields = {
      date: v.date ?? '',
      org_unit_id: Number(v.org_unit_id),
      supplier_id: Number(v.supplier_id),
      expected_date: v.expected_date,
      payment_terms: orNull(v.payment_terms),
      note: orNull(v.note),
      lines: v.lines.map((l): LineInput => ({
        item_id: Number(l.item_id),
        description: l.description.trim(),
        quantity: l.quantity,
        unit_price: l.unit_price ?? 0,
        discount_percent: l.discount_percent || '0',
        vat_rate: (l.vat_rate ?? 'none') as LineInput['vat_rate'],
      })),
    }
    if (doc) {
      await mutation.run((version) =>
        unwrap(api.PUT('/purchase/orders/{id}', { params: { path: { id: doc.id } }, body: { version, ...body } })),
      )
      return
    }
    const { id } = await unwrap(api.POST('/purchase/orders', { body: { request_id: requestId, ...body } }))
    await onCreated?.(id)
  }

  // The item cell fills description, price and VAT from the catalogue; each stays editable.
  // Read-only lines show the stored amounts: the discount and the net instead of the percentage.
  const lineColumns: LineColumn<Line>[] = [
    {
      key: 'item',
      header: t('purchase.line.item_col'),
      required: true,
      width: 200,
      cell: (i, bare, line) => (
        <ItemField
          name={`lines.${i}.item_id`}
          label={t('purchase.line.item', { n: i + 1 })}
          bare={bare}
          current={
            doc?.lines[i] && String(doc.lines[i].item_id) === line.item_id ? { id: doc.lines[i].item_id, code: doc.lines[i].item_code, name: doc.lines[i].description } : undefined
          }
          onPicked={(it) => {
            form.setValue(`lines.${i}.description`, it.name, { shouldDirty: true })
            form.setValue(`lines.${i}.unit_price`, it.price, { shouldDirty: true })
            form.setValue(`lines.${i}.vat_rate`, it.vat_rate, { shouldDirty: true })
          }}
        />
      ),
    },
    {
      key: 'description',
      header: t('purchase.line.description_col'),
      required: true,
      value: (l) => l.description,
      cell: (i, bare) => <TextField name={`lines.${i}.description`} label={t('purchase.line.description', { n: i + 1 })} bare={bare} required maxLength={500} />,
    },
    {
      key: 'quantity',
      header: t('purchase.line.quantity_col'),
      numeric: true,
      required: true,
      width: 96,
      value: (l) => `${formatDecimal(l.quantity, 3)} ${l.unit}`,
      cell: (i, bare) => <DecimalField name={`lines.${i}.quantity`} label={t('purchase.line.quantity', { n: i + 1 })} bare={bare} required scale={3} min={0} />,
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
      header: t(ro ? 'purchase.line.discount_col' : 'purchase.line.discount_percent_col'),
      numeric: true,
      width: 88,
      value: (l) => (l.discount ? formatNumber(l.discount) : ''),
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
    { key: 'net', header: t('purchase.line.net_col'), numeric: true, value: (l) => formatNumber(l.amount - l.discount) },
    // Only stock lines carry what was received.
    ...(ro && doc?.status === 'posted' ? [{ key: 'received', header: t('purchase.line.received_col'), numeric: true, value: (l: Line) => l.received && formatDecimal(l.received, 3) }] : []),
    ...(ro && doc?.status === 'posted' ? [{ key: 'invoiced', header: t('purchase.line.invoiced_col'), numeric: true, value: (l: Line) => l.invoiced && formatDecimal(l.invoiced, 3) }] : []),
  ]

  return (
    <Form form={form} onSubmit={save} errorFields={errorFields}>
      <FormSection title={t('purchase.order.section.general')} description={t('purchase.order.section.general_hint')} icon={IconFileDollar}>
        <DateField name="date" label={t('purchase.order.date')} required readOnly={ro} />
        <DateField name="expected_date" label={t('purchase.order.expected_date')} clearable readOnly={ro} />
        <OrgUnitField
          name="org_unit_id"
          label={t('purchase.doc.org_unit')}
          required
          product="purchase"
          permission={`purchase.order.${ro ? 'view' : 'edit'}`}
          readOnly={ro}
        />
        <SupplierField
          name="supplier_id"
          label={t('purchase.doc.supplier')}
          readOnly={ro}
          current={doc ? { id: doc.supplier.id, code: doc.supplier.code, name: doc.supplier.name } : undefined}
          onPicked={(c) => {
            if (!form.getValues('payment_terms') && c.payment_terms) form.setValue('payment_terms', c.payment_terms, { shouldDirty: true })
          }}
        />
      </FormSection>
      <FormSection
        title={t('purchase.doc.section.lines')}
        description={t(ro ? 'purchase.doc.section.lines_hint_ro' : 'purchase.doc.section.lines_hint')}
        icon={IconListDetails}
        wide
      >
        <FormRow>
          {ro && doc ? (
            <LinesTable
              readOnly
              label={t('purchase.doc.section.lines')}
              columns={lineColumns}
              lines={doc.lines}
              lineLabel={(n) => t('purchase.line.title', { n })}
              footer={<Totals doc={doc} saved={false} />}
            />
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
              footer={doc && <Totals doc={doc} saved />}
            />
          )}
        </FormRow>
      </FormSection>
      <FormSection title={t('purchase.doc.section.terms')} description={t('purchase.doc.section.terms_hint')} icon={IconNotes}>
        <TextField name="payment_terms" label={t('purchase.doc.payment_terms')} maxLength={1000} readOnly={ro} />
        <TextField name="note" label={t('purchase.doc.note')} maxLength={2000} multiline readOnly={ro} />
      </FormSection>
      {editable && <FormActions submitLabel={t(doc ? 'purchase.common.save' : 'purchase.common.create_draft')} cancelTo={doc ? undefined : '/purchase/orders'} />}
      {mutation.dialog}
    </Form>
  )
}

// Totals are the stored amounts; on a draft being edited they are those of the last save.
function Totals({ doc, saved }: { doc: Order; saved: boolean }) {
  const { t } = useTranslation()
  return (
    <LinesTotals
      note={saved ? t('purchase.doc.totals_saved') : undefined}
      rows={[
        [t('purchase.doc.subtotal'), formatNumber(doc.subtotal)],
        [t('purchase.doc.discount_total'), formatNumber(doc.discount_total)],
        [t('purchase.doc.vat_total'), formatNumber(doc.vat_total)],
        [t('purchase.doc.total'), formatNumber(doc.total)],
      ]}
    />
  )
}
