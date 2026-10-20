import type * as React from 'react'
import { useRef, useState } from 'react'
import { ActionIcon } from '../input/ActionIcon.tsx'
import type { FieldProps } from '../input/Field.tsx'
import { TextInput } from '../input/TextInput.tsx'
import { Popover } from '../overlay/Popover.tsx'
import { Calendar } from './Calendar.tsx'
import { clamp, type CalendarLocale } from './calendarGrid.ts'

type DateInputClearProps = { clearable?: false; clearLabel?: undefined } | { clearable: true; clearLabel: string }

export type DateInputProps = FieldProps &
  Omit<React.ComponentPropsWithRef<'input'>, 'value' | 'defaultValue' | 'onChange' | 'size' | 'type' | 'min' | 'max'> &
  DateInputClearProps & {
    value: string | null
    onChange(iso: string | null): void
    parse(text: string): string | null
    format(iso: string): string
    locale: CalendarLocale
    labels: { previousMonth: string; nextMonth: string }
    minDate?: string
    maxDate?: string
    placeholder?: string
    leftSection?: React.ReactNode
  }

export function ClearIcon() {
  return (
    <svg width="10" height="10" viewBox="0 0 10 10" aria-hidden="true">
      <path d="M1 1 9 9M9 1 1 9" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" />
    </svg>
  )
}

export function DateInput({
  value,
  onChange,
  parse,
  format,
  locale,
  labels,
  minDate,
  maxDate,
  clearable,
  clearLabel,
  disabled,
  readOnly,
  ref,
  onClick,
  onKeyDown,
  onBlur,
  ...props
}: DateInputProps) {
  const inputRef = useRef<HTMLInputElement | null>(null)
  const calendarRef = useRef<HTMLDivElement>(null)
  const [open, setOpen] = useState(false)
  const [openedByKeyboard, setOpenedByKeyboard] = useState(false)
  // The popup has no trigger to borrow a name from, so it takes the field's label.
  const [popupLabelledBy, setPopupLabelledBy] = useState<string>()
  const shown = value === null ? '' : format(value)
  const [text, setText] = useState(shown)
  const [lastValue, setLastValue] = useState(value)
  if (value !== lastValue) {
    setLastValue(value)
    setText(shown)
  }
  const interactive = !disabled && !readOnly

  const openCalendar = (byKeyboard: boolean) => {
    if (!interactive) return
    setOpenedByKeyboard(byKeyboard)
    setPopupLabelledBy(inputRef.current?.getAttribute('aria-labelledby') ?? undefined)
    setOpen(true)
  }

  // Free text is only interpreted here; anything unusable puts back the current value.
  const commit = () => {
    const trimmed = text.trim()
    if (trimmed === '' && clearable) {
      if (value !== null) onChange(null)
      return
    }
    const iso = trimmed === '' ? null : parse(trimmed)
    if (iso !== null && clamp(iso, minDate, maxDate) === iso) {
      setText(format(iso))
      if (iso !== value) onChange(iso)
      return
    }
    setText(shown)
  }

  const setRefs = (node: HTMLInputElement | null) => {
    inputRef.current = node
    if (typeof ref === 'function') ref(node)
    else if (ref) ref.current = node
  }

  return (
    <>
      <TextInput
        {...props}
        ref={setRefs}
        value={text}
        disabled={disabled}
        readOnly={readOnly}
        onChange={(event) => setText(event.target.value)}
        onClick={(event) => {
          onClick?.(event)
          openCalendar(false)
        }}
        onKeyDown={(event) => {
          onKeyDown?.(event)
          if (event.key === 'ArrowDown') {
            event.preventDefault()
            openCalendar(true)
          } else if (event.key === 'Enter') {
            commit()
            setOpen(false)
          }
        }}
        onBlur={(event) => {
          onBlur?.(event)
          commit()
        }}
        rightSection={
          clearable &&
          value !== null &&
          interactive && (
            <ActionIcon
              variant="subtle"
              color="gray"
              size="xs"
              aria-label={clearLabel}
              onClick={() => {
                setText('')
                onChange(null)
                inputRef.current?.focus()
              }}
            >
              <ClearIcon />
            </ActionIcon>
          )
        }
      />
      <Popover.Root open={open && interactive} onOpenChange={setOpen}>
        <Popover.Popup
          side="bottom"
          align="start"
          aria-labelledby={popupLabelledBy}
          aria-label={props['aria-label']}
          // The bordered box, not the bare <input>, so the popup lines up with what the user sees.
          anchor={() => inputRef.current?.parentElement ?? null}
          // A click keeps focus in the input for typing; ArrowDown moves it into the days.
          initialFocus={openedByKeyboard ? () => calendarRef.current?.querySelector<HTMLElement>('[role="grid"] [tabindex="0"]') ?? true : false}
        >
          <Calendar
            ref={calendarRef}
            value={value}
            onChange={(iso) => {
              setText(format(iso))
              onChange(iso)
              setOpen(false)
              inputRef.current?.focus()
            }}
            minDate={minDate}
            maxDate={maxDate}
            locale={locale}
            labels={labels}
          />
        </Popover.Popup>
      </Popover.Root>
    </>
  )
}
