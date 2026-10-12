import { Select } from '@mantine/core'
import { IconCalendar } from '@tabler/icons-react'
import { filterWidth, icon } from '../theme'

// YearSelect picks a year in a filter bar, from the years that can hold data, latest first.
export function YearSelect({ label, value, years, onChange }: { label: string; value: number; years: number[]; onChange: (year: number) => void }) {
  return (
    <Select
      aria-label={label}
      placeholder={label}
      leftSection={<IconCalendar {...icon.text} />}
      w={{ base: '100%', md: filterWidth.year }}
      data={[...years].sort((a, b) => b - a).map((y) => String(y))}
      value={String(value)}
      onChange={(v) => v && onChange(Number(v))}
      allowDeselect={false}
    />
  )
}
