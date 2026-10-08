import { describe, expect, it } from 'vitest'
import { pickLocale } from '.'

describe('pickLocale', () => {
  it('prefers the stored locale', () => expect(pickLocale('en', 'vi-VN')).toBe('en'))
  it('falls back to the browser language', () => expect(pickLocale(null, 'en-US')).toBe('en'))
  it('ignores an unknown stored value', () => expect(pickLocale('fr', 'vi')).toBe('vi'))
  it('defaults to vi for unsupported browsers', () => expect(pickLocale(null, 'ja-JP')).toBe('vi'))
})
