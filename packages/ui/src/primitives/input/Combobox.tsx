import { Combobox as BaseCombobox } from '@base-ui/react/combobox'
import type * as React from 'react'
import { useId, useRef, useState } from 'react'
import { ClearButton } from '../../internal/ClearButton.tsx'
import { cx } from '../../internal/cx.ts'
import { mergeRefs } from '../../internal/mergeRefs.ts'
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
  ref,
  ...props
}: ComboboxProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  const selectedLabel = data.find((item) => item.value === value)?.label ?? ''
  const [query, setQuery] = useState(selectedLabel)
  const [open, setOpen] = useState(false)
  const [synced, setSynced] = useState({ value, label: selectedLabel })
  // The text follows the value's label when the value changes, or when the label does while the
  // user isn't searching: remote results that drop the selected item must not wipe typed text.
  if (value !== synced.value || (!open && selectedLabel !== synced.label)) {
    setSynced({ value, label: selectedLabel })
    setQuery(selectedLabel)
  }

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
        onOpenChange={(next) => setOpen(next)}
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
            ref={mergeRefs(inputRef, ref)}
          />
          {loading && (
            <span className={textInputStyles.section} aria-hidden="true">
              <Loader size="xs" label="" />
            </span>
          )}
          {clearable && value !== null && (
            <ClearButton
              label={clearLabel}
              onClear={() => {
                setQuery('')
                onSearchChange?.('')
                onChange(null)
              }}
              focusTarget={inputRef}
              disabled={disabled}
              readOnly={readOnly}
            />
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
