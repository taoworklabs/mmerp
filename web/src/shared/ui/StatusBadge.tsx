import { Badge, type Tone } from '@mmerp/ui'
import type { ReactNode } from 'react'

// Tone of a catalog record's state; document states use DocumentStatus instead.
export type StatusTone = Extract<Tone, 'success' | 'neutral' | 'info' | 'warning' | 'danger'>

// StatusBadge shows a state as tinted text; the label always names the state, so colour is never the only signal.
// A span, since it also sits inside sentences (the activity feed's "moved to Pending").
export function StatusBadge({ tone, children }: { tone: StatusTone; children: ReactNode }) {
  return <Badge tone={tone}>{children}</Badge>
}
