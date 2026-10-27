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
import styles from './MultiSelect.module.css'

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
          className={cx(textInputStyles.wrapper, styles.wrapper)}
          data-mmerp-control=""
          data-error={error ? '' : undefined}
          data-disabled={disabled ? '' : undefined}
        >
          <BaseCombobox.Chips className={styles.chips}>
            {value.map((itemValue) => {
              const item = data.find((candidate) => candidate.value === itemValue)
              if (!item) return null
              return (
                <BaseCombobox.Chip key={itemValue} className={styles.chip} aria-readonly={undefined}>
                  <span className={styles.label}>{item.label}</span>
                  {!disabled && !readOnly && (
                    <BaseCombobox.ChipRemove className={styles.remove} aria-label={removeLabel(item.label)}>
                      <CloseIcon />
                    </BaseCombobox.ChipRemove>
                  )}
                </BaseCombobox.Chip>
              )
            })}
          </BaseCombobox.Chips>
          <BaseCombobox.Input
            placeholder={value.length === 0 ? placeholder : undefined}
            className={cx(textInputStyles.input, styles.input)}
            style={style}
            aria-describedby={describedBy}
            aria-invalid={error ? true : undefined}
            aria-required={required || undefined}
            {...props}
            onKeyDown={(event) => {
              props.onKeyDown?.(event)
              // With the list shut Base UI would clear every chosen value; leaving a field must not lose them.
              if (event.key === 'Escape' && event.currentTarget.getAttribute('aria-expanded') !== 'true') event.preventBaseUIHandler()
            }}
          />
        </BaseCombobox.InputGroup>
        <BaseCombobox.Portal>
          <BaseCombobox.Positioner sideOffset={4} className={optionStyles.positioner}>
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
