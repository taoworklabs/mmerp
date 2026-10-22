import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import type { StyleVars } from '../../internal/style.ts'
import styles from './Title.module.css'

export interface TitleProps extends React.ComponentPropsWithRef<'h1'> {
  order?: 1 | 2 | 3
}

export function Title({ order = 1, className, style, ...props }: TitleProps) {
  const vars: StyleVars = {
    ...style,
    fontSize: `var(--mmerp-heading-h${order}-size)`,
    lineHeight: `var(--mmerp-heading-h${order}-line-height)`,
  }
  const Tag = `h${order}` as 'h1' | 'h2' | 'h3'
  return <Tag className={cx(styles.root, className)} style={vars} {...props} />
}
