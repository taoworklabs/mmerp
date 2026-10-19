import { Anchor, Button, SegmentedControl, Stack, Text, UnstyledButton } from '@mantine/core'
import { IconChevronDown, IconEye } from '@tabler/icons-react'
import { Fragment, useId, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import type { HistoryEntry } from '@/shared/api/core'
import { useMe } from '@/shared/auth/me'
import { errorText, formatDateTime } from '@/shared/i18n'
import { PersonAvatar } from '@/shared/ui/avatar'
import { ContentSkeleton, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import classes from './ActivityFeed.module.css'
import { CommentBox, CommentText, useDiscussion } from './DiscussionPanel'
import { Bullet, Did, FeedRow, HistoryRow, isView, runs, useHistory, Who } from './DocumentHistory'
import history from './DocumentHistory.module.css'

type Filter = 'all' | 'comments' | 'changes' | 'views'
type Item = { key: string; at: string; kind: Exclude<Filter, 'all'>; quiet?: boolean; node: ReactNode }

// page is how many events show before "Show more".
const page = 30

// feedItems is what the feed lists: those of the filter, or with recent the latest that many
// comments and changes, reads left out (they are the history's to show).
export function feedItems<T extends { kind: Exclude<Filter, 'all'> }>(items: T[], filter: Filter, recent?: number): T[] {
  if (recent) return items.filter((i) => i.kind !== 'views').slice(0, recent)
  return items.filter((i) => filter === 'all' || i.kind === filter)
}

// ActivityFeed is everything that happened to a record in one timeline, newest first: comments
// in their wells among the changes and reads of the audit log, so each reads in its context.
// The box to comment heads it; a filter narrows it to comments, changes or reads; days head
// their events, indented under them, and open or close them. With recent it is a digest instead: no filter, the latest comments and changes only.
export function ActivityFeed({ docType, id, recent }: { docType: string; id: number; recent?: number }) {
  const { t } = useTranslation()
  const me = useMe()
  const log = useHistory(docType, id)
  const talk = useDiscussion(docType, id)
  const [filter, setFilter] = useState<Filter>('all')
  const [shown, setShown] = useState(page)
  // Days the reader folded away; each day heading opens and closes its own events.
  const [closed, setClosed] = useState<ReadonlySet<string>>(new Set())
  const ids = useId()
  if (log.isPending || talk.isPending) return <ContentSkeleton />
  if (log.isError) return <ErrorState message={errorText(t, log.error)} onRetry={() => void log.refetch()} />
  if (talk.isError) return <ErrorState message={errorText(t, talk.error)} onRetry={() => void talk.refetch()} />

  // The log also notes each comment; the comment itself, with its text, stands for it.
  const audit = [...(log.data ?? [])].reverse().filter((e: HistoryEntry) => e.action !== 'discussion.commented')
  const items: Item[] = [
    ...runs(audit).map((r) => ({
      key: `h${r.entry.id}`,
      at: r.entry.at,
      kind: isView(r.entry) ? ('views' as const) : ('changes' as const),
      // Reads of sensitive data stay in view however many: an audit looks for them.
      quiet: isView(r.entry) && !r.entry.action.endsWith('sensitive_viewed'),
      node: <HistoryRow docType={docType} run={r} />,
    })),
    ...talk.data.items.map((c) => ({
      key: `c${c.id}`,
      at: c.created_at,
      kind: 'comments' as const,
      node: (
        <FeedRow at={c.created_at} bullet={<PersonAvatar name={c.author_name} size={20} />} detail={<CommentText body={c.body} />}>
          <Who name={c.author_name} /> <Did>{t('shared.activity.commented')}</Did>
        </FeedRow>
      ),
    })),
  ].sort((a, b) => b.at.localeCompare(a.at))
  const visible = feedItems(items, filter, recent)

  // The day an event happened in the tenant's time zone, today and yesterday by name.
  const dayOf = (iso: string) => formatDateTime(iso, me.timezone).slice(0, 10)
  const today = dayOf(new Date().toISOString())
  const yesterday = dayOf(new Date(Date.now() - 86_400_000).toISOString())
  const dayLabel = (day: string) => (day === today ? t('shared.activity.today') : day === yesterday ? t('shared.activity.yesterday') : day)
  // Under "All", plain reads in a row fold into one line that opens on demand: what changed
  // leads, who looked stays one click away.
  type Row = Item | { key: string; fold: Item[] }
  const days: { day: string; rows: Row[] }[] = []
  for (const i of visible.slice(0, shown)) {
    const day = dayOf(i.at)
    let last = days[days.length - 1]
    if (last?.day !== day) days.push((last = { day, rows: [] }))
    const prev = last.rows[last.rows.length - 1]
    if (filter === 'all' && i.quiet && prev && 'fold' in prev) prev.fold.push(i)
    else if (filter === 'all' && i.quiet) last.rows.push({ key: `f${i.key}`, fold: [i] })
    else last.rows.push(i)
  }

  return (
    <Stack gap="lg" className={classes.feed}>
      {talk.data.allowed_actions.includes('comment') && <CommentBox docType={docType} id={id} maxLength={talk.data.max_length} />}
      {!recent && (
        <SegmentedControl
          value={filter}
          onChange={(v) => {
            setFilter(v as Filter)
            setShown(page)
          }}
          aria-label={t('shared.activity.filter')}
          data={(['all', 'comments', 'changes', 'views'] as const).map((f) => ({ value: f, label: t(`shared.activity.${f}`) }))}
          w="fit-content"
        />
      )}
      {days.length === 0 ? (
        <Text size="sm" c="dimmed">
          {t('shared.activity.empty')}
        </Text>
      ) : (
        days.map((d) => {
          const open = !closed.has(d.day)
          const id = `${ids}-${d.day}`
          const count = d.rows.reduce((n, r) => n + ('fold' in r ? r.fold.length : 1), 0)
          const toggle = () =>
            setClosed((c) => {
              const next = new Set(c)
              if (open) next.add(d.day)
              else next.delete(d.day)
              return next
            })
          return (
            <Stack key={d.day} gap="md">
              <UnstyledButton className={classes.day} onClick={toggle} aria-expanded={open} aria-controls={id}>
                <IconChevronDown {...icon.text} aria-hidden className={classes.chevron} data-closed={!open || undefined} />
                <Text span size="xs" fw={600} c="dimmed">
                  {dayLabel(d.day)}
                  {!open && ` · ${t('shared.activity.day_count', { count })}`}
                </Text>
              </UnstyledButton>
              {open && (
                <ol id={id} className={`${history.list} ${classes.events}`}>
                  {d.rows.map((r) => ('fold' in r && r.fold.length > 1 ? <Fold key={r.key} items={r.fold} /> : <Fragment key={r.key}>{'fold' in r ? r.fold[0]!.node : r.node}</Fragment>))}
                </ol>
              )}
            </Stack>
          )
        })
      )}
      {visible.length > shown && (
        <Button variant="default" w="fit-content" onClick={() => setShown(shown + page)}>
          {t('shared.activity.more')}
        </Button>
      )}
    </Stack>
  )
}

// Fold stands for several reads in a row: how many, and the reads themselves once opened.
function Fold({ items }: { items: Item[] }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  if (open)
    return (
      <>
        {items.map((i) => (
          <Fragment key={i.key}>{i.node}</Fragment>
        ))}
      </>
    )
  return (
    <FeedRow at={items[0]!.at} bullet={<Bullet look={{ color: 'gray', icon: IconEye }} />}>
      <Did>{t('shared.activity.folded', { count: items.length })}</Did>{' '}
      <Anchor component="button" type="button" size="sm" onClick={() => setOpen(true)}>
        {t('shared.activity.show')}
      </Anchor>
    </FeedRow>
  )
}
