import { IconAdjustmentsHorizontal, IconListDetails } from '@tabler/icons-react'
import { useState } from 'react'
import { useForm, type Path } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, ApiError, unwrap, type Adjustment, type AdjustmentLine } from '@/shared/api/core'
import type { LineItem } from '@/shared/api/sales'
import { useDocumentMutation } from '@/shared/document'
import { errorText, formatDate, formatDecimal } from '@/shared/i18n'
import { DateField, DecimalField, Form, FormActions, FormRow, FormSection, LinesTable, SelectField, TextField, type ErrorFields, type LineColumn } from '@/shared/ui/form'
import { adjustmentDocType } from '../keys'
import { useWarehouseOptions } from '../hooks/useStock'
import { ItemField } from './pickers'

type Tracking = LineItem['tracking']

type LineValues = {
  item_id: string | null
  // From the picked item; null until one is picked.
  tracking: Tracking | null
  lot_code: string
  expiry: string | null
  serial: string
  quantity: string
}

type Values = {
  date: string | null
  warehouse_id: string | null
  reason: string
  lines: LineValues[]
}

const emptyLine: LineValues = { item_id: null, tracking: null, lot_code: '', expiry: null, serial: '', quantity: '' }

// A saved line does not carry its item's tracking; its lot or serial tells it.
const trackingOf = (l: AdjustmentLine): Tracking => (l.lot ? 'lot' : l.serial ? 'serial' : 'quantity')

function defaults(doc?: Adjustment): Values {
  return {
    date: doc?.date ?? null,
    warehouse_id: doc ? String(doc.warehouse_id) : null,
    reason: doc?.reason ?? '',
    lines: doc
      ? doc.lines.map((l) => ({
          item_id: String(l.item_id),
          tracking: trackingOf(l),
          lot_code: l.lot?.code ?? '',
          expiry: l.lot?.expiry ?? null,
          serial: l.serial ?? '',
          quantity: l.quantity,
        }))
      : [{ ...emptyLine }],
  }
}

// Only stock coming in can name a lot's expiry; a lot going out already has one.
const incoming = (quantity: string) => !quantity.trim().startsWith('-')

// adjustmentLineErrors are the error codes a line would get at posting for its item's tracking,
// checked before sending: a lot item needs a lot, a serial item a serial and one unit in or out.
export function adjustmentLineErrors(line: Pick<LineValues, 'lot_code' | 'serial' | 'quantity'>, tracking: Tracking): string[] {
  switch (tracking) {
    case 'none':
      return ['stock_untracked']
    case 'lot':
      return line.lot_code.trim() ? [] : ['stock_lot_required']
    case 'serial':
      if (!line.serial.trim()) return ['stock_serial_required']
      return /^-?1(\.0*)?$/.test(line.quantity.trim()) ? [] : ['stock_serial_quantity']
    default:
      return []
  }
}

const errorFields: ErrorFields<Values> = {
  stock_reason_required: 'reason',
  stock_warehouse_inactive: 'warehouse_id',
  stock_quantity_invalid: 'lines.*.quantity',
  stock_serial_quantity: 'lines.*.quantity',
  stock_untracked: 'lines.*.item_id',
  stock_lot_not_tracked: 'lines.*.item_id',
  stock_lot_required: 'lines.*.lot_code',
  stock_lot_expiry_mismatch: 'lines.*.expiry',
  stock_serial_required: 'lines.*.serial',
}

type Line = Adjustment['lines'][number]

type Props = {
  doc?: Adjustment
  // A new adjustment: called with its id once created.
  onCreated?: (id: number) => Promise<void>
}

