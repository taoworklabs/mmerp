import { useRender } from '@base-ui/react/use-render'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import type { RenderProp } from '../../internal/render.ts'
import { spaceVar } from '../../internal/space.ts'
import type { StyleVars } from '../../internal/style.ts'
import type { Space } from '../../tokens/tokens.ts'
import styles from './Stack.module.css'

export type StackAlign = 'start' | 'center' | 'end' | 'stretch'
export type StackJustify = 'start' | 'center' | 'end' | 'space-between'

export interface StackProps extends React.ComponentPropsWithRef<'div'> {
  gap?: Space | 0
  align?: StackAlign
  justify?: StackJustify
  render?: RenderProp
}

const alignItems: Record<StackAlign, React.CSSProperties['alignItems']> = {
  start: 'flex-start',
  center: 'center',
  end: 'flex-end',
  stretch: 'stretch',
}

const justifyContent: Record<StackJustify, React.CSSProperties['justifyContent']> = {
  start: 'flex-start',
  center: 'center',
  end: 'flex-end',
  'space-between': 'space-between',
}

export function Stack({ gap = 'md', align = 'stretch', justify, className, style, render, ...props }: StackProps) {
  const vars: StyleVars = {
    ...style,
    '--mmerp-gap': spaceVar(gap, 'md'),
    alignItems: alignItems[align],
    justifyContent: justify ? justifyContent[justify] : undefined,
  }
  return useRender({
    defaultTagName: 'div',
    render,
    props: { className: cx(styles.root, className), style: vars, ...props },
  })
}
