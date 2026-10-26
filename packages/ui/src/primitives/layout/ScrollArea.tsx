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
    <BaseScrollArea.Root data-scroll-area="" className={cx(styles.root, className)} {...props}>
      <BaseScrollArea.Viewport data-scroll-area-viewport="" ref={viewportRef} className={styles.viewport}>
        <BaseScrollArea.Content>{children}</BaseScrollArea.Content>
      </BaseScrollArea.Viewport>
      <BaseScrollArea.Scrollbar
        data-scroll-area-scrollbar=""
        orientation="vertical"
        className={styles.scrollbar}
        style={{ top: 'var(--mmerp-scrollbar-inset-top, 0px)', bottom: 'max(var(--mmerp-scrollbar-inset-bottom, 0px), var(--scroll-area-corner-height, 0px))' }}
      >
        <BaseScrollArea.Thumb data-scroll-area-thumb="" className={styles.thumb} />
      </BaseScrollArea.Scrollbar>
      <BaseScrollArea.Scrollbar data-scroll-area-scrollbar="" orientation="horizontal" className={styles.scrollbar}>
        <BaseScrollArea.Thumb data-scroll-area-thumb="" className={styles.thumb} />
      </BaseScrollArea.Scrollbar>
      <BaseScrollArea.Corner className={styles.corner} />
    </BaseScrollArea.Root>
  )
}
