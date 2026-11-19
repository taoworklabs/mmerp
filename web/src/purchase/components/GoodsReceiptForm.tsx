import { Stack, Text } from '@mmerp/ui'
import { IconListDetails, IconPackageImport } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, ApiError, unwrap, type GoodsReceipt, type ReceiptLineInput, type ReceiptOptions } from '@/shared/api/purchase'
import { useMe } from '@/shared/auth/me'
import { useDocumentMutation } from '@/shared/document'
import { errorText, formatDate, formatDecimal, today } from '@/shared/i18n'
import { DateField, DecimalField, Form, FormActions, FormRow, FormSection, LinesTable, SelectField, TextField, type ErrorFields, type LineColumn, type Option } from '@/shared/ui/form'
import { Notice } from '@/shared/ui/Notice'
import { ContentSkeleton, ErrorState, ForbiddenPage } from '@/shared/ui/states'
import { receiptOptions } from '../hooks/usePurchase'
import { goodsReceiptDocType } from '../keys'
import { linesFromOptions, withOwnLines } from '../receiptLines'

type LineValues = {
  order_position: string | null
  lot_code: string
  lot_expiry: string | null
  serial: string
  quantity: string
}

type Values = {
  date: string | null
  warehouse_id: string | null
  note: string
  lines: LineValues[]
}

const emptyLine: LineValues = { order_position: null, lot_code: '', lot_expiry: null, serial: '', quantity: '' }

const toValues = (l: ReceiptLineInput): LineValues => ({
  order_position: String(l.order_position),
  lot_code: l.lot?.code ?? '',
  lot_expiry: l.lot?.expiry ?? null,
  serial: l.serial ?? '',
  quantity: l.quantity,
})

// suggested are the lines a new receipt starts with; at least one, for the user to fill.
function suggested(options: ReceiptOptions): LineValues[] {
  const lines = linesFromOptions(options).map(toValues)
  return lines.length ? lines : [{ ...emptyLine }]
}

const errorFields: ErrorFields<Values> = {
  purchase_receipt_before_order: 'date',
  purchase_receipt_warehouse: 'warehouse_id',
  purchase_receipt_position: 'lines.*.order_position',
  purchase_receipt_line_not_stock: 'lines.*.order_position',
  stock_lot_not_tracked: 'lines.*.order_position',
  purchase_receipt_over_order: 'lines.*.quantity',
  invalid_quantity: 'lines.*.quantity',
  stock_serial_quantity: 'lines.*.quantity',
  stock_lot_required: 'lines.*.lot_code',
  stock_lot_expiry_mismatch: 'lines.*.lot_expiry',
  stock_serial_required: 'lines.*.serial',
}

type Props = {
  order: number
  doc?: GoodsReceipt
  // A new receipt: called with its id once created.
  onCreated?: (id: number) => Promise<void>
}

