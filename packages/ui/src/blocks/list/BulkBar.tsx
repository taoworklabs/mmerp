import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import { CloseIcon } from '../../internal/icons.tsx'
import { Button } from '../../primitives/input/Button.tsx'
import { Text } from '../../primitives/typography/Text.tsx'
import styles from './List.module.css'

export interface BulkBarProps extends Omit<React.ComponentPropsWithRef<'div'>, 'role' | 'children'> {
  countText: string
  label: string
  clearLabel: string
  onClear(): void
  children: React.ReactNode
}

// BulkBar takes FilterBar's place while rows are selected: how many, the actions that apply to
// all of them, and a way to clear the selection.
export function BulkBar({ countText, label, clearLabel, onClear, children, className, ...props }: BulkBarProps) {
  return (
    <div role="toolbar" aria-label={label} className={cx(styles.bulk, className)} {...props}>
      <Text size="sm" weight={600}>
        {countText}
      </Text>
      {children}
      <Button variant="subtle" color="gray" leftSection={<CloseIcon size={12} />} onClick={onClear}>
        {clearLabel}
      </Button>
    </div>
  )
}
