import type * as React from 'react'
import { useRef } from 'react'
import { cx } from '../../internal/cx.ts'
import { VisuallyHidden } from '../layout/VisuallyHidden.tsx'
import { ActionIcon } from './ActionIcon.tsx'
import { Field, type FieldProps } from './Field.tsx'
import styles from './FileInput.module.css'

export interface FileInputProps extends FieldProps, Omit<React.ComponentPropsWithRef<'div'>, 'children' | 'onChange' | 'value'> {
  value: File | null
  onChange(file: File | null): void
  accept?: string
  placeholder?: string
  clearLabel: string
}

export function FileInput({
  label,
  description,
  error,
  required,
  disabled,
  readOnly,
  value,
  onChange,
  accept,
  placeholder,
  clearLabel,
  className,
  ...props
}: FileInputProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  const visibleText = value ? value.name : placeholder
  return (
    <Field label={label} description={description} error={error} required={required} disabled={disabled} readOnly={readOnly} className={className}>
      <div className={styles.wrapper} data-error={error ? '' : undefined} data-disabled={disabled ? '' : undefined} {...props}>
        <button
          type="button"
          data-mmerp-control=""
          className={styles.trigger}
          disabled={disabled || readOnly}
          onClick={() => inputRef.current?.click()}
        >
          {/* No visible text yet: fall back to the field's own label so the trigger still names itself. */}
          {visibleText ? (
            <span className={cx(styles.fileName, !value && styles.placeholder)}>{visibleText}</span>
          ) : (
            label !== undefined && <VisuallyHidden>{label}</VisuallyHidden>
          )}
        </button>
        {value && (
          <ActionIcon variant="subtle" color="gray" size="xs" disabled={disabled} aria-label={clearLabel} onClick={() => onChange(null)}>
            <svg width="10" height="10" viewBox="0 0 10 10" aria-hidden="true">
              <path d="M1 1 9 9M9 1 1 9" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" />
            </svg>
          </ActionIcon>
        )}
        <input
          ref={inputRef}
          type="file"
          accept={accept}
          disabled={disabled || readOnly}
          className={styles.hiddenInput}
          onChange={(event) => onChange(event.target.files?.[0] ?? null)}
        />
      </div>
    </Field>
  )
}
