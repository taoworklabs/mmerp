import { describe, expect, it } from 'vitest'
import { listState } from './listState'

const query = (over: object) => ({ isPending: false, isError: false, data: { items: [1], total: 1 }, ...over })

describe('listState', () => {
  it('waits while the first page loads', () => {
    expect(listState(query({ isPending: true, data: undefined }), false)).toBe('loading')
  })
  it('reports a failed load', () => {
    expect(listState(query({ isError: true, data: undefined }), false)).toBe('error')
  })
  it('tells an empty list from one the filters emptied', () => {
    const none = query({ data: { items: [], total: 0 } })
    expect(listState(none, false)).toBe('empty')
    expect(listState(none, true)).toBe('no_results')
  })
  it('shows rows when there are some', () => {
    expect(listState(query({}), true)).toBe('rows')
  })
})
