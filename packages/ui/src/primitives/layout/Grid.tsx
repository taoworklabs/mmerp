import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import { spaceVar } from '../../internal/space.ts'
import type { StyleVars } from '../../internal/style.ts'
import type { Space } from '../../tokens/tokens.ts'
import styles from './Grid.module.css'

export interface GridProps extends React.ComponentPropsWithRef<'div'> {
  columns?: number
  gap?: Space
}

function GridRoot({ columns = 12, gap = 'md', className, style, ...props }: GridProps) {
  const vars: StyleVars = { ...style, '--mmerp-grid-columns': columns, '--mmerp-gap': spaceVar(gap, 'md') }
  return <div className={cx(styles.root, className)} style={vars} {...props} />
}

export interface GridColProps extends React.ComponentPropsWithRef<'div'> {
  span?: number | 'auto'
}

function Col({ span, className, style, ...props }: GridColProps) {
  const vars: StyleVars = span === undefined ? { ...style } : { ...style, gridColumn: span === 'auto' ? 'auto' : `span ${span} / span ${span}` }
  return <div className={cx(styles.col, className)} style={vars} {...props} />
}

export const Grid = Object.assign(GridRoot, { Col })
