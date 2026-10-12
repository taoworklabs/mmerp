import { Group, Stack, Text, Title } from '@mantine/core'
import { useId, type ReactNode } from 'react'
import classes from './Section.module.css'

// Section is a titled region of a page (a table, a group of facts), set off from the section
// before it by a rule. Its own buttons sit right of the title, at section size (xs).
export function Section({ title, description, actions, children }: { title: string; description?: ReactNode; actions?: ReactNode; children: ReactNode }) {
  const id = useId()
  return (
    <Stack component="section" gap="sm" className={classes.section} aria-labelledby={id}>
      <Group justify="space-between" align="flex-start" wrap="nowrap" gap="md">
        <Stack gap="xxxs" miw={0}>
          <Title order={2} id={id}>
            {title}
          </Title>
          {description && (
            <Text size="sm" c="dimmed" component="div">
              {description}
            </Text>
          )}
        </Stack>
        {actions && (
          <Group gap="xs" wrap="nowrap">
            {actions}
          </Group>
        )}
      </Group>
      {children}
    </Stack>
  )
}
