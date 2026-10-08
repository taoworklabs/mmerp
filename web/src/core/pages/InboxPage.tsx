import { Anchor, Divider, Group, Stack, Text, Tooltip } from '@mantine/core'
import { IconExternalLink } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link, useLocation, useNavigate, useSearchParams } from 'react-router'
import { api, unwrap, type InboxItem } from '@/shared/api/core'
import { RecordPreview, useRecordLink } from '@/shared/area'
import { useMe } from '@/shared/auth/me'
import { ApprovalPanel, documentKeys } from '@/shared/document'
import { errorText, formatAgo, formatDateTime } from '@/shared/i18n'
import { InboxList, InboxRow, InboxPage as Template } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'

// The approval inbox: submissions waiting for the user. The selected one is ?item= on the URL.
export default function InboxPage() {
  const { t } = useTranslation()
  const me = useMe()
  const [search, setSearch] = useSearchParams()
  const inbox = useQuery({ queryKey: documentKeys.inbox(), queryFn: () => unwrap(api.GET('/approvals/inbox')) })
  const selected = Number(search.get('item'))
  const item = inbox.data?.items.find((i) => i.instance_id === selected)
  const navigate = useNavigate()
  const location = useLocation()
  // Picking an item pushed a history entry; going back pops it, like the browser's Back.
  // Opened straight from a link, there is nothing to pop.
  const back = () => (location.key === 'default' ? setSearch({}, { replace: true }) : navigate(-1))
  // After a decision, move on to the next waiting item, or the one before when it was the last.
  const advance = () => {
    const items = inbox.data?.items ?? []
    const at = items.findIndex((i) => i.instance_id === selected)
    const next = items[at + 1] ?? items[at - 1]
    if (next) setSearch({ item: String(next.instance_id) }, { replace: true })
    else back()
  }

  let list
  if (inbox.isPending) list = <ContentSkeleton />
  else if (inbox.isError) list = <ErrorState message={errorText(t, inbox.error)} onRetry={() => void inbox.refetch()} />
  else if (inbox.data.items.length === 0) list = <EmptyState title={t('core.inbox.empty')} description={t('core.inbox.empty_description')} />
  else
    list = (
      <InboxList label={t('core.inbox.title')}>
        {inbox.data.items.map((i) => (
          <InboxRow
            key={i.instance_id}
            active={i.instance_id === selected}
            onClick={() => setSearch({ item: String(i.instance_id) })}
            label={
              <Group justify="space-between" wrap="nowrap">
                <Text fw={500} size="sm">
                  {i.number}
                </Text>
                <Tooltip label={formatDateTime(i.submitted_at, me.timezone)}>
                  <Text size="xs" c="dimmed">
                    {formatAgo(i.submitted_at, me.timezone)}
                  </Text>
                </Tooltip>
              </Group>
            }
            description={`${t(`${i.doc_type}.name`, { defaultValue: i.doc_type })} · ${i.submitted_by_name}`}
          />
        ))}
      </InboxList>
    )

  return (
    <Template title={t('core.inbox.title')} description={t('core.inbox.description')} list={list} detail={item ? <Detail item={item} onDone={advance} /> : null} onBack={back} />
  )
}

function Detail({ item, onDone }: { item: InboxItem; onDone: () => void }) {
  const { t } = useTranslation()
  const link = useRecordLink(item.doc_type, item.doc_id)
  return (
    <Stack gap="md">
      <Group justify="space-between">
        <Text fw={600}>
          {item.number}{' '}
          <Text span c="dimmed" fw={400}>
            · {t(`${item.doc_type}.name`, { defaultValue: item.doc_type })}
          </Text>
        </Text>
        {link && (
          <Anchor component={Link} to={link} size="sm">
            <Group gap={4} component="span">
              {t('shared.document.open')}
              <IconExternalLink {...icon.text} aria-hidden />
            </Group>
          </Anchor>
        )}
      </Group>
      <RecordPreview docType={item.doc_type} id={item.doc_id} />
      <Divider />
      <Stack gap="xs">
        <Text fw={600}>{t('shared.document.tab.approval')}</Text>
        <ApprovalPanel docType={item.doc_type} id={item.doc_id} onDone={onDone} sticky />
      </Stack>
    </Stack>
  )
}
