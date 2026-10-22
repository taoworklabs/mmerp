import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import type { StyleVars } from '../../internal/style.ts'
import styles from './Skeleton.module.css'

export interface SkeletonProps extends React.ComponentPropsWithRef<'div'> {
  width?: number | string
  height?: number | string
  circle?: boolean
  visible?: boolean
}

export function Skeleton({ width, height, circle, visible = true, className, style, children, ...props }: SkeletonProps) {
  if (!visible) return <>{children}</>
  const vars: StyleVars = { ...style, width, height }
  return (
    <div className={cx(styles.root, className)} style={vars} data-circle={circle ? '' : undefined} {...props}>
      {children}
    </div>
  )
}
