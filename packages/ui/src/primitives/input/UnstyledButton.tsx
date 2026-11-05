import { ButtonBase, type ButtonBaseProps } from '../../internal/ButtonBase.tsx'
import { cx } from '../../internal/cx.ts'
import styles from './UnstyledButton.module.css'

export type UnstyledButtonProps = ButtonBaseProps

// A bare button for compound components (NavLink, menu items, ...) that bring their own look.
export function UnstyledButton({ className, render, ...props }: UnstyledButtonProps) {
  return <ButtonBase render={render} className={cx(styles.root, className)} {...props} />
}
