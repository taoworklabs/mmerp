import type * as React from 'react'
import { useEffect, useId, useRef, useState } from 'react'
import { cx } from '../../internal/cx.ts'
import { ActionIcon } from '../input/ActionIcon.tsx'
import { addDays, addMonths, clamp, localToday, monthGrid, weekdayIndex, type CalendarLocale } from './calendarGrid.ts'
import styles from './Calendar.module.css'

export interface CalendarProps extends Omit<React.ComponentPropsWithRef<'div'>, 'onChange' | 'defaultValue'> {
  value: string | null
  onChange(iso: string): void
  defaultMonth?: string
  minDate?: string
  maxDate?: string
  locale: CalendarLocale
  labels: { previousMonth: string; nextMonth: string }
  today?: string
}

function Chevron({ flip }: { flip?: boolean }) {
  return (
    <svg width="10" height="10" viewBox="0 0 10 10" aria-hidden="true" style={flip ? { transform: 'scaleX(-1)' } : undefined}>
      <path d="M6.5 2 3.5 5l3 3" stroke="currentColor" strokeWidth="1.4" fill="none" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

interface CalendarHeaderProps {
  title: string
  titleId: string
  previousLabel: string
  nextLabel: string
  previousDisabled: boolean
  nextDisabled: boolean
  onPrevious(): void
  onNext(): void
}

export function CalendarHeader(props: CalendarHeaderProps) {
  return (
    <div className={styles.header}>
      <ActionIcon variant="subtle" color="gray" size="xs" aria-label={props.previousLabel} disabled={props.previousDisabled} onClick={props.onPrevious}>
        <Chevron />
      </ActionIcon>
      <span id={props.titleId} className={styles.title} aria-live="polite">
        {props.title}
      </span>
      <ActionIcon variant="subtle" color="gray" size="xs" aria-label={props.nextLabel} disabled={props.nextDisabled} onClick={props.onNext}>
        <Chevron flip />
      </ActionIcon>
    </div>
  )
}

// Roving focus: `focused` is the one tabbable cell. Keyboard moves set the flag so the new cell
// takes DOM focus after render; header clicks move the cursor without stealing focus.
export function useRovingCursor(initial: () => string, normalize: (cursor: string) => string) {
  const [cursor, setCursor] = useState(() => normalize(initial()))
  const gridRef = useRef<HTMLDivElement>(null)
  const pendingFocus = useRef(false)
  useEffect(() => {
    if (!pendingFocus.current) return
    pendingFocus.current = false
    gridRef.current?.querySelector<HTMLElement>('[tabindex="0"]')?.focus()
  }, [cursor])
  const move = (next: string, focus: boolean) => {
    pendingFocus.current = focus
    setCursor(normalize(next))
  }
  return { cursor, move, gridRef }
}

export function Calendar({
  value,
  onChange,
  defaultMonth,
  minDate,
  maxDate,
  locale,
  labels,
  today,
  className,
  ...props
}: CalendarProps) {
  const todayIso = today ?? localToday()
  const firstDayOfWeek = locale.firstDayOfWeek ?? 1
  const titleId = useId()
  const { cursor, move, gridRef } = useRovingCursor(
    () => value ?? (defaultMonth ? `${defaultMonth}-01` : todayIso),
    (iso) => clamp(iso, minDate, maxDate),
  )
  // Follow a value set from outside (e.g. typed into an input) while the calendar is shown.
  const [lastValue, setLastValue] = useState(value)
  if (value !== lastValue) {
    setLastValue(value)
    if (value !== null) move(value, false)
  }

  const year = Number(cursor.slice(0, 4))
  const month = Number(cursor.slice(5, 7))
  const monthStart = `${cursor.slice(0, 7)}-01`
  const days = monthGrid(year, month, firstDayOfWeek)

  const onKeyDown = (event: React.KeyboardEvent) => {
    const index = weekdayIndex(cursor, firstDayOfWeek)
    const targets: Record<string, () => string> = {
      ArrowLeft: () => addDays(cursor, -1),
      ArrowRight: () => addDays(cursor, 1),
      ArrowUp: () => addDays(cursor, -7),
      ArrowDown: () => addDays(cursor, 7),
      PageUp: () => addMonths(cursor, -1),
      PageDown: () => addMonths(cursor, 1),
      Home: () => addDays(cursor, -index),
      End: () => addDays(cursor, 6 - index),
    }
    const target = targets[event.key]
    if (!target) return
    event.preventDefault()
    move(target(), true)
  }

  return (
    <div className={cx(styles.root, className)} {...props}>
      <CalendarHeader
        title={`${locale.months[month - 1]} ${year}`}
        titleId={titleId}
        previousLabel={labels.previousMonth}
        nextLabel={labels.nextMonth}
        previousDisabled={minDate !== undefined && addDays(monthStart, -1) < minDate}
        nextDisabled={maxDate !== undefined && addMonths(monthStart, 1) > maxDate}
        onPrevious={() => move(addMonths(cursor, -1), false)}
        onNext={() => move(addMonths(cursor, 1), false)}
      />
      <div ref={gridRef} role="grid" aria-labelledby={titleId} onKeyDown={onKeyDown}>
        <div role="row" className={styles.row}>
          {Array.from({ length: 7 }, (_, i) => (
            <div key={i} role="columnheader" className={styles.weekday}>
              {locale.weekdays[(firstDayOfWeek + i) % 7]}
            </div>
          ))}
        </div>
        {Array.from({ length: 6 }, (_, week) => (
          <div key={week} role="row" className={styles.row}>
            {days.slice(week * 7, week * 7 + 7).map((iso) =>
              // Days of the neighbouring months stay blank so every visible number is unique.
              iso.slice(0, 7) !== cursor.slice(0, 7) ? (
                <div key={iso} role="gridcell" className={styles.cell} />
              ) : (
                <button
                  key={iso}
                  type="button"
                  role="gridcell"
                  className={styles.cell}
                  tabIndex={iso === cursor ? 0 : -1}
                  aria-selected={iso === value}
                  aria-current={iso === todayIso ? 'date' : undefined}
                  disabled={clamp(iso, minDate, maxDate) !== iso}
                  onClick={() => {
                    move(iso, false)
                    onChange(iso)
                  }}
                >
                  {Number(iso.slice(8))}
                </button>
              ),
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
