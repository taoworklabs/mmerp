import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import type { StyleVars } from '../../internal/style.ts'
import type { ColorName } from '../../tokens/tokens.ts'
import styles from './Indicator.module.css'

export interface IndicatorProps extends React.ComponentPropsWithRef<'div'> {
  label?: React.ReactNode
  color?: ColorName
  disabled?: boolean
}

export function Indicator({ label, color = 'primary', disabled, className, style, children, ...props }: IndicatorProps) {
  const vars: StyleVars = { ...style, '--mmerp-indicator-bg': `var(--mmerp-color-${color}-6)` }
  return (
    <div className={cx(styles.root, className)} style={vars} {...props}>
      {children}
      {!disabled && <span className={cx(styles.badge, label === undefined && styles.dot)}>{label}</span>}
    </div>
  )
}
