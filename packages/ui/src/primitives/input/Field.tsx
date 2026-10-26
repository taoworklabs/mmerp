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
  // Lets a control that Base UI doesn't auto-wire (NumberField.Input, a custom trigger) point its
  // own `aria-describedby` at these, instead of the id Base UI would otherwise generate internally.
  descriptionId?: string
  errorId?: string
  labelId?: string
  controlId?: string
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
  descriptionId,
  errorId,
  labelId,
  controlId,
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
        <BaseField.Label id={labelId} htmlFor={controlId} className={styles.label}>
          {label}
          {required && (
            <span aria-hidden="true" className={styles.required}>
              *
            </span>
          )}
        </BaseField.Label>
      )}
      {children}
      {description !== undefined && (
        <BaseField.Description id={descriptionId} className={styles.description}>
          {description}
        </BaseField.Description>
      )}
      {error !== undefined && (
        // `match` forces the message to render from our own `error` prop instead of native validity.
        <BaseField.Error id={errorId} match className={styles.error}>
          {error}
        </BaseField.Error>
      )}
    </BaseField.Root>
  )
}
