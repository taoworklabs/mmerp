import { KeyFacts, Stack, Text } from '@mmerp/ui'
import { useQueries } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router'
import { api, unwrap } from '@/shared/api/hrm'
import { useMe } from '@/shared/auth/me'
import { errorText, formatDate, formatMonth, formatNumber, formatWeekday, today } from '@/shared/i18n'
import { DayGrid, PeriodNav, type DayCell } from '@/shared/ui/DayGrid'
import { Page } from '@/shared/ui/page'
import { ContentSkeleton, ErrorState } from '@/shared/ui/states'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { hrmKeys } from '../keys'
import { monthGrid, monthsOf, shiftDate, shiftMonth, weekOf } from './days'

type Mode = 'month' | 'week'
type Day = NonNullable<Awaited<ReturnType<typeof fetchMonth>>>['days'][number]

const fetchMonth = (month: string) => unwrap(api.GET('/hrm/me/days', { params: { query: { month } } }))
const isDate = (s: string | null): s is string => !!s && /^\d{4}-\d{2}-\d{2}$/.test(s)
const sum = (xs: (string | null | undefined)[]) => xs.reduce((a, x) => a + Number(x ?? 0), 0)

// DaysPage: the user's own days, by month or by week, with what the timesheet, posted leave and
// overtime say about each. The month and the mode are on the URL.
export function DaysPage() {
  const { t } = useTranslation()
  const me = useMe()
  const [params, setParams] = useSearchParams()
  const mode: Mode = params.get('mode') === 'week' ? 'week' : 'month'
  const anchor = isDate(params.get('date')) ? params.get('date')! : today(me.timezone)
  const set = (next: { mode?: Mode; date?: string }) =>
    setParams({ mode: next.mode ?? mode, date: next.date ?? anchor })

  const month = anchor.slice(0, 7)
  const week = weekOf(anchor)
  const months = mode === 'month' ? [month] : monthsOf(week)
  const results = useQueries({ queries: months.map((m) => ({ queryKey: hrmKeys.me.days(m), queryFn: () => fetchMonth(m) })) })

  const byDate = new Map<string, Day>()
  for (const r of results) for (const d of r.data?.days ?? []) byDate.set(d.date, d)
  const failed = results.find((r) => r.isError)
  const loading = results.some((r) => r.isPending)

  const step = (n: number) => set({ date: mode === 'month' ? `${shiftMonth(month, n)}-01` : shiftDate(anchor, 7 * n) })
  const title = mode === 'month' ? formatMonth(month) : `${formatDate(week[0])} – ${formatDate(week[6])}`
  const dates = mode === 'month' ? monthGrid(month) : week
  const weekdays = week.map((d) => formatWeekday(d))

  const cell = (date: string | null): DayCell | null => {
    if (!date) return null
    return dayCell(date, byDate.get(date), t, formatWeekday(date))
  }

  // The summary counts the month only: a week may reach into months not shown.
  const data = results[0]?.data
  const summary =
    mode === 'month' && data ? (
      <KeyFacts
        facts={[
          [t('hrm.me.days.sum_worked'), formatNumber(sum(data.days.map((d) => d.worked)))],
          [t('hrm.me.days.sum_leave'), formatNumber(sum(data.days.map((d) => (d.leave?.paid ? d.leave.days : null))))],
          [t('hrm.me.days.sum_overtime'), formatNumber(sum(data.days.map((d) => d.overtime_hours)))],
          [t('hrm.me.days.sum_standard'), formatNumber(data.standard_days)],
        ]}
      />
    ) : undefined

  return (
    <Page title={t('hrm.me.days.title')} description={t('hrm.me.days.description')} summary={summary}>
      <Stack gap="md">
        <PeriodNav
          modes={[
            { value: 'month', label: t('hrm.me.days.month') },
            { value: 'week', label: t('hrm.me.days.week') },
          ]}
          mode={mode}
          onMode={(m) => set({ mode: m })}
          modeLabel={t('hrm.me.days.mode')}
          title={title}
          onPrev={() => step(-1)}
          onNext={() => step(1)}
          prevLabel={t(`hrm.me.days.prev_${mode}`)}
          nextLabel={t(`hrm.me.days.next_${mode}`)}
        />
        {failed ? (
          <ErrorState message={errorText(t, failed.error)} onRetry={() => results.forEach((r) => void r.refetch())} />
        ) : loading ? (
          <ContentSkeleton />
        ) : (
          <DayGrid label={t('hrm.me.days.calendar')} weekdays={weekdays} cells={dates.map(cell)} />
        )}
      </Stack>
    </Page>
  )
}

function dayCell(date: string, d: Day | undefined, t: (k: string, o?: Record<string, unknown>) => string, weekday: string): DayCell {
  const tone = d?.kind === 'holiday' || d?.kind === 'rest' ? d.kind : undefined
  return {
    key: date,
    day: String(Number(date.slice(8))),
    weekday,
    tone,
    children: d && (
      <>
        {d.kind === 'holiday' && <Text size="xs">{t('hrm.me.days.holiday')}</Text>}
        {d.kind === 'rest' && <Text size="xs" tone="dimmed">{t('hrm.me.days.rest')}</Text>}
        {d.worked != null && <Text size="xs">{t('hrm.me.days.worked', { n: formatNumber(Number(d.worked)) })}</Text>}
        {d.worked != null && !d.worked_closed && <StatusBadge tone="warning">{t('hrm.me.days.not_closed')}</StatusBadge>}
        {d.leave && (
          <Text size="xs">
            {t('hrm.me.days.leave', { type: d.leave.type_name, n: formatNumber(Number(d.leave.days)) })}
            {d.leave.paid ? '' : ` (${t('hrm.me.days.unpaid')})`}
          </Text>
        )}
        {d.overtime_hours != null && <Text size="xs">{t('hrm.me.days.overtime', { n: formatNumber(Number(d.overtime_hours)) })}</Text>}
      </>
    ),
  }
}
