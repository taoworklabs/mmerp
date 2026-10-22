import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import type { StyleVars } from '../../internal/style.ts'
import type { ColorName } from '../../tokens/tokens.ts'
import styles from './Badge.module.css'

export interface BadgeProps extends React.ComponentPropsWithRef<'span'> {
  color?: ColorName
  variant?: 'light' | 'filled' | 'outline'
}

export function Badge({ color = 'primary', variant = 'light', className, style, ...props }: BadgeProps) {
  // Badge reads raw ramp steps rather than the shared "-light"/"-light-color" tokens: it needs more
  // contrast than other tinted surfaces at this small a size.
  const lightBgStep = color === 'gray' ? 2 : 1
  const vars: StyleVars = {
    ...style,
    '--mmerp-badge-bg':
      variant === 'filled' ? `var(--mmerp-color-${color}-6)` : variant === 'outline' ? 'transparent' : `var(--mmerp-color-${color}-${lightBgStep})`,
    '--mmerp-badge-color': variant === 'filled' ? 'var(--mmerp-color-body)' : variant === 'outline' ? `var(--mmerp-color-${color}-6)` : `var(--mmerp-color-${color}-8)`,
    '--mmerp-badge-border': variant === 'outline' ? `var(--mmerp-color-${color}-6)` : 'transparent',
  }
  return <span className={cx(styles.root, className)} style={vars} data-variant={variant} {...props} />
}
