import { describe, expect, it } from 'vitest'
import { previewRows } from './DocumentPreview'

describe('previewRows', () => {
  it('keeps the rows in order and drops the ones a document leaves out', () => {
    const reason = ''
    expect(previewRows([['Type', 'Annual'], null, reason && ['Reason', reason], false, undefined, ['Days', '2']])).toEqual([
      ['Type', 'Annual'],
      ['Days', '2'],
    ])
  })
})
