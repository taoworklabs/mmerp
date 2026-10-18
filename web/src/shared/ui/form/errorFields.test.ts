import { describe, expect, it } from 'vitest'
import { ApiError } from '@/shared/api/error'
import { fieldOfError } from './errorFields'

const map = { customer_inactive: 'customer_id', invalid_quantity: 'lines.*.quantity' }

describe('fieldOfError', () => {
  it('maps a code to its field', () => {
    expect(fieldOfError(map, new ApiError(422, 'customer_inactive'))).toBe('customer_id')
  })
  it('puts a line error on the field of that line, counted from 1', () => {
    expect(fieldOfError(map, new ApiError(422, 'invalid_quantity', { line: 3 }))).toBe('lines.2.quantity')
  })
  it('leaves a line error without a usable line to the banner', () => {
    expect(fieldOfError(map, new ApiError(422, 'invalid_quantity'))).toBeUndefined()
    expect(fieldOfError(map, new ApiError(422, 'invalid_quantity', { line: 0 }))).toBeUndefined()
    expect(fieldOfError(map, new ApiError(422, 'invalid_quantity', { line: '2' }))).toBeUndefined()
  })
  it('leaves unknown codes and inherited keys to the banner', () => {
    expect(fieldOfError(map, new ApiError(409, 'version_conflict'))).toBeUndefined()
    expect(fieldOfError(map, new ApiError(409, 'toString'))).toBeUndefined()
    expect(fieldOfError(undefined, new ApiError(422, 'customer_inactive'))).toBeUndefined()
  })
})
