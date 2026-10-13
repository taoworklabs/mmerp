import { Alert, Button, Menu, Stack, Text } from '@mantine/core'
import { IconAlertCircle, IconBuildingSkyscraper, IconPencil, IconPlus, IconTrash } from '@tabler/icons-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Holiday, type WorkWeek } from '@/shared/api/hrm'
import { useMe } from '@/shared/auth/me'
import { currentYear, errorText, formatDate } from '@/shared/i18n'
import { useConfirm } from '@/shared/ui/confirm'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { CheckboxField, DateField, FilterChip, Form, FormActions, TextField, useOrgUnits, YearFilter } from '@/shared/ui/form'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, ListPage } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { Section } from '@/shared/ui/Section'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import { hrmKeys } from '../keys'

const defaults = { filters: { legal_entity: '', year: '' }, sort: '', sorts: [], pageSize: 50 }
// Monday first, as the week is shown everywhere else.
const weekdays = [1, 2, 3, 4, 5, 6, 0]

// WorkCalendarPage: a legal entity's weekly days off, by the date each version starts, and its holidays of a year.
export function WorkCalendarPage() {
  const { t } = useTranslation()
  const me = useMe()
  const qc = useQueryClient()
  const [confirm, confirmDialog] = useConfirm()
  const [params, set] = useListParams(defaults)
  const [editingWeek, setEditingWeek] = useState<WorkWeek | 'new' | null>(null)
  const [editingHoliday, setEditingHoliday] = useState<Holiday | 'new' | null>(null)
  // A failed delete, shown above the tables until the next attempt.
  const [error, setError] = useState<string | null>(null)
  const units = useOrgUnits({ product: 'hrm', permission: 'hrm.calendar.manage' })
  const entities = (units.data ?? []).filter((u) => u.kind === 'company')
  const entity = Number(params.filters.legal_entity) || entities[0]?.id
  const thisYear = currentYear(me.timezone)
  const year = Number(params.filters.year) || thisYear
  const cal = useQuery({
    queryKey: hrmKeys.calendar(entity ?? 0, year),
    queryFn: () => unwrap(api.GET('/hrm/calendars/{legal_entity}', { params: { path: { legal_entity: entity! }, query: { year } } })),
    enabled: entity !== undefined,
  })
  const manage = cal.data?.allowed_actions.includes('manage') ?? false
  const days = (w: WorkWeek) => weekdays.filter((d) => w.off_days.includes(d)).map((d) => t(`hrm.calendar.day.${d}`)).join(', ') || t('hrm.calendar.no_days_off')

  async function remove(path: () => Promise<unknown>, message: string) {
    if (!(await confirm({ title: t('hrm.calendar.delete_title'), message, confirmLabel: t('hrm.calendar.delete'), danger: true }))) return
    setError(null)
    try {
      await path()
      await qc.invalidateQueries({ queryKey: hrmKeys.calendars() })
      notifySuccess(t('hrm.calendar.saved'))
    } catch (err) {
      setError(errorText(t, err))
    }
  }

  const weekColumns: Column<WorkWeek>[] = [
    { key: 'from', header: t('hrm.calendar.effective_from'), role: 'title', kind: 'date', render: (w) => formatDate(w.effective_from) },
    { key: 'off', header: t('hrm.calendar.off_days'), role: 'meta', render: days },
  ]
  const holidayColumns: Column<Holiday>[] = [
    { key: 'date', header: t('hrm.calendar.date'), role: 'title', kind: 'date', render: (h) => formatDate(h.date) },
    { key: 'name', header: t('hrm.calendar.name'), role: 'meta', render: (h) => h.name },
  ]

  const weekMenu = (w: WorkWeek) => (
    <>
      <Menu.Item leftSection={<IconPencil {...icon.text} />} onClick={() => setEditingWeek(w)}>
        {t('hrm.calendar.edit')}
      </Menu.Item>
      <Menu.Item
        color="danger"
        leftSection={<IconTrash {...icon.text} />}
        onClick={() =>
          void remove(
            () => unwrap(api.DELETE('/hrm/calendars/{legal_entity}/weeks/{effective_from}', { params: { path: { legal_entity: entity!, effective_from: w.effective_from } } })),
            t('hrm.calendar.delete_week', { date: formatDate(w.effective_from) }),
          )
        }
      >
        {t('hrm.calendar.delete')}
      </Menu.Item>
    </>
  )
  const holidayMenu = (h: Holiday) => (
    <>
      <Menu.Item leftSection={<IconPencil {...icon.text} />} onClick={() => setEditingHoliday(h)}>
        {t('hrm.calendar.edit')}
      </Menu.Item>
      <Menu.Item
        color="danger"
        leftSection={<IconTrash {...icon.text} />}
        onClick={() =>
          void remove(
            () => unwrap(api.DELETE('/hrm/calendars/{legal_entity}/holidays/{date}', { params: { path: { legal_entity: entity!, date: h.date } } })),
            t('hrm.calendar.delete_holiday', { name: h.name }),
          )
        }
      >
        {t('hrm.calendar.delete')}
      </Menu.Item>
    </>
  )

  let body
  if (units.isPending || (entity !== undefined && cal.isPending)) body = <ContentSkeleton />
  else if (units.isError || cal.isError) body = <ErrorState message={errorText(t, units.error ?? cal.error)} onRetry={() => void cal.refetch()} />
  else if (entity === undefined || !cal.data) body = <EmptyState title={t('hrm.calendar.no_entities')} />
  else
    body = (
      <Stack gap="lg">
        <Section
          title={t('hrm.calendar.weeks')}
          actions={
            manage && (
              <Button variant="default" leftSection={<IconPlus {...icon.button} />} onClick={() => setEditingWeek('new')}>
                {t('hrm.calendar.add_week')}
              </Button>
            )
          }
        >
          {cal.data.work_weeks.length === 0 ? (
            <EmptyState title={t('hrm.calendar.default_week')} />
          ) : (
            <DataTable
              label={t('hrm.calendar.weeks')}
              columns={weekColumns}
              rows={cal.data.work_weeks}
              rowKey={(w) => w.effective_from}
              onRowClick={manage ? (w) => setEditingWeek(w) : undefined}
              menu={manage ? weekMenu : undefined}
            />
          )}
        </Section>
        <Section title={t('hrm.calendar.holidays', { year })}>
          {cal.data.holidays.length === 0 ? (
            <EmptyState title={t('hrm.calendar.no_holidays')} />
          ) : (
            <DataTable
              label={t('hrm.calendar.holidays', { year })}
              columns={holidayColumns}
              rows={cal.data.holidays}
              rowKey={(h) => h.date}
              onRowClick={manage ? (h) => setEditingHoliday(h) : undefined}
              menu={manage ? holidayMenu : undefined}
            />
          )}
        </Section>
      </Stack>
    )

  return (
    <ListPage
      title={t('hrm.calendar.title')}
      description={t('hrm.calendar.description')}
      action={
        manage && (
          <Button leftSection={<IconPlus {...icon.button} />} onClick={() => setEditingHoliday('new')}>
            {t('hrm.calendar.add_holiday')}
          </Button>
        )
      }
      filters={
        <>
          <FilterChip
            label={t('hrm.calendar.legal_entity')}
            icon={IconBuildingSkyscraper}
            data={entities.map((u) => ({ value: String(u.id), label: u.name }))}
            value={entity ? String(entity) : null}
            onChange={(v) => set({ filters: { legal_entity: v ?? '' } })}
            clearable={false}
          />
          <YearFilter label={t('hrm.calendar.year')} value={year} years={[thisYear - 1, thisYear, thisYear + 1]} onChange={(y) => set({ filters: { year: String(y) } })} />
        </>
      }
    >
      {error && (
        <Alert color="danger" icon={<IconAlertCircle {...icon.button} />} role="alert">
          {error}
        </Alert>
      )}
      {body}
      {entity !== undefined && editingWeek && <WeekModal entity={entity} week={editingWeek === 'new' ? null : editingWeek} onClose={() => setEditingWeek(null)} />}
      {entity !== undefined && editingHoliday && (
        <HolidayModal entity={entity} holiday={editingHoliday === 'new' ? null : editingHoliday} onClose={() => setEditingHoliday(null)} />
      )}
      {confirmDialog}
    </ListPage>
  )
}

