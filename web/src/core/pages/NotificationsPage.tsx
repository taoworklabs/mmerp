import { Button, Group, Text, Tooltip } from '@mantine/core'
import { IconChecks } from '@tabler/icons-react'
import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router'
import { api, unwrap, type Notification } from '@/shared/api/core'
import { useRecordLinks } from '@/shared/area'
import { useMe } from '@/shared/auth/me'
import { errorText, formatAgo, formatDateTime } from '@/shared/i18n'
import { notifyError } from '@/shared/ui/notify'
import { InboxList, InboxRow, ListPage } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { notificationKeys } from '../notificationKeys'

// The user's notifications, newest first. Opening one marks it read and goes to its
// record, or to the background jobs; a record the user may no longer view only gets read.
export default function NotificationsPage() {
  const { t } = useTranslation()
  const me = useMe()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const linkOf = useRecordLinks()
  const list = useInfiniteQuery({
    queryKey: notificationKeys.list(),
    queryFn: ({ pageParam }) => unwrap(api.GET('/notifications', { params: { query: pageParam ? { before: pageParam } : {} } })),
    initialPageParam: 0,
    getNextPageParam: (last) => last.next_before ?? undefined,
  })
  const refresh = () => qc.invalidateQueries({ queryKey: notificationKeys.all() })
  const readAll = useMutation({
    mutationFn: () => unwrap(api.POST('/notifications/read-all')),
    onSuccess: refresh,
    onError: (err) => notifyError(errorText(t, err)),
  })

  const open = async (n: Notification) => {
    if (!n.read) {
      try {
        await unwrap(api.POST('/notifications/{id}/read', { params: { path: { id: n.id } } }))
      } catch (err) {
        notifyError(errorText(t, err))
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

  const items = list.data?.pages.flatMap((p) => p.items) ?? []
  let body
  if (list.isPending) body = <ContentSkeleton />
  else if (list.isError) body = <ErrorState message={errorText(t, list.error)} onRetry={() => void list.refetch()} />
  else if (items.length === 0) body = <EmptyState title={t('core.notifications.empty')} description={t('core.notifications.empty_description')} />
  else
    body = (
      <>
        <InboxList label={t('core.notifications.title')}>
          {items.map((n) => (
            <InboxRow
              key={n.id}
              active={false}
              unread={!n.read}
              onClick={() => void open(n)}
              label={
                <Group justify="space-between" wrap="nowrap">
                  <Text size="sm" fw="inherit">
                    {t(`core.notifications.kind.${n.kind}`)}
                  </Text>
                  <Tooltip label={formatDateTime(n.created_at, me.timezone)}>
                    <Text size="xs" c="dimmed">
                      {formatAgo(n.created_at, me.timezone)}
                    </Text>
                  </Tooltip>
                </Group>
              }
              description={describe(n)}
            />
          ))}
        </InboxList>
        {list.hasNextPage && (
          <Group justify="center">
            <Button size="xs" variant="default" loading={list.isFetchingNextPage} onClick={() => void list.fetchNextPage()}>
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
      {body}
    </ListPage>
  )
}
