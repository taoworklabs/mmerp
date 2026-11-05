import { describe, expect, it } from 'vitest'
import type { Account } from '@/shared/api/accounting'
import { hasLinesBelow, keepUnitsIn, parentCandidates, subtree } from './tree'

const acc = (id: number, type: Account['type'], parent_id: number | null = null): Account => ({
  id,
  code: String(id),
  name: `A${id}`,
  type,
  parent_id,
  active: true,
  postable: true,
  has_lines: false,
})

describe('subtree', () => {
  it('holds the root and every descendant, nothing else', () => {
    const items = [
      { id: 1, parent_id: null },
      { id: 2, parent_id: 1 },
      { id: 3, parent_id: 2 },
      { id: 4, parent_id: null },
    ]
    expect([...subtree(items, 1)].sort()).toEqual([1, 2, 3])
    expect([...subtree(items, 4)]).toEqual([4])
  })
})

describe('parentCandidates', () => {
  const accounts = [acc(1, 'asset'), acc(2, 'asset', 1), acc(3, 'asset', 2), acc(4, 'expense'), acc(5, 'asset')]
  it('offers accounts of the same type', () => {
    expect(parentCandidates(accounts, 'asset').map((a) => a.id)).toEqual([1, 2, 3, 5])
  })
  it('leaves out the account itself and its descendants', () => {
    expect(parentCandidates(accounts, 'asset', 2).map((a) => a.id)).toEqual([1, 5])
  })
})

describe('hasLinesBelow', () => {
  it('is true when the account or a descendant has lines', () => {
    const accounts = [acc(1, 'asset'), { ...acc(2, 'asset', 1), has_lines: true }, acc(3, 'asset')]
    expect(hasLinesBelow(accounts, 1)).toBe(true)
    expect(hasLinesBelow(accounts, 2)).toBe(true)
    expect(hasLinesBelow(accounts, 3)).toBe(false)
  })
})

describe('keepUnitsIn', () => {
  const units = [
    { id: 1, parent_id: null },
    { id: 2, parent_id: 1 },
    { id: 3, parent_id: null },
  ]
  it('clears org units outside the legal entity subtree and keeps the rest', () => {
    const lines = [{ org_unit_id: '2' }, { org_unit_id: '3' }, { org_unit_id: null }]
    expect(keepUnitsIn(lines, units, 1)).toEqual([{ org_unit_id: '2' }, { org_unit_id: null }, { org_unit_id: null }])
  })
})
