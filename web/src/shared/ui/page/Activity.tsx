import { Group, Paper, SimpleGrid, Stack, Text } from '@mmerp/ui'
import type { Icon as TablerIcon } from '@tabler/icons-react'
import type { ReactNode } from 'react'
import { icon } from '../theme'
import classes from './Activity.module.css'

// DocumentSection is one more part of a document's side column (attachments), its title
// also the tab's label on a narrow screen.
export type DocumentSection = { key: string; title: string; content: ReactNode }

// Panel is the one card of a detail page, in either column: a part of the record's values, the
// approval, the attachments, the activity. icon and actions (an Edit button) sit on its title line.
// plain drops the frame, for the history in a full-width document's activity tab.
export function Panel({ title, icon: Icon, actions, plain, children }: { title: string; icon?: TablerIcon; actions?: ReactNode; plain?: boolean; children: ReactNode }) {
  return (
    <Paper render={<section />} aria-label={title} withBorder={!plain} padding="md" className={plain ? classes.plain : undefined}>
      <Stack gap="sm">
        {/* On a narrow card the actions drop under the title rather than squeezing it. */}
        <Group justify="space-between" gap="xs" className={classes.head}>
          <Group gap="xs" wrap={false}>
            {Icon && <Icon {...icon.text} aria-hidden />}
            <Text weight={600}>{title}</Text>
          </Group>
          {actions}
        </Group>
        {children}
      </Stack>
    </Paper>
  )
}

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
