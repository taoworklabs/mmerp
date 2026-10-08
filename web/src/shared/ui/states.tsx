import { Alert, Button, Paper, Skeleton, Stack, Text, ThemeIcon } from '@mantine/core'
import { IconAlertCircle, IconInbox, IconInfoCircle, type Icon as TablerIcon } from '@tabler/icons-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { Page, pageFrame } from './page'
import classes from './states.module.css'
import { icon } from './theme'

// EmptyState explains why a region is empty and offers the next step.
export function EmptyState({ title, description, action, icon: Icon = IconInbox }: { title: string; description?: string; action?: ReactNode; icon?: TablerIcon }) {
  return (
    <Paper withBorder p="xl" className={classes.empty}>
      <Stack align="center" gap="xs">
        <ThemeIcon variant="light" color="gray" size={48} radius="xl" aria-hidden>
          <Icon size={24} stroke={1.5} />
        </ThemeIcon>
        <Text fw={600}>{title}</Text>
        {description && (
          <Text c="dimmed" size="sm" ta="center" maw={420}>
            {description}
          </Text>
        )}
        {action}
      </Stack>
    </Paper>
  )
}

// ErrorState shows what failed, with text and an icon, and a way to retry.
export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  const { t } = useTranslation()
  return (
    <Alert color="danger" icon={<IconAlertCircle {...icon.button} />} role="alert">
      <Stack gap="xs" align="flex-start">
        <Text>{message}</Text>
        {onRetry && (
          <Button variant="default" onClick={onRetry}>
            {t('shared.state.retry')}
          </Button>
        )}
      </Stack>
    </Alert>
  )
}

// PageSkeleton stands in for a whole page while its code or data loads, framed like Page.
export function PageSkeleton() {
  return (
    <Stack gap="lg" {...pageFrame} aria-busy="true">
      <Skeleton height={30} width="30%" />
      <Skeleton height={200} />
    </Stack>
  )
}

// ContentSkeleton stands in for a region inside a page (body, tab, panel), aligned with its header.
export function ContentSkeleton() {
  return <Skeleton height={200} aria-busy="true" />
}

// NotFoundPage is the 404 of any route, with a way back.
// back points to the list the missing record would be in.
export function NotFoundPage({ back }: { back?: { to: string; label: string } }) {
  return <BlockedPage kind="notfound" back={back} />
}

// ForbiddenPage explains a 403 instead of hiding the page silently.
export function ForbiddenPage() {
  return <BlockedPage kind="forbidden" />
}

function BlockedPage({ kind, back }: { kind: 'notfound' | 'forbidden'; back?: { to: string; label: string } }) {
  const { t } = useTranslation()
  return (
    <Page title={t(`shared.${kind}.title`)}>
      <EmptyState
        title={t(`shared.${kind}.description`)}
        action={
          <Button component={Link} to={back?.to ?? '/'} variant="default">
            {back?.label ?? t('shared.state.home')}
          </Button>
        }
      />
    </Page>
  )
}

// ReadOnlyBanner sits on every page of an area whose product is off but still has data.
export function ReadOnlyBanner() {
  const { t } = useTranslation()
  return (
    <Alert color="info" icon={<IconInfoCircle {...icon.button} />}>
      {t('shared.state.read_only')}
    </Alert>
  )
}
