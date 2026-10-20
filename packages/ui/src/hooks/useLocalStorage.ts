import { useCallback, useState } from 'react'

function read<T>(key: string, defaultValue: T): T {
  try {
    const raw = window.localStorage.getItem(key)
    return raw === null ? defaultValue : (JSON.parse(raw) as T)
  } catch {
    return defaultValue
  }
}

export function useLocalStorage<T>(key: string, defaultValue: T): [T, (value: T) => void] {
  const [value, setValue] = useState<T>(() => read(key, defaultValue))

  const set = useCallback(
    (next: T) => {
      setValue(next)
      try {
        window.localStorage.setItem(key, JSON.stringify(next))
      } catch {
        // storage unavailable (quota, privacy mode); state still updates in memory
      }
    },
    [key],
  )

  return [value, set]
}
