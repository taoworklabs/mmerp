import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import styles from './Code.module.css'

export interface CodeProps extends React.ComponentPropsWithRef<'code'> {
  block?: boolean
}

export function Code({ block, className, children, ...props }: CodeProps) {
  if (block) {
    return (
      <pre className={cx(styles.root, styles.block, className)}>
        <code {...props}>{children}</code>
      </pre>
    )
  }
  return (
    <code className={cx(styles.root, className)} {...props}>
      {children}
    </code>
  )
}
