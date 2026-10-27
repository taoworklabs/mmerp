import { Text, ThemeIcon, type ThemeIconProps, Tooltip } from '@mmerp/ui'
import {
  IconArrowBackUp,
  IconArrowRight,
  IconBan,
  IconCheck,
  IconCircleCheck,
  IconDownload,
  IconEye,
  IconMessage,
  IconPaperclip,
  IconPencil,
  IconPlus,
  IconSend,
  IconShieldLock,
  IconTrash,
  IconUserShare,
  IconX,
  type Icon as TablerIcon,
} from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { Fragment, type ReactNode } from 'react'
import type { TFunction } from 'i18next'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type HistoryEntry } from '@/shared/api/core'
import { useMe } from '@/shared/auth/me'
import { formatAgo, formatDate, formatDateTime } from '@/shared/i18n'
import { PersonAvatar } from '@/shared/ui/avatar'
import { icon } from '@/shared/ui/theme'
import classes from './DocumentHistory.module.css'
import { DocumentStatus, type DocStatus } from './DocumentStatus'
import { documentKeys } from './keys'

type Change = { old: unknown; new: unknown; sensitive?: boolean }

// Run is one line of history: an entry and the repeats folded into it, newest first.
export type Run = { entry: HistoryEntry; count: number; entries: HistoryEntry[] }

// isView: the event only read the record. isCreated: it is the record's first event.
export const isView = (e: HistoryEntry) => e.action.endsWith('viewed')
const isCreated = (e: HistoryEntry) => e.action === 'record.created' || e.action.endsWith('_created')

// foldWithin is how far apart two repeats may be and still read as one line.
const foldWithin = 30 * 60_000

// runs folds entries (newest first) of the same person doing the same thing within half an hour
// of each other into one line with a count, e.g. viewing four sensitive fields in a row. Entries
// that change fields always stand alone: each change is worth reading. Reads fold even when they
// read different things; the line then names them.
export function runs(all: HistoryEntry[]): Run[] {
  const out: Run[] = []
  const entries = all.filter((e, i) => !createdTwin(e, all[i - 1]) && !createdTwin(e, all[i + 1]))
  for (const e of entries) {
    const last = out[out.length - 1]
    const prev = last?.entries[last.entries.length - 1]
    const same =
      last &&
      prev &&
      !e.data?.changes &&
      !last.entry.data?.changes &&
      last.entry.action === e.action &&
      last.entry.actor_name === e.actor_name &&
      // Reads fold whatever they read; other repeats only when they did the very same thing.
      (isView(e) || JSON.stringify(last.entry.data) === JSON.stringify(e.data)) &&
      Math.abs(new Date(prev.at).getTime() - new Date(e.at).getTime()) <= foldWithin
    if (same) {
      last.count++
      last.entries.push(e)
    } else out.push({ entry: e, count: 1, entries: [e] })
  }
  return out
}

// createdTwin: e is the record's generic "created" written together with the module's own
// ("created request"), which says the same thing better, so the line shows once.
function createdTwin(e: HistoryEntry, other: HistoryEntry | undefined) {
  return (
    e.action === 'record.created' &&
    !!other &&
    other.action !== 'record.created' &&
    isCreated(other) &&
    other.actor_name === e.actor_name &&
    Math.abs(new Date(other.at).getTime() - new Date(e.at).getTime()) <= 60_000
  )
}

export type Look = { color: ThemeIconProps['color']; icon: TablerIcon; filled?: boolean }

// A status reached has its status colour; the rest by what was done, grey when it changed nothing.
// Creation is the milestone the rest follows, so it alone is filled.
const statusLook: Record<DocStatus, Look> = {
  draft: { color: 'gray', icon: IconArrowBackUp },
  pending_approval: { color: 'warning', icon: IconSend },
  posted: { color: 'success', icon: IconCircleCheck },
  cancelled: { color: 'danger', icon: IconBan },
}
const actionLook: Record<string, Look> = {
  'record.deleted': { color: 'danger', icon: IconTrash },
  'approval.approved': { color: 'success', icon: IconCheck },
  'approval.rejected': { color: 'danger', icon: IconX },
  'approval.reassigned': { color: 'info', icon: IconUserShare },
  'attachment.added': { color: 'gray', icon: IconPaperclip },
  'attachment.removed': { color: 'gray', icon: IconPaperclip },
  'discussion.commented': { color: 'info', icon: IconMessage },
}

