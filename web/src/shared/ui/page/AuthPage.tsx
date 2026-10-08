import { Center, Paper, Stack, Title } from '@mantine/core'
import type { ReactNode } from 'react'

// AuthPage frames screens outside the app shell, such as sign-in.
export function AuthPage({ title, children }: { title: string; children: ReactNode }) {
  return (
    <Center mih="100vh" p="md">
      <Paper withBorder p="xl" w="100%" maw={400}>
        <Stack gap="md">
          <Title order={1}>{title}</Title>
          {children}
        </Stack>
      </Paper>
    </Center>
  )
}
