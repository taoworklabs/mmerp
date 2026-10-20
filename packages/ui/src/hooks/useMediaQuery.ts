import { useSyncExternalStore } from 'react'

// useSyncExternalStore calls getSnapshot during render itself, so the
// first client render already matches matchMedia instead of flashing `initial`.
export function useMediaQuery(query: string, initial = false): boolean {
  return useSyncExternalStore(
    (onChange) => {
      const mql = window.matchMedia(query)
      mql.addEventListener('change', onChange)
      return () => mql.removeEventListener('change', onChange)
    },
    () => window.matchMedia(query).matches,
    () => initial,
  )
}
