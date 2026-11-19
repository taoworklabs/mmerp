import { IconArrowsExchange, IconListDetails } from '@tabler/icons-react'
import { useState } from 'react'
import { useForm, type Path } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, ApiError, unwrap, type StockItem, type Transfer, type TransferLine } from '@/shared/api/core'
import { useDocumentMutation } from '@/shared/document'
import { errorText, formatDecimal } from '@/shared/i18n'
import { DateField, DecimalField, Form, FormActions, FormRow, FormSection, LinesTable, SelectField, TextField, type ErrorFields, type LineColumn } from '@/shared/ui/form'
import { useWarehouseOptions } from '../hooks/useStock'
import { transferDocType } from '../keys'
import { StockItemField } from './pickers'

type Tracking = StockItem['tracking']

type LineValues = {
  item_id: string | null
  // From the picked item; null until one is picked.
  tracking: Tracking | null
  lot_code: string
  serial: string
  quantity: string
}

type Values = {
  date: string | null
  from_warehouse_id: string | null
  to_warehouse_id: string | null
  note: string
  lines: LineValues[]
}

const emptyLine: LineValues = { item_id: null, tracking: null, lot_code: '', serial: '', quantity: '' }

// A saved line does not carry its item's tracking; its lot or serial tells it.
const trackingOf = (l: TransferLine): Tracking => (l.lot ? 'lot' : l.serial ? 'serial' : 'quantity')

function defaults(doc?: Transfer): Values {
  return {
    date: doc?.date ?? null,
    from_warehouse_id: doc ? String(doc.from_warehouse_id) : null,
    to_warehouse_id: doc ? String(doc.to_warehouse_id) : null,
    note: doc?.note ?? '',
    lines: doc
      ? doc.lines.map((l) => ({ item_id: String(l.item_id), tracking: trackingOf(l), lot_code: l.lot ?? '', serial: l.serial ?? '', quantity: l.quantity }))
      : [{ ...emptyLine }],
  }
}

// transferLineErrors are the error codes a line would get at posting for its item's tracking,
// checked before sending: a lot item needs a lot, a serial item a serial moving one unit.
export function transferLineErrors(line: LineValues): string[] {
  switch (line.tracking) {
    case 'lot':
      return line.lot_code.trim() ? [] : ['stock_lot_required']
    case 'serial':
      if (!line.serial.trim()) return ['stock_serial_required']
      return /^1(\.0*)?$/.test(line.quantity.trim()) ? [] : ['stock_serial_quantity']
    default:
      return []
  }
}

// transferBody is what the API takes for a transfer's values.
export function transferBody(v: Values) {
  return {
    date: v.date ?? '',
    from_warehouse_id: Number(v.from_warehouse_id),
    to_warehouse_id: Number(v.to_warehouse_id),
    note: v.note.trim(),
    lines: v.lines.map(
      (l): TransferLine => ({
        item_id: Number(l.item_id),
        quantity: l.quantity,
        lot: l.tracking === 'lot' ? l.lot_code.trim() : undefined,
        serial: l.tracking === 'serial' ? l.serial.trim() : undefined,
      }),
    ),
  }
}

const errorFields: ErrorFields<Values> = {
  stock_transfer_same_warehouse: 'to_warehouse_id',
  stock_warehouse_entity: 'to_warehouse_id',
  stock_warehouse_inactive: 'from_warehouse_id',
  stock_quantity_invalid: 'lines.*.quantity',
  stock_serial_quantity: 'lines.*.quantity',
  stock_insufficient: 'lines.*.quantity',
  stock_untracked: 'lines.*.item_id',
  stock_lot_not_tracked: 'lines.*.item_id',
  stock_lot_required: 'lines.*.lot_code',
  stock_lot_not_found: 'lines.*.lot_code',
  stock_serial_required: 'lines.*.serial',
  stock_serial_not_found: 'lines.*.serial',
}

type Line = Transfer['lines'][number]

type Props = {
  doc?: Transfer
  // A new transfer: called with its id once created.
  onCreated?: (id: number) => Promise<void>
}

