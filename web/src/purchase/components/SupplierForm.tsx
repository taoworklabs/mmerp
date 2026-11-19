import { Button, Stack, Text } from '@mmerp/ui'
import { IconAddressBook, IconBuildingStore } from '@tabler/icons-react'
import { useEffect, useState } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { ApiError, type Partner, type PartnerListItem } from '@/shared/api/core'
import { api, unwrap, type Supplier, type SupplierUpdate, type NewSupplier } from '@/shared/api/purchase'
import { useCan } from '@/shared/auth/me'
import { PartnerFields, PartnerPicker, partnerBody, partnerValues, usePartner, type PartnerValues } from '@/shared/master'
import { Notice } from '@/shared/ui/Notice'
import { CheckboxField, CodeField, Form, FormActions, FormRow, FormSection, OrgUnitField, TextField, type RemoteRow } from '@/shared/ui/form'

// partner_id: an existing partner picked for a new supplier; its identity is then only shown.
export type SupplierValues = PartnerValues & {
  code: string
  partner_id: string | null
  payment_terms: string
  org_unit_id: string | null
  active: boolean
}

const orNull = (s: string) => (s.trim() === '' ? null : s.trim())

const roleBody = (v: SupplierValues) => ({ payment_terms: orNull(v.payment_terms), org_unit_id: Number(v.org_unit_id), active: v.active })

// newSupplierBody: the supplier over the picked partner, or over a new one with the typed identity.
export function newSupplierBody(v: SupplierValues): NewSupplier {
  const role = { code: v.code.trim(), ...roleBody(v) }
  return v.partner_id ? { ...role, partner_id: Number(v.partner_id) } : { ...role, partner: partnerBody(v) }
}

// supplierUpdateBody: the identity only from the card that edits it, so saving the role alone
// leaves the partner as others last saved it.
export const supplierUpdateBody = (v: SupplierValues, identity: boolean): SupplierUpdate =>
  identity ? { partner: partnerBody(v), ...roleBody(v) } : roleBody(v)

// takenPartnerId: the partner already holding the typed tax code, when that is why the save failed.
export function takenPartnerId(err: unknown): number | undefined {
  if (!(err instanceof ApiError) || err.code !== 'partner_tax_code_taken') return undefined
  const id = err.params.partner_id
  return typeof id === 'number' ? id : undefined
}

// withPartner puts an existing partner, its identity shown, in place of the typed one.
export const withPartner = (v: SupplierValues, p: PartnerListItem | Partner): SupplierValues => ({ ...v, ...partnerValues(p), partner_id: String(p.id) })

// useSupplierForm holds the whole supplier whichever part of it is on screen; an edit sends the
// identity only when it is the part on screen.
export function useSupplierForm(supplier: Supplier | undefined, onSaved: (id: number) => Promise<void> | void, identity = false) {
  const [taken, setTaken] = useState<number>()
  const form = useForm<SupplierValues>({
    mode: 'onBlur',
    defaultValues: {
      code: supplier?.code ?? '',
      partner_id: null,
      ...partnerValues(supplier?.partner),
      payment_terms: supplier?.payment_terms ?? '',
      org_unit_id: supplier ? String(supplier.org_unit_id) : null,
      active: supplier?.active ?? true,
    },
  })

  async function save(v: SupplierValues) {
    setTaken(undefined)
    if (supplier) {
      await unwrap(api.PUT('/purchase/suppliers/{id}', { params: { path: { id: supplier.id } }, body: supplierUpdateBody(v, identity) }))
      await onSaved(supplier.id)
      return
    }
    try {
      const { id } = await unwrap(api.POST('/purchase/suppliers', { body: newSupplierBody(v) }))
      await onSaved(id)
    } catch (err) {
      // The company is known already: offer it rather than a second one.
      const id = takenPartnerId(err)
      if (id === undefined) throw err
      setTaken(id)
      return false
    }
  }

  return { form, save, taken, errorFields: { supplier_code_taken: 'code', partner_tax_code_taken: 'tax_code' } as const }
}

