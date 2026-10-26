import type * as React from 'react'
import { useRef, useState } from 'react'
import { cx } from '../../internal/cx.ts'
import { ChevronDownIcon, CloseIcon } from '../../internal/icons.tsx'
import type { CalendarLocale } from '../../primitives/dates/calendarGrid.ts'
import { MonthPicker } from '../../primitives/dates/MonthPicker.tsx'
import { Button } from '../../primitives/input/Button.tsx'
import { Popover } from '../../primitives/overlay/Popover.tsx'
import { Tooltip } from '../../primitives/overlay/Tooltip.tsx'
import styles from './List.module.css'

export interface MonthFilterChipProps extends Omit<React.ComponentPropsWithRef<'div'>, 'children' | 'onChange'> {
  label: string
  icon: React.ReactNode
  // YYYY-MM.
  value: string | null
  onChange(month: string | null): void
  // How the chosen month reads once set, e.g. "Month: 03/2026".
  valueText(label: string, month: string): string
  clearLabel: string
  locale: CalendarLocale
  labels: { previousYear: string; nextYear: string }
}

// MonthFilterChip is FilterChip for a month: "Label" and a chevron while unset, "Label: 03/2026"
// tinted with its own clear button once set.
export function MonthFilterChip({ label, icon, value, onChange, valueText, clearLabel, locale, labels, className, ...props }: MonthFilterChipProps) {
  const [open, setOpen] = useState(false)
  const pickerRef = useRef<HTMLDivElement>(null)
  const variant = value !== null ? 'light' : 'default'
  const text = value !== null ? valueText(label, value) : label
  return (
    <Button.Group className={cx(styles.chip, className)} {...props}>
      <Popover.Root open={open} onOpenChange={setOpen}>
        <Popover.Trigger
          render={
            <Button
              variant={variant}
              leftSection={icon}
              rightSection={value === null && <ChevronDownIcon />}
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
          <MonthPicker
            ref={pickerRef}
            value={value}
            onChange={(month) => {
              onChange(month)
              setOpen(false)
            }}
            locale={locale}
            labels={labels}
          />
        </Popover.Popup>
      </Popover.Root>
      {value !== null && (
        <Tooltip label={clearLabel}>
          <Button variant={variant} aria-label={clearLabel} onClick={() => onChange(null)} className={styles.clear}>
            <CloseIcon />
          </Button>
        </Tooltip>
      )}
    </Button.Group>
  )
}
