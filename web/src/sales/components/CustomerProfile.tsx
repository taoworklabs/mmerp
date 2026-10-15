import { Divider, Stack } from '@mantine/core'
import { IconAddressBook, IconBuildingStore, type Icon as TablerIcon } from '@tabler/icons-react'
import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { Customer } from '@/shared/api/sales'
import { FieldList } from '@/shared/ui/FieldList'
import { EditableSection, Form, FormActions } from '@/shared/ui/form'
import { FormModal } from '@/shared/ui/page'
import { CustomerStatus } from './CustomerStatus'
import { GeneralFields, SalesFields, useCustomerForm } from './CustomerForm'

type Part = 'general' | 'sales'

const orDash = (s: string | null | undefined) => s || '—'

// CustomerProfile is a customer's values in two parts, each read as label–value pairs and
// edited on its own in a modal: the column is too narrow for a form.
export function CustomerProfile({ customer: c, onSaved }: { customer: Customer; onSaved: () => Promise<void> }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState<Part | null>(null)
  // The modal unmounts on closing, before it could hand the focus back; this does it.
  const opener = useRef<HTMLElement | null>(null)
  const canEdit = c.allowed_actions.includes('edit')
  const part = (p: Part, title: string, icon: TablerIcon) => ({
    title,
    icon,
    editable: canEdit,
    onEdit: () => {
      opener.current = document.activeElement as HTMLElement | null
      setOpen(p)
    },
  })
  const close = () => {
    setOpen(null)
    requestAnimationFrame(() => opener.current?.focus())
  }

  return (
    <Stack gap="md">
      <EditableSection {...part('general', t('sales.customer.section.general'), IconAddressBook)}>
        <FieldList
          rows={[
            [t('sales.customer.code'), c.code],
            [t('sales.customer.name'), c.name],
            [t('sales.customer.tax_code'), orDash(c.tax_code)],
            [t('sales.customer.contact_name'), orDash(c.contact_name)],
            [t('sales.customer.phone'), orDash(c.phone)],
            [t('sales.customer.email'), orDash(c.email)],
            [t('sales.customer.address'), orDash(c.address)],
          ]}
        />
      </EditableSection>
      <Divider />
      <EditableSection {...part('sales', t('sales.customer.section.sales'), IconBuildingStore)}>
        <FieldList
          rows={[
            [t('sales.customer.org_unit'), c.org_unit_name],
            [t('sales.customer.payment_terms'), orDash(c.payment_terms)],
            [t('sales.customer.status'), <CustomerStatus key="status" active={c.active} />],
          ]}
        />
      </EditableSection>
      {open && <PartModal customer={c} part={open} onClose={close} onSaved={onSaved} />}
    </Stack>
  )
}

// PartModal edits one part with a form holding the whole customer: the API saves it as a whole.
function PartModal({ customer: c, part, onClose, onSaved }: { customer: Customer; part: Part; onClose: () => void; onSaved: () => Promise<void> }) {
  const { t } = useTranslation()
  const { form, save, fieldOf } = useCustomerForm(c, async () => {
    await onSaved()
    onClose()
  })
  return (
    <FormModal opened title={t(`sales.customer.edit.${part}`)} onClose={onClose}>
      {/* The one error mapped to a field (code taken) belongs to the general part. */}
      <Form form={form} onSubmit={save} fieldOf={part === 'general' ? fieldOf : undefined}>
        {part === 'general' ? <GeneralFields existing /> : <SalesFields />}
        <FormActions submitLabel={t('sales.common.save')} onCancel={onClose} cancelLabel={t('sales.common.cancel')} />
      </Form>
    </FormModal>
  )
}
