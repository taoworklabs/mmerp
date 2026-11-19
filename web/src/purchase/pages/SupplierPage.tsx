import { RecordColumns } from '@mmerp/ui'
import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router'
import { useInvalidateRecord } from '@/shared/area'
import { ActivityFeed, useAttachmentSection } from '@/shared/document'
import { PersonAvatar } from '@/shared/ui/avatar'
import { loaded } from '@/shared/ui/loaded'
import { notifySuccess } from '@/shared/ui/notify'
import { RecordPage } from '@/shared/ui/page'
import { SupplierDigest, SupplierOrdersTab, SupplierSummary } from '../components/SupplierOverview'
import { SupplierProfile } from '../components/SupplierProfile'
import { SupplierStatus } from '../components/SupplierStatus'
import { useSupplier, useSupplierOrders } from '../hooks/usePurchase'
import { supplierDocType } from '../keys'

export function SupplierPage() {
  const { t } = useTranslation()
  const invalidate = useInvalidateRecord()
  const id = Number(useParams().id)
  const supplier = useSupplier(id)
  const attachments = useAttachmentSection(supplierDocType, id)
  const orders = useSupplierOrders(id)

  const state = loaded(supplier, t, { to: '/purchase/suppliers', label: t('purchase.suppliers.back') })
  if (!state.ok) return state.fallback
  const c = state.data
  const saved = async () => {
    await invalidate(supplierDocType, c.id)
    notifySuccess(t('purchase.supplier.saved'))
  }
  // A list the user may not read has no tab.
  const count = orders.data?.total
  const ordersTab = orders.isError
    ? []
    : [{ value: 'orders', label: count ? t('purchase.orders.title_count', { count }) : t('purchase.orders.title'), content: <SupplierOrdersTab supplier={c} /> }]
  return (
    <RecordPage
      title={c.partner.name}
      breadcrumbs={[{ label: t('purchase.suppliers.title'), to: '/purchase/suppliers' }]}
      leading={<PersonAvatar name={c.partner.name} size="lg" />}
      status={<SupplierStatus active={c.active} />}
      description={[c.code, c.org_unit_name].join(' · ')}
      summary={<SupplierSummary supplier={c} />}
      tabs={[
        {
          value: 'overview',
          label: t('shared.record.overview'),
          content: <RecordColumns main={<SupplierProfile key={c.id} supplier={c} onSaved={saved} />} side={<SupplierDigest supplier={c} />} />,
        },
        ...ordersTab,
        { value: attachments.key, label: attachments.title, content: attachments.content },
        { value: 'activity', label: t('shared.document.tab.activity'), content: <ActivityFeed docType={supplierDocType} id={c.id} /> },
      ]}
      // The activity was a card beside the profile; its old links land on its tab.
      aliases={{ history: 'activity', discussion: 'activity' }}
    />
  )
}
