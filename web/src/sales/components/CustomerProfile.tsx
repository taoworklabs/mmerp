import { Anchor } from '@mantine/core'
import { IconAddressBook, IconBuildingStore } from '@tabler/icons-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import type { Customer } from '@/shared/api/sales'
import { FieldList, MissingValue } from '@/shared/ui/FieldList'
import { EditablePanel, Form, FormActions, FormFields } from '@/shared/ui/form'
import { CustomerStatus } from './CustomerStatus'
import { GeneralFields, SalesFields, useCustomerForm } from './CustomerForm'

type Part = 'general' | 'sales'

// CustomerProfile is a customer's values in two cards, each read as label–value pairs and edited
// in place. A card shows every value its form edits, those in the page header included, so what
// can be changed is where it is read. An empty value offers to fill it in.
export function CustomerProfile({ customer: c, onSaved }: { customer: Customer; onSaved: () => Promise<void> }) {
  const { t } = useTranslation()
  const canEdit = c.allowed_actions.includes('edit')
  const edit = (part: Part) => (close: () => void) => <PartForm customer={c} part={part} onClose={close} onSaved={onSaved} />
  // An empty value: "+ Bổ sung" opening the card's form for who may edit.
  const or = (open: () => void, label: string, value: ReactNode) => value || <MissingValue label={label} onAdd={canEdit ? open : undefined} />
  return (
    <>
      <EditablePanel title={t('sales.customer.section.general')} icon={IconAddressBook} editable={canEdit} edit={edit('general')}>
        {(open) => (
          <FieldList
            stacked
            rows={[
              [t('sales.customer.name'), c.name],
              [t('sales.customer.code'), c.code],
              [t('sales.customer.tax_code'), or(open, t('sales.customer.tax_code'), c.tax_code)],
              [t('sales.customer.contact_name'), or(open, t('sales.customer.contact_name'), c.contact_name)],
              [
                t('sales.customer.phone'),
                or(open, t('sales.customer.phone'), c.phone && <Anchor href={`tel:${c.phone.replace(/\s/g, '')}`} size="sm" inherit>{c.phone}</Anchor>),
              ],
              [t('sales.customer.email'), or(open, t('sales.customer.email'), c.email && <Anchor href={`mailto:${c.email}`} size="sm" inherit>{c.email}</Anchor>)],
              [t('sales.customer.address'), or(open, t('sales.customer.address'), c.address), 'wide'],
            ]}
          />
        )}
      </EditablePanel>
      <EditablePanel title={t('sales.customer.section.sales')} icon={IconBuildingStore} editable={canEdit} edit={edit('sales')}>
        {(open) => (
          <FieldList
            stacked
            rows={[
              [t('sales.customer.org_unit'), c.org_unit_name],
              [t('sales.customer.status'), <CustomerStatus key="status" active={c.active} />],
              [t('sales.customer.payment_terms'), or(open, t('sales.customer.payment_terms'), c.payment_terms), 'wide'],
            ]}
          />
        )}
      </EditablePanel>
    </>
  )
}

// PartForm edits one part in its card with a form holding the whole customer: the API saves it as a whole.
function PartForm({ customer: c, part, onClose, onSaved }: { customer: Customer; part: Part; onClose: () => void; onSaved: () => Promise<void> }) {
  const { t } = useTranslation()
  const { form, save, errorFields } = useCustomerForm(c, async () => {
    await onSaved()
    onClose()
  })
  return (
    // The one error mapped to a field (code taken) belongs to the general part.
    <Form form={form} onSubmit={save} errorFields={part === 'general' ? errorFields : undefined}>
      <FormFields>{part === 'general' ? <GeneralFields existing /> : <SalesFields />}</FormFields>
      <FormActions submitLabel={t('sales.common.save')} onCancel={onClose} cancelLabel={t('sales.common.cancel')} />
    </Form>
  )
}
