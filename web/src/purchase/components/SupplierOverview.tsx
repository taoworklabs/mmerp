import { Anchor, Group, KeyFacts, Panel, Stack, Text } from '@mmerp/ui'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import type { Supplier } from '@/shared/api/purchase'
import { ActivityFeed, DocumentStatus } from '@/shared/document'
import { formatDate, formatNumber } from '@/shared/i18n'
import { DataTable } from '@/shared/ui/DataTable'
import { MissingValue } from '@/shared/ui/MissingValue'
import { SeeAll, TabContent } from '@/shared/ui/page'
import { QueryView } from '@/shared/ui/QueryView'
import { ContentSkeleton, EmptyState } from '@/shared/ui/states'
import { useSupplierOrders } from '../hooks/usePurchase'
import { orderDocType, supplierDocType } from '../keys'
import { useOrderColumns } from './docColumns'

// SupplierSummary is the supplier's key facts under the page title: who to talk to, the default
// terms, and how many orders they have. A list the user may not read is left out.
export function SupplierSummary({ supplier: c }: { supplier: Supplier }) {
  const { t } = useTranslation()
  const orders = useSupplierOrders(c.id)
  const count = (q: typeof orders) => (q.isPending ? '…' : formatNumber(q.data?.total ?? 0))
  return (
    <KeyFacts
      facts={[
        [t('purchase.supplier.contact_name'), c.partner.contact_name || <MissingValue label={t('purchase.supplier.contact_name')} />],
        [t('purchase.supplier.payment_terms'), c.payment_terms || <MissingValue label={t('purchase.supplier.payment_terms')} />],
        ...(!orders.isError ? [[t('purchase.orders.title'), count(orders)] as [string, string]] : []),
      ]}
    />
  )
}

// SupplierOrdersTab: the supplier's orders, newest first, each opening its page.
export function SupplierOrdersTab({ supplier: c }: { supplier: Supplier }) {
  const { t } = useTranslation()
  const list = useSupplierOrders(c.id)
  const columns = useOrderColumns({ supplier: false })
  return (
    <TabContent>
      <QueryView query={list} select={(d) => d.items} empty={<EmptyState title={t('purchase.orders.empty')} />}>
        {(rows) => <DataTable label={t('purchase.orders.title')} columns={columns} rows={rows} rowKey={(d) => d.id} href={(d) => `/purchase/orders/${d.id}`} />}
      </QueryView>
    </TabContent>
  )
}

// Latest is a digest of the supplier's three latest orders.
function Latest({ supplier: c }: { supplier: Supplier }) {
  const { t } = useTranslation()
  const list = useSupplierOrders(c.id)
  if (list.isError) return null
  return (
    <Panel plain title={t('purchase.supplier.latest_order')} actions={<SeeAll tab="orders" />}>
      {list.isPending ? (
        <ContentSkeleton />
      ) : list.data.items.length === 0 ? (
        <Text size="sm" tone="dimmed">
          {t('purchase.orders.empty')}
        </Text>
      ) : (
        <Stack gap="sm">
          {list.data.items.slice(0, 3).map((d) => (
            <Stack key={d.id} gap={0}>
              <Group gap="sm" justify="space-between" wrap={false}>
                <Anchor render={<Link to={`/purchase/orders/${d.id}`} />} weight={600}>
                  {d.number}
                </Anchor>
                <DocumentStatus docType={orderDocType} status={d.status} />
              </Group>
              <Text size="sm" tone="dimmed">
                {t('purchase.supplier.doc_line', { date: formatDate(d.date), total: formatNumber(d.total) })}
              </Text>
            </Stack>
          ))}
        </Stack>
      )}
    </Panel>
  )
}

// SupplierDigest is the overview's right column: the latest orders, then the latest comments and
// changes, each leading to its tab.
export function SupplierDigest({ supplier: c }: { supplier: Supplier }) {
  const { t } = useTranslation()
  return (
    <>
      <Latest supplier={c} />
      <Panel plain title={t('shared.record.recent_activity')} actions={<SeeAll tab="activity" />}>
        <ActivityFeed docType={supplierDocType} id={c.id} recent={5} />
      </Panel>
    </>
  )
}
