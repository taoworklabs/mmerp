import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import type { StyleVars } from '../../internal/style.ts'
import type { ColorName } from '../../tokens/tokens.ts'
import styles from './Timeline.module.css'

export type TimelineProps = React.ComponentPropsWithRef<'div'>

function TimelineRoot({ className, ...props }: TimelineProps) {
  return <div className={cx(styles.root, className)} {...props} />
}

export interface TimelineItemProps extends Omit<React.ComponentPropsWithRef<'div'>, 'title'> {
  bullet?: React.ReactNode
  title?: React.ReactNode
  active?: boolean
  // An active item's bullet and line take this colour (primary by default).
  color?: ColorName
  // Dashed: the step after this one is still to come.
  lineVariant?: 'solid' | 'dashed'
}

function Item({ bullet, title, active, color, lineVariant, className, style, children, ...props }: TimelineItemProps) {
  const vars: StyleVars = { ...style, ...(color && { '--mmerp-timeline-color': `var(--mmerp-color-${color}-6)` }) }
  return (
    <div
      className={cx(styles.item, className)}
      style={vars}
      data-active={active ? '' : undefined}
      data-line={lineVariant === 'dashed' ? 'dashed' : undefined}
      {...props}
    >
      <span className={styles.bullet}>{bullet}</span>
      <div className={styles.content}>
        {title && <div className={styles.title}>{title}</div>}
        {children && <div className={styles.body}>{children}</div>}
      </div>
    </div>
  )
}

export const Timeline = Object.assign(TimelineRoot, { Item })
