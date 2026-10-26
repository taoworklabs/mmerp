import { Button, useDisclosure } from '@mmerp/ui'
import { Popover, Tooltip } from '@mantine/core'
import { MonthPicker } from '@mantine/dates'
import { IconCalendarMonth, IconChevronDown, IconX } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import { formatMonthShort } from '@/shared/i18n'
import { filterWidth, icon } from '../theme'

// MonthFilter is a filter chip that picks a month (YYYY-MM): "Name" and a chevron while unset;
// "Name: month", tinted, with its own clear button once set.
// ponytail: a hand-built chip around a Mantine popover until the month picker comes from the kit.
export function MonthFilter({ label, value, onChange }: { label: string; value: string | null; onChange: (month: string | null) => void }) {
  const { t } = useTranslation()
  const [opened, { toggle, close }] = useDisclosure()
  const variant = value ? 'light' : 'default'
  const text = value ? t('shared.list.filter_value', { name: label, value: formatMonthShort(value) }) : label
  const clear = t('shared.list.clear_filter', { name: label })
  return (
    <Button.Group>
      <Popover opened={opened} onChange={(o) => !o && close()} position="bottom-start" trapFocus returnFocus>
        <Popover.Target>
          <Button
            variant={variant}
            leftSection={<IconCalendarMonth {...icon.button} />}
            rightSection={!value && <IconChevronDown {...icon.text} />}
            style={{ maxWidth: filterWidth.chip }}
            onClick={toggle}
          >
            {text}
          </Button>
        </Popover.Target>
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
      {value && (
        <Tooltip label={clear}>
          <Button variant={variant} aria-label={clear} onClick={() => onChange(null)}>
            <IconX {...icon.button} />
          </Button>
        </Tooltip>
      )}
    </Button.Group>
  )
}
