import { Center, Paper, Stack, Title } from '@mmerp/ui'
import type { ReactNode } from 'react'
import classes from './AuthPage.module.css'

// AuthPage frames screens outside the app shell, such as sign-in.
export function AuthPage({ title, children }: { title: string; children: ReactNode }) {
  return (
    <Center className={classes.root}>
      <Paper withBorder padding="xl" className={classes.card}>
        <Stack gap="md">
          <Title order={1}>{title}</Title>
          {children}
        </Stack>
      </Paper>
    </Center>
  )
}
