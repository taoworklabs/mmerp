import { useState } from 'react'

// An input's value: the `value` prop when given (controlled), its own state from `defaultValue`
// otherwise. Setting it updates that state when uncontrolled and always reports through onChange.
export function useControllable<T>(value: T | undefined, defaultValue: T, onChange?: (next: T) => void): [T, (next: T) => void] {
  const [own, setOwn] = useState(defaultValue)
  const controlled = value !== undefined
  return [
    controlled ? value : own,
    (next) => {
      if (!controlled) setOwn(next)
      onChange?.(next)
    },
  ]
}
