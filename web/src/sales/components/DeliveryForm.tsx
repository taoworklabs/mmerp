import { Stack, Text } from '@mmerp/ui'
import { IconListDetails, IconTruckDelivery } from '@tabler/icons-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useRef, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, ApiError, unwrap, type Delivery, type DeliveryLineInput, type DeliveryOptions } from '@/shared/api/sales'
import { useMe } from '@/shared/auth/me'
import { useDocumentMutation } from '@/shared/document'
import { errorText, formatDate, formatDecimal, today } from '@/shared/i18n'
import { useConfirm } from '@/shared/ui/confirm'
import { DateField, DecimalField, Form, FormActions, FormRow, FormSection, LinesTable, SelectField, TextField, type ErrorFields, type LineColumn, type Option } from '@/shared/ui/form'
import { Notice } from '@/shared/ui/Notice'
import { ContentSkeleton, ErrorState, ForbiddenPage } from '@/shared/ui/states'
import { linesFromOptions, withOwnLines } from '../deliveryLines'
import { deliveryOptions } from '../hooks/useSales'
import { deliveryDocType } from '../keys'

type LineValues = {
  order_position: string | null
  lot_code: string | null
  serial_code: string | null
  quantity: string
}

type Values = {
  date: string | null
  warehouse_id: string | null
  note: string
  lines: LineValues[]
}

const emptyLine: LineValues = { order_position: null, lot_code: null, serial_code: null, quantity: '' }

const toValues = (l: DeliveryLineInput): LineValues => ({
  order_position: String(l.order_position),
  lot_code: l.lot_code ?? null,
  serial_code: l.serial_code ?? null,
  quantity: l.quantity,
})

// suggested are the lines a new set of options starts with; at least one, for the user to fill.
function suggested(options: DeliveryOptions): LineValues[] {
  const lines = linesFromOptions(options).map(toValues)
  return lines.length ? lines : [{ ...emptyLine }]
}

const errorFields: ErrorFields<Values> = {
  sales_delivery_date: 'date',
  sales_delivery_warehouse: 'warehouse_id',
  sales_delivery_position: 'lines.*.order_position',
  sales_delivery_line_not_stock: 'lines.*.order_position',
  stock_lot_not_tracked: 'lines.*.order_position',
  sales_delivery_over_ordered: 'lines.*.quantity',
  invalid_quantity: 'lines.*.quantity',
  stock_serial_quantity: 'lines.*.quantity',
  sales_lot_expired: 'lines.*.lot_code',
  stock_lot_required: 'lines.*.lot_code',
  stock_lot_not_found: 'lines.*.lot_code',
  stock_serial_required: 'lines.*.serial_code',
}

type Props = {
  order: number
  doc?: Delivery
  // A new delivery: called with its id once created.
  onCreated?: (id: number) => Promise<void>
}

// DeliveryForm makes a delivery from a posted order, or edits a draft whose allowed_actions has
// edit; otherwise it shows the delivery read-only. Lines start from what is left on the order,
// lots first expiry first, in the chosen warehouse.
export function DeliveryForm(props: Props) {
  const { t } = useTranslation()
  const ro = props.doc ? !props.doc.allowed_actions.includes('edit') : false
  const options = useQuery({ ...deliveryOptions(props.order, props.doc?.warehouse_id ?? 0), enabled: !ro })
  if (!ro) {
    if (options.isPending) return <ContentSkeleton />
    // A draft that cannot be edited now (warehouse retired, order no longer open) still shows its lines.
    if (options.isError && props.doc)
      return (
        <Stack gap="md">
          <Notice tone="danger">{errorText(t, options.error)}</Notice>
          <Fields {...props} ro />
        </Stack>
      )
    if (options.error instanceof ApiError && options.error.status === 403) return <ForbiddenPage />
    if (options.isError) return <ErrorState message={errorText(t, options.error)} onRetry={() => void options.refetch()} />
  }
  // The options count a draft's own lines as taken; editing it may take them again.
  const data = options.data && props.doc ? withOwnLines(options.data, props.doc.lines) : options.data
  return <Fields {...props} ro={ro} initial={data} />
}

