import { Select as BaseSelect } from '@base-ui/react/select'
import type * as React from 'react'
import { useId, useRef } from 'react'
import { ClearButton } from '../../internal/ClearButton.tsx'
import { cx } from '../../internal/cx.ts'
import { mergeRefs } from '../../internal/mergeRefs.ts'
import { Field, type FieldProps } from './Field.tsx'
import optionStyles from './options.module.css'
import textInputStyles from './TextInput.module.css'

export interface SelectItem {
  value: string
  label: string
  disabled?: boolean
}

type SelectClearProps = { clearable?: false; clearLabel?: undefined } | { clearable: true; clearLabel: string }

export type SelectProps = FieldProps &
  Omit<React.ComponentPropsWithRef<'button'>, 'value' | 'onChange' | 'children'> &
  SelectClearProps & {
    data: SelectItem[]
    value: string | null
    onChange(value: string | null): void
    placeholder?: string
  }

// Basis `auto` (not the `.input` class's `0%`) so an explicit width from an ancestor still
// shrinks the trigger instead of the flex algorithm ignoring it.
const triggerStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  flex: '1 1 auto',
  minWidth: 0,
  padding: 0,
  textAlign: 'start',
}

const valueStyle: React.CSSProperties = {
  flex: '1 1 auto',
  minWidth: 0,
  overflow: 'hidden',
  textOverflow: 'ellipsis',
  whiteSpace: 'nowrap',
  textAlign: 'start',
}

function ChevronIcon() {
  return (
    <svg width="10" height="10" viewBox="0 0 10 10" aria-hidden="true">
      <path d="M2 4l3 3 3-3" stroke="currentColor" strokeWidth="1.4" fill="none" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

export function Select({
  label,
  description,
  error,
  required,
  disabled,
  readOnly,
  data,
  value,
  onChange,
  placeholder,
  clearable,
  clearLabel,
  className,
  style,
  ref,
  ...props
}: SelectProps) {
  const triggerRef = useRef<HTMLButtonElement>(null)
  // The trigger isn't a Field control, so Base UI never links it to the description/error text itself.
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
      {/* The visible control box: the trigger (and the clear button, when clearable) are siblings
          inside it, never nested — a <button> cannot contain another interactive descendant. */}
      <div className={textInputStyles.wrapper} data-mmerp-control="" data-error={error ? '' : undefined} data-disabled={disabled ? '' : undefined}>
        <BaseSelect.Root<string> items={data} value={value} onValueChange={onChange} disabled={disabled} readOnly={readOnly}>
          <BaseSelect.Trigger
            ref={mergeRefs(triggerRef, ref)}
            className={textInputStyles.input}
            style={{ ...triggerStyle, ...style }}
            aria-describedby={describedBy}
            aria-invalid={error ? true : undefined}
            aria-required={required || undefined}
            {...props}
          >
            <BaseSelect.Value placeholder={placeholder} style={valueStyle} />
            <BaseSelect.Icon className={textInputStyles.section}>
              <ChevronIcon />
            </BaseSelect.Icon>
          </BaseSelect.Trigger>
          <BaseSelect.Portal>
            <BaseSelect.Positioner sideOffset={4}>
              <BaseSelect.Popup className={optionStyles.popup}>
                <BaseSelect.List>
                  {data.map((item) => (
                    <BaseSelect.Item key={item.value} value={item.value} disabled={item.disabled} className={optionStyles.option}>
                      {item.label}
                    </BaseSelect.Item>
                  ))}
                </BaseSelect.List>
              </BaseSelect.Popup>
            </BaseSelect.Positioner>
          </BaseSelect.Portal>
        </BaseSelect.Root>
        {clearable && value !== null && (
          <ClearButton label={clearLabel} onClear={() => onChange(null)} focusTarget={triggerRef} disabled={disabled} readOnly={readOnly} />
        )}
      </div>
    </Field>
  )
}
