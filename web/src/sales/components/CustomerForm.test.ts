import { describe, expect, it } from 'vitest'
import { ApiError } from '@/shared/api/core'
import { canEditPartner } from '@/shared/master'
import { customerUpdateBody, newCustomerBody, takenPartnerId, withPartner, type CustomerValues } from './CustomerForm'

const typed: CustomerValues = {
  code: '',
  partner_id: null,
  name: ' Ánh Dương ',
  tax_code: '0312345678',
  address: '',
  phone: '',
  email: '',
  contact_name: '',
  payment_terms: '',
  org_unit_id: '4',
  active: true,
}
const existing = { id: 7, code: 'DT00007', name: 'Công ty Ánh Dương', tax_code: '0312345678', address: 'Q1', phone: null, email: null, contact_name: 'Lan', allowed_actions: [] }

describe('new customer', () => {
  it('sends a typed identity as a new partner', () => {
    const body = newCustomerBody(typed)
    expect(body.partner_id).toBeUndefined()
    expect(body.partner).toEqual({ name: 'Ánh Dương', tax_code: '0312345678', address: null, phone: null, email: null, contact_name: null })
  })
  it('names the partner already holding the tax code', () => {
    expect(takenPartnerId(new ApiError(409, 'partner_tax_code_taken', { partner_id: 7 }))).toBe(7)
    expect(takenPartnerId(new ApiError(409, 'customer_code_taken'))).toBeUndefined()
    expect(takenPartnerId(new Error('x'))).toBeUndefined()
  })
  it('resubmits on the existing partner with its id and no identity', () => {
    const v = withPartner(typed, existing)
    expect(v.name).toBe('Công ty Ánh Dương')
    expect(v.contact_name).toBe('Lan')
    const body = newCustomerBody(v)
    expect(body.partner_id).toBe(7)
    expect(body).not.toHaveProperty('partner')
    expect(body).toMatchObject({ org_unit_id: 4, active: true, payment_terms: null })
  })
})

describe('customer edit', () => {
  it('sends the identity only from the general card', () => {
    expect(customerUpdateBody(typed, true).partner).toMatchObject({ name: 'Ánh Dương' })
    const role = customerUpdateBody(typed, false)
    expect(role).not.toHaveProperty('partner')
    expect(role).toEqual({ payment_terms: null, org_unit_id: 4, active: true })
  })
})

describe('identity inputs', () => {
  it('are editable only when the partner allows edit', () => {
    expect(canEditPartner({ allowed_actions: ['edit', 'attach'] })).toBe(true)
    expect(canEditPartner({ allowed_actions: [] })).toBe(false)
    expect(canEditPartner(undefined)).toBe(false)
  })
})