export function GeneralFields({ existing, identityReadOnly }: { existing?: boolean; identityReadOnly?: boolean }) {
  const { t } = useTranslation()
  return (
    <>
      <CodeField name="code" label={t('purchase.supplier.code')} existing={!!existing} />
      <PartnerFields nameLabel={t('purchase.supplier.name')} readOnly={identityReadOnly} />
    </>
  )
}

export function SalesFields() {
  const { t } = useTranslation()
  return (
    <>
      <OrgUnitField name="org_unit_id" label={t('purchase.supplier.org_unit')} required product="purchase" permission="purchase.supplier.edit" />
      <TextField name="payment_terms" label={t('purchase.supplier.payment_terms')} description={t('purchase.supplier.payment_terms_hint')} maxLength={1000} />
      <CheckboxField name="active" label={t('purchase.supplier.active')} description={t('purchase.supplier.active_hint')} />
    </>
  )
}

// TakenPartner names the partner holding the typed tax code and offers to make the supplier
// over it; the button submits the form once the partner is in place.
function TakenPartner({ id, onUse }: { id: number; onUse: (p: Partner) => void }) {
  const { t } = useTranslation()
  const partner = usePartner(id)
  if (partner.isPending) return null
  if (partner.isError) return <Notice tone="danger">{t('shared.error.partner_tax_code_taken')}</Notice>
  const p = partner.data
  return (
    <Notice tone="warning" announce="alert" title={t('purchase.supplier.partner_taken_title')}>
      <Stack gap="sm" align="start">
        <Text size="sm">{t('purchase.supplier.partner_taken', { name: p.name, code: p.code })}</Text>
        <Button type="submit" variant="default" onClick={() => onUse(p)}>
          {t('purchase.supplier.use_partner')}
        </Button>
      </Stack>
    </Notice>
  )
}

const partnerKeys = ['name', 'tax_code', 'address', 'phone', 'email', 'contact_name'] as const
const identityKeys = ['partner_id', ...partnerKeys] as const

// SupplierForm creates a supplier over an existing partner, picked or found by its tax code, or
// over a new one; an existing supplier is edited a part at a time on its page.
export function SupplierForm({ onSaved }: { onSaved: (id: number) => Promise<void> | void }) {
  const { t } = useTranslation()
  const canPick = useCan('partner.view')
  const { form, save, taken, errorFields } = useSupplierForm(undefined, onSaved)
  const [picked, setPicked] = useState<RemoteRow>()
  const partnerId = useWatch({ control: form.control, name: 'partner_id' })
  const use = (p: PartnerListItem | Partner) => {
    setPicked(p)
    const v = withPartner(form.getValues(), p)
    for (const k of identityKeys) form.setValue(k, v[k], { shouldDirty: true })
    form.clearErrors()
  }
  // A cleared pick leaves blank inputs for a new company, not the dropped partner's identity.
  useEffect(() => {
    if (partnerId || !picked) return
    setPicked(undefined)
    const blank = partnerValues(undefined)
    for (const k of partnerKeys) form.setValue(k, blank[k], { shouldDirty: true })
  }, [partnerId, picked, form])
  return (
    <Form form={form} onSubmit={save} errorFields={errorFields}>
      <FormSection title={t('purchase.supplier.section.general')} description={t('purchase.supplier.section.general_hint')} icon={IconAddressBook}>
        {taken !== undefined && (
          <FormRow>
            <TakenPartner id={taken} onUse={use} />
          </FormRow>
        )}
        {canPick && <PartnerPicker name="partner_id" label={t('purchase.supplier.existing_partner')} current={picked} onPicked={use} />}
        <GeneralFields identityReadOnly={!!partnerId} />
      </FormSection>
      <FormSection title={t('purchase.supplier.section.purchase')} description={t('purchase.supplier.section.purchase_hint')} icon={IconBuildingStore}>
        <SalesFields />
      </FormSection>
      <FormActions submitLabel={t('purchase.suppliers.create')} cancelTo="/purchase/suppliers" />
    </Form>
  )
}
