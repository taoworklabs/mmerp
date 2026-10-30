import { Anchor, Button, Group } from '@mmerp/ui'
import { IconShoppingCartPlus, IconTruckDelivery } from '@tabler/icons-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useNavigate, useParams } from 'react-router'
import { api, unwrap, type Doc } from '@/shared/api/sales'
import { useInvalidateRecord } from '@/shared/area'
import { DocumentScreen, DocumentStatus } from '@/shared/document'
import { errorText } from '@/shared/i18n'
import { loaded } from '@/shared/ui/loaded'
import { notifySuccess } from '@/shared/ui/notify'
import { icon } from '@/shared/ui/theme'
import { DocForm } from '../components/DocForm'
import { DocProgress } from '../components/DocProgress'
import { useDoc } from '../hooks/useSales'
import { deliveryDocType, docTypeOf, pathOf, quoteDocType, type Kind } from '../keys'

// DocPage shows a quotation or order; a draft the user may edit is edited in place.
export function DocPage({ kind }: { kind: Kind }) {
  const { t } = useTranslation()
  const id = Number(useParams().id)
  const docType = docTypeOf(kind)
  const list = `/sales/${pathOf(kind)}`
  const doc = useDoc(kind, id)

  const state = loaded(doc, t, { to: list, label: t(`sales.${pathOf(kind)}.back`) })
  if (!state.ok) return state.fallback
  const d = state.data
  const linked = d.order ?? d.quote
  const deliveries = d.deliveries ?? []
  return (
    <DocumentScreen
      docType={docType}
      doc={d}
      list={{ label: t(`sales.${pathOf(kind)}.title`), to: list }}
      facts={[
        d.customer.name,
        d.org_unit_name,
        deliveries.length > 0 && (
          <Group gap="xs" render={<span />}>
            {deliveries.map((r) => (
              <Group key={r.id} gap="xs" wrap={false} render={<span />}>
                <Anchor render={<Link to={`/sales/deliveries/${r.id}`} />} size="sm">
                  {t('sales.order.delivery_link', { number: r.number })}
                </Anchor>
                <DocumentStatus docType={deliveryDocType} status={r.status} />
              </Group>
            ))}
            <Anchor render={<Link to={`/sales/deliveries?order_id=${d.id}`} />} size="sm">
              {t('sales.order.deliveries_all')}
            </Anchor>
          </Group>
        ),
      ]}
      badges={
        <>
          <DocProgress doc={d} />
          {linked && (
            <Anchor render={<Link to={`/sales/${d.order ? 'orders' : 'quotes'}/${linked.id}`} />} size="sm">
              {t(d.order ? 'sales.quote.order_link' : 'sales.order.quote_link', { number: linked.number })}
            </Anchor>
          )}
        </>
      }
      attachments
      // The editable lines table needs the whole width; approval and history become tabs.
      fullWidth
      extraActions={(onError) => (
        <>
          {d.allowed_actions.includes('create_order') && <CreateOrderButton quote={d} onError={onError} />}
          {d.allowed_actions.includes('create_delivery') && (
            <Button variant="default" leftSection={<IconTruckDelivery {...icon.button} />} render={<Link to={`/sales/deliveries/new?order=${d.id}`} />}>
              {t('sales.order.create_delivery')}
            </Button>
          )}
        </>
      )}
      printLabel={t(`sales.${kind}.print`)}
    >
      {/* Keyed by version: a reload after a conflict starts from the saved data. */}
      <DocForm key={d.version} kind={kind} doc={d} />
    </DocumentScreen>
  )
}

// CreateOrderButton makes an order from the quotation and opens it; when a rule makes that wait
// for approval, it stays on the quotation.
function CreateOrderButton({ quote, onError }: { quote: Doc; onError: (message: string | null) => void }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const invalidate = useInvalidateRecord()
  const [ordering, setOrdering] = useState(false)
  async function makeOrder() {
    setOrdering(true)
    onError(null)
    try {
      const { id: order, held } = await unwrap(api.POST('/sales/quotes/{id}/order', { params: { path: { id: quote.id } } }))
      await invalidate(quoteDocType, quote.id)
      if (held) return notifySuccess(t('shared.document.sent_for_approval', { number: quote.number }))
      notifySuccess(t('sales.quote.ordered'))
      navigate(`/sales/orders/${order}`)
    } catch (err) {
      onError(errorText(t, err))
    } finally {
      setOrdering(false)
    }
  }
  return (
    <Button variant="default" leftSection={<IconShoppingCartPlus {...icon.button} />} loading={ordering} onClick={() => void makeOrder()}>
      {t('sales.quote.create_order')}
    </Button>
  )
}
