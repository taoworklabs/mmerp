import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router'
import { CreatePage } from '@/shared/document'
import { loaded } from '@/shared/ui/loaded'
import { NotFoundPage } from '@/shared/ui/states'
import { GoodsReceiptForm } from '../components/GoodsReceiptForm'
import { useOrder } from '../hooks/usePurchase'
import { goodsReceiptDocType } from '../keys'

const orders = '/purchase/orders'

// NewGoodsReceiptPage makes a goodsReceipt from the order in ?order=, when the order allows it.
export function NewGoodsReceiptPage() {
  const { t } = useTranslation()
  const order = Number(useSearchParams()[0].get('order'))
  if (!Number.isInteger(order) || order <= 0) return <NotFoundPage back={{ to: orders, label: t('purchase.orders.back') }} />
  return <NewGoodsReceipt order={order} />
}

function NewGoodsReceipt({ order: id }: { order: number }) {
  const { t } = useTranslation()
  const state = loaded(useOrder(id), t, { to: orders, label: t('purchase.orders.back') })
  if (!state.ok) return state.fallback
  const order = state.data
  return (
    <CreatePage
      docType={goodsReceiptDocType}
      actions={order.allowed_actions}
      action="create_goods_receipt"
      title={t('purchase.goods_receipts.create')}
      breadcrumbs={[
        { label: t('purchase.orders.title'), to: orders },
        { label: order.number, to: `${orders}/${order.id}` },
      ]}
      description={t('purchase.goods_receipts.create_description', { number: order.number, supplier: order.supplier.name })}
      created={t('purchase.goods_receipt.created')}
    >
      {(onCreated) => <GoodsReceiptForm order={order.id} onCreated={onCreated} />}
    </CreatePage>
  )
}
