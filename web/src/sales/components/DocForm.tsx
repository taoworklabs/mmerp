import { ActionIcon, Button, Group, Stack, Text, Tooltip } from '@mantine/core'
import { IconFileDollar, IconListDetails, IconNotes, IconPlus, IconTrash } from '@tabler/icons-react'
import { useState } from 'react'
import { useFieldArray, useForm, type Path } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Doc, type DocFields, type LineInput } from '@/shared/api/sales'
import { useDocumentMutation } from '@/shared/document'
import { formatDecimal, formatNumber } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { FieldList } from '@/shared/ui/FieldList'
import { DateField, DecimalField, Form, FormActions, FormRow, FormSection, InputRow, MoneyField, OrgUnitField, SelectField, TextField } from '@/shared/ui/form'
import { icon } from '@/shared/ui/theme'
import { docTypeOf, type Kind } from '../keys'
import { CustomerField, ItemField } from './pickers'
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

const lineFields: Record<string, keyof LineValues> = {
  invalid_quantity: 'quantity',
  line_amount_too_large: 'quantity',
  invalid_discount: 'discount_percent',
  invalid_vat_rate: 'vat_rate',
  item_inactive: 'item_id',
}

const docFields: Record<string, Path<Values>> = {
  customer_inactive: 'customer_id',
  quote_valid_until_before_date: 'valid_until',
  order_delivery_before_date: 'delivery_date',
}

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
  const lines = useFieldArray({ control: form.control, name: 'lines' })
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

  function fieldOf(err: { code: string; params: Record<string, unknown> }): Path<Values> | undefined {
    const f = lineFields[err.code]
    if (f && typeof err.params.line === 'number') return `lines.${err.params.line - 1}.${f}` as Path<Values>
    return docFields[err.code]
  }

  return (
    <Form form={form} onSubmit={save} fieldOf={fieldOf}>
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
      >
        {ro && doc ? (
          <FormRow>
            <LinesTable doc={doc} />
          </FormRow>
        ) : (
          <>
            {lines.fields.map((line, i) => (
              <LineFields
                key={line.id}
                index={i}
                current={
                  doc?.lines[i] && String(doc.lines[i].item_id) === line.item_id
                    ? { id: doc.lines[i].item_id, code: doc.lines[i].item_code, name: doc.lines[i].description }
                    : undefined
                }
                vatOptions={vat.options}
                removable={lines.fields.length > 1}
                onRemove={() => lines.remove(i)}
                onPicked={(it) => {
                  form.setValue(`lines.${i}.description`, it.name, { shouldDirty: true })
                  form.setValue(`lines.${i}.unit_price`, it.price, { shouldDirty: true })
                  form.setValue(`lines.${i}.vat_rate`, it.vat_rate, { shouldDirty: true })
                }}
              />
            ))}
            <Group>
              <Button size="xs" variant="light" leftSection={<IconPlus {...icon.text} />} onClick={() => lines.append({ ...emptyLine })}>
                {t('sales.line.add')}
              </Button>
            </Group>
          </>
        )}
        {doc && <Totals doc={doc} saved={!ro} />}
      </FormSection>
      <FormSection title={t('sales.doc.section.terms')} description={t('sales.doc.section.terms_hint')} icon={IconNotes}>
        <TextField name="payment_terms" label={t('sales.doc.payment_terms')} maxLength={1000} readOnly={ro} />
        <TextField name="delivery_terms" label={t('sales.doc.delivery_terms')} maxLength={1000} readOnly={ro} />
        <TextField name="note" label={t('sales.doc.note')} maxLength={2000} multiline readOnly={ro} />
      </FormSection>
      {editable && <FormActions submitLabel={t(doc ? 'sales.common.save' : 'sales.common.create_draft')} />}
      {mutation.dialog}
    </Form>
  )
}

function LineFields({
  index,
  current,
  vatOptions,
  removable,
  onRemove,
  onPicked,
}: {
  index: number
  current?: { id: number; code: string; name: string }
  vatOptions: { value: string; label: string }[]
  removable: boolean
  onRemove: () => void
  onPicked: (it: { name: string; price: number; vat_rate: string }) => void
}) {
  const { t } = useTranslation()
  const n = index + 1
  return (
    <>
      <ItemField name={`lines.${index}.item_id`} label={t('sales.line.item', { n })} current={current} onPicked={onPicked} />
      <TextField name={`lines.${index}.description`} label={t('sales.line.description', { n })} required maxLength={500} />
      <DecimalField name={`lines.${index}.quantity`} label={t('sales.line.quantity', { n })} required scale={3} min={0} />
      <MoneyField name={`lines.${index}.unit_price`} label={t('sales.line.unit_price', { n })} required />
      <DecimalField name={`lines.${index}.discount_percent`} label={t('sales.line.discount', { n })} scale={2} min={0} />
      <InputRow>
        <SelectField name={`lines.${index}.vat_rate`} label={t('sales.line.vat_rate', { n })} required data={vatOptions} />
        {removable && (
          <Tooltip label={t('sales.line.remove', { n })}>
            <ActionIcon variant="subtle" color="danger" size="lg" aria-label={t('sales.line.remove', { n })} onClick={onRemove}>
              <IconTrash {...icon.button} />
            </ActionIcon>
          </Tooltip>
        )}
      </InputRow>
    </>
  )
}

type LineRow = Doc['lines'][number] & { n: number }

// LinesTable shows the lines of a document that is not edited, with their amounts.
export function LinesTable({ doc }: { doc: Doc }) {
  const { t } = useTranslation()
  const vat = useVatOptions()
  const columns: Column<LineRow>[] = [
    { key: 'description', header: t('sales.line.description_col'), role: 'title', render: (l) => l.description },
    { key: 'quantity', header: t('sales.line.quantity_col'), role: 'meta', numeric: true, render: (l) => `${formatDecimal(l.quantity, 3)} ${l.unit}` },
    { key: 'unit_price', header: t('sales.line.unit_price_col'), role: 'meta', numeric: true, render: (l) => formatNumber(l.unit_price) },
    { key: 'discount', header: t('sales.line.discount_col'), role: 'hidden', numeric: true, render: (l) => (l.discount ? formatNumber(l.discount) : '–') },
    { key: 'vat_rate', header: t('sales.line.vat_rate_col'), role: 'hidden', render: (l) => vat.label(l.vat_rate) },
    { key: 'net', header: t('sales.line.net_col'), role: 'meta', numeric: true, strong: true, render: (l) => formatNumber(l.amount - l.discount) },
  ]
  return <DataTable label={t('sales.doc.section.lines')} columns={columns} rows={doc.lines.map((l, i) => ({ ...l, n: i + 1 }))} rowKey={(l) => l.n} />
}

// Totals are the stored amounts; on a draft being edited they are those of the last save.
function Totals({ doc, saved }: { doc: Doc; saved: boolean }) {
  const { t } = useTranslation()
  return (
    <FormRow>
      <Stack gap="xs">
        {saved && (
          <Text size="sm" c="dimmed">
            {t('sales.doc.totals_saved')}
          </Text>
        )}
        <FieldList
          rows={[
            [t('sales.doc.subtotal'), formatNumber(doc.subtotal)],
            [t('sales.doc.discount_total'), formatNumber(doc.discount_total)],
            [t('sales.doc.vat_total'), formatNumber(doc.vat_total)],
            [t('sales.doc.total'), formatNumber(doc.total)],
          ]}
        />
      </Stack>
    </FormRow>
  )
}
