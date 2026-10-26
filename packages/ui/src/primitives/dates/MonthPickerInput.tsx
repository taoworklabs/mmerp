import type * as React from 'react'
import { useId, useRef, useState } from 'react'
import { ClearButton } from '../../internal/ClearButton.tsx'
import { cx } from '../../internal/cx.ts'
import { mergeRefs } from '../../internal/mergeRefs.ts'
import { VisuallyHidden } from '../layout/VisuallyHidden.tsx'
import { Field, type FieldProps } from '../input/Field.tsx'
import textInputStyles from '../input/TextInput.module.css'
import { Popover } from '../overlay/Popover.tsx'
import type { CalendarLocale } from './calendarGrid.ts'
import { MonthPicker } from './MonthPicker.tsx'
import styles from './MonthPickerInput.module.css'

type MonthPickerInputClearProps = { clearable?: false; clearLabel?: undefined } | { clearable: true; clearLabel: string }

export type MonthPickerInputProps = FieldProps &
  Omit<React.ComponentPropsWithRef<'button'>, 'value' | 'defaultValue' | 'onChange' | 'children' | 'type'> &
  MonthPickerInputClearProps & {
    value: string | null
    onChange(month: string | null): void
    format(month: string): string
    locale: CalendarLocale
    labels: { previousYear: string; nextYear: string }
    minDate?: string
    maxDate?: string
    placeholder?: string
    leftSection?: React.ReactNode
  }

export function MonthPickerInput({
  label,
  description,
  error,
  required,
  disabled,
  readOnly,
  value,
  onChange,
  format,
  locale,
  labels,
  minDate,
  maxDate,
  placeholder,
  leftSection,
  clearable,
  clearLabel,
  className,
  ref,
  ...props
}: MonthPickerInputProps) {
  const boxRef = useRef<HTMLDivElement>(null)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const pickerRef = useRef<HTMLDivElement>(null)
  const [open, setOpen] = useState(false)
  // The trigger isn't a Field control, so Base UI never links it to the description/error text itself.
  const descriptionId = useId()
  const errorId = useId()
  const describedBy = cx(description !== undefined && descriptionId, error !== undefined && errorId) || undefined
  const interactive = !disabled && !readOnly
  const visibleText = value === null ? placeholder : format(value)
  const hasOwnName = props['aria-label'] !== undefined || props['aria-labelledby'] !== undefined
  return (
    <Field
      label={label}
      description={description}
      error={error}
      required={required}
      disabled={disabled}
      readOnly={readOnly}
      className={className}
      descriptionId={descriptionId}
      errorId={errorId}
    >
      <div ref={boxRef} className={textInputStyles.wrapper} data-mmerp-control="" data-error={error ? '' : undefined} data-disabled={disabled ? '' : undefined}>
        {leftSection && <span className={textInputStyles.section}>{leftSection}</span>}
        <Popover.Root open={open && interactive} onOpenChange={setOpen}>
          <Popover.Trigger
            {...props}
            ref={mergeRefs(triggerRef, ref)}
            className={styles.trigger}
            disabled={!interactive}
            aria-describedby={describedBy}
            aria-invalid={error ? true : undefined}
          >
            {/* The field label is part of the name, so the chosen month is announced with it. */}
            {!hasOwnName && label !== undefined && <VisuallyHidden>{label}</VisuallyHidden>}
            {visibleText !== undefined && <span className={cx(styles.value, value === null && styles.placeholder)}>{visibleText}</span>}
          </Popover.Trigger>
          <Popover.Popup
            side="bottom"
            align="start"
            anchor={boxRef}
            initialFocus={() => pickerRef.current?.querySelector<HTMLElement>('[role="grid"] [tabindex="0"]') ?? true}
          >
            <MonthPicker
              ref={pickerRef}
              value={value}
              onChange={(month) => {
                onChange(month)
                setOpen(false)
              }}
              minDate={minDate}
              maxDate={maxDate}
              locale={locale}
              labels={labels}
            />
          </Popover.Popup>
        </Popover.Root>
        {clearable && value !== null && (
          <ClearButton label={clearLabel} onClear={() => onChange(null)} focusTarget={triggerRef} disabled={disabled} readOnly={readOnly} />
        )}
      </div>
    </Field>
  )
}
