import { Anchor, Button, Group } from '@mmerp/ui'
import { IconCash } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import { Link, useParams } from 'react-router'
import { DocumentScreen, DocumentStatus } from '@/shared/document'
import { formatDate, formatNumber } from '@/shared/i18n'
import { loaded } from '@/shared/ui/loaded'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { icon } from '@/shared/ui/theme'
import { InvoiceForm } from '../components/InvoiceForm'
import { useInvoice } from '../hooks/usePurchase'
import { paymentStateBadge } from '../invoiceLines'
import { invoiceDocType, paymentDocType } from '../keys'

const list = '/purchase/invoices'

// InvoicePage shows an invoice with what has been received of it; a draft the user may edit is
// edited in place.
export function InvoicePage() {
  const { t } = useTranslation()
  const id = Number(useParams().id)
  const state = loaded(useInvoice(id), t, { to: list, label: t('purchase.invoices.back') })
  if (!state.ok) return state.fallback
  const d = state.data
  const payment = d.status === 'posted' ? paymentStateBadge(d.payment_state) : null
  return (
    <DocumentScreen
      docType={invoiceDocType}
      doc={d}
      list={{ label: t('purchase.invoices.title'), to: list }}
      facts={[
        d.supplier.name,
        <Anchor key="order" render={<Link to={`/purchase/orders/${d.order_id}`} />} size="sm">
          {t('purchase.goods_receipt.order_link', { number: d.order_number })}
        </Anchor>,
        formatDate(d.date),
        d.status === 'posted' && t('purchase.invoice.paid_of', { paid: formatNumber(d.paid), total: formatNumber(d.totals.total) }),
        d.payments.length > 0 && (
          <Group gap="xs" render={<span />}>
            {d.payments.map((r) => (
              <Group key={r.id} gap="xs" wrap={false} render={<span />}>
                <Anchor render={<Link to={`/purchase/payments/${r.id}`} />} size="sm">
                  {t('purchase.invoice.payment_link', { number: r.number, amount: formatNumber(r.amount) })}
                </Anchor>
                <DocumentStatus docType={paymentDocType} status={r.status} />
              </Group>
            ))}
          </Group>
        ),
      ]}
      badges={payment && <StatusBadge tone={payment.tone}>{t(payment.label)}</StatusBadge>}
      attachments
      fullWidth
      extraActions={() =>
        d.allowed_actions.includes('create_payment') && (
          <Button variant="default" leftSection={<IconCash {...icon.button} />} render={<Link to={`/purchase/payments/new?invoice=${d.id}`} />}>
            {t('purchase.invoice.create_payment')}
          </Button>
        )
      }
      printLabel={t('purchase.invoice.print')}
    >
      {/* Keyed by version: a reload after a conflict starts from the saved data. */}
      <InvoiceForm key={d.version} order={d.order_id} doc={d} />
    </DocumentScreen>
  )
}
