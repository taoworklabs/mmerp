import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import styles from './Loader.module.css'

export interface LoaderProps extends React.ComponentPropsWithRef<'span'> {
  size?: 'xs' | 'sm' | 'md'
  label: string
}

export function Loader({ size = 'sm', label, className, ...props }: LoaderProps) {
  return <span role="status" aria-label={label} className={cx(styles.root, className)} data-size={size} {...props} />
}
