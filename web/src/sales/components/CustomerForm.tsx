import { IconAddressBook, IconBuildingStore } from '@tabler/icons-react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Customer, type CustomerFields } from '@/shared/api/sales'
import { CheckboxField, CodeField, Form, FormActions, FormSection, OrgUnitField, TextField } from '@/shared/ui/form'

type Values = {
  code: string
  name: string
  tax_code: string
  address: string
  phone: string
  email: string
  contact_name: string
  payment_terms: string
  org_unit_id: string | null
  active: boolean
}

const orNull = (s: string) => (s.trim() === '' ? null : s.trim())

// useCustomerForm holds the whole customer whichever part of it is on screen: the API saves it as a whole.
export function useCustomerForm(customer: Customer | undefined, onSaved: (id: number) => Promise<void> | void) {
  const form = useForm<Values>({
    mode: 'onBlur',
    defaultValues: {
      code: customer?.code ?? '',
      name: customer?.name ?? '',
      tax_code: customer?.tax_code ?? '',
      address: customer?.address ?? '',
      phone: customer?.phone ?? '',
      email: customer?.email ?? '',
      contact_name: customer?.contact_name ?? '',
      payment_terms: customer?.payment_terms ?? '',
      org_unit_id: customer ? String(customer.org_unit_id) : null,
      active: customer?.active ?? true,
    },
  })

  async function save(v: Values) {
    const body: CustomerFields = {
      name: v.name.trim(),
      tax_code: orNull(v.tax_code),
      address: orNull(v.address),
      phone: orNull(v.phone),
      email: orNull(v.email),
      contact_name: orNull(v.contact_name),
      payment_terms: orNull(v.payment_terms),
      org_unit_id: Number(v.org_unit_id),
      active: v.active,
    }
    if (customer) {
      await unwrap(api.PUT('/sales/customers/{id}', { params: { path: { id: customer.id } }, body }))
      await onSaved(customer.id)
    } else {
      const { id } = await unwrap(api.POST('/sales/customers', { body: { ...body, code: v.code.trim() } }))
      await onSaved(id)
    }
  }

  return { form, save, fieldOf: (err: { code: string }) => (err.code === 'customer_code_taken' ? ('code' as const) : undefined) }
}

export function GeneralFields({ existing }: { existing?: boolean }) {
  const { t } = useTranslation()
  return (
    <>
      <CodeField name="code" label={t('sales.customer.code')} existing={!!existing} />
      <TextField name="name" label={t('sales.customer.name')} required maxLength={300} />
      <TextField name="tax_code" label={t('sales.customer.tax_code')} maxLength={20} />
      <TextField name="contact_name" label={t('sales.customer.contact_name')} maxLength={200} />
      <TextField name="phone" label={t('sales.customer.phone')} type="tel" maxLength={50} />
      <TextField name="email" label={t('sales.customer.email')} type="email" maxLength={200} />
      <TextField name="address" label={t('sales.customer.address')} maxLength={500} wide />
    </>
  )
}

export function SalesFields() {
  const { t } = useTranslation()
  return (
    <>
      <OrgUnitField
        name="org_unit_id"
        label={t('sales.customer.org_unit')}
        required
        product="sales"
        permission="sales.customer.edit"
       
      />
      <TextField name="payment_terms" label={t('sales.customer.payment_terms')} description={t('sales.customer.payment_terms_hint')} maxLength={1000} />
      <CheckboxField name="active" label={t('sales.customer.active')} description={t('sales.customer.active_hint')} />
    </>
  )
}

// CustomerForm creates a customer; an existing one is edited a part at a time on its page.
export function CustomerForm({ onSaved }: { onSaved: (id: number) => Promise<void> | void }) {
  const { t } = useTranslation()
  const { form, save, fieldOf } = useCustomerForm(undefined, onSaved)
  return (
    <Form form={form} onSubmit={save} fieldOf={fieldOf}>
      <FormSection title={t('sales.customer.section.general')} description={t('sales.customer.section.general_hint')} icon={IconAddressBook}>
        <GeneralFields />
      </FormSection>
      <FormSection title={t('sales.customer.section.sales')} description={t('sales.customer.section.sales_hint')} icon={IconBuildingStore}>
        <SalesFields />
      </FormSection>
      <FormActions submitLabel={t('sales.customers.create')} cancelTo="/sales/customers" />
    </Form>
  )
}
