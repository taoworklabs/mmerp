import { Input as BaseInput } from '@base-ui/react/input'
import type * as React from 'react'
import { Field, type FieldProps } from './Field.tsx'
import styles from './TextInput.module.css'

export interface TextInputProps extends FieldProps, Omit<React.ComponentPropsWithRef<'input'>, 'size'> {
  leftSection?: React.ReactNode
  rightSection?: React.ReactNode
}

export function TextInput({
  label,
  description,
  error,
  required,
  disabled,
  readOnly,
  leftSection,
  rightSection,
  className,
  style,
  ...props
}: TextInputProps) {
  return (
    <Field label={label} description={description} error={error} required={required} disabled={disabled} readOnly={readOnly} className={className}>
      <div className={styles.wrapper} data-error={error ? '' : undefined} data-disabled={disabled ? '' : undefined}>
        {leftSection && <span className={styles.section}>{leftSection}</span>}
        <BaseInput
          data-mmerp-control=""
          disabled={disabled}
          readOnly={readOnly}
          className={styles.input}
          style={style}
          {...props}
        />
        {rightSection && <span className={styles.section}>{rightSection}</span>}
      </div>
    </Field>
  )
}
