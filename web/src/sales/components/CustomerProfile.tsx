import { Anchor } from '@mantine/core'
import { IconAddressBook, IconBuildingStore } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import type { Customer } from '@/shared/api/sales'
import { FieldList } from '@/shared/ui/FieldList'
import { EditablePanel, Form, FormActions, FormFields } from '@/shared/ui/form'
import { CustomerStatus } from './CustomerStatus'
import { GeneralFields, SalesFields, useCustomerForm } from './CustomerForm'

type Part = 'general' | 'sales'

const orDash = (s: string | null | undefined) => s || '—'

// CustomerProfile is a customer's values in two cards, each read as label–value pairs and edited
// in place. A card shows every value its form edits, those in the page header included.
export function CustomerProfile({ customer: c, onSaved }: { customer: Customer; onSaved: () => Promise<void> }) {
  const { t } = useTranslation()
  const canEdit = c.allowed_actions.includes('edit')
  const edit = (part: Part) => (close: () => void) => <PartForm customer={c} part={part} onClose={close} onSaved={onSaved} />
  return (
    <>
      <EditablePanel title={t('sales.customer.section.general')} icon={IconAddressBook} editable={canEdit} edit={edit('general')}>
        <FieldList
          rows={[
            [t('sales.customer.code'), c.code],
            [t('sales.customer.name'), c.name],
            [t('sales.customer.tax_code'), orDash(c.tax_code)],
            [t('sales.customer.contact_name'), orDash(c.contact_name)],
            [t('sales.customer.phone'), c.phone ? <Anchor href={`tel:${c.phone.replace(/\s/g, '')}`} size="sm" inherit>{c.phone}</Anchor> : '—'],
            [t('sales.customer.email'), c.email ? <Anchor href={`mailto:${c.email}`} size="sm" inherit>{c.email}</Anchor> : '—'],
            [t('sales.customer.address'), orDash(c.address)],
          ]}
        />
      </EditablePanel>
      <EditablePanel title={t('sales.customer.section.sales')} icon={IconBuildingStore} editable={canEdit} edit={edit('sales')}>
        <FieldList
          rows={[
            [t('sales.customer.org_unit'), c.org_unit_name],
            [t('sales.customer.payment_terms'), orDash(c.payment_terms)],
            [t('sales.customer.status'), <CustomerStatus key="status" active={c.active} />],
          ]}
        />
      </EditablePanel>
    </>
  )
}

// PartForm edits one part in its card with a form holding the whole customer: the API saves it as a whole.
function PartForm({ customer: c, part, onClose, onSaved }: { customer: Customer; part: Part; onClose: () => void; onSaved: () => Promise<void> }) {
  const { t } = useTranslation()
  const { form, save, fieldOf } = useCustomerForm(c, async () => {
    await onSaved()
    onClose()
  })
  return (
    // The one error mapped to a field (code taken) belongs to the general part.
    <Form form={form} onSubmit={save} fieldOf={part === 'general' ? fieldOf : undefined}>
      <FormFields>{part === 'general' ? <GeneralFields existing /> : <SalesFields />}</FormFields>
      <FormActions submitLabel={t('sales.common.save')} onCancel={onClose} cancelLabel={t('sales.common.cancel')} />
    </Form>
  )
}
