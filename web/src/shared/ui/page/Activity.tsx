import { Panel, SimpleGrid, Stack } from '@mmerp/ui'
import type { ReactNode } from 'react'

// DocumentSection is one more part of a document's side column (attachments), its title
// also the tab's label on a narrow screen.
export type DocumentSection = { key: string; title: string; content: ReactNode }

// Activity lays out a wide document's side column in one tab: the framed parts (approval and
// the extra sections) on the left, the activity feed, usually the longest, on the right; one column below 1024px.
export function Activity({ top, history }: { top: DocumentSection[]; history?: DocumentSection }) {
  return (
    <SimpleGrid cols={{ base: 1, md: 2 }} spacing="xl">
      <Stack gap="md">
        {top.map((p) => (
          <Panel key={p.key} title={p.title}>
            {p.content}
          </Panel>
        ))}
      </Stack>
      {/* In its own stack so the history ends with its content instead of the row. */}
      {history && (
        <Stack>
          <Panel title={history.title} plain>
            {history.content}
          </Panel>
        </Stack>
      )}
    </SimpleGrid>
  )
}
