import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import { Stack } from '../../primitives/layout/Stack.tsx'
import styles from './RecordLayout.module.css'

export interface RecordColumnsProps extends React.ComponentPropsWithRef<'div'> {
  main: React.ReactNode
  side: React.ReactNode
  // 'document' gives the side column more room (≈62/38) than a record's digest cards (≈1.6/1).
  ratio?: 'record' | 'document'
}

// RecordColumns lays out a record's or document's overview: its values on the left, digests or
// parts of it on the right; one column below 64em, the side column under the main one.
export function RecordColumns({ main, side, ratio = 'record', className, ...props }: RecordColumnsProps) {
  return (
    <div className={cx(styles.columns, className)} data-ratio={ratio} {...props}>
      <Stack gap="md" className={styles.main}>
        {main}
      </Stack>
      <Stack gap={0} className={styles.side}>
        {side}
      </Stack>
    </div>
  )
}

export type ActionBarProps = React.ComponentPropsWithRef<'div'>

// ActionBar keeps a record or document's actions reachable at the bottom of the screen below 64em,
// where the page header no longer has room to row them; above it, it renders nothing.
export function ActionBar({ children, className, ...props }: ActionBarProps) {
  return (
    <>
      <div className={styles.spacer} />
      <div className={cx(styles.bar, className)} {...props}>
        {children}
      </div>
    </>
  )
}