function look(e: HistoryEntry): Look {
  if (e.action === 'record.transitioned') return statusLook[e.data?.to as DocStatus] ?? statusLook.draft
  if (isCreated(e)) return { color: 'primary', icon: IconPlus, filled: true }
  // Reading sensitive data is what an audit looks for: it stands out in the warning colour.
  if (e.action.endsWith('sensitive_viewed')) return { color: 'warning', icon: IconShieldLock }
  if (isView(e)) return { color: 'gray', icon: IconEye }
  if (e.action.endsWith('exported')) return { color: 'gray', icon: IconDownload }
  return actionLook[e.action] ?? { color: 'gray', icon: IconPencil }
}

// useHistory is a record's audit log, oldest first as the API sends it.
export function useHistory(docType: string, id: number) {
  return useQuery({
    queryKey: documentKeys.history(docType, id),
    queryFn: () => unwrap(api.GET('/documents/{type}/{id}/history', { params: { path: { type: docType, id } } })),
  })
}

// Parts is what any timeline shows of one run: what it did (and the status it reached, or the
// fields it read), how it looks, the fields it changed and how many times it happened.
export type Parts = { verb: string; status?: DocStatus; read: string[]; look: Look; changes: [string, Change][]; count: number }

export function describeRun(t: TFunction, docType: string, { entry: e, count, entries }: Run): Parts {
  const d = e.data ?? {}
  // The fields read across the run, once each: "Xem thông tin nhạy cảm: Số CCCD, Tài khoản ngân hàng".
  const read = [...new Set(entries.map((x) => x.data?.field).filter((f): f is string => typeof f === 'string'))].map((f) => fieldLabel(t, docType, f))
  const transitioned = e.action === 'record.transitioned'
  return {
    verb: transitioned ? t('shared.history.transitioned') : t(`shared.history.${e.action}`, { ...d, defaultValue: t(`${e.action}`, { ...d, defaultValue: e.action }) }),
    status: transitioned ? (d.to as DocStatus) : undefined,
    read,
    look: look(e),
    // A new record's first values are on its page; listing them all here repeats the form.
    changes: isCreated(e) ? [] : Object.entries((d.changes ?? {}) as Record<string, Change>),
    count,
  }
}

const fieldLabel = (t: TFunction, docType: string, field: string) => t(`${docType}.field.${field}`, { defaultValue: field })

// ChangeValue is one changed field's old value struck out, then the new one, inline.
export function ChangeValue({ docType, field, change: c }: { docType: string; field: string; change: Change }) {
  const { t } = useTranslation()
  const empty = (v: unknown) => v === null || v === undefined || v === ''
  const value = (v: unknown) => {
    if (c.sensitive) return '••••••'
    if (empty(v)) return '—'
    if (typeof v === 'string' && /^\d{4}-\d{2}-\d{2}$/.test(v)) return formatDate(v)
    // A coded value (male, weekday) reads in words when the area names it.
    if (typeof v === 'string' && /^[a-z_]+$/.test(v)) return t(`${docType}.${field}.${v}`, { defaultValue: v })
    return String(v)
  }
  // A reference is stored as an id, which means nothing to read: say only that it changed.
  if (field.endsWith('_id')) return <>{t('shared.history.changed')}</>
  return (
    <>
      {/* A first value has nothing to strike out. */}
      {!empty(c.old) && (
        <>
          <Text render={<span />} size="inherit" tone="dimmed" className={classes.old}>
            {value(c.old)}
          </Text>
          <IconArrowRight {...icon.text} className={classes.arrow} aria-label={t('shared.history.to')} />
        </>
      )}
      <Text render={<span />} size="inherit" weight={500}>
        {value(c.new)}
      </Text>
    </>
  )
}

