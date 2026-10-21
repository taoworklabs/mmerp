import type * as React from 'react'
import { useId, useState } from 'react'
import { cx } from '../../internal/cx.ts'
import { CalendarHeader, useRovingCursor } from './Calendar.tsx'
import { addMonths, clamp, localToday, type CalendarLocale } from './calendarGrid.ts'
import styles from './Calendar.module.css'

export interface MonthPickerProps extends Omit<React.ComponentPropsWithRef<'div'>, 'onChange' | 'defaultValue'> {
  value: string | null
  onChange(month: string): void
  minDate?: string
  maxDate?: string
  locale: CalendarLocale
  labels: { previousYear: string; nextYear: string }
}

const shiftMonth = (month: string, n: number) => addMonths(`${month}-01`, n).slice(0, 7)

export function MonthPicker({ value, onChange, minDate, maxDate, locale, labels, className, ...props }: MonthPickerProps) {
  // Bounds may be given as days or months; months compare on their first seven characters.
  const min = minDate?.slice(0, 7)
  const max = maxDate?.slice(0, 7)
  const titleId = useId()
  const { cursor, move, gridRef } = useRovingCursor(
    () => value ?? localToday().slice(0, 7),
    (month) => clamp(month, min, max),
  )
  const [lastValue, setLastValue] = useState(value)
  if (value !== lastValue) {
    setLastValue(value)
    if (value !== null) move(value, false)
  }

  const year = Number(cursor.slice(0, 4))
  const months = Array.from({ length: 12 }, (_, i) => `${year}-${String(i + 1).padStart(2, '0')}`)

  const onKeyDown = (event: React.KeyboardEvent) => {
    const index = (Number(cursor.slice(5, 7)) - 1) % 3
    const steps: Record<string, number> = {
      ArrowLeft: -1,
      ArrowRight: 1,
      ArrowUp: -3,
      ArrowDown: 3,
      PageUp: -12,
      PageDown: 12,
      Home: -index,
      End: 2 - index,
    }
    const step = steps[event.key]
    if (step === undefined) return
    event.preventDefault()
    move(shiftMonth(cursor, step), true)
  }

  return (
    <div className={cx(styles.root, styles.months, className)} {...props}>
      <CalendarHeader
        title={String(year)}
        titleId={titleId}
        previousLabel={labels.previousYear}
        nextLabel={labels.nextYear}
        previousDisabled={min !== undefined && `${year - 1}-12` < min}
        nextDisabled={max !== undefined && `${year + 1}-01` > max}
        onPrevious={() => move(shiftMonth(cursor, -12), false)}
        onNext={() => move(shiftMonth(cursor, 12), false)}
      />
      <div ref={gridRef} role="grid" aria-labelledby={titleId} onKeyDown={onKeyDown}>
        {Array.from({ length: 4 }, (_, row) => (
          <div key={row} role="row" className={styles.row}>
            {months.slice(row * 3, row * 3 + 3).map((month, i) => (
              <button
                key={month}
                type="button"
                role="gridcell"
                className={styles.cell}
                tabIndex={month === cursor ? 0 : -1}
                aria-selected={month === value}
                disabled={clamp(month, min, max) !== month}
                onClick={() => {
                  move(month, false)
                  onChange(month)
                }}
              >
                {locale.months[row * 3 + i]}
              </button>
            ))}
          </div>
        ))}
      </div>
    </div>
  )
}
