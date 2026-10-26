import { Checkbox as BaseCheckbox } from '@base-ui/react/checkbox'
import { Field as BaseField } from '@base-ui/react/field'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import styles from './Checkbox.module.css'

export interface CheckboxProps extends Omit<React.ComponentPropsWithRef<'div'>, 'children' | 'onChange'> {
  checked?: boolean
  defaultChecked?: boolean
  indeterminate?: boolean
  onChange?(checked: boolean): void
  label?: React.ReactNode
  description?: React.ReactNode
  error?: React.ReactNode
  disabled?: boolean
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
  className,
  'aria-label': ariaLabel,
  'aria-labelledby': ariaLabelledBy,
  'aria-describedby': ariaDescribedBy,
  ...props
}: CheckboxProps) {
  return (
    <BaseField.Root invalid={!!error} disabled={disabled} className={cx(styles.root, className)} {...props}>
      <BaseField.Label className={styles.label} data-disabled={disabled ? '' : undefined}>
        <BaseCheckbox.Root
          aria-label={ariaLabel}
          aria-labelledby={ariaLabelledBy}
          aria-describedby={ariaDescribedBy}
          checked={checked}
          defaultChecked={defaultChecked}
          indeterminate={indeterminate}
          disabled={disabled}
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
      {description !== undefined && <BaseField.Description className={styles.description}>{description}</BaseField.Description>}
      {error !== undefined && (
        <BaseField.Error match className={styles.error}>
          {error}
        </BaseField.Error>
      )}
    </BaseField.Root>
  )
}
