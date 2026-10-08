import { MonthPickerInput } from '@mantine/dates'
import { IconCalendarMonth } from '@tabler/icons-react'
import { icon } from '../theme'

// MonthFilter picks a month (YYYY-MM) in a list's filter bar; its name is the placeholder.
export function MonthFilter({ label, value, onChange }: { label: string; value: string | null; onChange: (month: string | null) => void }) {
  return (
    <MonthPickerInput
      aria-label={label}
      placeholder={label}
      leftSection={<IconCalendarMonth {...icon.text} />}
      w={{ base: '100%', md: 160 }}
      value={value ? `${value}-01` : null}
      onChange={(v) => onChange(v ? v.slice(0, 7) : null)}
      valueFormat="MM/YYYY"
      clearable
    />
  )
}
