import { NumberField } from '@base-ui/react/number-field'
import type * as React from 'react'
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
  return (
    <Field label={label} description={description} error={error} required={required} disabled={disabled} readOnly={readOnly} className={className}>
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
        data-error={error ? '' : undefined}
        data-disabled={disabled ? '' : undefined}
        {...props}
      >
        <NumberField.Input data-mmerp-control="" placeholder={placeholder} className={styles.input} />
      </NumberField.Root>
    </Field>
  )
}
