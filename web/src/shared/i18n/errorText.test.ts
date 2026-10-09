import type { TFunction } from 'i18next'
import { afterEach, describe, expect, it } from 'vitest'
import { ApiError } from '@/shared/api/error'
import { errorText, setErrorPrefixes } from '.'

// t stands in for i18next: it answers the first key it knows, else the default value.
const catalog: Record<string, string> = {
  'shared.error.unexpected_error': 'Có lỗi xảy ra.',
  'shared.error.network_error': 'Không kết nối được.',
  'shared.error.version_conflict': 'Bản ghi đã thay đổi.',
  'hrm.error.contract_overlaps': 'Hợp đồng trùng với hợp đồng đang hiệu lực.',
  'hrm.error.invalid_leave_days': 'Số ngày nghỉ không hợp lệ: {{max}}.',
}
const t = ((keys: string | string[], opts?: Record<string, unknown>) => {
  const found = [keys].flat().find((k) => k in catalog)
  const text = (found && catalog[found]) || String(opts?.defaultValue ?? '')
  return text.replace(/\{\{(\w+)\}\}/g, (_, name: string) => String(opts?.[name] ?? ''))
}) as unknown as TFunction

afterEach(() => setErrorPrefixes([]))

describe('errorText', () => {
  it('reads a core code from the shared catalogue', () => {
    setErrorPrefixes(['hrm', 'core'])
    expect(errorText(t, new ApiError(409, 'version_conflict', {}))).toBe('Bản ghi đã thay đổi.')
  })

  it("reads a product's code from that product's own catalogue", () => {
    setErrorPrefixes(['hrm', 'core'])
    expect(errorText(t, new ApiError(422, 'contract_overlaps', {}))).toBe('Hợp đồng trùng với hợp đồng đang hiệu lực.')
  })

  it('fills the parameters of a product code', () => {
    setErrorPrefixes(['hrm', 'core'])
    expect(errorText(t, new ApiError(422, 'invalid_leave_days', { max: 12 }))).toBe('Số ngày nghỉ không hợp lệ: 12.')
  })

  it("falls back to the generic message when the owning area's wording is not loaded", () => {
    expect(errorText(t, new ApiError(422, 'contract_overlaps', {}))).toBe('Có lỗi xảy ra.')
  })

  it('reports a thrown non-API error as a network failure', () => {
    setErrorPrefixes(['hrm'])
    expect(errorText(t, new Error('offline'))).toBe('Không kết nối được.')
  })

  it('falls back to the generic message for a code nobody translates', () => {
    setErrorPrefixes(['hrm'])
    expect(errorText(t, new ApiError(500, 'brand_new_code', {}))).toBe('Có lỗi xảy ra.')
  })
})
