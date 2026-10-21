import { Toggle } from '@base-ui/react/toggle'
import { ToggleGroup } from '@base-ui/react/toggle-group'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import styles from './SegmentedControl.module.css'

export interface SegmentedControlProps extends Omit<React.ComponentPropsWithRef<'div'>, 'children' | 'onChange' | 'value' | 'defaultValue'> {
  data: { value: string; label: React.ReactNode }[]
  value: string
  onChange(value: string): void
  fullWidth?: boolean
  disabled?: boolean
  'aria-label': string
}

export function SegmentedControl({ data, value, onChange, fullWidth, disabled, className, ...props }: SegmentedControlProps) {
  return (
    <ToggleGroup
      value={[value]}
      onValueChange={(next) => {
        // Single-select: ignore a deselect (clicking the pressed item) so a value always stays chosen.
        const picked = next.find((item) => item !== value)
        if (picked !== undefined) onChange(picked)
      }}
      disabled={disabled}
      className={cx(styles.root, className)}
      data-full-width={fullWidth ? '' : undefined}
      {...props}
    >
      {data.map((item) => (
        <Toggle key={item.value} value={item.value} data-mmerp-control="" className={styles.item}>
          {item.label}
        </Toggle>
      ))}
    </ToggleGroup>
  )
}
