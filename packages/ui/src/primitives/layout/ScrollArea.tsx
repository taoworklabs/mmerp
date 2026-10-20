import { ScrollArea as BaseScrollArea } from '@base-ui/react/scroll-area'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import styles from './ScrollArea.module.css'

export interface ScrollAreaProps extends Omit<React.ComponentPropsWithRef<'div'>, 'children'> {
  children: React.ReactNode
  viewportRef?: React.Ref<HTMLDivElement>
}

export function ScrollArea({ children, className, viewportRef, ...props }: ScrollAreaProps) {
  return (
    <BaseScrollArea.Root className={cx(styles.root, className)} {...props}>
      <BaseScrollArea.Viewport ref={viewportRef} className={styles.viewport}>
        <BaseScrollArea.Content>{children}</BaseScrollArea.Content>
      </BaseScrollArea.Viewport>
      <BaseScrollArea.Scrollbar orientation="vertical" className={styles.scrollbar}>
        <BaseScrollArea.Thumb className={styles.thumb} />
      </BaseScrollArea.Scrollbar>
      <BaseScrollArea.Scrollbar orientation="horizontal" className={styles.scrollbar}>
        <BaseScrollArea.Thumb className={styles.thumb} />
      </BaseScrollArea.Scrollbar>
      <BaseScrollArea.Corner className={styles.corner} />
    </BaseScrollArea.Root>
  )
}
