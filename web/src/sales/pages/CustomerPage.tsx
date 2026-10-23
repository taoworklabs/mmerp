import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router'
import { useInvalidateRecord } from '@/shared/area'
import { ActivityFeed, useAttachmentSection } from '@/shared/document'
import { PersonAvatar } from '@/shared/ui/avatar'
import { loaded } from '@/shared/ui/loaded'
import { notifySuccess } from '@/shared/ui/notify'
import { RecordColumns, RecordPage } from '@/shared/ui/page'
import { CustomerDigest, CustomerDocsTab, CustomerSummary } from '../components/CustomerOverview'
import { CustomerProfile } from '../components/CustomerProfile'
import { CustomerStatus } from '../components/CustomerStatus'
import { useCustomer, useCustomerDocs } from '../hooks/useSales'
import { customerDocType, pathOf, type Kind } from '../keys'

export function CustomerPage() {
  const { t } = useTranslation()
  const invalidate = useInvalidateRecord()
  const id = Number(useParams().id)
  const customer = useCustomer(id)
  const attachments = useAttachmentSection(customerDocType, id)
  const docs = { quote: useCustomerDocs('quote', id), order: useCustomerDocs('order', id) }

  const state = loaded(customer, t, { to: '/sales/customers', label: t('sales.customers.back') })
  if (!state.ok) return state.fallback
  const c = state.data
  const saved = async () => {
    await invalidate(customerDocType, c.id)
    notifySuccess(t('sales.customer.saved'))
  }
  // A list the user may not read has no tab.
  const docTab = (k: Kind) => {
    const list = docs[k]
    if (list.isError) return []
    const count = list.data?.total
    return [{ value: pathOf(k), label: count ? t(`sales.${pathOf(k)}.title_count`, { count }) : t(`sales.${pathOf(k)}.title`), content: <CustomerDocsTab customer={c} kind={k} /> }]
  }
  return (
    <RecordPage
      title={c.partner.name}
      breadcrumbs={[{ label: t('sales.customers.title'), to: '/sales/customers' }]}
      leading={<PersonAvatar name={c.partner.name} size="lg" />}
      status={<CustomerStatus active={c.active} />}
      description={[c.code, c.org_unit_name].join(' · ')}
      summary={<CustomerSummary customer={c} />}
      tabs={[
        {
          value: 'overview',
          label: t('shared.record.overview'),
          content: <RecordColumns main={<CustomerProfile key={c.id} customer={c} onSaved={saved} />} side={<CustomerDigest customer={c} />} />,
        },
        ...docTab('quote'),
        ...docTab('order'),
        { value: attachments.key, label: attachments.title, content: attachments.content },
        { value: 'activity', label: t('shared.document.tab.activity'), content: <ActivityFeed docType={customerDocType} id={c.id} /> },
      ]}
      // The activity was a card beside the profile; its old links land on its tab.
      aliases={{ history: 'activity', discussion: 'activity' }}
    />
  )
}
