import { Combobox as BaseCombobox } from '@base-ui/react/combobox'
import type * as React from 'react'
import { useId } from 'react'
import { cx } from '../../internal/cx.ts'
import { CloseIcon } from '../../internal/icons.tsx'
import { Field, type FieldProps } from './Field.tsx'
import { matchesQuery } from './matchesQuery.ts'
import optionStyles from './options.module.css'
import type { SelectItem } from './Select.tsx'
import textInputStyles from './TextInput.module.css'

export interface MultiSelectProps extends FieldProps, Omit<React.ComponentPropsWithRef<'input'>, 'value' | 'onChange' | 'size'> {
  data: SelectItem[]
  value: string[]
  onChange(value: string[]): void
  nothingFound: React.ReactNode
  placeholder?: string
  removeLabel: (itemLabel: string) => string
}

function filterByLabel(item: SelectItem, query: string) {
  return matchesQuery(item.label, query)
}

const wrapperStyle: React.CSSProperties = {
  height: 'auto',
  flexWrap: 'wrap',
  paddingBlock: 'var(--mmerp-space-xxs)',
  rowGap: 'var(--mmerp-space-xxs)',
}

const chipsStyle: React.CSSProperties = { display: 'contents' }

const chipStyle: React.CSSProperties = {
  display: 'inline-flex',
  alignItems: 'center',
  gap: 'var(--mmerp-space-xxs)',
  padding: '0 var(--mmerp-space-xxs) 0 var(--mmerp-space-xs)',
  borderRadius: 'var(--mmerp-radius-sm)',
  background: 'var(--mmerp-color-gray-light)',
  color: 'var(--mmerp-color-gray-light-color)',
  fontSize: 'var(--mmerp-font-size-xs)',
}

export function MultiSelect({
  label,
  description,
  error,
  required,
  disabled,
  readOnly,
  data,
  value,
  onChange,
  nothingFound,
  placeholder,
  removeLabel,
  className,
  style,
  ...props
}: MultiSelectProps) {
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
      <BaseCombobox.Root<string, true, SelectItem>
        multiple
        items={data}
        value={value}
        onValueChange={onChange}
        filter={filterByLabel}
        disabled={disabled}
        readOnly={readOnly}
      >
        <BaseCombobox.InputGroup
          className={textInputStyles.wrapper}
          style={wrapperStyle}
          data-mmerp-control=""
          data-error={error ? '' : undefined}
          data-disabled={disabled ? '' : undefined}
        >
          <BaseCombobox.Chips style={chipsStyle}>
            {value.map((itemValue) => {
              const item = data.find((candidate) => candidate.value === itemValue)
              if (!item) return null
              return (
                <BaseCombobox.Chip key={itemValue} style={chipStyle} aria-readonly={undefined}>
                  {item.label}
                  <BaseCombobox.ChipRemove aria-label={removeLabel(item.label)}>
                    <CloseIcon size={8} />
                  </BaseCombobox.ChipRemove>
                </BaseCombobox.Chip>
              )
            })}
          </BaseCombobox.Chips>
          <BaseCombobox.Input
            placeholder={value.length === 0 ? placeholder : undefined}
            className={textInputStyles.input}
            style={{ ...style, flex: '1 1 auto', minWidth: '4ch' }}
            aria-describedby={describedBy}
            aria-invalid={error ? true : undefined}
            aria-required={required || undefined}
            {...props}
          />
        </BaseCombobox.InputGroup>
        <BaseCombobox.Portal>
          <BaseCombobox.Positioner sideOffset={4}>
            <BaseCombobox.Popup className={optionStyles.popup}>
              <BaseCombobox.Empty className={optionStyles.empty}>{nothingFound}</BaseCombobox.Empty>
              <BaseCombobox.List>
                {(item: SelectItem) => (
                  <BaseCombobox.Item key={item.value} value={item.value} disabled={item.disabled} className={optionStyles.option}>
                    {item.label}
                  </BaseCombobox.Item>
                )}
              </BaseCombobox.List>
            </BaseCombobox.Popup>
          </BaseCombobox.Positioner>
        </BaseCombobox.Portal>
      </BaseCombobox.Root>
    </Field>
  )
}