// ChangeList lists the fields a run changed, a field a row.
export function ChangeList({ docType, changes }: { docType: string; changes: [string, Change][] }) {
  const { t } = useTranslation()
  return (
    <dl className={classes.changes}>
      {changes.map(([field, c]) => (
        <Fragment key={field}>
          <Text render={<dt />} size="xs" tone="dimmed">
            {fieldLabel(t, docType, field)}
          </Text>
          <Text render={<dd />} size="xs">
            <ChangeValue docType={docType} field={field} change={c} />
          </Text>
        </Fragment>
      ))}
    </dl>
  )
}

// FeedRow is one line of a timeline: the time of day (how long ago on hover; the day heads the
// group), a bullet, one sentence, and under it any detail, joined to the bullet by a thread.
export function FeedRow({ at, bullet, detail, children }: { at: string; bullet: ReactNode; detail?: ReactNode; children: ReactNode }) {
  const me = useMe()
  return (
    <li className={classes.row} data-detail={detail ? true : undefined}>
      <Tooltip label={formatAgo(at, me.timezone)}>
        <Text render={<time dateTime={at} />} size="xs" tone="dimmed" className={classes.time}>
          {formatDateTime(at, me.timezone).slice(11)}
        </Text>
      </Tooltip>
      <span className={classes.bullet}>{bullet}</span>
      <div className={classes.body}>
        <Text size="sm">{children}</Text>
        {detail && <div className={classes.detail}>{detail}</div>}
      </div>
    </li>
  )
}

// Who did it and what they did it to lead the sentence in the text colour; the doing steps back.
export function Who({ name }: { name: string }) {
  return (
    <Text render={<span />} size="inherit" weight={600}>
      {name}
    </Text>
  )
}

export function Did({ children }: { children: ReactNode }) {
  return (
    <Text render={<span />} size="inherit" tone="dimmed">
      {children}
    </Text>
  )
}

export function Bullet({ look }: { look: Look }) {
  return (
    <ThemeIcon variant={look.filled ? 'filled' : 'light'} color={look.color} size={20} circle aria-hidden>
      <look.icon size={12} stroke={1.5} />
    </ThemeIcon>
  )
}

// shortcut: lower-cases the first letter of the action's label to set it mid-sentence; a label
// that starts with a name or an acronym would need its own sentence key.
const midSentence = (s: string) => s.charAt(0).toLocaleLowerCase() + s.slice(1)

// HistoryRow is one run of the audit log as a sentence: who, what they did, what to. A person's
// plain doings carry their avatar; a status reached, a decision, a creation or a sensitive read
// keeps its coloured icon, since the colour says what happened. One changed field reads in the
// sentence; more go in a list under it. Sensitive values stay masked.
export function HistoryRow({ docType, run }: { docType: string; run: Run }) {
  const { t } = useTranslation()
  const p = describeRun(t, docType, run)
  const name = run.entry.actor_name
  const [one] = p.changes
  return (
    <FeedRow
      at={run.entry.at}
      bullet={name && p.look.color === 'gray' ? <PersonAvatar name={name} size={20} /> : <Bullet look={p.look} />}
      detail={p.changes.length > 1 && <ChangeList docType={docType} changes={p.changes} />}
    >
      {/* A system job acts without a person. */}
      <Who name={name ?? t('shared.history.system')} /> <Did>{midSentence(p.verb)}</Did>
      {p.status && (
        <>
          {' '}
          <DocumentStatus docType={docType} status={p.status} />
        </>
      )}
      {p.read.length > 0 && <Who name={` ${p.read.join(', ')}`} />}
      {p.changes.length === 1 && one && (
        <>
          <Did>{`: ${fieldLabel(t, docType, one[0])} `}</Did>
          <ChangeValue docType={docType} field={one[0]} change={one[1]} />
        </>
      )}
      {/* Named reads count themselves. */}
      {p.count > 1 && p.read.length === 0 && <Did>{` · ${t('shared.history.times', { count: p.count })}`}</Did>}
    </FeedRow>
  )
}