type WeekForm = { effective_from: string | null } & Record<`d${number}`, boolean>

function WeekModal({ entity, week, onClose }: { entity: number; week: WorkWeek | null; onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const off = week?.off_days ?? [0, 6]
  const form = useForm<WeekForm>({
    mode: 'onBlur',
    defaultValues: { effective_from: week?.effective_from ?? null, ...Object.fromEntries(weekdays.map((d) => [`d${d}`, off.includes(d)])) },
  })
  async function save(v: WeekForm) {
    const days = weekdays.filter((d) => v[`d${d}`])
    await unwrap(
      api.PUT('/hrm/calendars/{legal_entity}/weeks/{effective_from}', {
        params: { path: { legal_entity: entity, effective_from: v.effective_from ?? '' } },
        body: { off_days: days },
      }),
    )
    await qc.invalidateQueries({ queryKey: hrmKeys.calendars() })
    notifySuccess(t('hrm.calendar.saved'))
    onClose()
  }
  return (
    <FormModal opened title={t(week ? 'hrm.calendar.edit_week' : 'hrm.calendar.add_week')} onClose={onClose}>
      <Form form={form} onSubmit={save}>
        <DateField name="effective_from" label={t('hrm.calendar.effective_from')} description={t('hrm.calendar.effective_from_hint')} readOnly={!!week} required />
        <Text size="sm" fw={500} c="dimmed">
          {t('hrm.calendar.off_days')}
        </Text>
        {weekdays.map((d) => (
          <CheckboxField key={d} name={`d${d}`} label={t(`hrm.calendar.day.${d}`)} />
        ))}
        <FormActions submitLabel={t('hrm.common.save')} onCancel={onClose} cancelLabel={t('hrm.common.cancel')} />
      </Form>
    </FormModal>
  )
}

function HolidayModal({ entity, holiday, onClose }: { entity: number; holiday: Holiday | null; onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const form = useForm<{ date: string | null; name: string }>({ mode: 'onBlur', defaultValues: { date: holiday?.date ?? null, name: holiday?.name ?? '' } })
  async function save(v: { date: string | null; name: string }) {
    await unwrap(
      api.PUT('/hrm/calendars/{legal_entity}/holidays/{date}', { params: { path: { legal_entity: entity, date: v.date ?? '' } }, body: { name: v.name.trim() } }),
    )
    await qc.invalidateQueries({ queryKey: hrmKeys.calendars() })
    notifySuccess(t('hrm.calendar.saved'))
    onClose()
  }
  return (
    <FormModal opened title={t(holiday ? 'hrm.calendar.edit_holiday' : 'hrm.calendar.add_holiday')} onClose={onClose}>
      <Form form={form} onSubmit={save}>
        <DateField name="date" label={t('hrm.calendar.date')} description={t('hrm.calendar.date_hint')} readOnly={!!holiday} required />
        <TextField name="name" label={t('hrm.calendar.name')} required maxLength={200} />
        <FormActions submitLabel={t('hrm.common.save')} onCancel={onClose} cancelLabel={t('hrm.common.cancel')} />
      </Form>
    </FormModal>
  )
}
