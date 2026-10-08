import { describe, expect, it } from 'vitest'
import { initials } from './avatar'

describe('initials', () => {
  it('takes the first and last words', () => {
    expect(initials('Phạm Bảo Hà')).toBe('PH')
    expect(initials('  admin ')).toBe('A')
    expect(initials('')).toBe('')
  })
})
