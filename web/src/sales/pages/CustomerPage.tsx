import { Group } from '@mantine/core'
import { useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router'
import { ActivityFeed, documentKeys, useAttachmentSection } from '@/shared/document'
import { PersonAvatar } from '@/shared/ui/avatar'
import { loaded } from '@/shared/ui/loaded'
import { notifySuccess } from '@/shared/ui/notify'
import { RecordPage } from '@/shared/ui/page'
import { CustomerProfile } from '../components/CustomerProfile'
import { CustomerStatus } from '../components/CustomerStatus'
import { useCustomer } from '../hooks/useSales'
import { customerDocType, salesKeys } from '../keys'

export function CustomerPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const id = Number(useParams().id)
  const customer = useCustomer(id)
  const attachments = useAttachmentSection(customerDocType, id)

  const state = loaded(customer, t, { to: '/sales/customers', label: t('sales.customers.back') })
  if (!state.ok) return state.fallback
  const c = state.data
  return (
    <RecordPage
      title={c.name}
      breadcrumbs={[{ label: t('sales.customers.title'), to: '/sales/customers' }]}
      leading={<PersonAvatar name={c.name} size="lg" />}
      description={
        <Group gap="xs" component="span">
          <span>{c.code}</span>
          <span aria-hidden>·</span>
          <span>{c.org_unit_name}</span>
          <CustomerStatus active={c.active} />
        </Group>
      }
      aside={{
        label: t('sales.customer.tab.info'),
        content: (
          <CustomerProfile
            key={c.id}
            customer={c}
            onSaved={async () => {
              await Promise.all([
                qc.invalidateQueries({ queryKey: salesKeys.customers.all() }),
                qc.invalidateQueries({ queryKey: documentKeys.history(customerDocType, c.id) }),
              ])
              notifySuccess(t('sales.customer.saved'))
            }}
          />
        ),
      }}
      tabs={[{ value: attachments.key, label: attachments.title, content: attachments.content, panel: true }]}
      activity={<ActivityFeed docType={customerDocType} id={c.id} />}
    />
  )
}
