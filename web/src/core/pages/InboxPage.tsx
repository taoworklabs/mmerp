import { Anchor, Button, Divider, Group, Stack, Text, Tooltip } from '@mmerp/ui'
import { IconExternalLink, IconUserShare } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useLocation, useNavigate, useSearchParams } from 'react-router'
import { api, unwrap, type InboxItem } from '@/shared/api/core'
import { RecordPreview, useRecordLink } from '@/shared/area'
import { useMe } from '@/shared/auth/me'
import { actionLabel, ApprovalPanel, documentKeys } from '@/shared/document'
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
  // The chosen submission stays on screen after it leaves the list. Someone else handling it,
  // or the submitter withdrawing it, drops it from the inbox, and the panel is where the user
  // reads why their decision was refused; dropping the panel with it explains nothing.
  const held = useRef<InboxItem | null>(null)
  const found = inbox.data?.items.find((i) => i.instance_id === selected) ?? null
  if (found) held.current = found
  const item = found ?? (held.current?.instance_id === selected ? held.current : null)
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
              <Group justify="space-between" wrap={false}>
                <Text weight={500} size="sm">
                  {i.number}
                </Text>
                <Tooltip label={formatDateTime(i.submitted_at, me.timezone)}>
                  <Text size="xs" tone="dimmed">
                    {formatAgo(i.submitted_at, me.timezone)}
                  </Text>
                </Tooltip>
              </Group>
            }
            description={[
              t(`${i.doc_type}.name`, { defaultValue: i.doc_type }),
              ...(i.action === 'post' ? [] : [actionLabel(t, i.doc_type, i.action)]),
              i.submitted_by_name,
              ...(i.on_behalf_of_name ? [t('core.inbox.on_behalf', { name: i.on_behalf_of_name })] : []),
            ].join(' · ')}
          />
        ))}
      </InboxList>
    )

  return (
    <Template
      title={t('core.inbox.title')}
      description={t('core.inbox.description')}
      actions={
        // Relative, so inside an area the page keeps that area's menu.
        <Button render={<Link to="../delegations" relative="path" />} variant="default" leftSection={<IconUserShare {...icon.button} />}>
          {t('core.delegations.title')}
        </Button>
      }
      list={list}
      detail={item ? <Detail item={item} onDone={advance} /> : null}
      onBack={back}
    />
  )
}

function Detail({ item, onDone }: { item: InboxItem; onDone: () => void }) {
  const { t } = useTranslation()
  const link = useRecordLink(item.doc_type, item.doc_id)
  return (
    <Stack gap="md">
      <Group justify="space-between">
        <Text weight={600}>
          {item.number}{' '}
          <Text render={<span />} tone="dimmed" weight={400}>
            · {t(`${item.doc_type}.name`, { defaultValue: item.doc_type })}
          </Text>
        </Text>
        {link && (
          <Anchor render={<Link to={link} />} size="sm">
            <Group gap="xxs" render={<span />}>
              {t('shared.document.open')}
              <IconExternalLink {...icon.text} aria-hidden />
            </Group>
          </Anchor>
        )}
      </Group>
      <RecordPreview docType={item.doc_type} id={item.doc_id} />
      <Divider />
      <Stack gap="xs">
        <Text weight={600}>{t('shared.document.tab.approval')}</Text>
        <ApprovalPanel docType={item.doc_type} id={item.doc_id} onDone={onDone} sticky />
      </Stack>
    </Stack>
  )
}
