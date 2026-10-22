import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import styles from './Timeline.module.css'

export type TimelineProps = React.ComponentPropsWithRef<'div'>

function TimelineRoot({ className, ...props }: TimelineProps) {
  return <div className={cx(styles.root, className)} {...props} />
}

export interface TimelineItemProps extends Omit<React.ComponentPropsWithRef<'div'>, 'title'> {
  bullet?: React.ReactNode
  title?: React.ReactNode
  active?: boolean
}

function Item({ bullet, title, active, className, children, ...props }: TimelineItemProps) {
  return (
    <div className={cx(styles.item, className)} data-active={active ? '' : undefined} {...props}>
      <span className={styles.bullet}>{bullet}</span>
      <div className={styles.content}>
        {title && <div className={styles.title}>{title}</div>}
        {children && <div className={styles.body}>{children}</div>}
      </div>
    </div>
  )
}

export const Timeline = Object.assign(TimelineRoot, { Item })
