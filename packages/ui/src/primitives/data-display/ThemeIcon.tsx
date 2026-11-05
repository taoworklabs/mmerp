import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import type { StyleVars } from '../../internal/style.ts'
import { ramp, type Tone } from '../../internal/tone.ts'
import styles from './ThemeIcon.module.css'

export interface ThemeIconProps extends Omit<React.ComponentPropsWithRef<'div'>, 'color'> {
  tone?: Tone
  variant?: 'light' | 'filled'
  size?: number
  circle?: boolean
}

export function ThemeIcon({ tone = 'primary', variant = 'light', size = 30, circle, className, style, children, ...props }: ThemeIconProps) {
  const color = ramp(tone)
  const vars: StyleVars = {
    ...style,
    '--mmerp-theme-icon-size': `${size}px`,
    '--mmerp-theme-icon-bg': variant === 'filled' ? `var(--mmerp-color-${color}-6)` : `var(--mmerp-color-${color}-light)`,
    '--mmerp-theme-icon-color': variant === 'filled' ? 'var(--mmerp-color-body)' : `var(--mmerp-color-${color}-light-color)`,
  }
  return (
    <div className={cx(styles.root, className)} style={vars} data-circle={circle ? '' : undefined} {...props}>
      {children}
    </div>
  )
}
