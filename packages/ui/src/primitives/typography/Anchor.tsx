import { useRender } from '@base-ui/react/use-render'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import type { RenderProp } from '../../internal/render.ts'
import type { StyleVars } from '../../internal/style.ts'
import type { FontSize } from '../../tokens/tokens.ts'
import { toneColor, type Tone } from '../../internal/tone.ts'
import styles from './Anchor.module.css'

export interface AnchorProps extends React.ComponentPropsWithRef<'a'> {
  size?: FontSize
  tone?: Tone
  weight?: 400 | 500 | 600 | 650
  underline?: 'hover' | 'always' | 'never'
  render?: RenderProp
}

const weightName = { 400: 'regular', 500: 'medium', 600: 'semibold', 650: 'heading' } as const

export function Anchor({ size, tone, weight, underline = 'hover', className, style, render, ...props }: AnchorProps) {
  const vars: StyleVars = {
    ...style,
    color: toneColor(tone, 'var(--mmerp-color-primary-6)'),
    fontWeight: weight && `var(--mmerp-font-weight-${weightName[weight]})`,
    fontSize: size && `var(--mmerp-font-size-${size})`,
    lineHeight: size && `var(--mmerp-line-height-${size})`,
  }
  return useRender({
    defaultTagName: 'a',
    render,
    props: { className: cx(styles.root, className), style: vars, 'data-underline': underline, ...props },
  })
}
