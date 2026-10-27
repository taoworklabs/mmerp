import { ButtonBase } from '../../internal/ButtonBase.tsx'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import type { RenderProp } from '../../internal/render.ts'
import styles from './UnstyledButton.module.css'

export interface UnstyledButtonProps extends React.ComponentPropsWithRef<'button'> {
  render?: RenderProp
}

// A bare button for compound components (NavLink, menu items, ...) that bring their own look.
export function UnstyledButton({ className, render, ...props }: UnstyledButtonProps) {
  return <ButtonBase render={render} className={cx(styles.root, className)} {...props} />
}
