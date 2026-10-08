import { describe, expect, it } from 'vitest'
import { pollDelay } from './schedule'

describe('pollDelay', () => {
  it('polls every 2 s for the first 30 s, then every 10 s', () => {
    expect(pollDelay(0)).toBe(2_000)
    expect(pollDelay(29_999)).toBe(2_000)
    expect(pollDelay(30_000)).toBe(10_000)
    expect(pollDelay(600_000)).toBe(10_000)
  })
})
