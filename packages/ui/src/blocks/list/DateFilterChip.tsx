import type * as React from 'react'
import { useRef, useState } from 'react'
import { cx } from '../../internal/cx.ts'
import { ChevronDownIcon, CloseIcon } from '../../internal/icons.tsx'
import type { CalendarLocale } from '../../primitives/dates/calendarGrid.ts'
import { Calendar } from '../../primitives/dates/Calendar.tsx'
import { Button } from '../../primitives/input/Button.tsx'
import { Popover } from '../../primitives/overlay/Popover.tsx'
import { Tooltip } from '../../primitives/overlay/Tooltip.tsx'
import styles from './List.module.css'

export interface DateFilterChipProps extends Omit<React.ComponentPropsWithRef<'div'>, 'children' | 'onChange'> {
  label: string
  icon: React.ReactNode
  // YYYY-MM-DD.
  value: string | null
  onChange(date: string | null): void
  // How the chosen day reads once set, e.g. "Expiring before: 15/11/2026".
  valueText(label: string, date: string): string
  clearLabel: string
  locale: CalendarLocale
  labels: { previousMonth: string; nextMonth: string }
  // A filter that always holds a day (a period's start or end) is never cleared nor tinted.
  clearable?: boolean
}

// DateFilterChip is FilterChip for a day: "Label" and a chevron while unset, "Label: 15/11/2026"
// tinted with its own clear button once set.
export function DateFilterChip({ label, icon, value, onChange, valueText, clearLabel, locale, labels, clearable = true, className, ...props }: DateFilterChipProps) {
  const [open, setOpen] = useState(false)
  const pickerRef = useRef<HTMLDivElement>(null)
  const active = value !== null && clearable
  const variant = active ? 'light' : 'default'
  const text = value !== null ? valueText(label, value) : label
  return (
    <Button.Group className={cx(styles.chip, className)} {...props}>
      <Popover.Root open={open} onOpenChange={setOpen}>
        <Popover.Trigger
          render={
            <Button
              variant={variant}
              leftSection={icon}
              rightSection={!active && <ChevronDownIcon />}
              title={value !== null ? text : undefined}
              className={styles.main}
            >
              <span className={styles.text}>{text}</span>
            </Button>
          }
        />
        <Popover.Popup
          side="bottom"
          align="start"
          initialFocus={() => pickerRef.current?.querySelector<HTMLElement>('[role="grid"] [tabindex="0"]') ?? true}
        >
          <Calendar
            ref={pickerRef}
            value={value}
            onChange={(date) => {
              onChange(date)
              setOpen(false)
            }}
            locale={locale}
            labels={labels}
          />
        </Popover.Popup>
      </Popover.Root>
      {active && (
        <Tooltip label={clearLabel}>
          <Button variant={variant} aria-label={clearLabel} onClick={() => onChange(null)} className={styles.clear}>
            <CloseIcon />
          </Button>
        </Tooltip>
      )}
    </Button.Group>
  )
}
