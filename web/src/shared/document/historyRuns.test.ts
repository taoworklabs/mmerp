import { describe, expect, it } from 'vitest'
import type { HistoryEntry } from '@/shared/api/core'
import { runs } from './DocumentHistory'

const entry = (id: number, action: string, actor: string, data?: Record<string, unknown>) => ({ id, action, actor_name: actor, at: '2026-10-06T07:00:00Z', data }) as unknown as HistoryEntry

describe('runs', () => {
  it('merges back-to-back repeats of one person and keeps the rest apart', () => {
    const out = runs([entry(1, 'v', 'Hà'), entry(2, 'v', 'Hà'), entry(3, 'v', 'An'), entry(4, 'v', 'Hà')])
    expect(out.map((r) => [r.entry.id, r.count])).toEqual([[1, 2], [3, 1], [4, 1]])
  })
  it('never merges entries that change fields', () => {
    const changed = { changes: { days: { old: 1, new: 2 } } }
    expect(runs([entry(1, 'u', 'Hà', changed), entry(2, 'u', 'Hà', changed)])).toHaveLength(2)
  })
})
