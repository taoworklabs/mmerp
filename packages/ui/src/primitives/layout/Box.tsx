import { useRender } from '@base-ui/react/use-render'
import type * as React from 'react'
import type { RenderProp } from '../../internal/render.ts'

export interface BoxProps extends React.ComponentPropsWithRef<'div'> {
  render?: RenderProp
}

// The bare polymorphic primitive: a div by default, anything else through render.
export function Box({ render, ...props }: BoxProps) {
  return useRender({ defaultTagName: 'div', render, props })
}
