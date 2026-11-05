import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import type { StyleVars } from '../../internal/style.ts'
import type { Tone } from '../../internal/tone.ts'
import styles from './Alert.module.css'

export interface AlertProps extends Omit<React.ComponentPropsWithRef<'div'>, 'title' | 'color'> {
  tone?: Extract<Tone, 'info' | 'success' | 'warning' | 'danger' | 'discovery'>
  title?: React.ReactNode
  icon?: React.ReactNode
}

export function Alert({ tone = 'info', title, icon, className, style, children, ...props }: AlertProps) {
  const vars: StyleVars = {
    ...style,
    '--mmerp-alert-bg': `var(--mmerp-color-${tone}-light)`,
    '--mmerp-alert-color': `var(--mmerp-color-${tone}-light-color)`,
  }
  return (
    <div role={tone === 'danger' ? 'alert' : 'status'} className={cx(styles.root, className)} style={vars} {...props}>
      {icon && <span className={styles.icon}>{icon}</span>}
      <div className={styles.body}>
        {title && <div className={styles.title}>{title}</div>}
        {children}
      </div>
    </div>
  )
}
