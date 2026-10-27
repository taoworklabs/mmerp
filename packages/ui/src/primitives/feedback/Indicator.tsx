import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import type { StyleVars } from '../../internal/style.ts'
import type { ColorName } from '../../tokens/tokens.ts'
import styles from './Indicator.module.css'

export interface IndicatorProps extends React.ComponentPropsWithRef<'div'> {
  label?: React.ReactNode
  color?: ColorName
  disabled?: boolean
  // Pulls the badge in from the corner, in px; a round child (an icon button) needs it.
  offset?: number
}

export function Indicator({ label, color = 'primary', disabled, offset = 0, className, style, children, ...props }: IndicatorProps) {
  const vars: StyleVars = { ...style, '--mmerp-indicator-bg': `var(--mmerp-color-${color}-6)`, '--mmerp-indicator-offset': `${offset}px` }
  return (
    <div className={cx(styles.root, className)} style={vars} {...props}>
      {children}
      {!disabled && <span className={cx(styles.badge, label === undefined && styles.dot)}>{label}</span>}
    </div>
  )
}
