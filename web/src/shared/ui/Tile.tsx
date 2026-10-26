import { Group, Paper, SimpleGrid, Skeleton, Stack, Text, ThemeIcon } from '@mmerp/ui'
import type { ComponentType, ReactNode } from 'react'
import { Link } from 'react-router'
import { formatNumber } from '@/shared/i18n'
import { ErrorState } from './states'
import { icon } from './theme'
import classes from './Tile.module.css'

// TileGrid lays tiles out: one column on phones, up to three on wide screens.
export function TileGrid({ children }: { children: ReactNode }) {
  return <SimpleGrid cols={{ base: 1, sm: 2, lg: 3 }} spacing="md">{children}</SimpleGrid>
}

// StatTile is a count that opens the list it counts; value undefined means loading.
// A failed count is not a link: it shows the error and a retry.
export function StatTile(props: { label: string; value: number | undefined; to: string; error?: string; onRetry?: () => void }) {
  const { label, value, to, error, onRetry } = props
  const heading = (
    <Text size="sm" tone="dimmed">
      {label}
    </Text>
  )
  if (error)
    return (
      <Paper withBorder padding="md">
        <Stack gap="xs">
          {heading}
          <ErrorState message={error} onRetry={onRetry} />
        </Stack>
      </Paper>
    )
  return (
    <Paper withBorder padding="md" render={<Link to={to} />} className={classes.tile} aria-busy={value === undefined}>
      <Stack gap="xxs">
        {heading}
        {value === undefined ? (
          <Skeleton className={classes.loading} />
        ) : (
          <Text size="xl" weight={600} className={classes.value}>
            {formatNumber(value)}
          </Text>
        )}
      </Stack>
    </Paper>
  )
}

// LinkTile is a destination: an icon and a name, e.g. a product on the home page.
export function LinkTile({ icon: Icon, title, to }: { icon: ComponentType<{ size?: number; stroke?: number }>; title: string; to: string }) {
  return (
    <Paper withBorder padding="md" render={<Link to={to} />} className={classes.tile}>
      <Group gap="sm" wrap={false}>
        <ThemeIcon variant="light" size={36} aria-hidden>
          <Icon {...icon.button} />
        </ThemeIcon>
        <Text weight={600}>{title}</Text>
      </Group>
    </Paper>
  )
}
