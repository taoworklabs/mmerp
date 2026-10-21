import { Field as BaseField } from '@base-ui/react/field'
import type * as React from 'react'
import type { StyleVars } from '../../internal/style.ts'
import { Field, type FieldProps } from './Field.tsx'
import styles from './Textarea.module.css'

export interface TextareaProps extends FieldProps, React.ComponentPropsWithRef<'textarea'> {
  autosize?: boolean
  minRows?: number
  maxRows?: number
}

// Field.Control types itself against a native <input>; swapping its rendered tag to <textarea>
// via `render` is the documented way to reuse Field's label/validation wiring for other elements.
const TextareaControl = BaseField.Control as unknown as React.FC<
  React.ComponentPropsWithRef<'textarea'> & { render: React.ReactElement; 'data-error'?: string; 'data-autosize'?: string }
>

export function Textarea({
  label,
  description,
  error,
  required,
  disabled,
  readOnly,
  autosize,
  minRows,
  maxRows,
  className,
  style,
  ...props
}: TextareaProps) {
  // field-sizing: content autosizes natively in Chromium today; older engines just keep a fixed height.
  // ponytail: Chromium-only field-sizing, add a JS resize-on-input fallback if other engines matter.
  const vars: StyleVars = {
    ...style,
    minHeight: autosize && minRows !== undefined ? `calc(${minRows} * var(--mmerp-line-height-sm) * 1em + var(--mmerp-space-xxs) * 2 + 2px)` : undefined,
    maxHeight: autosize && maxRows !== undefined ? `calc(${maxRows} * var(--mmerp-line-height-sm) * 1em + var(--mmerp-space-xxs) * 2 + 2px)` : undefined,
  }
  return (
    <Field label={label} description={description} error={error} required={required} disabled={disabled} readOnly={readOnly} className={className}>
      <TextareaControl
        render={<textarea />}
        data-mmerp-control="multiline"
        data-error={error ? '' : undefined}
        data-autosize={autosize ? '' : undefined}
        disabled={disabled}
        readOnly={readOnly}
        aria-required={required || undefined}
        className={styles.root}
        style={vars}
        {...props}
      />
    </Field>
  )
}

