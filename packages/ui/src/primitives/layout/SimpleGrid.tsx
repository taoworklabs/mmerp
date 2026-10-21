import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import { spaceVar } from '../../internal/space.ts'
import type { StyleVars } from '../../internal/style.ts'
import type { Space } from '../../tokens/tokens.ts'
import styles from './SimpleGrid.module.css'

export interface SimpleGridCols {
  base: number
  sm?: number
  md?: number
  lg?: number
}

export interface SimpleGridProps extends React.ComponentPropsWithRef<'div'> {
  cols: number | SimpleGridCols
  spacing?: Space
}

export function SimpleGrid({ cols, spacing = 'md', className, style, ...props }: SimpleGridProps) {
  const resolved = typeof cols === 'number' ? { base: cols } : cols
  const vars: StyleVars = {
    ...style,
    '--mmerp-cols-base': resolved.base,
    '--mmerp-cols-sm': resolved.sm,
    '--mmerp-cols-md': resolved.md,
    '--mmerp-cols-lg': resolved.lg,
    '--mmerp-gap': spaceVar(spacing, 'md'),
  }
  return <div className={cx(styles.root, className)} style={vars} {...props} />
}
