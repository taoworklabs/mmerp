import { Anchor, Group, Stack, Text } from '@mmerp/ui'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import type { Customer } from '@/shared/api/sales'
import { ActivityFeed, DocumentStatus } from '@/shared/document'
import { formatDate, formatNumber } from '@/shared/i18n'
import { DataTable } from '@/shared/ui/DataTable'
import { MissingValue } from '@/shared/ui/FieldList'
import { KeyFacts } from '@/shared/ui/KeyFacts'
import { Panel, SeeAll, TabContent } from '@/shared/ui/page'
import { QueryView } from '@/shared/ui/QueryView'
import { ContentSkeleton, EmptyState } from '@/shared/ui/states'
import { useCustomerDocs } from '../hooks/useSales'
import { customerDocType, docTypeOf, pathOf, type Kind } from '../keys'
import { useDocColumns } from './docColumns'

const kinds: Kind[] = ['quote', 'order']

// CustomerSummary is the customer's key facts under the page title: who to talk to, the default
// terms, and how many quotations and orders they have. A list the user may not read is left out.
export function CustomerSummary({ customer: c }: { customer: Customer }) {
  const { t } = useTranslation()
  const quotes = useCustomerDocs('quote', c.id)
  const orders = useCustomerDocs('order', c.id)
  const count = (q: typeof quotes) => (q.isPending ? '…' : formatNumber(q.data?.total ?? 0))
  return (
    <KeyFacts
      facts={[
        [t('sales.customer.contact_name'), c.partner.contact_name || <MissingValue label={t('sales.customer.contact_name')} />],
        [t('sales.customer.payment_terms'), c.payment_terms || <MissingValue label={t('sales.customer.payment_terms')} />],
        ...(!quotes.isError ? [[t('sales.quotes.title'), count(quotes)] as [string, string]] : []),
        ...(!orders.isError ? [[t('sales.orders.title'), count(orders)] as [string, string]] : []),
      ]}
    />
  )
}

// CustomerDocsTab: the customer's quotations or orders, newest first, each opening its page.
export function CustomerDocsTab({ customer: c, kind }: { customer: Customer; kind: Kind }) {
  const { t } = useTranslation()
  const list = useCustomerDocs(kind, c.id)
  const columns = useDocColumns(kind, { customer: false })
  return (
    <TabContent>
      <QueryView query={list} select={(d) => d.items} empty={<EmptyState title={t(`sales.${pathOf(kind)}.empty`)} />}>
        {(rows) => <DataTable label={t(`sales.${pathOf(kind)}.title`)} columns={columns} rows={rows} rowKey={(d) => d.id} href={(d) => `/sales/${pathOf(kind)}/${d.id}`} />}
      </QueryView>
    </TabContent>
  )
}

// Latest is a digest of the customer's three latest quotations or orders.
function Latest({ customer: c, kind }: { customer: Customer; kind: Kind }) {
  const { t } = useTranslation()
  const list = useCustomerDocs(kind, c.id)
  if (list.isError) return null
  return (
    <Panel plain title={t(`sales.customer.latest_${kind}`)} actions={<SeeAll tab={pathOf(kind)} />}>
      {list.isPending ? (
        <ContentSkeleton />
      ) : list.data.items.length === 0 ? (
        <Text size="sm" tone="dimmed">
          {t(`sales.${pathOf(kind)}.empty`)}
        </Text>
      ) : (
        <Stack gap="sm">
          {list.data.items.slice(0, 3).map((d) => (
            <Stack key={d.id} gap={0}>
              <Group gap="sm" justify="space-between" wrap={false}>
                <Anchor render={<Link to={`/sales/${pathOf(kind)}/${d.id}`} />} weight={600}>
                  {d.number}
                </Anchor>
                <DocumentStatus docType={docTypeOf(kind)} status={d.status} />
              </Group>
              <Text size="sm" tone="dimmed">
                {t('sales.customer.doc_line', { date: formatDate(d.date), total: formatNumber(d.total) })}
              </Text>
            </Stack>
          ))}
        </Stack>
      )}
    </Panel>
  )
}

// CustomerDigest is the overview's right column: the latest quotations and orders, then the latest
// comments and changes, each leading to its tab.
export function CustomerDigest({ customer: c }: { customer: Customer }) {
  const { t } = useTranslation()
  return (
    <>
      {kinds.map((k) => (
        <Latest key={k} customer={c} kind={k} />
      ))}
      <Panel plain title={t('shared.record.recent_activity')} actions={<SeeAll tab="activity" />}>
        <ActivityFeed docType={customerDocType} id={c.id} recent={5} />
      </Panel>
    </>
  )
}
