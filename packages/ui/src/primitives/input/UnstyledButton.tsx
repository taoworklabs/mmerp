import { Button as BaseButton } from '@base-ui/react/button'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import type { RenderProp } from '../../internal/render.ts'
import styles from './UnstyledButton.module.css'

export interface UnstyledButtonProps extends React.ComponentPropsWithRef<'button'> {
  render?: RenderProp
}

// A bare button for compound components (NavLink, menu items, ...) that bring their own look.
export function UnstyledButton({ className, render, ...props }: UnstyledButtonProps) {
  return <BaseButton render={render as React.ComponentProps<typeof BaseButton>['render']} className={cx(styles.root, className)} {...props} />
}
