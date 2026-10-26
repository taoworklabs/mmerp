import { useTranslation } from 'react-i18next'
import type { PartnerIdentity } from '@/shared/api/core'
import { TextField } from '@/shared/ui/form'

// A partner's identity as a form holds it: every value a string.
export type PartnerValues = { name: string; tax_code: string; address: string; phone: string; email: string; contact_name: string }

export const partnerValues = (p?: PartnerIdentity): PartnerValues => ({
  name: p?.name ?? '',
  tax_code: p?.tax_code ?? '',
  address: p?.address ?? '',
  phone: p?.phone ?? '',
  email: p?.email ?? '',
  contact_name: p?.contact_name ?? '',
})

const orNull = (s: string) => (s.trim() === '' ? null : s.trim())

export const partnerBody = (v: PartnerValues): PartnerIdentity => ({
  name: v.name.trim(),
  tax_code: orNull(v.tax_code),
  address: orNull(v.address),
  phone: orNull(v.phone),
  email: orNull(v.email),
  contact_name: orNull(v.contact_name),
})

export const canEditPartner = (p?: { allowed_actions: string[] }) => !!p?.allowed_actions.includes('edit')

// PartnerFields are the identity inputs of a form holding PartnerValues. A product may name the
// partner its own way (customer name); readOnly shows an identity the user may not change.
export function PartnerFields({ nameLabel, readOnly }: { nameLabel?: string; readOnly?: boolean }) {
  const { t } = useTranslation()
  return (
    <>
      <TextField name="name" label={nameLabel ?? t('shared.partner.name')} required maxLength={300} readOnly={readOnly} />
      <TextField name="tax_code" label={t('shared.partner.tax_code')} maxLength={20} readOnly={readOnly} />
      <TextField name="contact_name" label={t('shared.partner.contact_name')} maxLength={200} readOnly={readOnly} />
      <TextField name="phone" label={t('shared.partner.phone')} type="tel" maxLength={50} readOnly={readOnly} />
      <TextField name="email" label={t('shared.partner.email')} type="email" maxLength={200} readOnly={readOnly} />
      <TextField name="address" label={t('shared.partner.address')} maxLength={500} wide readOnly={readOnly} />
    </>
  )
}
