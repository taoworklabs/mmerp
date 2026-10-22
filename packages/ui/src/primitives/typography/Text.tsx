import { useRender } from '@base-ui/react/use-render'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import type { RenderProp } from '../../internal/render.ts'
import type { StyleVars } from '../../internal/style.ts'
import { toneColor, type Tone } from '../../internal/tone.ts'
import type { FontSize } from '../../tokens/tokens.ts'
import styles from './Text.module.css'

export interface TextProps extends React.ComponentPropsWithRef<'p'> {
  size?: FontSize
  tone?: Tone
  weight?: 400 | 500 | 600 | 650
  align?: 'start' | 'center' | 'end'
  truncate?: boolean
  lineClamp?: number
  render?: RenderProp
}

const fontWeight: Record<400 | 500 | 600 | 650, string> = {
  400: 'var(--mmerp-font-weight-regular)',
  500: 'var(--mmerp-font-weight-medium)',
  600: 'var(--mmerp-font-weight-semibold)',
  650: 'var(--mmerp-font-weight-heading)',
}

export function Text({
  size = 'sm',
  tone = 'default',
  weight,
  align,
  truncate,
  lineClamp,
  className,
  style,
  render,
  ...props
}: TextProps) {
  const vars: StyleVars = {
    ...style,
    fontSize: `var(--mmerp-font-size-${size})`,
    lineHeight: `var(--mmerp-line-height-${size})`,
    color: toneColor(tone, 'var(--mmerp-color-text)'),
    textAlign: align,
    fontWeight: weight === undefined ? undefined : fontWeight[weight],
    '--mmerp-line-clamp': lineClamp,
  }
  return useRender({
    defaultTagName: 'p',
    render,
    props: {
      className: cx(styles.root, className),
      style: vars,
      'data-truncate': truncate ? '' : undefined,
      'data-line-clamp': lineClamp === undefined ? undefined : '',
      ...props,
    },
  })
}
