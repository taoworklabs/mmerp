import type * as React from 'react'
import { ActionIcon } from '../primitives/input/ActionIcon.tsx'
import { CloseIcon } from './icons.tsx'

interface ClearButtonProps {
  label: string
  onClear(): void
  // The field's focusable control: clearing removes this button, so focus would otherwise drop to the page.
  focusTarget: React.RefObject<HTMLElement | null>
  disabled?: boolean
  readOnly?: boolean
}

export function ClearButton({ label, onClear, focusTarget, disabled, readOnly }: ClearButtonProps) {
  if (disabled || readOnly) return null
  return (
    <ActionIcon
      variant="subtle"
      color="gray"
      size="xs"
      aria-label={label}
      onClick={() => {
        onClear()
        focusTarget.current?.focus()
      }}
    >
      <CloseIcon />
    </ActionIcon>
  )
}
