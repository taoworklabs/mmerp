import type { TFunction } from 'i18next'
import { describe, expect, it } from 'vitest'
import { kindLabel, rowBalance, savedPartner, splitBalance } from './balance'

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

describe('rowBalance', () => {
  it('splits the balance of a row', () => {
    expect(rowBalance({ opening: 0, closing: -15 }, 'closing', 'credit')).toBe(15)
  })
  // A totals row of balances on both sides, or of several accounts, would net them.
  it('leaves the balance cells of a totals row empty', () => {
    expect(rowBalance({ total: true, opening: 40, closing: 25 }, 'closing', 'debit')).toBeNull()
    expect(rowBalance({ total: true, opening: 40, closing: 25 }, 'opening', 'debit')).toBeNull()
  })
})

describe('savedPartner', () => {
  const line = { partner_id: 7, partner_code: 'DT7', partner_name: 'Khách 7' }
  it('names the saved partner of the same line', () => {
    expect(savedPartner(line, '7')).toEqual({ id: 7, code: 'DT7', name: 'Khách 7' })
  })
  // After a line is removed, index i holds another line: its saved partner is not this one.
  it('ignores a saved line whose partner differs', () => {
    expect(savedPartner(line, '8')).toBeUndefined()
    expect(savedPartner(undefined, '7')).toBeUndefined()
    expect(savedPartner({ partner_id: null, partner_code: null, partner_name: null }, null)).toBeUndefined()
  })
})
