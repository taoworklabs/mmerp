import { Anchor, FieldList } from '@mmerp/ui'
import { IconAddressBook, IconBuildingStore } from '@tabler/icons-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import type { Supplier } from '@/shared/api/purchase'
import { useCan } from '@/shared/auth/me'
import { canEditPartner } from '@/shared/master'
import { MissingValue } from '@/shared/ui/MissingValue'
import { EditablePanel, Form, FormActions, FormFields } from '@/shared/ui/form'
import { SupplierStatus } from './SupplierStatus'
import { GeneralFields, SalesFields, useSupplierForm } from './SupplierForm'

type Part = 'general' | 'purchase'

// SupplierProfile is a supplier's values in two cards, each read as label–value pairs and edited
// in place. A card shows every value its form edits, those in the page header included, so what
// can be changed is where it is read. An empty value offers to fill it in. The identity is the
// partner's: only who may edit the partner edits it.
export function SupplierProfile({ supplier: c, onSaved }: { supplier: Supplier; onSaved: () => Promise<void> }) {
  const { t } = useTranslation()
  const canEdit = c.allowed_actions.includes('edit')
  const canSeePartner = useCan('partner.view')
  const canEditIdentity = canEdit && canEditPartner({ allowed_actions: c.partner_actions })
  const edit = (part: Part) => (close: () => void) => <PartForm supplier={c} part={part} onClose={close} onSaved={onSaved} />
  // An empty value: "+ Bổ sung" opening the card's form for who may edit it.
  const or = (may: boolean) => (open: () => void, label: string, value: ReactNode) => value || <MissingValue label={label} onAdd={may ? open : undefined} />
  const orGeneral = or(canEditIdentity)
  const orSales = or(canEdit)
  return (
    <>
      <EditablePanel title={t('purchase.supplier.section.general')} icon={IconAddressBook} editable={canEditIdentity} edit={edit('general')}>
        {(open) => (
          <FieldList
            stacked
            rows={[
              [t('purchase.supplier.name'), c.partner.name],
              [t('purchase.supplier.code'), c.code],
              [
                t('purchase.supplier.partner'),
                canSeePartner ? (
                  <Anchor render={<Link to={`/sales/partners/${c.partner_id}`} />}>
                    {c.partner_code}
                  </Anchor>
                ) : (
                  c.partner_code
                ),
              ],
              [t('purchase.supplier.tax_code'), orGeneral(open, t('purchase.supplier.tax_code'), c.partner.tax_code)],
              [t('purchase.supplier.contact_name'), orGeneral(open, t('purchase.supplier.contact_name'), c.partner.contact_name)],
              [
                t('purchase.supplier.phone'),
                orGeneral(open, t('purchase.supplier.phone'), c.partner.phone && <Anchor href={`tel:${c.partner.phone.replace(/\s/g, '')}`}>{c.partner.phone}</Anchor>),
              ],
              [t('purchase.supplier.email'), orGeneral(open, t('purchase.supplier.email'), c.partner.email && <Anchor href={`mailto:${c.partner.email}`}>{c.partner.email}</Anchor>)],
              [t('purchase.supplier.address'), orGeneral(open, t('purchase.supplier.address'), c.partner.address), 'wide'],
            ]}
          />
        )}
      </EditablePanel>
      <EditablePanel title={t('purchase.supplier.section.purchase')} icon={IconBuildingStore} editable={canEdit} edit={edit('purchase')}>
        {(open) => (
          <FieldList
            stacked
            rows={[
              [t('purchase.supplier.org_unit'), c.org_unit_name],
              [t('purchase.supplier.status'), <SupplierStatus key="status" active={c.active} />],
              [t('purchase.supplier.payment_terms'), orSales(open, t('purchase.supplier.payment_terms'), c.payment_terms), 'wide'],
            ]}
          />
        )}
      </EditablePanel>
    </>
  )
}

// PartForm edits one part in its card with a form holding the whole supplier; only the general
// part sends the partner's identity.
function PartForm({ supplier: c, part, onClose, onSaved }: { supplier: Supplier; part: Part; onClose: () => void; onSaved: () => Promise<void> }) {
  const { t } = useTranslation()
  const { form, save, errorFields } = useSupplierForm(c, async () => {
    await onSaved()
    onClose()
  }, part === 'general')
  return (
    // The one error mapped to a field (code taken) belongs to the general part.
    <Form form={form} onSubmit={save} errorFields={part === 'general' ? errorFields : undefined}>
      <FormFields>{part === 'general' ? <GeneralFields existing /> : <SalesFields />}</FormFields>
      <FormActions submitLabel={t('purchase.common.save')} onCancel={onClose} cancelLabel={t('purchase.common.cancel')} />
    </Form>
  )
}
