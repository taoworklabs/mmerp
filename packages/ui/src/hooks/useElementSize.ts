import { useCallback, useRef, useState } from 'react'

export interface ElementSize<T extends HTMLElement> {
  ref: (el: T | null) => void
  width: number
  height: number
}

export function useElementSize<T extends HTMLElement>(): ElementSize<T> {
  const [size, setSize] = useState({ width: 0, height: 0 })
  const observer = useRef<ResizeObserver | null>(null)

  const ref = useCallback((el: T | null) => {
    observer.current?.disconnect()
    observer.current = null
    if (!el) return
    observer.current = new ResizeObserver(([entry]) => {
      if (!entry) return
      setSize({ width: entry.contentRect.width, height: entry.contentRect.height })
    })
    observer.current.observe(el)
  }, [])

  return { ref, width: size.width, height: size.height }
}