// TransferForm creates a transfer, or edits one whose allowed_actions has edit; otherwise it
// shows it read-only. Each line leaves the source warehouse and enters the destination.
export function TransferForm({ doc, onCreated }: Props) {
  const { t } = useTranslation()
  const mutation = useDocumentMutation(transferDocType, doc?.id ?? 0, doc?.version ?? 0)
  // One id per opened form: sending it twice returns the transfer made the first time.
  const [requestId] = useState(() => crypto.randomUUID())
  const form = useForm<Values>({ mode: 'onBlur', defaultValues: defaults(doc) })
  const ro = doc ? !doc.allowed_actions.includes('edit') : false
  const lines = form.watch('lines')
  const options = useWarehouseOptions()
  // A retired warehouse is no longer offered, but a saved one still shows.
  const withSaved = (id: number | undefined, code: string | undefined, name: string | undefined) =>
    id && !options.some((o) => o.value === String(id)) ? [{ value: String(id), label: `${code} · ${name}` }, ...options] : options

  async function save(v: Values) {
    const errors = v.lines.flatMap((l, i) => transferLineErrors(l).map((code) => new ApiError(422, code, { line_no: i + 1 })))
    if (errors.length) {
      errors.forEach((err, n) => {
        const field = String(errorFields[err.code]).replace('*', String(Number(err.params.line_no) - 1)) as Path<Values>
        form.setError(field, { message: errorText(t, err) }, { shouldFocus: n === 0 })
      })
      return false
    }
    const body = transferBody(v)
    if (doc) {
      await mutation.run((version) => unwrap(api.PUT('/stock/transfers/{id}', { params: { path: { id: doc.id } }, body: { version, ...body } })))
      return
    }
    const { id } = await unwrap(api.POST('/stock/transfers', { body: { request_id: requestId, ...body } }))
    await onCreated?.(id)
  }

  // Lot and serial cells show only for items tracked that way.
  const lineColumns: LineColumn<Line>[] = [
    {
      key: 'item',
      header: t('stock.line.item_col'),
      required: true,
      value: (l) => `${l.item_code} · ${l.item_name}`,
      cell: (i, bare, line) => (
        <StockItemField
          name={`lines.${i}.item_id`}
          label={t('stock.line.item', { n: i + 1 })}
          bare={bare}
          current={
            doc?.lines[i] && String(doc.lines[i].item_id) === line.item_id ? { id: doc.lines[i].item_id, code: doc.lines[i].item_code, name: doc.lines[i].item_name } : undefined
          }
          onPicked={(it) => {
            form.setValue(`lines.${i}.tracking`, it.tracking)
            form.setValue(`lines.${i}.lot_code`, '')
            form.setValue(`lines.${i}.serial`, '')
          }}
        />
      ),
    },
    {
      key: 'lot',
      header: t('stock.on_hand.lot'),
      width: 144,
      value: (l) => l.lot ?? '',
      cell: (i, bare) =>
        lines[i]?.tracking === 'lot' && <TextField name={`lines.${i}.lot_code`} label={t('stock.adjustment.line.lot', { n: i + 1 })} bare={bare} required maxLength={100} />,
    },
    {
      key: 'serial',
      header: t('stock.on_hand.serial'),
      width: 144,
      value: (l) => l.serial ?? '',
      cell: (i, bare) =>
        lines[i]?.tracking === 'serial' && <TextField name={`lines.${i}.serial`} label={t('stock.adjustment.line.serial', { n: i + 1 })} bare={bare} required maxLength={100} />,
    },
    {
      key: 'quantity',
      header: t('stock.line.quantity_col'),
      numeric: true,
      required: true,
      width: 112,
      value: (l) => `${formatDecimal(l.quantity, 3)} ${l.unit}`,
      cell: (i, bare) => <DecimalField name={`lines.${i}.quantity`} label={t('stock.line.quantity', { n: i + 1 })} bare={bare} required scale={3} />,
    },
  ]

  return (
    <Form form={form} onSubmit={save} errorFields={errorFields}>
      <FormSection title={t('stock.transfer.section.general')} description={t('stock.transfer.section.general_hint')} icon={IconArrowsExchange}>
        <DateField name="date" label={t('stock.transfer.date')} required readOnly={ro} />
        <SelectField
          name="from_warehouse_id"
          label={t('stock.transfer.from_warehouse')}
          required
          readOnly={ro}
          data={withSaved(doc?.from_warehouse_id, doc?.from_warehouse_code, doc?.from_warehouse_name)}
        />
        <SelectField
          name="to_warehouse_id"
          label={t('stock.transfer.to_warehouse')}
          required
          readOnly={ro}
          data={withSaved(doc?.to_warehouse_id, doc?.to_warehouse_code, doc?.to_warehouse_name)}
        />
        <TextField name="note" label={t('stock.transfer.note')} maxLength={500} wide readOnly={ro} />
      </FormSection>
      <FormSection
        title={t('stock.doc.section.lines')}
        description={t(ro ? 'stock.transfer.section.lines_hint_ro' : 'stock.transfer.section.lines_hint')}
        icon={IconListDetails}
        wide
      >
        <FormRow>
          {ro && doc ? (
            <LinesTable readOnly label={t('stock.doc.section.lines')} columns={lineColumns} lines={doc.lines} lineLabel={(n) => t('stock.line.title', { n })} />
          ) : (
            <LinesTable
              name="lines"
              label={t('stock.doc.section.lines')}
              columns={lineColumns}
              newLine={() => ({ ...emptyLine })}
              addLabel={t('stock.line.add')}
              lineLabel={(n) => t('stock.line.title', { n })}
              removeLabel={(n) => t('stock.line.remove', { n })}
              minLines={1}
            />
          )}
        </FormRow>
      </FormSection>
      {!ro && <FormActions submitLabel={t(doc ? 'stock.common.save' : 'stock.common.create_draft')} cancelTo={doc ? undefined : '/stock/transfers'} />}
      {mutation.dialog}
    </Form>
  )
}
