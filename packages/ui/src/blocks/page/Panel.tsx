import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import { Text } from '../../primitives/typography/Text.tsx'
import styles from './Panel.module.css'

export interface PanelProps extends React.ComponentPropsWithRef<'section'> {
  title: string
  icon?: React.ReactNode
  actions?: React.ReactNode
  // Drops the frame for a full-width history feed; keeps the gap, loses the border and inline padding.
  plain?: boolean
  children: React.ReactNode
}

// Panel is the one card of a record or document page: its values, the approval, the attachments,
// the activity. Icon and actions (an Edit button) sit on its title line, dropping under it on a
// narrow card rather than squeezing the title.
export function Panel({ title, icon, actions, plain, children, className, ...props }: PanelProps) {
  return (
    <section aria-label={title} className={cx(styles.root, className)} data-plain={plain ? '' : undefined} {...props}>
      <div className={styles.titleRow}>
        <div className={styles.titleGroup}>
          {icon}
          <Text weight={600} render={<span />}>
            {title}
          </Text>
        </div>
        {actions}
      </div>
      {children}
    </section>
  )
}
