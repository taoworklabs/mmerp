import { describe, expect, it, vi } from 'vitest'
import { mergeRefs } from './mergeRefs.ts'

describe('mergeRefs', () => {
  it('sets every ref object and calls every callback ref with the node', () => {
    const objectRef = { current: null as string | null }
    const callbackRef = vi.fn()
    const node = 'node'

    mergeRefs(objectRef, callbackRef)(node)

    expect(objectRef.current).toBe(node)
    expect(callbackRef).toHaveBeenCalledExactlyOnceWith(node)
  })

  it('on detach, runs a callback ref\'s own returned cleanup instead of calling it with null', () => {
    const cleanup = vi.fn()
    const callbackRef = vi.fn().mockReturnValue(cleanup)
    const detach = mergeRefs(callbackRef)('node')!

    detach()

    expect(cleanup).toHaveBeenCalledOnce()
    expect(callbackRef).toHaveBeenCalledOnce() // only the attach call, never called again with null
  })

  it("on detach, calls a callback ref with null when it returned no cleanup", () => {
    const callbackRef = vi.fn()
    const detach = mergeRefs(callbackRef)('node')!

    detach()

    expect(callbackRef).toHaveBeenNthCalledWith(1, 'node')
    expect(callbackRef).toHaveBeenNthCalledWith(2, null)
  })

  it('on detach, clears an object ref to null', () => {
    const objectRef = { current: null as string | null }
    const detach = mergeRefs(objectRef)('node')!

    detach()

    expect(objectRef.current).toBeNull()
  })
})
