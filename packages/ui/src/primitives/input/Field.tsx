import { Field as BaseField } from '@base-ui/react/field'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import styles from './Field.module.css'

export interface FieldProps {
  label?: React.ReactNode
  description?: React.ReactNode
  error?: React.ReactNode
  required?: boolean
  disabled?: boolean
  readOnly?: boolean
}

interface FieldComponentProps extends FieldProps, Omit<React.ComponentPropsWithRef<'div'>, 'children'> {
  children: React.ReactNode
}

export function Field({
  label,
  description,
  error,
  required,
  disabled,
  readOnly,
  className,
  children,
  ...props
}: FieldComponentProps) {
  return (
    <BaseField.Root
      invalid={!!error}
      disabled={disabled}
      data-read-only={readOnly ? '' : undefined}
      className={cx(styles.root, className)}
      {...props}
    >
      {label !== undefined && (
        <BaseField.Label className={styles.label}>
          {label}
          {required && (
            <span aria-hidden="true" className={styles.required}>
              *
            </span>
          )}
        </BaseField.Label>
      )}
      {children}
      {description !== undefined && <BaseField.Description className={styles.description}>{description}</BaseField.Description>}
      {error !== undefined && (
        // `match` forces the message to render from our own `error` prop instead of native validity.
        <BaseField.Error match className={styles.error}>
          {error}
        </BaseField.Error>
      )}
    </BaseField.Root>
  )
}
