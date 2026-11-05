import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import { UnstyledButton } from '../input/UnstyledButton.tsx'
import styles from './Burger.module.css'

export interface BurgerProps extends Omit<React.ComponentPropsWithRef<'button'>, 'children'> {
  open: boolean
  onOpenChange(open: boolean): void
  'aria-label': string
}

export function Burger({ open, onOpenChange, onClick, className, ...props }: BurgerProps) {
  return (
    <UnstyledButton
      type="button"
      data-mmerp-control=""
      aria-expanded={open}
      className={cx(styles.root, className)}
      {...props}
      // After the spread: a wrapping trigger (Tooltip) passes its own onClick, which must not replace ours.
      onClick={(event) => {
        onClick?.(event)
        onOpenChange(!open)
      }}
    >
      <span className={styles.box} data-open={open ? '' : undefined}>
        <span className={styles.line} />
        <span className={styles.line} />
        <span className={styles.line} />
      </span>
    </UnstyledButton>
  )
}
