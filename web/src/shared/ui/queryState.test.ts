import { describe, expect, it } from 'vitest'
import { ApiError } from '@/shared/api/error'
import { queryState } from './queryState'

const all = { empty: true, noResults: true, notFound: true }
const done = <T>(data: T) => ({ isPending: false, isError: false, error: null, data })
const failed = (error: unknown) => ({ isPending: false, isError: true, error })

describe('queryState', () => {
  it('waits while loading', () => {
    expect(queryState({ isPending: true, isError: false, error: null }, undefined, all)).toEqual({ state: 'loading' })
  })
  it('answers a 404 with not found only when offered', () => {
    const q = failed(new ApiError(404, 'not_found'))
    expect(queryState(q, undefined, all)).toEqual({ state: 'not_found' })
    expect(queryState(q, undefined, { ...all, notFound: false })).toEqual({ state: 'error' })
    expect(queryState(failed(new ApiError(500, 'internal')), undefined, all)).toEqual({ state: 'error' })
  })
  it('is empty for an empty list or no value, only when offered', () => {
    expect(queryState(done([]), undefined, all)).toEqual({ state: 'empty' })
    expect(queryState(done(null), undefined, all)).toEqual({ state: 'empty' })
    expect(queryState(done([]), undefined, { ...all, empty: false })).toEqual({ state: 'data', data: [] })
  })
  it('tells a filtered-out result from an empty one', () => {
    const none = (xs: number[]) => xs.filter((x) => x > 5)
    expect(queryState(done([1, 2]), none, all)).toEqual({ state: 'no_results' })
    expect(queryState(done([1, 2]), none, { ...all, noResults: false })).toEqual({ state: 'empty' })
    expect(queryState(done([]), none, all)).toEqual({ state: 'empty' })
  })
  it('hands over the selected data', () => {
    expect(queryState(done({ items: [3, 1] }), (d) => [...d.items].sort(), all)).toEqual({ state: 'data', data: [1, 3] })
  })
})
