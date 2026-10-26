import { Button as BaseButton } from '@base-ui/react/button'
import { useRender } from '@base-ui/react/use-render'
import * as React from 'react'
import type { RenderProp } from './render.ts'

// Base UI's Button insists on a native <button>; a link rendered through `render` must stay a plain
// element (no role, no type), so only real buttons go through it.
const isNative = (render: RenderProp | undefined) => render === undefined || (React.isValidElement(render) && render.type === 'button')

function Plain({ render, ...props }: React.ComponentPropsWithRef<'button'> & { render: RenderProp }) {
  const { disabled, ...rest } = props
  return useRender({ defaultTagName: 'button', render, props: { ...rest, 'aria-disabled': disabled || undefined, tabIndex: disabled ? -1 : undefined } })
}

export function ButtonBase({ render, ...props }: React.ComponentPropsWithRef<'button'> & { render?: RenderProp }) {
  if (render !== undefined && !isNative(render)) return <Plain render={render} {...props} />
  return <BaseButton render={render as React.ComponentProps<typeof BaseButton>['render']} {...props} />
}
