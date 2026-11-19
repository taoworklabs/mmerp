import { Anchor } from '@mmerp/ui'
import { useTranslation } from 'react-i18next'
import { Link, useParams } from 'react-router'
import { DocumentScreen } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import { loaded } from '@/shared/ui/loaded'
import { GoodsReceiptForm } from '../components/GoodsReceiptForm'
import { useGoodsReceipt } from '../hooks/usePurchase'
import { goodsReceiptDocType } from '../keys'

const list = '/purchase/goods-receipts'

// GoodsReceiptPage shows a goodsReceipt; a draft the user may edit is edited in place.
export function GoodsReceiptPage() {
  const { t } = useTranslation()
  const id = Number(useParams().id)
  const state = loaded(useGoodsReceipt(id), t, { to: list, label: t('purchase.goods_receipts.back') })
  if (!state.ok) return state.fallback
  const d = state.data
  return (
    <DocumentScreen
      docType={goodsReceiptDocType}
      doc={d}
      list={{ label: t('purchase.goods_receipts.title'), to: list }}
      facts={[
        <Anchor key="order" render={<Link to={`/purchase/orders/${d.order_id}`} />} size="sm">
          {t('purchase.goods_receipt.order_link', { number: d.order_number })}
        </Anchor>,
        `${d.warehouse_code} · ${d.warehouse_name}`,
        formatDate(d.date),
      ]}
      attachments
      // The editable lines table needs the whole width; approval and history become tabs.
      fullWidth
    >
      {/* Keyed by version: a reload after a conflict starts from the saved data. */}
      <GoodsReceiptForm key={d.version} order={d.order_id} doc={d} />
    </DocumentScreen>
  )
}
