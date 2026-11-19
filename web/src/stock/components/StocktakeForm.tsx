import { Button } from '@mmerp/ui'
import { IconClipboardCheck, IconListDetails, IconPackages } from '@tabler/icons-react'
import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm, type Path } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, ApiError, unwrap, type StockItem, type Stocktake, type StocktakeLine } from '@/shared/api/core'
import { useDocumentMutation } from '@/shared/document'
import { errorText, formatDate, formatDecimal } from '@/shared/i18n'
import { useConfirm } from '@/shared/ui/confirm'
import { DateField, DecimalField, Form, FormActions, FormRow, FormSection, LinesTable, SelectField, TextField, type ErrorFields, type LineColumn } from '@/shared/ui/form'
import { notifyError } from '@/shared/ui/notify'
import { icon } from '@/shared/ui/theme'
import { useWarehouseOptions } from '../hooks/useStock'
import { stockKeys, stocktakeDocType } from '../keys'
import { StockItemField } from './pickers'

type Tracking = StockItem['tracking']

type LineValues = {
  item_id: string | null
  // From the picked item; null until one is picked.
  tracking: Tracking | null
  lot_code: string
  expiry: string | null
  serial: string
  counted: string
  // What the warehouse holds now, when known: from the saved draft or from loading on hand.
  on_hand?: string
  // The item as loaded from on hand, which no picker has named yet.
  item?: { id: number; code: string; name: string }
}

type Values = {
  date: string | null
  warehouse_id: string | null
  note: string
  lines: LineValues[]
}

const emptyLine: LineValues = { item_id: null, tracking: null, lot_code: '', expiry: null, serial: '', counted: '' }

// A saved line does not carry its item's tracking; its lot or serial tells it.
const trackingOf = (l: StocktakeLine): Tracking => (l.lot ? 'lot' : l.serial ? 'serial' : 'quantity')

const fromLine = (l: StocktakeLine & { on_hand?: string }): LineValues => ({
  item_id: String(l.item_id),
  tracking: trackingOf(l),
  lot_code: l.lot?.code ?? '',
  expiry: l.lot?.expiry ?? null,
  serial: l.serial ?? '',
  counted: l.counted,
  on_hand: l.on_hand,
})

function defaults(doc?: Stocktake): Values {
  return { date: doc?.date ?? null, warehouse_id: doc ? String(doc.warehouse_id) : null, note: doc?.note ?? '', lines: doc ? doc.lines.map(fromLine) : [{ ...emptyLine }] }
}

// thousandths reads a quantity of up to three decimals exactly.
function thousandths(s: string): bigint {
  const neg = s.trim().startsWith('-')
  const [whole, frac = ''] = s.trim().replace(/^[-+]/, '').split('.')
  const v = BigInt(whole || '0') * 1000n + BigInt((frac + '000').slice(0, 3))
  return neg ? -v : v
}

// draftDifference is what posting would move for a line: counted less on hand, or null while
// either is unknown. Display only; posting takes on hand under the balance lock.
export function draftDifference(onHand: string | undefined, counted: string): string | null {
  if (onHand === undefined || counted.trim() === '') return null
  const d = thousandths(counted) - thousandths(onHand)
  const abs = d < 0n ? -d : d
  const frac = String(abs % 1000n).padStart(3, '0').replace(/0+$/, '')
  return `${d < 0n ? '-' : ''}${abs / 1000n}${frac ? `.${frac}` : ''}`
}

// stocktakeBody is what the API takes for a stocktake's values.
export function stocktakeBody(v: Values) {
  return {
    date: v.date ?? '',
    warehouse_id: Number(v.warehouse_id),
    note: v.note.trim(),
    lines: v.lines.map(
      (l): StocktakeLine => ({
        item_id: Number(l.item_id),
        counted: l.counted,
        lot: l.tracking === 'lot' ? { code: l.lot_code.trim(), expiry: l.expiry || undefined } : undefined,
        serial: l.tracking === 'serial' ? l.serial.trim() : undefined,
      }),
    ),
  }
}

