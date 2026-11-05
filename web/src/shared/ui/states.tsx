import { Button, Page as KitPage, Paper, Skeleton, Stack, Text, ThemeIcon } from '@mmerp/ui'
import { IconInbox, type Icon as TablerIcon } from '@tabler/icons-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { errorText } from '@/shared/i18n'
import { Page } from './page'
import { Notice } from './Notice'
import { failure, type Retryable } from './queryState'
import classes from './states.module.css'
import { icon } from './theme'

// EmptyState explains why a region is empty and offers the next step.
export function EmptyState({ title, description, action, icon: Icon = IconInbox }: { title: string; description?: string; action?: ReactNode; icon?: TablerIcon }) {
  return (
    <Paper withBorder padding="xl" className={classes.empty}>
      <Stack align="center" gap="xs">
        <ThemeIcon variant="light" tone="neutral" size={48} circle aria-hidden>
          <Icon {...icon.empty} />
        </ThemeIcon>
        <Text weight={600}>{title}</Text>
        {description && (
          <Text tone="dimmed" size="sm" align="center" className={classes.hint}>
            {description}
          </Text>
        )}
        {action}
      </Stack>
    </Paper>
  )
}

// ErrorState shows what failed, with text and an icon, and a way to retry. Given the queries a
// region loads from, it shows the first failure and Retry refetches every query that failed.
export function ErrorState(props: { message: string; onRetry?: () => void } | { queries: Retryable[] }) {
  const { t } = useTranslation()
  let message, onRetry
  if ('queries' in props) {
    const f = failure(props.queries)
    message = errorText(t, f.error)
    onRetry = f.retry
  } else ({ message, onRetry } = props)
  return (
    <Notice tone="danger">
      <Stack gap="xs" align="start">
        <Text>{message}</Text>
        {onRetry && (
          <Button variant="default" onClick={onRetry}>
            {t('shared.state.retry')}
          </Button>
        )}
      </Stack>
    </Notice>
  )
}

// PageSkeleton stands in for a whole page while its code or data loads, framed like Page.
export function PageSkeleton() {
  return (
    <KitPage header={<Skeleton height={30} width="30%" />} aria-busy="true">
      <Skeleton height={200} />
    </KitPage>
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
          <Button render={<Link to={back?.to ?? '/'} />} variant="default">
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
    <Notice tone="info">{t('shared.state.read_only')}</Notice>
  )
}
