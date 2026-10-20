import { renderHook } from 'vitest-browser-react'
import { page } from 'vitest/browser'
import { describe, expect, it, vi } from 'vitest'
import { useDisclosure } from './useDisclosure.ts'
import { useMediaQuery } from './useMediaQuery.ts'
import { useDebouncedValue } from './useDebouncedValue.ts'
import { useLocalStorage } from './useLocalStorage.ts'
import { useElementSize } from './useElementSize.ts'

describe('useDisclosure', () => {
  it('toggle flips false to true to false', async () => {
    const { result, act } = await renderHook(() => useDisclosure())
    expect(result.current[0]).toBe(false)
    await act(() => result.current[1].toggle())
    expect(result.current[0]).toBe(true)
    await act(() => result.current[1].toggle())
    expect(result.current[0]).toBe(false)
  })
})

describe('useMediaQuery', () => {
  it('reflects the viewport on first render and after a rerender', async () => {
    await page.viewport(800, 600)
    const { result, rerender } = await renderHook(() => useMediaQuery('(max-width: 63.99em)'))
    expect(result.current).toBe(true)

    await page.viewport(1280, 800)
    await rerender()
    expect(result.current).toBe(false)
  })
})

describe('useDebouncedValue', () => {
  it('delays the value until the timeout elapses', async () => {
    vi.useFakeTimers()
    try {
      const { result, rerender, act } = await renderHook((props?: { value: string }) => useDebouncedValue(props?.value ?? 'a', 300), {
        initialProps: { value: 'a' },
      })
      expect(result.current[0]).toBe('a')

      await rerender({ value: 'b' })
      expect(result.current[0]).toBe('a')

      await act(() => vi.advanceTimersByTimeAsync(300))
      expect(result.current[0]).toBe('b')
    } finally {
      vi.useRealTimers()
    }
  })

  it('cancel keeps the previous value', async () => {
    vi.useFakeTimers()
    try {
      const { result, rerender, act } = await renderHook((props?: { value: string }) => useDebouncedValue(props?.value ?? 'a', 300), {
        initialProps: { value: 'a' },
      })

      await rerender({ value: 'b' })
      await act(() => result.current[1]())
      await act(() => vi.advanceTimersByTimeAsync(300))
      expect(result.current[0]).toBe('a')
    } finally {
      vi.useRealTimers()
    }
  })
})

describe('useLocalStorage', () => {
  it('writes the serialised value under the key', async () => {
    localStorage.removeItem('k')
    const { result } = await renderHook(() => useLocalStorage('k', { a: 0 }))
    await result.current[1]({ a: 1 })
    expect(localStorage.getItem('k')).toBe('{"a":1}')
  })

  it('falls back to the default value when the entry is not valid JSON', async () => {
    localStorage.setItem('k', 'not json')
    const { result } = await renderHook(() => useLocalStorage('k', { a: 0 }))
    expect(result.current[0]).toEqual({ a: 0 })
  })
})

describe('useElementSize', () => {
  it('reports the size of the observed element', async () => {
    const { result } = await renderHook(() => useElementSize<HTMLDivElement>())
    const el = document.createElement('div')
    el.style.width = '120px'
    el.style.height = '40px'
    document.body.append(el)
    try {
      result.current.ref(el)
      await expect.poll(() => result.current.width).toBe(120)
      expect(result.current.height).toBe(40)
    } finally {
      el.remove()
    }
  })
})
