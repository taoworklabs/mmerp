import { NumberField } from '@base-ui/react/number-field'
import type * as React from 'react'
import { useId } from 'react'
import { cx } from '../../internal/cx.ts'
import { Field, type FieldProps } from './Field.tsx'
import styles from './NumberInput.module.css'

export interface NumberInputProps extends FieldProps, Omit<React.ComponentPropsWithRef<'div'>, 'children' | 'onChange' | 'value' | 'defaultValue'> {
  value: number | null
  onChange(value: number | null): void
  min?: number
  max?: number
  step?: number
  locale?: string
  format?: Intl.NumberFormatOptions
  placeholder?: string
}

export function NumberInput({
  label,
  description,
  error,
  required,
  disabled,
  readOnly,
  value,
  onChange,
  min,
  max,
  step,
  locale,
  format,
  placeholder,
  className,
  ...props
}: NumberInputProps) {
  // NumberField.Input never registers description ids with Field, so the describedby link is wired by hand.
  const descriptionId = useId()
  const errorId = useId()
  const describedBy = cx(description !== undefined && descriptionId, error !== undefined && errorId) || undefined
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
      {/* No Increment/Decrement parts rendered: arrow-key stepping on the input is Base UI's native behaviour. */}
      <NumberField.Root
        value={value}
        onValueChange={(next) => onChange(next)}
        min={min}
        max={max}
        step={step}
        locale={locale}
        format={format}
        disabled={disabled}
        readOnly={readOnly}
        className={styles.wrapper}
        data-mmerp-control=""
        data-error={error ? '' : undefined}
        data-disabled={disabled ? '' : undefined}
        {...props}
      >
        <NumberField.Input
          placeholder={placeholder}
          className={styles.input}
          aria-describedby={describedBy}
          aria-invalid={error ? true : undefined}
        />
      </NumberField.Root>
    </Field>
  )
}
