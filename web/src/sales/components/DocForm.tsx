import { IconFileDollar, IconListDetails, IconNotes } from '@tabler/icons-react'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Doc, type DocFields, type LineInput } from '@/shared/api/sales'
import { useDocumentMutation } from '@/shared/document'
import { formatDecimal, formatNumber } from '@/shared/i18n'
import { ItemPicker } from '@/shared/master'
import { DateField, DecimalField, Form, FormActions, FormRow, FormSection, LinesTable, LinesTotals, MoneyField, OrgUnitField, SelectField, TextField, type ErrorFields, type LineColumn } from '@/shared/ui/form'
import { docTypeOf, pathOf, type Kind } from '../keys'
import { CustomerField } from './pickers'
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
  valid_until: string | null
  delivery_date: string | null
  org_unit_id: string | null
  customer_id: string | null
  payment_terms: string
  delivery_terms: string
  note: string
  lines: LineValues[]
}

const emptyLine: LineValues = { item_id: null, item_code: '', description: '', quantity: '1', unit_price: null, discount_percent: '0', vat_rate: null }

function defaults(doc?: Doc): Values {
  return {
    date: doc?.date ?? null,
    valid_until: doc?.valid_until ?? null,
    delivery_date: doc?.delivery_date ?? null,
    org_unit_id: doc ? String(doc.org_unit_id) : null,
    customer_id: doc ? String(doc.customer.id) : null,
    payment_terms: doc?.payment_terms ?? '',
    delivery_terms: doc?.delivery_terms ?? '',
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
  customer_inactive: 'customer_id',
  quote_valid_until_before_date: 'valid_until',
  order_delivery_before_date: 'delivery_date',
  invalid_quantity: 'lines.*.quantity',
  line_amount_too_large: 'lines.*.quantity',
  invalid_discount: 'lines.*.discount_percent',
  invalid_vat_rate: 'lines.*.vat_rate',
  item_inactive: 'lines.*.item_id',
}

type Line = Doc['lines'][number]

type Props = {
  kind: Kind
  doc?: Doc
  // A new document: called with its id once created.
  onCreated?: (id: number) => Promise<void>
}

// DocForm creates a quotation or order, or edits a draft whose allowed_actions has edit;
// otherwise it shows the document read-only. Amounts come from the server only.
export function DocForm({ kind, doc, onCreated }: Props) {
  const { t } = useTranslation()
  const vat = useVatOptions()
  const mutation = useDocumentMutation(docTypeOf(kind), doc?.id ?? 0, doc?.version ?? 0)
  // One id per opened form: sending it twice returns the document made the first time.
  const [requestId] = useState(() => crypto.randomUUID())
  const form = useForm<Values>({ mode: 'onBlur', defaultValues: defaults(doc) })
  const editable = doc ? doc.allowed_actions.includes('edit') : true
  const ro = !editable
  const quote = kind === 'quote'

  async function save(v: Values) {
    const body: DocFields = {
      date: v.date ?? '',
      org_unit_id: Number(v.org_unit_id),
      customer_id: Number(v.customer_id),
      valid_until: quote ? v.valid_until : null,
      delivery_date: quote ? null : v.delivery_date,
      payment_terms: orNull(v.payment_terms),
      delivery_terms: orNull(v.delivery_terms),
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
        unwrap(
          quote
            ? api.PUT('/sales/quotes/{id}', { params: { path: { id: doc.id } }, body: { version, ...body } })
            : api.PUT('/sales/orders/{id}', { params: { path: { id: doc.id } }, body: { version, ...body } }),
        ),
      )
      return
    }
    const { id } = await unwrap(
      quote ? api.POST('/sales/quotes', { body: { request_id: requestId, ...body } }) : api.POST('/sales/orders', { body: { request_id: requestId, ...body } }),
    )
    await onCreated?.(id)
  }

  // The item cell fills description, price and VAT from the catalogue; each stays editable.
  // Read-only lines show the stored amounts: the discount and the net instead of the percentage.
  const lineColumns: LineColumn<Line>[] = [
    {
      key: 'item',
      header: t('sales.line.item_col'),
      required: true,
      width: 200,
      cell: (i, bare, line) => (
        <ItemPicker
          name={`lines.${i}.item_id`}
          label={t('sales.line.item', { n: i + 1 })}
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
      header: t('sales.line.description_col'),
      required: true,
      value: (l) => l.description,
      cell: (i, bare) => <TextField name={`lines.${i}.description`} label={t('sales.line.description', { n: i + 1 })} bare={bare} required maxLength={500} />,
    },
    {
      key: 'quantity',
      header: t('sales.line.quantity_col'),
      numeric: true,
      required: true,
      width: 96,
      value: (l) => `${formatDecimal(l.quantity, 3)} ${l.unit}`,
      cell: (i, bare) => <DecimalField name={`lines.${i}.quantity`} label={t('sales.line.quantity', { n: i + 1 })} bare={bare} required scale={3} min={0} />,
    },
    {
      key: 'unit_price',
      header: t('sales.line.unit_price_col'),
      numeric: true,
      required: true,
      width: 128,
      value: (l) => formatNumber(l.unit_price),
      cell: (i, bare) => <MoneyField name={`lines.${i}.unit_price`} label={t('sales.line.unit_price', { n: i + 1 })} bare={bare} required />,
    },
    {
      key: 'discount',
      header: t(ro ? 'sales.line.discount_col' : 'sales.line.discount_percent_col'),
      numeric: true,
      width: 88,
      value: (l) => (l.discount ? formatNumber(l.discount) : ''),
      cell: (i, bare) => <DecimalField name={`lines.${i}.discount_percent`} label={t('sales.line.discount', { n: i + 1 })} bare={bare} scale={2} min={0} />,
    },
    {
      key: 'vat_rate',
      header: t('sales.line.vat_rate_col'),
      required: true,
      width: 112,
      value: (l) => vat.label(l.vat_rate),
      cell: (i, bare) => <SelectField name={`lines.${i}.vat_rate`} label={t('sales.line.vat_rate', { n: i + 1 })} bare={bare} required data={vat.options} />,
    },
    { key: 'net', header: t('sales.line.net_col'), numeric: true, value: (l) => formatNumber(l.amount - l.discount) },
  ]

  return (
    <Form form={form} onSubmit={save} errorFields={errorFields}>
      <FormSection title={t(`sales.${kind}.section.general`)} description={t(`sales.${kind}.section.general_hint`)} icon={IconFileDollar}>
        <DateField name="date" label={t(`sales.${kind}.date`)} required readOnly={ro} />
        {quote ? (
          <DateField name="valid_until" label={t('sales.quote.valid_until')} description={t('sales.quote.valid_until_hint')} required readOnly={ro} />
        ) : (
          <DateField name="delivery_date" label={t('sales.order.delivery_date')} clearable readOnly={ro} />
        )}
        <OrgUnitField
          name="org_unit_id"
          label={t('sales.doc.org_unit')}
          required
          product="sales"
          permission={`sales.${kind}.${ro ? 'view' : 'edit'}`}
          readOnly={ro}
        />
        <CustomerField
          name="customer_id"
          label={t('sales.doc.customer')}
          readOnly={ro}
          current={doc ? { id: doc.customer.id, code: doc.customer.code, name: doc.customer.name } : undefined}
          onPicked={(c) => {
            if (!form.getValues('payment_terms') && c.payment_terms) form.setValue('payment_terms', c.payment_terms, { shouldDirty: true })
          }}
        />
      </FormSection>
      <FormSection
        title={t('sales.doc.section.lines')}
        description={t(ro ? 'sales.doc.section.lines_hint_ro' : 'sales.doc.section.lines_hint')}
        icon={IconListDetails}
        wide
      >
        <FormRow>
          {ro && doc ? (
            <LinesTable
              readOnly
              label={t('sales.doc.section.lines')}
              columns={lineColumns}
              lines={doc.lines}
              lineLabel={(n) => t('sales.line.title', { n })}
              footer={<Totals doc={doc} saved={false} />}
            />
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
              footer={doc && <Totals doc={doc} saved />}
            />
          )}
        </FormRow>
      </FormSection>
      <FormSection title={t('sales.doc.section.terms')} description={t('sales.doc.section.terms_hint')} icon={IconNotes}>
        <TextField name="payment_terms" label={t('sales.doc.payment_terms')} maxLength={1000} readOnly={ro} />
        <TextField name="delivery_terms" label={t('sales.doc.delivery_terms')} maxLength={1000} readOnly={ro} />
        <TextField name="note" label={t('sales.doc.note')} maxLength={2000} multiline readOnly={ro} />
      </FormSection>
      {editable && <FormActions submitLabel={t(doc ? 'sales.common.save' : 'sales.common.create_draft')} cancelTo={doc ? undefined : `/sales/${pathOf(kind)}`} />}
      {mutation.dialog}
    </Form>
  )
}

// Totals are the stored amounts; on a draft being edited they are those of the last save.
function Totals({ doc, saved }: { doc: Doc; saved: boolean }) {
  const { t } = useTranslation()
  return (
    <LinesTotals
      note={saved ? t('sales.doc.totals_saved') : undefined}
      rows={[
        [t('sales.doc.subtotal'), formatNumber(doc.subtotal)],
        [t('sales.doc.discount_total'), formatNumber(doc.discount_total)],
        [t('sales.doc.vat_total'), formatNumber(doc.vat_total)],
        [t('sales.doc.total'), formatNumber(doc.total)],
      ]}
    />
  )
}
