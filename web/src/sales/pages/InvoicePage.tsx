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
import { useInvoice } from '../hooks/useSales'
import { paymentStateBadge } from '../invoiceLines'
import { invoiceDocType, receiptDocType } from '../keys'

const list = '/sales/invoices'

// InvoicePage shows an invoice with what has been received of it; a draft the user may edit is
// edited in place.
export function InvoicePage() {
  const { t } = useTranslation()
  const id = Number(useParams().id)
  const state = loaded(useInvoice(id), t, { to: list, label: t('sales.invoices.back') })
  if (!state.ok) return state.fallback
  const d = state.data
  const payment = d.status === 'posted' ? paymentStateBadge(d.payment_state) : null
  return (
    <DocumentScreen
      docType={invoiceDocType}
      doc={d}
      list={{ label: t('sales.invoices.title'), to: list }}
      facts={[
        d.customer.name,
        <Anchor key="order" render={<Link to={`/sales/orders/${d.order_id}`} />} size="sm">
          {t('sales.delivery.order_link', { number: d.order_number })}
        </Anchor>,
        formatDate(d.date),
        d.status === 'posted' && t('sales.invoice.paid_of', { paid: formatNumber(d.paid), total: formatNumber(d.totals.total) }),
        d.receipts.length > 0 && (
          <Group gap="xs" render={<span />}>
            {d.receipts.map((r) => (
              <Group key={r.id} gap="xs" wrap={false} render={<span />}>
                <Anchor render={<Link to={`/sales/receipts/${r.id}`} />} size="sm">
                  {t('sales.invoice.receipt_link', { number: r.number, amount: formatNumber(r.amount) })}
                </Anchor>
                <DocumentStatus docType={receiptDocType} status={r.status} />
              </Group>
            ))}
          </Group>
        ),
      ]}
      badges={payment && <StatusBadge tone={payment.tone}>{t(payment.label)}</StatusBadge>}
      attachments
      fullWidth
      extraActions={() =>
        d.allowed_actions.includes('create_receipt') && (
          <Button variant="default" leftSection={<IconCash {...icon.button} />} render={<Link to={`/sales/receipts/new?invoice=${d.id}`} />}>
            {t('sales.invoice.create_receipt')}
          </Button>
        )
      }
      printLabel={t('sales.invoice.print')}
    >
      {/* Keyed by version: a reload after a conflict starts from the saved data. */}
      <InvoiceForm key={d.version} order={d.order_id} doc={d} />
    </DocumentScreen>
  )
}
