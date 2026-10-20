import { useRender } from '@base-ui/react/use-render'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import type { RenderProp } from '../../internal/render.ts'
import type { StyleVars } from '../../internal/style.ts'
import { toneColor, type Tone } from '../../internal/tone.ts'
import styles from './Anchor.module.css'

export interface AnchorProps extends React.ComponentPropsWithRef<'a'> {
  tone?: Tone
  underline?: 'hover' | 'always' | 'never'
  render?: RenderProp
}

export function Anchor({ tone, underline = 'hover', className, style, render, ...props }: AnchorProps) {
  const vars: StyleVars = { ...style, color: toneColor(tone, 'var(--mmerp-color-primary-6)') }
  return useRender({
    defaultTagName: 'a',
    render,
    props: { className: cx(styles.root, className), style: vars, 'data-underline': underline, ...props },
  })
}
