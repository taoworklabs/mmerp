import { DateFilterChip } from '@mmerp/ui'
import { IconCalendar } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import { calendarLocale, formatDate } from '@/shared/i18n'
import { icon } from '../theme'

// DateFilter is a filter chip that picks a day (YYYY-MM-DD), beside the FilterChips of a filter bar.
export function DateFilter({
  label,
  value,
  onChange,
  clearable,
}: {
  label: string
  value: string | null
  onChange: (date: string | null) => void
  clearable?: boolean
}) {
  const { t } = useTranslation()
  return (
    <DateFilterChip
      label={label}
      icon={<IconCalendar {...icon.button} />}
      value={value}
      onChange={onChange}
      clearable={clearable}
      valueText={(name, date) => t('shared.list.filter_value', { name, value: formatDate(date) })}
      clearLabel={t('shared.list.clear_filter', { name: label })}
      locale={calendarLocale()}
      labels={{ previousMonth: t('shared.form.previous_month'), nextMonth: t('shared.form.next_month') }}
    />
  )
}
