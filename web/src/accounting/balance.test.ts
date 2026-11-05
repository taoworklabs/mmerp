import type { TFunction } from 'i18next'
import { describe, expect, it } from 'vitest'
import { kindLabel, splitBalance } from './balance'

// t answers the keys it knows, else its default value.
const catalog: Record<string, string> = { 'accounting.kind.salary_expense': 'Chi phí lương' }
const t = ((key: string, opts?: { defaultValue?: string }) => catalog[key] ?? opts?.defaultValue ?? key) as unknown as TFunction

describe('splitBalance', () => {
  it('shows a positive balance as a debit', () => {
    expect(splitBalance(120)).toEqual({ debit: 120, credit: 0 })
  })
  it('shows a negative balance as a credit', () => {
    expect(splitBalance(-30)).toEqual({ debit: 0, credit: 30 })
  })
  it('shows zero on neither side', () => {
    expect(splitBalance(0)).toEqual({ debit: 0, credit: 0 })
  })
})

describe('kindLabel', () => {
  it('names a known posting kind', () => {
    expect(kindLabel(t, 'salary_expense')).toBe('Chi phí lương')
  })
  it('leaves an unknown kind raw', () => {
    expect(kindLabel(t, 'unknown_kind')).toBe('unknown_kind')
  })
})
