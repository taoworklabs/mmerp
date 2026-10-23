import { Anchor } from '@mantine/core'
import { IconAddressBook, IconBuildingStore } from '@tabler/icons-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import type { Customer } from '@/shared/api/sales'
import { useCan } from '@/shared/auth/me'
import { canEditPartner } from '@/shared/master'
import { FieldList, MissingValue } from '@/shared/ui/FieldList'
import { EditablePanel, Form, FormActions, FormFields } from '@/shared/ui/form'
import { CustomerStatus } from './CustomerStatus'
import { GeneralFields, SalesFields, useCustomerForm } from './CustomerForm'

type Part = 'general' | 'sales'

// CustomerProfile is a customer's values in two cards, each read as label–value pairs and edited
// in place. A card shows every value its form edits, those in the page header included, so what
// can be changed is where it is read. An empty value offers to fill it in. The identity is the
// partner's: only who may edit the partner edits it.
export function CustomerProfile({ customer: c, onSaved }: { customer: Customer; onSaved: () => Promise<void> }) {
  const { t } = useTranslation()
  const canEdit = c.allowed_actions.includes('edit')
  const canSeePartner = useCan('partner.view')
  const canEditIdentity = canEdit && canEditPartner({ allowed_actions: c.partner_actions })
  const edit = (part: Part) => (close: () => void) => <PartForm customer={c} part={part} onClose={close} onSaved={onSaved} />
  // An empty value: "+ Bổ sung" opening the card's form for who may edit it.
  const or = (may: boolean) => (open: () => void, label: string, value: ReactNode) => value || <MissingValue label={label} onAdd={may ? open : undefined} />
  const orGeneral = or(canEditIdentity)
  const orSales = or(canEdit)
  return (
    <>
      <EditablePanel title={t('sales.customer.section.general')} icon={IconAddressBook} editable={canEditIdentity} edit={edit('general')}>
        {(open) => (
          <FieldList
            stacked
            rows={[
              [t('sales.customer.name'), c.partner.name],
              [t('sales.customer.code'), c.code],
              [
                t('sales.customer.partner'),
                canSeePartner ? (
                  <Anchor component={Link} to={`/sales/partners/${c.partner_id}`} size="sm" inherit>
                    {c.partner_code}
                  </Anchor>
                ) : (
                  c.partner_code
                ),
              ],
              [t('sales.customer.tax_code'), orGeneral(open, t('sales.customer.tax_code'), c.partner.tax_code)],
              [t('sales.customer.contact_name'), orGeneral(open, t('sales.customer.contact_name'), c.partner.contact_name)],
              [
                t('sales.customer.phone'),
                orGeneral(open, t('sales.customer.phone'), c.partner.phone && <Anchor href={`tel:${c.partner.phone.replace(/\s/g, '')}`} size="sm" inherit>{c.partner.phone}</Anchor>),
              ],
              [t('sales.customer.email'), orGeneral(open, t('sales.customer.email'), c.partner.email && <Anchor href={`mailto:${c.partner.email}`} size="sm" inherit>{c.partner.email}</Anchor>)],
              [t('sales.customer.address'), orGeneral(open, t('sales.customer.address'), c.partner.address), 'wide'],
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
              [t('sales.customer.payment_terms'), orSales(open, t('sales.customer.payment_terms'), c.payment_terms), 'wide'],
            ]}
          />
        )}
      </EditablePanel>
    </>
  )
}

// PartForm edits one part in its card with a form holding the whole customer; only the general
// part sends the partner's identity.
function PartForm({ customer: c, part, onClose, onSaved }: { customer: Customer; part: Part; onClose: () => void; onSaved: () => Promise<void> }) {
  const { t } = useTranslation()
  const { form, save, errorFields } = useCustomerForm(c, async () => {
    await onSaved()
    onClose()
  }, part === 'general')
  return (
    // The one error mapped to a field (code taken) belongs to the general part.
    <Form form={form} onSubmit={save} errorFields={part === 'general' ? errorFields : undefined}>
      <FormFields>{part === 'general' ? <GeneralFields existing /> : <SalesFields />}</FormFields>
      <FormActions submitLabel={t('sales.common.save')} onCancel={onClose} cancelLabel={t('sales.common.cancel')} />
    </Form>
  )
}
