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

export type GridSpan = number | 'auto'
export type ResponsiveGridSpan = Partial<Record<'base' | 'sm' | 'md' | 'lg' | 'xl', GridSpan>>

export interface GridColProps extends React.ComponentPropsWithRef<'div'> {
  span?: GridSpan | ResponsiveGridSpan
}

function gridColumn(span: GridSpan | undefined): string | undefined {
  if (span === undefined) return undefined
  return span === 'auto' ? 'auto' : `span ${span} / span ${span}`
}

function Col({ span, className, style, ...props }: GridColProps) {
  const responsive: ResponsiveGridSpan | undefined = span === undefined ? undefined : typeof span === 'object' ? span : { base: span }
  const vars: StyleVars = {
    ...style,
    '--mmerp-span-base': gridColumn(responsive?.base),
    '--mmerp-span-sm': gridColumn(responsive?.sm),
    '--mmerp-span-md': gridColumn(responsive?.md),
    '--mmerp-span-lg': gridColumn(responsive?.lg),
    '--mmerp-span-xl': gridColumn(responsive?.xl),
  }
  return <div className={cx(styles.col, className)} style={vars} {...props} />
}

export const Grid = Object.assign(GridRoot, { Col })
