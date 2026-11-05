import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import type { StyleVars } from '../../internal/style.ts'
import { ramp, type Tone } from '../../internal/tone.ts'
import styles from './Badge.module.css'

export interface BadgeProps extends Omit<React.ComponentPropsWithRef<'span'>, 'color'> {
  tone?: Tone
  variant?: 'light' | 'filled' | 'outline'
}

export function Badge({ tone = 'primary', variant = 'light', className, style, ...props }: BadgeProps) {
  // Badge reads raw ramp steps rather than the shared "-light"/"-light-color" tokens: it needs more
  // contrast than other tinted surfaces at this small a size.
  const color = ramp(tone)
  const lightBgStep = tone === 'neutral' ? 2 : 1
  const vars: StyleVars = {
    ...style,
    '--mmerp-badge-bg':
      variant === 'filled' ? `var(--mmerp-color-${color}-6)` : variant === 'outline' ? 'transparent' : `var(--mmerp-color-${color}-${lightBgStep})`,
    '--mmerp-badge-color': variant === 'filled' ? 'var(--mmerp-color-body)' : variant === 'outline' ? `var(--mmerp-color-${color}-6)` : `var(--mmerp-color-${color}-8)`,
    '--mmerp-badge-border': variant === 'outline' ? `var(--mmerp-color-${color}-6)` : 'transparent',
  }
  return <span className={cx(styles.root, className)} style={vars} data-variant={variant} {...props} />
}
