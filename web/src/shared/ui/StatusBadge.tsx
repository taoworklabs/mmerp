import { Badge } from '@mantine/core'
import type { ReactNode } from 'react'

// Tone of a catalog record's state; document states use DocumentStatus instead.
export type StatusTone = 'positive' | 'neutral' | 'info' | 'warning' | 'negative'

const color: Record<StatusTone, string> = { positive: 'success', neutral: 'gray', info: 'info', warning: 'warning', negative: 'danger' }

// StatusBadge shows a state as tinted text; the label always names the state, so colour is never the only signal.
export function StatusBadge({ tone, children }: { tone: StatusTone; children: ReactNode }) {
  return (
    <Badge color={color[tone]} size="md">
      {children}
    </Badge>
  )
}
