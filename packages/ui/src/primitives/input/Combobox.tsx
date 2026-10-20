import { Combobox as BaseCombobox } from '@base-ui/react/combobox'
import type * as React from 'react'
import { useEffect, useId, useState } from 'react'
import { cx } from '../../internal/cx.ts'
import { Loader } from '../feedback/Loader.tsx'
import { Field, type FieldProps } from './Field.tsx'
import { matchesQuery } from './matchesQuery.ts'
import optionStyles from './options.module.css'
import type { SelectItem } from './Select.tsx'
import textInputStyles from './TextInput.module.css'

type ComboboxClearProps = { clearable?: false; clearLabel?: undefined } | { clearable: true; clearLabel: string }

export type ComboboxProps = FieldProps &
  Omit<React.ComponentPropsWithRef<'input'>, 'value' | 'onChange' | 'size'> &
  ComboboxClearProps & {
    data: SelectItem[]
    value: string | null
    onChange(value: string | null): void
    nothingFound: React.ReactNode
    placeholder?: string
    filter?: (item: SelectItem, query: string) => boolean
    onSearchChange?(query: string): void
    loading?: boolean
    loadingLabel?: string
  }

function defaultFilter(item: SelectItem, query: string) {
  return matchesQuery(item.label, query)
}

function ClearIcon() {
  return (
    <svg width="10" height="10" viewBox="0 0 10 10" aria-hidden="true">
      <path d="M1 1 9 9M9 1 1 9" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" />
    </svg>
  )
}

export function Combobox({
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
  filter = defaultFilter,
  onSearchChange,
  loading,
  loadingLabel,
  clearable,
  clearLabel,
  className,
  style,
  onKeyDown,
  onBlur,
  ...props
}: ComboboxProps) {
  const selectedLabel = data.find((item) => item.value === value)?.label ?? ''
  const [query, setQuery] = useState(selectedLabel)

  // The current value's label is the source of truth for the text whenever selection settles
  // (a fresh value from outside, or our own revert-on-escape/blur below).
  useEffect(() => {
    setQuery(selectedLabel)
  }, [selectedLabel])

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
      <BaseCombobox.Root<string, false, SelectItem>
        items={data}
        value={value}
        onValueChange={onChange}
        inputValue={query}
        onInputValueChange={(next) => {
          setQuery(next)
          onSearchChange?.(next)
        }}
        filter={filter}
        // Base UI can't resolve a plain string value back to its label on its own; this is
        // what it uses internally to restore the input text once the popup closes.
        itemToStringLabel={(itemValue) => data.find((item) => item.value === itemValue)?.label ?? ''}
        disabled={disabled}
        readOnly={readOnly}
      >
        <BaseCombobox.InputGroup
          className={textInputStyles.wrapper}
          data-mmerp-control=""
          data-error={error ? '' : undefined}
          data-disabled={disabled ? '' : undefined}
        >
          <BaseCombobox.Input
            placeholder={placeholder}
            className={textInputStyles.input}
            style={style}
            aria-describedby={describedBy}
            aria-invalid={error ? true : undefined}
            aria-required={required || undefined}
            onKeyDown={(event) => {
              onKeyDown?.(event)
              // Base UI only closes the popup on Escape; restoring the label is ours to do.
              if (event.key === 'Escape') setQuery(selectedLabel)
            }}
            onBlur={(event) => {
              onBlur?.(event)
              if (clearable && query.trim() === '') {
                if (value !== null) onChange(null)
                return
              }
              setQuery(selectedLabel)
            }}
            {...props}
          />
          {loading && (
            <span className={textInputStyles.section} aria-hidden="true">
              <Loader size="xs" label="" />
            </span>
          )}
          {clearable && (
            <BaseCombobox.Clear className={textInputStyles.section} aria-label={clearLabel}>
              <ClearIcon />
            </BaseCombobox.Clear>
          )}
        </BaseCombobox.InputGroup>
        <BaseCombobox.Portal>
          <BaseCombobox.Positioner sideOffset={4}>
            <BaseCombobox.Popup className={optionStyles.popup}>
              <BaseCombobox.Status>{loading ? loadingLabel : null}</BaseCombobox.Status>
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
