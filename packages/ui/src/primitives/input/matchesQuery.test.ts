import { describe, expect, it } from 'vitest'
import { matchesQuery } from './matchesQuery.ts'

describe('matchesQuery', () => {
  it('ignores case and Vietnamese diacritics', () => {
    expect(matchesQuery('Nguyễn Văn A', 'nguyen van')).toBe(true)
    expect(matchesQuery('Đặng Thị Bích', 'dang bich')).toBe(true)
  })
  it('needs every word', () => {
    expect(matchesQuery('Trần Văn C', 'tran b')).toBe(false)
  })
  it('matches everything on an empty query', () => {
    expect(matchesQuery('Any', '  ')).toBe(true)
  })
})
