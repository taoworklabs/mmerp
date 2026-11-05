import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import { UnstyledButton } from '../input/UnstyledButton.tsx'
import styles from './Burger.module.css'

export interface BurgerProps extends Omit<React.ComponentPropsWithRef<'button'>, 'onClick' | 'children'> {
  open: boolean
  onOpenChange(open: boolean): void
  'aria-label': string
}

export function Burger({ open, onOpenChange, className, ...props }: BurgerProps) {
  return (
    <UnstyledButton
      type="button"
      data-mmerp-control=""
      aria-expanded={open}
      onClick={() => onOpenChange(!open)}
      className={cx(styles.root, className)}
      {...props}
    >
      <span className={styles.box} data-open={open ? '' : undefined}>
        <span className={styles.line} />
        <span className={styles.line} />
        <span className={styles.line} />
      </span>
    </UnstyledButton>
  )
}
