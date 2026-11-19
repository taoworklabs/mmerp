import { Anchor, Button, Group } from '@mmerp/ui'
import { IconFileInvoice, IconPackageImport } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import { Link, useParams } from 'react-router'
import { DocumentScreen, DocumentStatus } from '@/shared/document'
import { formatNumber } from '@/shared/i18n'
import { loaded } from '@/shared/ui/loaded'
import { icon } from '@/shared/ui/theme'
import { DocForm } from '../components/DocForm'
import { DocProgress } from '../components/DocProgress'
import { useOrder } from '../hooks/usePurchase'
import { goodsReceiptDocType, invoiceDocType, orderDocType } from '../keys'

const list = '/purchase/orders'

// OrderPage shows a purchase order; a draft the user may edit is edited in place.
export function OrderPage() {
  const { t } = useTranslation()
  const id = Number(useParams().id)
  const doc = useOrder(id)

  const state = loaded(doc, t, { to: list, label: t('purchase.orders.back') })
  if (!state.ok) return state.fallback
  const d = state.data
  const goodsReceipts = d.goods_receipts ?? []
  const invoices = d.invoices ?? []
  return (
    <DocumentScreen
      docType={orderDocType}
      doc={d}
      list={{ label: t('purchase.orders.title'), to: list }}
      facts={[
        d.supplier.name,
        d.org_unit_name,
        invoices.length > 0 && d.paid !== undefined && t('purchase.order.paid_of', { paid: formatNumber(d.paid), total: formatNumber(d.total) }),
        goodsReceipts.length > 0 && (
          <Group gap="xs" render={<span />}>
            {goodsReceipts.map((r) => (
              <Group key={r.id} gap="xs" wrap={false} render={<span />}>
                <Anchor render={<Link to={`/purchase/goods-receipts/${r.id}`} />} size="sm">
                  {t('purchase.order.goods_receipt_link', { number: r.number })}
                </Anchor>
                <DocumentStatus docType={goodsReceiptDocType} status={r.status} />
              </Group>
            ))}
            <Anchor render={<Link to={`/purchase/goods-receipts?order_id=${d.id}`} />} size="sm">
              {t('purchase.order.goods_receipts_all')}
            </Anchor>
          </Group>
        ),
        invoices.length > 0 && (
          <Group gap="xs" render={<span />}>
            {invoices.map((r) => (
              <Group key={r.id} gap="xs" wrap={false} render={<span />}>
                <Anchor render={<Link to={`/purchase/invoices/${r.id}`} />} size="sm">
                  {t('purchase.order.invoice_link', { number: r.number })}
                </Anchor>
                <DocumentStatus docType={invoiceDocType} status={r.status} />
              </Group>
            ))}
            <Anchor render={<Link to={`/purchase/invoices?order_id=${d.id}`} />} size="sm">
              {t('purchase.order.invoices_all')}
            </Anchor>
          </Group>
        ),
      ]}
      badges={<DocProgress doc={d} />}
      attachments
      // The editable lines table needs the whole width; approval and history become tabs.
      fullWidth
      extraActions={() => (
        <>
          {d.allowed_actions.includes('create_goods_receipt') && (
            <Button variant="default" leftSection={<IconPackageImport {...icon.button} />} render={<Link to={`/purchase/goods-receipts/new?order=${d.id}`} />}>
              {t('purchase.order.create_goods_receipt')}
            </Button>
          )}
          {d.allowed_actions.includes('create_invoice') && (
            <Button variant="default" leftSection={<IconFileInvoice {...icon.button} />} render={<Link to={`/purchase/invoices/new?order=${d.id}`} />}>
              {t('purchase.order.create_invoice')}
            </Button>
          )}
        </>
      )}
      printLabel={t('purchase.order.print')}
    >
      {/* Keyed by version: a reload after a conflict starts from the saved data. */}
      <DocForm key={d.version} doc={d} />
    </DocumentScreen>
  )
}
