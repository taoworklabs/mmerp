import { IconCalendar } from '@tabler/icons-react'
import { FilterChip } from './FilterChip'

// YearFilter picks a year in a filter bar, from the years that can hold data, latest first; it
// always holds one.
export function YearFilter({ label, value, years, onChange }: { label: string; value: number; years: number[]; onChange: (year: number) => void }) {
  return (
    <FilterChip
      label={label}
      icon={IconCalendar}
      data={[...years].sort((a, b) => b - a).map((y) => ({ value: String(y), label: String(y) }))}
      value={String(value)}
      onChange={(v) => v && onChange(Number(v))}
      clearable={false}
    />
  )
}
