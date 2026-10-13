import { Popover } from '@mantine/core'
import { MonthPicker } from '@mantine/dates'
import { useDisclosure } from '@mantine/hooks'
import { IconCalendarMonth } from '@tabler/icons-react'
import { formatMonthShort } from '@/shared/i18n'
import { ChipShell } from './FilterChip'

// MonthFilter is a filter chip that picks a month (YYYY-MM).
export function MonthFilter({ label, value, onChange }: { label: string; value: string | null; onChange: (month: string | null) => void }) {
  const [opened, { toggle, close }] = useDisclosure()
  return (
    <Popover opened={opened} onChange={(o) => !o && close()} position="bottom-start" trapFocus returnFocus>
      <ChipShell
        label={label}
        icon={IconCalendarMonth}
        chosen={value ? formatMonthShort(value) : null}
        clearable
        onClear={() => onChange(null)}
        onToggle={toggle}
        target={(button) => <Popover.Target>{button}</Popover.Target>}
      />
      <Popover.Dropdown>
        <MonthPicker
          value={value ? `${value}-01` : null}
          onChange={(v) => {
            if (v) onChange(v.slice(0, 7))
            close()
          }}
        />
      </Popover.Dropdown>
    </Popover>
  )
}
