import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import { spaceVar } from '../../internal/space.ts'
import type { StyleVars } from '../../internal/style.ts'
import type { Space } from '../../tokens/tokens.ts'
import styles from './List.module.css'

export interface ListProps extends Omit<React.ComponentPropsWithRef<'ul'>, 'ref'> {
  ref?: React.Ref<HTMLUListElement | HTMLOListElement>
  type?: 'unordered' | 'ordered'
  spacing?: Space
}

function ListRoot({ type = 'unordered', spacing = 'xs', className, style, ref, ...props }: ListProps) {
  const vars: StyleVars = { ...style, '--mmerp-item-spacing': spaceVar(spacing, 'xs') }
  if (type === 'ordered') {
    return <ol ref={ref as React.Ref<HTMLOListElement>} className={cx(styles.root, className)} style={vars} {...props} />
  }
  return <ul ref={ref as React.Ref<HTMLUListElement>} className={cx(styles.root, className)} style={vars} {...props} />
}

export type ListItemProps = React.ComponentPropsWithRef<'li'>

function Item(props: ListItemProps) {
  return <li {...props} />
}

export const List = Object.assign(ListRoot, { Item })
