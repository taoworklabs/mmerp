import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import { spaceVar } from '../../internal/space.ts'
import type { StyleVars } from '../../internal/style.ts'
import type { Space } from '../../tokens/tokens.ts'
import styles from './Paper.module.css'

export interface PaperProps extends React.ComponentPropsWithRef<'div'> {
  withBorder?: boolean
  padding?: Space | 0
  shadow?: 'xs' | 'sm'
}

export function Paper({ withBorder, padding = 0, shadow, className, style, ...props }: PaperProps) {
  const vars: StyleVars = { ...style, '--mmerp-padding': spaceVar(padding, 'md') }
  return (
    <div
      className={cx(styles.root, className)}
      style={vars}
      data-with-border={withBorder ? '' : undefined}
      data-shadow={shadow}
      {...props}
    />
  )
}
