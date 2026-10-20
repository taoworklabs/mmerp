import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import { UnstyledButton } from '../input/UnstyledButton.tsx'
import styles from './Burger.module.css'

export interface BurgerProps extends Omit<React.ComponentPropsWithRef<'button'>, 'onClick' | 'children'> {
  opened: boolean
  onClick(): void
  'aria-label': string
}

export function Burger({ opened, onClick, className, ...props }: BurgerProps) {
  return (
    <UnstyledButton
      type="button"
      data-mmerp-control=""
      aria-expanded={opened}
      onClick={onClick}
      className={cx(styles.root, className)}
      {...props}
    >
      <span className={styles.box} data-opened={opened ? '' : undefined}>
        <span className={styles.line} />
        <span className={styles.line} />
        <span className={styles.line} />
      </span>
    </UnstyledButton>
  )
}
