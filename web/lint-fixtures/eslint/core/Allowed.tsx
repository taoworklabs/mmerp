// Allowed: an area using only the Mantine primitives on the allowlist.
import { Anchor, Button, Divider, Group, List, Loader, Skeleton, Stack, Text, Tooltip } from '@mantine/core'

export const Fixture = ({ label }: { label: string }) => (
  <Stack>
    <Group>
      <Tooltip label={label}>
        <Button>{label}</Button>
      </Tooltip>
      <Anchor>{label}</Anchor>
    </Group>
    <Divider />
    <List />
    <Loader />
    <Skeleton />
    <Text>{label}</Text>
  </Stack>
)
