import { useCallback, useState } from 'react'

export interface DisclosureHandlers {
  open: () => void
  close: () => void
  toggle: () => void
}

export function useDisclosure(initial = false): [boolean, DisclosureHandlers] {
  const [opened, setOpened] = useState(initial)
  const open = useCallback(() => setOpened(true), [])
  const close = useCallback(() => setOpened(false), [])
  const toggle = useCallback(() => setOpened((v) => !v), [])
  return [opened, { open, close, toggle }]
}
