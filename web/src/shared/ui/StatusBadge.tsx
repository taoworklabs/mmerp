import { Badge } from '@mantine/core'
import type { ReactNode } from 'react'
import classes from './StatusBadge.module.css'

// Tone of a catalog record's state; document states use DocumentStatus instead.
export type StatusTone = 'positive' | 'neutral' | 'info' | 'warning' | 'negative'

const color: Record<StatusTone, string> = { positive: 'success', neutral: 'gray', info: 'info', warning: 'warning', negative: 'danger' }

// StatusBadge shows a state as text with a coloured dot; colour is never the only signal.
export function StatusBadge({ tone, children }: { tone: StatusTone; children: ReactNode }) {
  return (
    <Badge color={color[tone]} leftSection={<span aria-hidden className={classes.dot} />} size="md">
      {children}
    </Badge>
  )
}
