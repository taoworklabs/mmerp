import { Anchor } from '@mantine/core'
import { IconAddressBook } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router'
import { api as coreApi, type Partner } from '@/shared/api/core'
import { api, unwrap, type CustomerListItem } from '@/shared/api/sales'
import { useInvalidateRecord } from '@/shared/area'
import { ActivityFeed, useAttachmentSection } from '@/shared/document'
import { canEditPartner, PartnerFields, partnerBody, partnerValues, usePartner, type PartnerValues } from '@/shared/master'
import { PersonAvatar } from '@/shared/ui/avatar'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { FieldList, MissingValue } from '@/shared/ui/FieldList'
import { EditablePanel, Form, FormActions, FormFields } from '@/shared/ui/form'
import { loaded } from '@/shared/ui/loaded'
import { notifySuccess } from '@/shared/ui/notify'
import { Panel, RecordColumns, RecordPage, SeeAll, TabContent } from '@/shared/ui/page'
import { QueryView } from '@/shared/ui/QueryView'
import { EmptyState } from '@/shared/ui/states'
import { CustomerStatus } from '../components/CustomerStatus'
import { partnerType, salesKeys } from '../keys'

// PartnerPage: a partner's identity, its own files and activity, and the customers it is in
// the units the user may see.
export function PartnerPage() {
  const { t } = useTranslation()
  const invalidate = useInvalidateRecord()
  const id = Number(useParams().id)
  const partner = usePartner(id)
  const attachments = useAttachmentSection(partnerType, id)
  const customers = usePartnerCustomers(id)

  const state = loaded(partner, t, { to: '/sales/customers', label: t('sales.customers.back') })
  if (!state.ok) return state.fallback
  const p = state.data
  const saved = async () => {
    await invalidate(partnerType, p.id)
    notifySuccess(t('sales.partner.saved'))
  }
  const count = customers.data?.total
  return (
    <RecordPage
      title={p.name}
      breadcrumbs={[{ label: t('sales.customers.title'), to: '/sales/customers' }]}
      leading={<PersonAvatar name={p.name} size="lg" />}
      description={[p.code, p.tax_code].filter(Boolean).join(' · ')}
      tabs={[
        {
          value: 'overview',
          label: t('shared.record.overview'),
          content: (
            <RecordColumns
              main={<PartnerProfile key={p.id} partner={p} onSaved={saved} />}
              side={
                <Panel plain title={t('shared.record.recent_activity')} actions={<SeeAll tab="activity" />}>
                  <ActivityFeed docType={partnerType} id={p.id} recent={5} />
                </Panel>
              }
            />
          ),
        },
        // A list the user may not read has no tab.
        ...(customers.isError
          ? []
          : [{ value: 'customers', label: count ? t('sales.partner.customers_count', { count }) : t('sales.partner.customers'), content: <CustomersTab partner={p.id} /> }]),
        { value: attachments.key, label: attachments.title, content: attachments.content },
        { value: 'activity', label: t('shared.document.tab.activity'), content: <ActivityFeed docType={partnerType} id={p.id} /> },
      ]}
    />
  )
}

// usePartnerCustomers: the partner's customers in the units the user may see.
function usePartnerCustomers(partner: number) {
  return useQuery({
    queryKey: salesKeys.customers.partner(partner),
    queryFn: () => unwrap(api.GET('/sales/customers', { params: { query: { partner_id: partner, page_size: 100 } } })),
  })
}

function CustomersTab({ partner }: { partner: number }) {
  const { t } = useTranslation()
  const list = usePartnerCustomers(partner)
  const columns: Column<CustomerListItem>[] = [
    { key: 'code', header: t('sales.customer.code'), role: 'title', kind: 'code', render: (c) => c.code },
    { key: 'org_unit', header: t('sales.customer.org_unit'), role: 'meta', render: (c) => c.org_unit_name },
    { key: 'active', header: t('sales.customer.status'), role: 'status', kind: 'status', render: (c) => <CustomerStatus active={c.active} /> },
  ]
  return (
    <TabContent>
      <QueryView query={list} select={(d) => d.items} empty={<EmptyState title={t('sales.partner.no_customers')} />}>
        {(rows) => <DataTable label={t('sales.partner.customers')} columns={columns} rows={rows} rowKey={(c) => c.id} href={(c) => `/sales/customers/${c.id}`} />}
      </QueryView>
    </TabContent>
  )
}

// PartnerProfile: the identity read as label–value pairs, edited in place by who may edit the partner.
function PartnerProfile({ partner: p, onSaved }: { partner: Partner; onSaved: () => Promise<void> }) {
  const { t } = useTranslation()
  const canEdit = canEditPartner(p)
  const or = (open: () => void, label: string, value: ReactNode) => value || <MissingValue label={label} onAdd={canEdit ? open : undefined} />
  return (
    <EditablePanel
      title={t('sales.partner.identity')}
      icon={IconAddressBook}
      editable={canEdit}
      edit={(close) => (
        <IdentityForm
          partner={p}
          onClose={close}
          onSaved={async () => {
            await onSaved()
            close()
          }}
        />
      )}
    >
      {(open) => (
        <FieldList
          stacked
          rows={[
            [t('shared.partner.name'), p.name],
            [t('shared.partner.code'), p.code],
            [t('shared.partner.tax_code'), or(open, t('shared.partner.tax_code'), p.tax_code)],
            [t('shared.partner.contact_name'), or(open, t('shared.partner.contact_name'), p.contact_name)],
            [
              t('shared.partner.phone'),
              or(open, t('shared.partner.phone'), p.phone && <Anchor href={`tel:${p.phone.replace(/\s/g, '')}`} size="sm" inherit>{p.phone}</Anchor>),
            ],
            [t('shared.partner.email'), or(open, t('shared.partner.email'), p.email && <Anchor href={`mailto:${p.email}`} size="sm" inherit>{p.email}</Anchor>)],
            [t('shared.partner.address'), or(open, t('shared.partner.address'), p.address), 'wide'],
          ]}
        />
      )}
    </EditablePanel>
  )
}

function IdentityForm({ partner: p, onClose, onSaved }: { partner: Partner; onClose: () => void; onSaved: () => Promise<void> }) {
  const { t } = useTranslation()
  const form = useForm<PartnerValues>({ mode: 'onBlur', defaultValues: partnerValues(p) })
  async function save(v: PartnerValues) {
    await unwrap(coreApi.PUT('/partners/{id}', { params: { path: { id: p.id } }, body: partnerBody(v) }))
    await onSaved()
  }
  return (
    <Form form={form} onSubmit={save} errorFields={{ partner_tax_code_taken: 'tax_code' }}>
      <FormFields>
        <PartnerFields />
      </FormFields>
      <FormActions submitLabel={t('sales.common.save')} onCancel={onClose} cancelLabel={t('sales.common.cancel')} />
    </Form>
  )
}
