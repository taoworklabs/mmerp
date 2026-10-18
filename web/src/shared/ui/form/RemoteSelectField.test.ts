import { describe, expect, it } from 'vitest'
import { remoteOptions } from './RemoteSelectField'

const an = { id: 1, code: 'NV001', name: 'An' }
const binh = { id: 2, code: 'NV002', name: 'Bình' }

describe('remoteOptions', () => {
  it('keeps the current value first when the search does not return it', () => {
    expect(remoteOptions([binh], [an])).toEqual([
      { value: '1', label: 'NV001 · An' },
      { value: '2', label: 'NV002 · Bình' },
    ])
  })
  it('does not repeat the current value when the search returns it', () => {
    expect(remoteOptions([an, binh], [an]).map((o) => o.value)).toEqual(['1', '2'])
  })
  it('keeps each kept row once and skips missing ones', () => {
    expect(remoteOptions([], [binh, undefined, binh]).map((o) => o.value)).toEqual(['2'])
  })
  it('labels a row without a code by its name alone', () => {
    expect(remoteOptions([{ id: 3, code: '', name: 'Chi' }], [])).toEqual([{ value: '3', label: 'Chi' }])
  })
})
