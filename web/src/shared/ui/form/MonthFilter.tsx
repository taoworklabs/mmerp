import { MonthFilterChip } from '@mmerp/ui'
import { IconCalendarMonth } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import { calendarLocale, formatMonthShort } from '@/shared/i18n'
import { icon } from '../theme'

// MonthFilter is a filter chip that picks a month (YYYY-MM), beside the FilterChips of a filter bar.
export function MonthFilter({ label, value, onChange }: { label: string; value: string | null; onChange: (month: string | null) => void }) {
  const { t } = useTranslation()
  return (
    <MonthFilterChip
      label={label}
      icon={<IconCalendarMonth {...icon.button} />}
      value={value}
      onChange={onChange}
      valueText={(name, month) => t('shared.list.filter_value', { name, value: formatMonthShort(month) })}
      clearLabel={t('shared.list.clear_filter', { name: label })}
      locale={calendarLocale()}
      labels={{ previousYear: t('shared.form.previous_year'), nextYear: t('shared.form.next_year') }}
    />
  )
}
