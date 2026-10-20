import { Alert, Button, Group, Text, ThemeIcon, Tooltip } from '@mantine/core'
import { IconAlertCircle, IconAlertTriangle, IconAt, IconBell, IconChecks, IconCircleCheck, IconCircleX, IconClockExclamation, IconHourglassHigh, type Icon as TablerIcon } from '@tabler/icons-react'
import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Fragment, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router'
import { api, unwrap, type Notification } from '@/shared/api/core'
import { useRecordLinks } from '@/shared/area'
import { useMe } from '@/shared/auth/me'
import { errorText, formatAgo, formatDateTime } from '@/shared/i18n'
import { InboxGroup, InboxList, InboxRow, ListPage } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { notificationKeys } from '../notificationKeys'

// Each core kind has its icon, in the colour of what it says: waiting, done, refused, overdue, a mention.
// A product's own kind (<product>.<name>) gets the plain bell and its label from the product's area.
const kindLooks: Record<string, { Icon: TablerIcon; color: string } | undefined> = {
  approval_requested: { Icon: IconHourglassHigh, color: 'warning' },
  approval_approved: { Icon: IconCircleCheck, color: 'success' },
  approval_rejected: { Icon: IconCircleX, color: 'danger' },
  approval_overdue: { Icon: IconClockExclamation, color: 'danger' },
  mentioned: { Icon: IconAt, color: 'info' },
  job_completed: { Icon: IconChecks, color: 'success' },
  job_failed: { Icon: IconAlertTriangle, color: 'danger' },
}
const productLook = { Icon: IconBell, color: 'gray' }
const kindLabelKey = (kind: string) => (kind.includes('.') ? `${kind}.notification` : `core.notifications.kind.${kind}`)

// The user's notifications, newest first, under the day they came. Opening one marks it read
// and goes to its record, or to the background jobs; a record the user may no longer view only gets read.
export default function NotificationsPage() {
  const { t } = useTranslation()
  const me = useMe()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const linkOf = useRecordLinks()
  // A failed read, shown above the list until the next attempt.
  const [error, setError] = useState<string | null>(null)
  const list = useInfiniteQuery({
    queryKey: notificationKeys.list(),
    queryFn: ({ pageParam }) => unwrap(api.GET('/notifications', { params: { query: pageParam ? { before: pageParam } : {} } })),
    initialPageParam: 0,
    getNextPageParam: (last) => last.next_before ?? undefined,
  })
  const refresh = () => qc.invalidateQueries({ queryKey: notificationKeys.all() })
  const readAll = useMutation({
    mutationFn: () => unwrap(api.POST('/notifications/read-all')),
    onMutate: () => setError(null),
    onSuccess: refresh,
    onError: (err) => setError(errorText(t, err)),
  })

  const open = async (n: Notification) => {
    if (!n.read) {
      setError(null)
      try {
        await unwrap(api.POST('/notifications/{id}/read', { params: { path: { id: n.id } } }))
      } catch (err) {
        setError(errorText(t, err))
        return
      }
      void refresh()
    }
    // Jobs stay in the current area, like this page.
    if (n.job_id) void navigate('../jobs', { relative: 'path' })
    else if (n.record_type && n.record_id) {
      const to = linkOf(n.record_type, n.record_id)
      if (to) void navigate(to)
    }
  }
  const describe = (n: Notification) => {
    if (n.job_id) return t('core.notifications.job')
    const type = t(`${n.record_type}.name`, { defaultValue: n.record_type ?? '' })
    if (!n.record_id) return `${type} · ${t('core.notifications.hidden')}`
    return [n.label ? `${type} ${n.label}` : type, n.actor_name].filter(Boolean).join(' · ')
  }
  // The day a notification came, in the tenant's time zone: today and yesterday by name.
  const dayOf = (iso: string) => formatDateTime(iso, me.timezone).slice(0, 10)
  const today = dayOf(new Date().toISOString())
  const yesterday = dayOf(new Date(Date.now() - 86_400_000).toISOString())
  const dayLabel = (day: string) => (day === today ? t('core.notifications.today') : day === yesterday ? t('core.notifications.yesterday') : day)

  const items = list.data?.pages.flatMap((p) => p.items) ?? []
  let body
  if (list.isPending) body = <ContentSkeleton />
  else if (list.isError) body = <ErrorState message={errorText(t, list.error)} onRetry={() => void list.refetch()} />
  else if (items.length === 0) body = <EmptyState title={t('core.notifications.empty')} description={t('core.notifications.empty_description')} />
  else
    body = (
      <>
        <InboxList label={t('core.notifications.title')} framed>
          {items.map((n, i) => {
            const { Icon, color } = kindLooks[n.kind] ?? productLook
            const day = dayOf(n.created_at)
            return (
              <Fragment key={n.id}>
                {(i === 0 || dayOf(items[i - 1]!.created_at) !== day) && <InboxGroup label={dayLabel(day)} />}
                <InboxRow
                  active={false}
                  unread={!n.read}
                  onClick={() => void open(n)}
                  icon={
                    <ThemeIcon variant="light" color={color} size={36} radius={1000} aria-hidden>
                      <Icon {...icon.button} />
                    </ThemeIcon>
                  }
                  label={
                    <Text size="sm" fw="inherit">
                      {t(kindLabelKey(n.kind))}
                    </Text>
                  }
                  description={describe(n)}
                  aside={
                    <Tooltip label={formatDateTime(n.created_at, me.timezone)}>
                      <Text size="xs" c="dimmed">
                        {formatAgo(n.created_at, me.timezone)}
                      </Text>
                    </Tooltip>
                  }
                />
              </Fragment>
            )
          })}
        </InboxList>
        {list.hasNextPage && (
          <Group justify="center">
            <Button variant="default" loading={list.isFetchingNextPage} onClick={() => void list.fetchNextPage()}>
              {t('core.notifications.more')}
            </Button>
          </Group>
        )}
      </>
    )

  return (
    <ListPage
      title={t('core.notifications.title')}
      description={t('core.notifications.description')}
      action={
        <Button variant="default" leftSection={<IconChecks {...icon.button} />} loading={readAll.isPending} disabled={!items.some((n) => !n.read)} onClick={() => readAll.mutate()}>
          {t('core.notifications.read_all')}
        </Button>
      }
    >
      {error && (
        <Alert color="danger" icon={<IconAlertCircle {...icon.button} />} role="alert">
          {error}
        </Alert>
      )}
      {body}
    </ListPage>
  )
}
