import { describe, expect, it } from 'vitest'
import { formatAgo } from '.'

describe('formatAgo', () => {
  const now = new Date('2026-10-06T10:00:00Z')
  it('counts minutes, at least one', () => expect(formatAgo('2026-10-06T09:59:50Z', 'UTC', now)).toMatch(/1/))
  it('counts hours under a day', () => expect(formatAgo('2026-10-06T07:00:00Z', 'UTC', now)).toMatch(/3/))
  it('shows the tenant date from a day on', () => expect(formatAgo('2026-10-04T23:30:00Z', 'Asia/Ho_Chi_Minh', now)).toBe('05/10/2026'))
})
