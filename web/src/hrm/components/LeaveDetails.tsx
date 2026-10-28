import { FieldList, Group, Panel, Stack, Text } from '@mmerp/ui'
import { IconCalendarEvent, IconUsers } from '@tabler/icons-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import type { Leave, LeaveListItem } from '@/shared/api/hrm'
import { DocumentStatus } from '@/shared/document'
import { formatDate, formatDecimal } from '@/shared/i18n'
import { MissingValue } from '@/shared/ui/MissingValue'
import { icon } from '@/shared/ui/theme'
import { EditablePanel } from '@/shared/ui/form'
import { leaveDocType } from '../keys'
import { EmployeeLink } from './EmployeeLink'
import { LeaveForm } from './LeaveForm'

// halves turns a day count in half-day steps ("2.5") into whole halves, so sums stay exact.
const halves = (s: string) => Math.round(Number(s) * 2)

// LeaveDetails is a leave request's values as label–value pairs; a draft the user may edit opens
// its form in the same card. The balance is of the request's own kind: what it deducts from, or
// that it deducts nothing.
export function LeaveDetails({ leave: l, deducts }: { leave: Leave; deducts: boolean | undefined }) {
  const { t } = useTranslation()
  const period = l.start_date === l.end_date ? formatDate(l.start_date) : `${formatDate(l.start_date)} – ${formatDate(l.end_date)}`
  let balance: string | undefined
  if (deducts === false) balance = t('hrm.leave.no_deduct')
  else if (deducts && l.balance !== null && l.balance !== undefined) {
    const left = formatDecimal(l.balance)
    // Once approved the days are already off the balance; until then, show what approving leaves.
    const waiting = l.status === 'draft' || l.status === 'pending_approval'
    // The API always returns days on a request; the field is optional only as an input.
    const after = formatDecimal(String((halves(l.balance) - halves(l.days ?? '0')) / 2))
    balance = waiting ? t('hrm.leave.balance_after', { left, after }) : t('hrm.leave.balance_left', { left })
  }
  const canEdit = l.allowed_actions.includes('edit')
  return (
    // Keyed by version: a save or a reload after a conflict starts from the saved data, closed.
    <EditablePanel key={l.version} title={t('hrm.leave.section')} icon={IconCalendarEvent} editable={canEdit} edit={(close) => <LeaveForm leave={l} onClose={close} />}>
      {(open) => (
        <FieldList
          stacked
          rows={[
            [t('hrm.leave.type'), l.leave_type_name],
            [t('hrm.leave.period'), period],
            [t('hrm.leave.days'), t('hrm.leave.days_value', { days: formatDecimal(l.days) })],
            ...(balance ? [[t('hrm.leave.balance'), balance] as [string, ReactNode]] : []),
            // A draft offers to fill an empty value in; a decided request just has none.
            [t('hrm.leave.reason'), l.reason || (canEdit ? <MissingValue label={t('hrm.leave.reason')} onAdd={open} /> : '—'), 'wide'],
          ]}
        />
      )}
    </EditablePanel>
  )
}

// LeaveImpact is who else is away from the department over the same days, and who takes the work over.
export function LeaveImpact({ overlapping }: { overlapping: LeaveListItem[] | undefined }) {
  const { t } = useTranslation()
  return (
    <Panel title={t('hrm.leave.impact')} icon={<IconUsers {...icon.text} aria-hidden />}>
      <FieldList
        stacked
        rows={[
          // A list of people and dates: the whole width of the card.
          [
            t('hrm.leave.overlapping'),
            overlapping === undefined ? (
              '—'
            ) : overlapping.length === 0 ? (
              t('hrm.leave.overlapping_none')
            ) : (
              <Stack gap="xxs">
                {overlapping.map((x) => (
                  <Group key={x.id} gap="xs" wrap={false}>
                    <EmployeeLink id={x.employee_id}>{x.employee_name}</EmployeeLink>
                    <Text size="sm" tone="dimmed">
                      {`${formatDate(x.start_date)} – ${formatDate(x.end_date)}`}
                    </Text>
                    <DocumentStatus docType={leaveDocType} status={x.status} />
                  </Group>
                ))}
              </Stack>
            ),
            'wide',
          ],
          [t('hrm.leave.handover'), '—'],
        ]}
      />
    </Panel>
  )
}