// AdjustmentForm creates a stock adjustment, or edits one whose allowed_actions has edit;
// otherwise it shows it read-only. A positive quantity brings stock in, a negative one takes it out.
export function AdjustmentForm({ doc, onCreated }: Props) {
  const { t } = useTranslation()
  const mutation = useDocumentMutation(adjustmentDocType, doc?.id ?? 0, doc?.version ?? 0)
  // One id per opened form: sending it twice returns the adjustment made the first time.
  const [requestId] = useState(() => crypto.randomUUID())
  const form = useForm<Values>({ mode: 'onBlur', defaultValues: defaults(doc) })
  const ro = doc ? !doc.allowed_actions.includes('edit') : false
  const lines = form.watch('lines')
  const options = useWarehouseOptions()
  // A retired warehouse is no longer offered, but the saved one still shows.
  const warehouses =
    doc && !options.some((o) => o.value === String(doc.warehouse_id))
      ? [{ value: String(doc.warehouse_id), label: `${doc.warehouse_code} · ${doc.warehouse_name}` }, ...options]
      : options

  async function save(v: Values) {
    const errors = v.lines.flatMap((l, i) => (l.tracking ? adjustmentLineErrors(l, l.tracking).map((code) => new ApiError(422, code, { line_no: i + 1 })) : []))
    if (errors.length) {
      errors.forEach((err, n) => {
        const field = String(errorFields[err.code]).replace('*', String(Number(err.params.line_no) - 1)) as Path<Values>
        form.setError(field, { message: errorText(t, err) }, { shouldFocus: n === 0 })
      })
      return false
    }
    const body = {
      date: v.date ?? '',
      warehouse_id: Number(v.warehouse_id),
      reason: v.reason.trim(),
      lines: v.lines.map(
        (l): AdjustmentLine => ({
          item_id: Number(l.item_id),
          quantity: l.quantity,
          lot: l.tracking === 'lot' ? { code: l.lot_code.trim(), expiry: (incoming(l.quantity) && l.expiry) || undefined } : undefined,
          serial: l.tracking === 'serial' ? l.serial.trim() : undefined,
        }),
      ),
    }
    if (doc) {
      await mutation.run((version) => unwrap(api.PUT('/stock/adjustments/{id}', { params: { path: { id: doc.id } }, body: { version, ...body } })))
      return
    }
    const { id } = await unwrap(api.POST('/stock/adjustments', { body: { request_id: requestId, ...body } }))
    await onCreated?.(id)
  }

  // Lot and serial cells show only for items tracked that way.
  const lineColumns: LineColumn<Line>[] = [
    {
      key: 'item',
      header: t('sales.line.item_col'),
      required: true,
      width: 240,
      value: (l) => `${l.item_code} · ${l.item_name}`,
      cell: (i, bare, line) => (
        <ItemField
          name={`lines.${i}.item_id`}
          label={t('sales.line.item', { n: i + 1 })}
          bare={bare}
          current={
            doc?.lines[i] && String(doc.lines[i].item_id) === line.item_id ? { id: doc.lines[i].item_id, code: doc.lines[i].item_code, name: doc.lines[i].item_name } : undefined
          }
          onPicked={(it) => {
            form.setValue(`lines.${i}.tracking`, it.tracking)
            form.setValue(`lines.${i}.lot_code`, '')
            form.setValue(`lines.${i}.expiry`, null)
            form.setValue(`lines.${i}.serial`, '')
          }}
        />
      ),
    },
    {
      key: 'lot',
      header: t('sales.on_hand.lot'),
      width: 128,
      value: (l) => l.lot?.code ?? '',
      cell: (i, bare) =>
        lines[i]?.tracking === 'lot' && <TextField name={`lines.${i}.lot_code`} label={t('sales.adjustment.line.lot', { n: i + 1 })} bare={bare} required maxLength={100} />,
    },
    {
      key: 'expiry',
      header: t('sales.on_hand.expiry'),
      width: 144,
      value: (l) => (l.lot?.expiry ? formatDate(l.lot.expiry) : ''),
      cell: (i, bare) =>
        lines[i]?.tracking === 'lot' &&
        incoming(lines[i].quantity) && <DateField name={`lines.${i}.expiry`} label={t('sales.adjustment.line.expiry', { n: i + 1 })} bare={bare} clearable />,
    },
    {
      key: 'serial',
      header: t('sales.on_hand.serial'),
      width: 144,
      value: (l) => l.serial ?? '',
      cell: (i, bare) =>
        lines[i]?.tracking === 'serial' && <TextField name={`lines.${i}.serial`} label={t('sales.adjustment.line.serial', { n: i + 1 })} bare={bare} required maxLength={100} />,
    },
    {
      key: 'quantity',
      header: t('sales.line.quantity_col'),
      numeric: true,
      required: true,
      width: 112,
      value: (l) => `${formatDecimal(l.quantity, 3)} ${l.unit}`,
      cell: (i, bare) => <DecimalField name={`lines.${i}.quantity`} label={t('sales.line.quantity', { n: i + 1 })} bare={bare} required scale={3} allowNegative />,
    },
  ]

  return (
    <Form form={form} onSubmit={save} errorFields={errorFields}>
      <FormSection title={t('sales.adjustment.section.general')} description={t('sales.adjustment.section.general_hint')} icon={IconAdjustmentsHorizontal}>
        <DateField name="date" label={t('sales.adjustment.date')} required readOnly={ro} />
        <SelectField name="warehouse_id" label={t('sales.adjustment.warehouse')} required readOnly={ro} data={warehouses} />
        <TextField name="reason" label={t('sales.adjustment.reason')} required maxLength={500} wide readOnly={ro} />
      </FormSection>
      <FormSection
        title={t('sales.doc.section.lines')}
        description={t(ro ? 'sales.adjustment.section.lines_hint_ro' : 'sales.adjustment.section.lines_hint')}
        icon={IconListDetails}
        wide
      >
        <FormRow>
          {ro && doc ? (
            <LinesTable readOnly label={t('sales.doc.section.lines')} columns={lineColumns} lines={doc.lines} lineLabel={(n) => t('sales.line.title', { n })} />
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
            />
          )}
        </FormRow>
      </FormSection>
      {!ro && <FormActions submitLabel={t(doc ? 'sales.common.save' : 'sales.common.create_draft')} cancelTo={doc ? undefined : '/sales/stock/adjustments'} />}
      {mutation.dialog}
    </Form>
  )
}
