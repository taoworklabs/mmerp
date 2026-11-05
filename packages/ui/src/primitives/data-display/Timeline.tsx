import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import type { StyleVars } from '../../internal/style.ts'
import { ramp, type Tone } from '../../internal/tone.ts'
import styles from './Timeline.module.css'

export type TimelineProps = React.ComponentPropsWithRef<'div'>

function TimelineRoot({ className, ...props }: TimelineProps) {
  return <div className={cx(styles.root, className)} {...props} />
}

export interface TimelineItemProps extends Omit<React.ComponentPropsWithRef<'div'>, 'title' | 'color'> {
  bullet?: React.ReactNode
  title?: React.ReactNode
  active?: boolean
  // An active item's bullet and line take this tone (primary by default).
  tone?: Tone
  // Dashed: the step after this one is still to come.
  lineVariant?: 'solid' | 'dashed'
}

function Item({ bullet, title, active, tone, lineVariant, className, style, children, ...props }: TimelineItemProps) {
  const vars: StyleVars = { ...style, ...(tone && { '--mmerp-timeline-color': `var(--mmerp-color-${ramp(tone)}-6)` }) }
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
