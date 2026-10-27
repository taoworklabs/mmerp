import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import type { StyleVars } from '../../internal/style.ts'
import styles from './Alert.module.css'

export interface AlertProps extends Omit<React.ComponentPropsWithRef<'div'>, 'title'> {
  color?: 'info' | 'success' | 'warning' | 'danger' | 'discovery'
  title?: React.ReactNode
  icon?: React.ReactNode
}

export function Alert({ color = 'info', title, icon, className, style, children, ...props }: AlertProps) {
  const vars: StyleVars = {
    ...style,
    '--mmerp-alert-bg': `var(--mmerp-color-${color}-light)`,
    '--mmerp-alert-color': `var(--mmerp-color-${color}-light-color)`,
  }
  return (
    <div role={color === 'danger' ? 'alert' : 'status'} className={cx(styles.root, className)} style={vars} {...props}>
      {icon && <span className={styles.icon}>{icon}</span>}
      <div className={styles.body}>
        {title && <div className={styles.title}>{title}</div>}
        {children}
      </div>
    </div>
  )
}
