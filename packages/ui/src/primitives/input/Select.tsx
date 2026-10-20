import { Select as BaseSelect } from '@base-ui/react/select'
import type * as React from 'react'
import { ActionIcon } from './ActionIcon.tsx'
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

function ClearIcon() {
  return (
    <svg width="10" height="10" viewBox="0 0 10 10" aria-hidden="true">
      <path d="M1 1 9 9M9 1 1 9" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" />
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
  ...props
}: SelectProps) {
  return (
    <Field label={label} description={description} error={error} required={required} disabled={disabled} readOnly={readOnly} className={className}>
      <BaseSelect.Root<string> items={data} value={value} onValueChange={onChange} disabled={disabled} readOnly={readOnly}>
        <BaseSelect.Trigger
          data-mmerp-control=""
          data-error={error ? '' : undefined}
          data-disabled={disabled ? '' : undefined}
          className={textInputStyles.wrapper}
          style={style}
          {...props}
        >
          <BaseSelect.Value placeholder={placeholder} style={valueStyle} />
          {clearable && value !== null && (
            <ActionIcon
              variant="subtle"
              color="gray"
              size="xs"
              aria-label={clearLabel}
              onClick={(event) => {
                event.stopPropagation()
                onChange(null)
              }}
            >
              <ClearIcon />
            </ActionIcon>
          )}
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
    </Field>
  )
}
