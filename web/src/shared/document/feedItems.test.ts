import { describe, expect, it } from 'vitest'
import { feedItems } from './ActivityFeed'

const items = [
  { key: 'v1', kind: 'views' as const },
  { key: 'c1', kind: 'comments' as const },
  { key: 'h1', kind: 'changes' as const },
  { key: 'v2', kind: 'views' as const },
  { key: 'h2', kind: 'changes' as const },
]
const keys = (xs: { key: string }[]) => xs.map((x) => x.key)

describe('feedItems', () => {
  it('lists everything under all, and one kind under its filter', () => {
    expect(keys(feedItems(items, 'all'))).toEqual(['v1', 'c1', 'h1', 'v2', 'h2'])
    expect(keys(feedItems(items, 'views'))).toEqual(['v1', 'v2'])
  })

  it('leaves reads out of the recent digest, whatever the filter, and keeps the latest that many', () => {
    expect(keys(feedItems(items, 'all', 5))).toEqual(['c1', 'h1', 'h2'])
    expect(keys(feedItems(items, 'views', 2))).toEqual(['c1', 'h1'])
  })
})
