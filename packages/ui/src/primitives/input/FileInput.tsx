import type * as React from 'react'
import { useId, useRef } from 'react'
import { ClearButton } from '../../internal/ClearButton.tsx'
import { cx } from '../../internal/cx.ts'
import { mergeRefs } from '../../internal/mergeRefs.ts'
import { VisuallyHidden } from '../layout/VisuallyHidden.tsx'
import { Field, type FieldProps } from './Field.tsx'
import styles from './FileInput.module.css'

export interface FileInputProps extends FieldProps, Omit<React.ComponentPropsWithRef<'button'>, 'children' | 'onChange' | 'value' | 'type'> {
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
  onClick,
  ref,
  ...props
}: FileInputProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const visibleText = value ? value.name : placeholder
  // The trigger isn't a Field control, so Base UI never links it to the description/error text itself.
  const descriptionId = useId()
  const errorId = useId()
  const describedBy = cx(description !== undefined && descriptionId, error !== undefined && errorId) || undefined
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
      <div className={styles.wrapper} data-mmerp-control="" data-error={error ? '' : undefined} data-disabled={disabled ? '' : undefined}>
        <button
          {...props}
          ref={mergeRefs(triggerRef, ref)}
          type="button"
          className={styles.trigger}
          disabled={disabled || readOnly}
          aria-describedby={describedBy}
          aria-invalid={error ? true : undefined}
          onClick={(event) => {
            onClick?.(event)
            inputRef.current?.click()
          }}
        >
          {/* No visible text and no caller-supplied name: fall back to the field's own label. */}
          {visibleText ? (
            <span className={cx(styles.fileName, !value && styles.placeholder)}>{visibleText}</span>
          ) : (
            !hasOwnName && label !== undefined && <VisuallyHidden>{label}</VisuallyHidden>
          )}
        </button>
        {value && <ClearButton label={clearLabel} onClear={() => onChange(null)} focusTarget={triggerRef} disabled={disabled} readOnly={readOnly} />}
        <input
          ref={inputRef}
          type="file"
          accept={accept}
          disabled={disabled || readOnly}
          className={styles.hiddenInput}
          onChange={(event) => {
            onChange(event.target.files?.[0] ?? null)
            // An emptied input fires change again when the same file is picked after a clear.
            event.target.value = ''
          }}
        />
      </div>
    </Field>
  )
}
