import { useCallback, useEffect, useRef, useState } from 'react'

export function useDebouncedValue<T>(value: T, ms: number): [T, () => void] {
  const [debounced, setDebounced] = useState(value)
  const timeout = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)

  const cancel = useCallback(() => clearTimeout(timeout.current), [])

  useEffect(() => {
    timeout.current = setTimeout(() => setDebounced(value), ms)
    return cancel
  }, [value, ms, cancel])

  return [debounced, cancel]
}
