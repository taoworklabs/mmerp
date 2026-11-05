import { Checkbox as BaseCheckbox } from '@base-ui/react/checkbox'
import { Field as BaseField } from '@base-ui/react/field'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import { Field } from './Field.tsx'
import styles from './Checkbox.module.css'

// Like TextInput and Select, `className` styles the field root and every other prop (id, ref,
// aria-*, handlers) goes to the focusable control, so a form can focus and label it.
export interface CheckboxProps extends Omit<React.ComponentPropsWithRef<'span'>, 'children' | 'onChange' | 'defaultChecked'> {
  checked?: boolean
  defaultChecked?: boolean
  indeterminate?: boolean
  onChange?(checked: boolean): void
  label?: React.ReactNode
  description?: React.ReactNode
  error?: React.ReactNode
  disabled?: boolean
  // Form props go to the hidden input Base UI renders, so the box works in a native form.
  name?: string
  value?: string
  required?: boolean
  readOnly?: boolean
  inputRef?: React.Ref<HTMLInputElement>
}

export function Checkbox({
  checked,
  defaultChecked,
  indeterminate,
  onChange,
  label,
  description,
  error,
  disabled,
  name,
  value,
  required,
  readOnly,
  inputRef,
  className,
  ...props
}: CheckboxProps) {
  return (
    <Field description={description} error={error} disabled={disabled} readOnly={readOnly} className={cx(styles.root, className)}>
      <BaseField.Label className={styles.label} data-disabled={disabled ? '' : undefined}>
        <BaseCheckbox.Root
          {...props}
          checked={checked}
          defaultChecked={defaultChecked}
          indeterminate={indeterminate}
          disabled={disabled}
          name={name}
          value={value}
          required={required}
          readOnly={readOnly}
          inputRef={inputRef}
          onCheckedChange={(next) => onChange?.(next)}
          className={styles.box}
        >
          <BaseCheckbox.Indicator className={styles.indicator}>
            {indeterminate ? (
              <svg width="10" height="2" viewBox="0 0 10 2" aria-hidden="true">
                <rect width="10" height="2" fill="currentColor" />
              </svg>
            ) : (
              <svg width="10" height="8" viewBox="0 0 10 8" fill="none" aria-hidden="true">
                <path d="M1 4 3.5 6.5 9 1" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            )}
          </BaseCheckbox.Indicator>
        </BaseCheckbox.Root>
        {label !== undefined && <span className={styles.text}>{label}</span>}
      </BaseField.Label>
    </Field>
  )
}
