import { describe, expect, it } from 'vitest'
import { mentionAt, splitMentions } from './mentions'

describe('splitMentions', () => {
  it('keeps the sentence end out of the login', () => {
    expect(splitMentions('Hỏi @boss, rồi @hr2.')).toEqual([
      { text: 'Hỏi ', mention: false },
      { text: '@boss', mention: true },
      { text: ', rồi ', mention: false },
      { text: '@hr2', mention: true },
      { text: '.', mention: false },
    ])
  })
})

describe('mentionAt', () => {
  it('finds the word being typed after @', () => {
    expect(mentionAt('anh @bo', 7)).toEqual({ start: 4, query: 'bo' })
    expect(mentionAt('@', 1)).toEqual({ start: 0, query: '' })
  })
  it('ignores an @ inside a word, such as an email address', () => {
    expect(mentionAt('a@b', 3)).toBeNull()
    expect(mentionAt('@boss xong', 10)).toBeNull()
  })
})
