import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import styles from './VisuallyHidden.module.css'

export type VisuallyHiddenProps = React.ComponentPropsWithRef<'span'>

export function VisuallyHidden({ className, ...props }: VisuallyHiddenProps) {
  return <span className={cx(styles.root, className)} {...props} />
}
