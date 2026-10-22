import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import styles from './Center.module.css'

export interface CenterProps extends React.ComponentPropsWithRef<'div'> {
  inline?: boolean
}

export function Center({ inline, className, ...props }: CenterProps) {
  return <div className={cx(styles.root, className)} data-inline={inline ? '' : undefined} {...props} />
}