// stocktakeLineErrors are what posting would refuse for a line's tracking, checked before sending.
function stocktakeLineErrors(l: LineValues): string[] {
  switch (l.tracking) {
    case 'lot':
      return l.lot_code.trim() ? [] : ['stock_lot_required']
    case 'serial':
      if (!l.serial.trim()) return ['stock_serial_required']
      return /^[01](\.0*)?$/.test(l.counted.trim()) ? [] : ['stock_serial_quantity']
    default:
      return []
  }
}

const errorFields: ErrorFields<Values> = {
  stock_warehouse_inactive: 'warehouse_id',
  stock_quantity_invalid: 'lines.*.counted',
  stock_serial_quantity: 'lines.*.counted',
  stock_count_duplicate: 'lines.*.item_id',
  stock_untracked: 'lines.*.item_id',
  stock_lot_not_tracked: 'lines.*.item_id',
  stock_lot_required: 'lines.*.lot_code',
  stock_lot_expiry_mismatch: 'lines.*.expiry',
  stock_serial_required: 'lines.*.serial',
}

type Line = Stocktake['lines'][number]

type Props = {
  doc?: Stocktake
  // A new stocktake: called with its id once created.
  onCreated?: (id: number) => Promise<void>
}

// StocktakeForm creates a stocktake, or edits one whose allowed_actions has edit; otherwise it
// shows it read-only, with book quantity and difference once posted. Only listed lines count.
export function StocktakeForm({ doc, onCreated }: Props) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const mutation = useDocumentMutation(stocktakeDocType, doc?.id ?? 0, doc?.version ?? 0)
  // One id per opened form: sending it twice returns the stocktake made the first time.
  const [requestId] = useState(() => crypto.randomUUID())
  const form = useForm<Values>({ mode: 'onBlur', defaultValues: defaults(doc) })
  const [confirm, confirmDialog] = useConfirm()
  const ro = doc ? !doc.allowed_actions.includes('edit') : false
  const posted = !!doc && doc.lines.some((l) => l.book !== undefined)
  const lines = form.watch('lines')
  const warehouse = form.watch('warehouse_id')
  const options = useWarehouseOptions()
  // A retired warehouse is no longer offered, but the saved one still shows.
  const warehouses =
    doc && !options.some((o) => o.value === String(doc.warehouse_id))
      ? [{ value: String(doc.warehouse_id), label: `${doc.warehouse_code} · ${doc.warehouse_name}` }, ...options]
      : options

  // loadOnHand replaces the lines with what the warehouse holds, each counted as on hand.
  async function loadOnHand() {
    if (!warehouse) return
    if (form.getValues('lines').some((l) => l.item_id)) {
      const ok = await confirm({ title: t('stock.stocktake.load.title'), message: t('stock.stocktake.load.message'), confirmLabel: t('stock.stocktake.load.confirm') })
      if (!ok) return
    }
    try {
      const id = Number(warehouse)
      const got = await qc.fetchQuery({
        queryKey: stockKeys.onHand.lines(id, 0),
        queryFn: () => unwrap(api.GET('/stock/warehouses/{id}/on-hand-lines', { params: { path: { id } } })),
        staleTime: 0,
      })
      form.setValue(
        'lines',
        got.items.length
          ? got.items.map((l) => ({ ...fromLine(l), on_hand: l.counted, item: { id: l.item_id, code: l.item_code, name: l.item_name } }))
          : [{ ...emptyLine }],
      )
    } catch (err) {
      notifyError(errorText(t, err))
    }
  }

  async function save(v: Values) {
    const errors = v.lines.flatMap((l, i) => stocktakeLineErrors(l).map((code) => new ApiError(422, code, { line_no: i + 1 })))
    if (errors.length) {
      errors.forEach((err, n) => {
        const field = String(errorFields[err.code]).replace('*', String(Number(err.params.line_no) - 1)) as Path<Values>
        form.setError(field, { message: errorText(t, err) }, { shouldFocus: n === 0 })
      })
      return false
    }
    const body = stocktakeBody(v)
    if (doc) {
      await mutation.run((version) => unwrap(api.PUT('/stock/stocktakes/{id}', { params: { path: { id: doc.id } }, body: { version, ...body } })))
      return
    }
    const { id } = await unwrap(api.POST('/stock/stocktakes', { body: { request_id: requestId, ...body } }))
    await onCreated?.(id)
  }

  const qty = (v: string | undefined) => (v === undefined ? '' : formatDecimal(v, 3))
  // Lot and serial cells show only for items tracked that way.
  const lineColumns: LineColumn<Line>[] = [
    {
      key: 'item',
      header: t('stock.line.item_col'),
      required: true,
      value: (l) => `${l.item_code} · ${l.item_name}`,
      cell: (i, bare, line) => {
        const saved = doc?.lines.find((l) => String(l.item_id) === line.item_id)
        const current = saved ? { id: saved.item_id, code: saved.item_code, name: saved.item_name } : lines[i]?.item
        return (
          <StockItemField
            name={`lines.${i}.item_id`}
            label={t('stock.line.item', { n: i + 1 })}
            bare={bare}
            current={current}
            onPicked={(it) => {
              form.setValue(`lines.${i}.tracking`, it.tracking)
              form.setValue(`lines.${i}.lot_code`, '')
              form.setValue(`lines.${i}.expiry`, null)
              form.setValue(`lines.${i}.serial`, '')
              form.setValue(`lines.${i}.on_hand`, undefined)
            }}
          />
        )
      },
    },
    {
      key: 'lot',
      header: t('stock.on_hand.lot'),
      width: 128,
      value: (l) => l.lot?.code ?? '',
      cell: (i, bare) =>
        lines[i]?.tracking === 'lot' && <TextField name={`lines.${i}.lot_code`} label={t('stock.adjustment.line.lot', { n: i + 1 })} bare={bare} required maxLength={100} />,
    },
    {
      key: 'expiry',
      header: t('stock.on_hand.expiry'),
      width: 144,
      value: (l) => (l.lot?.expiry ? formatDate(l.lot.expiry) : ''),
      cell: (i, bare) => lines[i]?.tracking === 'lot' && <DateField name={`lines.${i}.expiry`} label={t('stock.adjustment.line.expiry', { n: i + 1 })} bare={bare} clearable />,
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
      key: 'book',
      header: t(posted ? 'stock.stocktake.book' : 'stock.stocktake.on_hand'),
      numeric: true,
      width: 112,
      value: (l) => qty(l.book ?? l.on_hand),
      cell: (i) => qty(lines[i]?.on_hand),
    },
    {
      key: 'counted',
      header: t('stock.stocktake.counted'),
      numeric: true,
      required: true,
      width: 112,
      value: (l) => `${formatDecimal(l.counted, 3)} ${l.unit}`,
      cell: (i, bare) => <DecimalField name={`lines.${i}.counted`} label={t('stock.stocktake.counted_n', { n: i + 1 })} bare={bare} required scale={3} />,
    },
    {
      key: 'difference',
      header: t('stock.stocktake.difference'),
      numeric: true,
      width: 112,
      value: (l) => qty(l.difference ?? draftDifference(l.on_hand, l.counted) ?? undefined),
      cell: (i) => qty(draftDifference(lines[i]?.on_hand, lines[i]?.counted ?? '') ?? undefined),
    },
  ]

  return (
    <Form form={form} onSubmit={save} errorFields={errorFields}>
      <FormSection title={t('stock.stocktake.section.general')} description={t('stock.stocktake.section.general_hint')} icon={IconClipboardCheck}>
        <DateField name="date" label={t('stock.stocktake.date')} required readOnly={ro} />
        <SelectField name="warehouse_id" label={t('stock.stocktake.warehouse')} required readOnly={ro} data={warehouses} />
        <TextField name="note" label={t('stock.stocktake.note')} maxLength={500} wide readOnly={ro} />
      </FormSection>
      <FormSection
        title={t('stock.doc.section.lines')}
        description={t(ro ? 'stock.stocktake.section.lines_hint_ro' : 'stock.stocktake.section.lines_hint')}
        icon={IconListDetails}
        wide
      >
        {!ro && (
          <FormRow>
            <Button variant="default" leftSection={<IconPackages {...icon.button} />} disabled={!warehouse} onClick={loadOnHand}>
              {t('stock.stocktake.load.action')}
            </Button>
          </FormRow>
        )}
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
      {!ro && <FormActions submitLabel={t(doc ? 'stock.common.save' : 'stock.common.create_draft')} cancelTo={doc ? undefined : '/stock/stocktakes'} />}
      {mutation.dialog}
      {confirmDialog}
    </Form>
  )
}