// GoodsReceiptForm makes a goods receipt from a posted order, or edits a draft whose allowed_actions
// has edit; otherwise it shows the receipt read-only. Lines start from what is left on the order;
// the user types the lot or serial of what arrived.
export function GoodsReceiptForm(props: Props) {
  const { t } = useTranslation()
  const ro = props.doc ? !props.doc.allowed_actions.includes('edit') : false
  const options = useQuery({ ...receiptOptions(props.order), enabled: !ro })
  if (!ro) {
    if (options.isPending) return <ContentSkeleton />
    // A draft that cannot be edited now (order no longer open) still shows its lines.
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
  // The options count a draft's own lines as taken; editing it may take them again.
  const data = options.data && props.doc ? withOwnLines(options.data, props.doc.lines) : options.data
  return <Fields {...props} ro={ro} options={data} />
}

function Fields({ order, doc, onCreated, ro, options }: Props & { ro: boolean; options?: ReceiptOptions }) {
  const { t } = useTranslation()
  const me = useMe()
  const mutation = useDocumentMutation(goodsReceiptDocType, doc?.id ?? 0, doc?.version ?? 0)
  // One id per opened form: sending it twice returns the receipt made the first time.
  const [requestId] = useState(() => crypto.randomUUID())
  const defaultWarehouse = options?.warehouses.find((w) => w.is_default) ?? options?.warehouses[0]
  const form = useForm<Values>({
    mode: 'onBlur',
    defaultValues: {
      date: doc?.date ?? today(me.timezone),
      warehouse_id: doc ? String(doc.warehouse_id) : defaultWarehouse ? String(defaultWarehouse.id) : null,
      note: doc?.note ?? '',
      lines: doc ? doc.lines.map(toValues) : options ? suggested(options) : [],
    },
  })
  const lines = form.watch('lines')

  const warehouses: Option[] = (options?.warehouses ?? []).map((w) => ({ value: String(w.id), label: `${w.code} · ${w.name}` }))
  // Read-only, or a warehouse retired since: the saved one still shows.
  if (doc && !warehouses.some((w) => w.value === String(doc.warehouse_id)))
    warehouses.unshift({ value: String(doc.warehouse_id), label: `${doc.warehouse_code} · ${doc.warehouse_name}` })
  const orderLines = new Map((options?.lines ?? []).map((l) => [String(l.order_position), l]))
  const trackingOf = (i: number) => orderLines.get(lines[i]?.order_position ?? '')?.tracking

  async function save(v: Values) {
    const body = {
      date: v.date ?? '',
      warehouse_id: Number(v.warehouse_id),
      note: v.note.trim() || null,
      lines: v.lines.map(
        (l): ReceiptLineInput => ({
          order_position: Number(l.order_position),
          lot: l.lot_code.trim() ? { code: l.lot_code.trim(), expiry: l.lot_expiry ?? undefined } : undefined,
          serial: l.serial.trim() || undefined,
          quantity: l.quantity,
        }),
      ),
    }
    if (doc) {
      await mutation.run((version) => unwrap(api.PUT('/purchase/goods-receipts/{id}', { params: { path: { id: doc.id } }, body: { version, ...body } })))
      return
    }
    const { id } = await unwrap(api.POST('/purchase/goods-receipts', { body: { request_id: requestId, order_id: order, ...body } }))
    await onCreated?.(id)
  }

  // Lot and serial cells show only for order lines tracked that way.
  const lineColumns: LineColumn<GoodsReceipt['lines'][number]>[] = [
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
            form.setValue(`lines.${i}.lot_code`, '')
            form.setValue(`lines.${i}.lot_expiry`, null)
            form.setValue(`lines.${i}.serial`, '')
            // A serial comes in one unit at a time.
            if (orderLines.get(v ?? '')?.tracking === 'serial') form.setValue(`lines.${i}.quantity`, '1')
          }}
        />
      ),
    },
    {
      key: 'remaining',
      header: t('purchase.goods_receipt.line.remaining'),
      numeric: true,
      width: 128,
      cell: (i, bare) => {
        const l = orderLines.get(lines[i]?.order_position ?? '')
        if (!l) return null
        const quantity = `${formatDecimal(l.remaining, 3)} ${l.unit}`
        return (
          <Text size="sm" tone="dimmed">
            {bare ? quantity : t('purchase.goods_receipt.line.remaining_hint', { quantity })}
          </Text>
        )
      },
    },
    {
      key: 'lot',
      header: t('purchase.goods_receipt.line.lot_col'),
      width: 176,
      value: (l) => l.lot?.code ?? '',
      cell: (i, bare) =>
        trackingOf(i) === 'lot' && (
          <TextField name={`lines.${i}.lot_code`} label={t('purchase.goods_receipt.line.lot', { n: i + 1 })} bare={bare} required maxLength={100} />
        ),
    },
    {
      key: 'expiry',
      header: t('purchase.goods_receipt.line.expiry_col'),
      width: 152,
      value: (l) => (l.lot?.expiry ? formatDate(l.lot.expiry) : ''),
      cell: (i, bare) =>
        trackingOf(i) === 'lot' && <DateField name={`lines.${i}.lot_expiry`} label={t('purchase.goods_receipt.line.expiry', { n: i + 1 })} bare={bare} clearable />,
    },
    {
      key: 'serial',
      header: t('purchase.goods_receipt.line.serial_col'),
      width: 176,
      value: (l) => l.serial ?? '',
      cell: (i, bare) =>
        trackingOf(i) === 'serial' && (
          <TextField name={`lines.${i}.serial`} label={t('purchase.goods_receipt.line.serial', { n: i + 1 })} bare={bare} required maxLength={100} />
        ),
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
  ]

  return (
    <Form form={form} onSubmit={save} errorFields={errorFields}>
      {!ro && !options?.warehouses.length && (
        <Notice tone="warning" title={t('purchase.goods_receipt.no_warehouse')}>
          {t('purchase.goods_receipt.no_warehouse_hint')}
        </Notice>
      )}
      <FormSection title={t('purchase.goods_receipt.section.general')} description={t('purchase.goods_receipt.section.general_hint')} icon={IconPackageImport}>
        <DateField name="date" label={t('purchase.goods_receipt.date')} required readOnly={ro} />
        <SelectField name="warehouse_id" label={t('purchase.goods_receipt.warehouse')} required readOnly={ro} data={warehouses} />
        <TextField name="note" label={t('purchase.goods_receipt.note')} multiline maxLength={2000} readOnly={ro} />
      </FormSection>
      <FormSection
        title={t('purchase.doc.section.lines')}
        description={t(ro ? 'purchase.goods_receipt.section.lines_hint_ro' : 'purchase.goods_receipt.section.lines_hint')}
        icon={IconListDetails}
        wide
      >
        <FormRow>
          {ro && doc ? (
            <LinesTable readOnly label={t('purchase.doc.section.lines')} columns={lineColumns} lines={doc.lines} lineLabel={(n) => t('purchase.line.title', { n })} />
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
            />
          )}
        </FormRow>
      </FormSection>
      {!ro && <FormActions submitLabel={t(doc ? 'purchase.common.save' : 'purchase.common.create_draft')} cancelTo={doc ? undefined : `/purchase/orders/${order}`} />}
      {mutation.dialog}
    </Form>
  )
}
