import { Button, Stack, Text } from '@mantine/core'
import { IconAddressBook, IconBuildingStore } from '@tabler/icons-react'
import { useState } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { ApiError, type Partner, type PartnerListItem } from '@/shared/api/core'
import { api, unwrap, type Customer, type CustomerUpdate, type NewCustomer } from '@/shared/api/sales'
import { useCan } from '@/shared/auth/me'
import { PartnerFields, PartnerPicker, partnerBody, partnerValues, usePartner, type PartnerValues } from '@/shared/master'
import { Notice } from '@/shared/ui/Notice'
import { CheckboxField, CodeField, Form, FormActions, FormRow, FormSection, OrgUnitField, TextField, type RemoteRow } from '@/shared/ui/form'

// partner_id: an existing partner picked for a new customer; its identity is then only shown.
export type CustomerValues = PartnerValues & {
  code: string
  partner_id: string | null
  payment_terms: string
  org_unit_id: string | null
  active: boolean
}

const orNull = (s: string) => (s.trim() === '' ? null : s.trim())

const roleBody = (v: CustomerValues) => ({ payment_terms: orNull(v.payment_terms), org_unit_id: Number(v.org_unit_id), active: v.active })

// newCustomerBody: the customer over the picked partner, or over a new one with the typed identity.
export function newCustomerBody(v: CustomerValues): NewCustomer {
  const role = { code: v.code.trim(), ...roleBody(v) }
  return v.partner_id ? { ...role, partner_id: Number(v.partner_id) } : { ...role, partner: partnerBody(v) }
}

// customerUpdateBody: the identity only from the card that edits it, so saving the role alone
// leaves the partner as others last saved it.
export const customerUpdateBody = (v: CustomerValues, identity: boolean): CustomerUpdate =>
  identity ? { partner: partnerBody(v), ...roleBody(v) } : roleBody(v)

// takenPartnerId: the partner already holding the typed tax code, when that is why the save failed.
export function takenPartnerId(err: unknown): number | undefined {
  if (!(err instanceof ApiError) || err.code !== 'partner_tax_code_taken') return undefined
  const id = err.params.partner_id
  return typeof id === 'number' ? id : undefined
}

// withPartner puts an existing partner, its identity shown, in place of the typed one.
export const withPartner = (v: CustomerValues, p: PartnerListItem | Partner): CustomerValues => ({ ...v, ...partnerValues(p), partner_id: String(p.id) })

// useCustomerForm holds the whole customer whichever part of it is on screen; an edit sends the
// identity only when it is the part on screen.
export function useCustomerForm(customer: Customer | undefined, onSaved: (id: number) => Promise<void> | void, identity = false) {
  const [taken, setTaken] = useState<number>()
  const form = useForm<CustomerValues>({
    mode: 'onBlur',
    defaultValues: {
      code: customer?.code ?? '',
      partner_id: null,
      ...partnerValues(customer?.partner),
      payment_terms: customer?.payment_terms ?? '',
      org_unit_id: customer ? String(customer.org_unit_id) : null,
      active: customer?.active ?? true,
    },
  })

  async function save(v: CustomerValues) {
    setTaken(undefined)
    if (customer) {
      await unwrap(api.PUT('/sales/customers/{id}', { params: { path: { id: customer.id } }, body: customerUpdateBody(v, identity) }))
      await onSaved(customer.id)
      return
    }
    try {
      const { id } = await unwrap(api.POST('/sales/customers', { body: newCustomerBody(v) }))
      await onSaved(id)
    } catch (err) {
      // The company is known already: offer it rather than a second one.
      const id = takenPartnerId(err)
      if (id === undefined) throw err
      setTaken(id)
      return false
    }
  }

  return { form, save, taken, errorFields: { customer_code_taken: 'code', partner_tax_code_taken: 'tax_code' } as const }
}

export function GeneralFields({ existing, identityReadOnly }: { existing?: boolean; identityReadOnly?: boolean }) {
  const { t } = useTranslation()
  return (
    <>
      <CodeField name="code" label={t('sales.customer.code')} existing={!!existing} />
      <PartnerFields nameLabel={t('sales.customer.name')} readOnly={identityReadOnly} />
    </>
  )
}

export function SalesFields() {
  const { t } = useTranslation()
  return (
    <>
      <OrgUnitField name="org_unit_id" label={t('sales.customer.org_unit')} required product="sales" permission="sales.customer.edit" />
      <TextField name="payment_terms" label={t('sales.customer.payment_terms')} description={t('sales.customer.payment_terms_hint')} maxLength={1000} />
      <CheckboxField name="active" label={t('sales.customer.active')} description={t('sales.customer.active_hint')} />
    </>
  )
}

// TakenPartner names the partner holding the typed tax code and offers to make the customer
// over it; the button submits the form once the partner is in place.
function TakenPartner({ id, onUse }: { id: number; onUse: (p: Partner) => void }) {
  const { t } = useTranslation()
  const partner = usePartner(id)
  if (partner.isPending) return null
  if (partner.isError) return <Notice tone="danger">{t('shared.error.partner_tax_code_taken')}</Notice>
  const p = partner.data
  return (
    <Notice tone="warning" announce="alert" title={t('sales.customer.partner_taken_title')}>
      <Stack gap="sm" align="flex-start">
        <Text size="sm">{t('sales.customer.partner_taken', { name: p.name, code: p.code })}</Text>
        <Button type="submit" variant="default" onClick={() => onUse(p)}>
          {t('sales.customer.use_partner')}
        </Button>
      </Stack>
    </Notice>
  )
}

// CustomerForm creates a customer over an existing partner, picked or found by its tax code, or
// over a new one; an existing customer is edited a part at a time on its page.
export function CustomerForm({ onSaved }: { onSaved: (id: number) => Promise<void> | void }) {
  const { t } = useTranslation()
  const canPick = useCan('partner.view')
  const { form, save, taken, errorFields } = useCustomerForm(undefined, onSaved)
  const [picked, setPicked] = useState<RemoteRow>()
  const partnerId = useWatch({ control: form.control, name: 'partner_id' })
  const use = (p: PartnerListItem | Partner) => {
    setPicked(p)
    const v = withPartner(form.getValues(), p)
    for (const k of ['partner_id', 'name', 'tax_code', 'address', 'phone', 'email', 'contact_name'] as const) form.setValue(k, v[k], { shouldDirty: true })
    form.clearErrors()
  }
  return (
    <Form form={form} onSubmit={save} errorFields={errorFields}>
      <FormSection title={t('sales.customer.section.general')} description={t('sales.customer.section.general_hint')} icon={IconAddressBook}>
        {taken !== undefined && (
          <FormRow>
            <TakenPartner id={taken} onUse={use} />
          </FormRow>
        )}
        {canPick && <PartnerPicker name="partner_id" label={t('sales.customer.existing_partner')} current={picked} onPicked={use} />}
        <GeneralFields identityReadOnly={!!partnerId} />
      </FormSection>
      <FormSection title={t('sales.customer.section.sales')} description={t('sales.customer.section.sales_hint')} icon={IconBuildingStore}>
        <SalesFields />
      </FormSection>
      <FormActions submitLabel={t('sales.customers.create')} cancelTo="/sales/customers" />
    </Form>
  )
}
