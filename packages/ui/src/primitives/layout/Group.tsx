import { useRender } from '@base-ui/react/use-render'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import type { RenderProp } from '../../internal/render.ts'
import { spaceVar } from '../../internal/space.ts'
import type { StyleVars } from '../../internal/style.ts'
import type { Space } from '../../tokens/tokens.ts'
import styles from './Group.module.css'

export type GroupAlign = 'start' | 'center' | 'end' | 'stretch'
export type GroupJustify = 'start' | 'center' | 'end' | 'space-between'

export interface GroupProps extends React.ComponentPropsWithRef<'div'> {
  gap?: Space | 0
  align?: GroupAlign
  justify?: GroupJustify
  wrap?: boolean
  grow?: boolean
  shrink?: boolean
  fill?: boolean
  render?: RenderProp
}

const alignItems: Record<GroupAlign, React.CSSProperties['alignItems']> = {
  start: 'flex-start',
  center: 'center',
  end: 'flex-end',
  stretch: 'stretch',
}

const justifyContent: Record<GroupJustify, React.CSSProperties['justifyContent']> = {
  start: 'flex-start',
  center: 'center',
  end: 'flex-end',
  'space-between': 'space-between',
}

export function Group({ gap = 'md', align = 'center', justify, wrap = true, grow, shrink, fill, className, style, render, ...props }: GroupProps) {
  const vars: StyleVars = {
    ...style,
    '--mmerp-gap': spaceVar(gap, 'md'),
    alignItems: alignItems[align],
    justifyContent: justify ? justifyContent[justify] : undefined,
  }
  return useRender({
    defaultTagName: 'div',
    render,
    props: {
      className: cx(styles.root, className),
      style: vars,
      'data-wrap': wrap ? '' : undefined,
      'data-grow': grow ? '' : undefined,
      'data-shrink': shrink ? '' : undefined,
      'data-fill': fill ? '' : undefined,
      ...props,
    },
  })
}
