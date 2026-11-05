import { Button as BaseButton } from '@base-ui/react/button'
import { useRender } from '@base-ui/react/use-render'
import * as React from 'react'
import type { RenderProp } from './render.ts'

// Base UI's Button insists on a native <button>; a link rendered through `render` must stay a plain
// element (no role, no type), so only real buttons go through it.
const isNative = (render: RenderProp | undefined) => render === undefined || (React.isValidElement(render) && render.type === 'button')

export type ButtonBaseProps = React.ComponentPropsWithRef<'button'> & {
  render?: RenderProp
  // Disabled but still focusable (aria-disabled): a focused control that becomes disabled keeps focus.
  focusableWhenDisabled?: boolean
}

function Plain({ render, focusableWhenDisabled, ...props }: ButtonBaseProps & { render: RenderProp }) {
  const { disabled, onClick, ...rest } = props
  // A link has no native disabled; block activation (click and Enter, which fires click) like Base UI does.
  const guardedClick = disabled ? (event: React.MouseEvent<HTMLButtonElement>) => event.preventDefault() : onClick
  return useRender({ defaultTagName: 'button', render, props: { ...rest, onClick: guardedClick, 'aria-disabled': disabled || undefined, tabIndex: disabled && !focusableWhenDisabled ? -1 : undefined } })
}

export function ButtonBase({ render, ...props }: ButtonBaseProps) {
  if (render !== undefined && !isNative(render)) return <Plain render={render} {...props} />
  return <BaseButton render={render as React.ComponentProps<typeof BaseButton>['render']} {...props} />
}