function Fields({ order, doc, onCreated, ro, initial }: Props & { ro: boolean; initial?: DeliveryOptions }) {
  const { t } = useTranslation()
  const me = useMe()
  const client = useQueryClient()
  const [ask, dialog] = useConfirm()
  const mutation = useDocumentMutation(deliveryDocType, doc?.id ?? 0, doc?.version ?? 0)
  // One id per opened form: sending it twice returns the delivery made the first time.
  const [requestId] = useState(() => crypto.randomUUID())
  const [options, setOptions] = useState(initial)
  // The lines as last filled in for the user: edits beyond them are asked about before a rebuild.
  const [start, setStart] = useState(() => (doc ? doc.lines.map(toValues) : initial ? suggested(initial) : []))
  const [warehouse, setWarehouse] = useState(doc ? String(doc.warehouse_id) : initial?.warehouse_id ? String(initial.warehouse_id) : null)
  const form = useForm<Values>({
    mode: 'onBlur',
    defaultValues: { date: doc?.date ?? today(me.timezone), warehouse_id: warehouse, note: doc?.note ?? '', lines: start },
  })
  const lines = form.watch('lines')

  const warehouses: Option[] = (options?.warehouses ?? []).map((w) => ({ value: String(w.id), label: `${w.code} · ${w.name}` }))
  // Read-only, or a warehouse retired since: the saved one still shows.
  if (doc && !warehouses.some((w) => w.value === String(doc.warehouse_id)))
    warehouses.unshift({ value: String(doc.warehouse_id), label: `${doc.warehouse_code} · ${doc.warehouse_name}` })
  const orderLines = new Map((options?.lines ?? []).map((l) => [String(l.order_position), l]))
  const lineOf = (i: number) => orderLines.get(lines[i]?.order_position ?? '')
  // Set while the options of a newly picked warehouse load: lot codes of the old one must not be sent.
  const switching = useRef(false)
  // A saved lot or serial no longer suggested stays selectable.
  const withCurrent = (data: Option[], current: string | null | undefined) => (current && !data.some((o) => o.value === current) ? [{ value: current, label: current }, ...data] : data)

  async function changeWarehouse(v: string | null) {
    if (!v || v === warehouse) return
    // A draft's lines are saved work: always ask before replacing them.
    if (doc || JSON.stringify(form.getValues('lines')) !== JSON.stringify(start)) {
      const ok = await ask({
        title: t('sales.delivery.change_warehouse'),
        message: t('sales.delivery.change_warehouse_confirm'),
        confirmLabel: t('sales.delivery.change_warehouse'),
        danger: true,
      })
      if (!ok) return form.setValue('warehouse_id', warehouse)
    }
    switching.current = true
    try {
      const fetched = await client.fetchQuery(deliveryOptions(order, Number(v)))
      const next = doc ? withOwnLines(fetched, doc.lines) : fetched
      const rebuilt = suggested(next)
      setOptions(next)
      setStart(rebuilt)
      setWarehouse(v)
      form.setValue('lines', rebuilt, { shouldDirty: true })
    } catch (err) {
      form.setValue('warehouse_id', warehouse)
      form.setError('warehouse_id', { message: errorText(t, err) })
    } finally {
      switching.current = false
    }
  }

  async function save(v: Values) {
    // Shown in the form's banner; the edits stay.
    if (switching.current) throw new ApiError(409, 'delivery_options_loading')
    const body = {
      date: v.date ?? '',
      warehouse_id: Number(v.warehouse_id),
      note: v.note.trim() || null,
      lines: v.lines.map(
        (l): DeliveryLineInput => ({
          order_position: Number(l.order_position),
          lot_code: l.lot_code ?? undefined,
          serial_code: l.serial_code ?? undefined,
          quantity: l.quantity,
        }),
      ),
    }
    if (doc) {
      await mutation.run((version) => unwrap(api.PUT('/sales/deliveries/{id}', { params: { path: { id: doc.id } }, body: { version, ...body } })))
      return
    }
    const { id } = await unwrap(api.POST('/sales/deliveries', { body: { request_id: requestId, order_id: order, ...body } }))
    await onCreated?.(id)
  }

  // Lot and serial cells show only for order lines tracked that way, picked from the warehouse's stock.
  const lineColumns: LineColumn<Delivery['lines'][number]>[] = [
    {
      key: 'order_line',
      header: t('sales.delivery.line.order_line_col'),
      required: true,
      width: 240,
      value: (l) => `${l.item_code} · ${l.description}`,
      cell: (i, bare) => (
        <SelectField
          name={`lines.${i}.order_position`}
          label={t('sales.delivery.line.order_line', { n: i + 1 })}
          bare={bare}
          required
          data={[...orderLines.values()].map((l) => ({ value: String(l.order_position), label: `${l.item_code} · ${l.description}` }))}
          onChange={(v) => {
            form.setValue(`lines.${i}.lot_code`, null)
            form.setValue(`lines.${i}.serial_code`, null)
            // A serial goes out one unit at a time.
            if (orderLines.get(v ?? '')?.tracking === 'serial') form.setValue(`lines.${i}.quantity`, '1')
          }}
        />
      ),
    },
    {
      key: 'remaining',
      header: t('sales.delivery.line.remaining'),
      numeric: true,
      width: 128,
      cell: (i, bare) => {
        const l = lineOf(i)
        if (!l) return null
        const quantity = `${formatDecimal(l.remaining, 3)} ${l.unit}`
        return (
          <Text size="sm" tone="dimmed">
            {bare ? quantity : t('sales.delivery.line.remaining_hint', { quantity })}
          </Text>
        )
      },
    },
    {
      key: 'lot',
      header: t('sales.on_hand.lot'),
      width: 240,
      value: (l) => l.lot_code ?? '',
      cell: (i, bare) => {
        const l = lineOf(i)
        if (l?.tracking !== 'lot') return null
        const data = l.lots.map((lot) => ({
          value: lot.code,
          label: [lot.code, lot.expiry && formatDate(lot.expiry), `${formatDecimal(lot.quantity, 3)} ${l.unit}`].filter(Boolean).join(' — '),
        }))
        return (
          <SelectField name={`lines.${i}.lot_code`} label={t('sales.delivery.line.lot', { n: i + 1 })} bare={bare} required data={withCurrent(data, lines[i]?.lot_code)} />
        )
      },
    },
    {
      key: 'serial',
      header: t('sales.on_hand.serial'),
      width: 176,
      value: (l) => l.serial_code ?? '',
      cell: (i, bare) => {
        const l = lineOf(i)
        if (l?.tracking !== 'serial') return null
        const data = l.serials.map((s) => ({ value: s, label: s }))
        return (
          <SelectField
            name={`lines.${i}.serial_code`}
            label={t('sales.delivery.line.serial', { n: i + 1 })}
            bare={bare}
            required
            data={withCurrent(data, lines[i]?.serial_code)}
          />
        )
      },
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
  ]

  return (
    <Form form={form} onSubmit={save} errorFields={errorFields}>
      <FormSection title={t('sales.delivery.section.general')} description={t('sales.delivery.section.general_hint')} icon={IconTruckDelivery}>
        <DateField name="date" label={t('sales.delivery.date')} required readOnly={ro} />
        <SelectField name="warehouse_id" label={t('sales.delivery.warehouse')} required readOnly={ro} data={warehouses} onChange={(v) => void changeWarehouse(v)} />
        <TextField name="note" label={t('sales.delivery.note')} multiline maxLength={2000} readOnly={ro} />
      </FormSection>
      <FormSection
        title={t('sales.doc.section.lines')}
        description={t(ro ? 'sales.delivery.section.lines_hint_ro' : 'sales.delivery.section.lines_hint')}
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
      {!ro && <FormActions submitLabel={t(doc ? 'sales.common.save' : 'sales.common.create_draft')} cancelTo={doc ? undefined : `/sales/orders/${order}`} />}
      {mutation.dialog}
      {dialog}
    </Form>
  )
}
