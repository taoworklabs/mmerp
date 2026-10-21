import { Combobox as BaseCombobox } from '@base-ui/react/combobox'
import type * as React from 'react'
import { useId, useState } from 'react'
import { tokens } from '../../tokens/tokens.ts'
import { matchesQuery } from './matchesQuery.ts'
import optionStyles from './options.module.css'
import type { SelectItem } from './Select.tsx'
import textInputStyles from './TextInput.module.css'

export type ComboboxButtonProps = {
  // Rendered as the Base UI trigger: its own onClick/aria attributes are merged onto it.
  trigger: React.ReactElement
  data: SelectItem[]
  value: string | null
  onChange(value: string): void
  nothingFound: React.ReactNode
  searchable?: boolean
  // aria-label and placeholder of the search box; required when searchable.
  searchLabel?: string
  renderOption?: (item: SelectItem) => React.ReactNode
  filter?: (item: SelectItem, query: string) => boolean
  popupWidth?: number
}

function defaultFilter(item: SelectItem, query: string) {
  return matchesQuery(item.label, query)
}

// The popup always needs an input to drive keyboard navigation (Base UI focuses it on open and on
// ArrowDown/ArrowUp from the trigger); without `searchable` it stays mounted but invisible.
const hiddenInputStyle: React.CSSProperties = {
  position: 'absolute',
  width: 1,
  height: 1,
  margin: -1,
  padding: 0,
  overflow: 'hidden',
  clip: 'rect(0, 0, 0, 0)',
  whiteSpace: 'nowrap',
  border: 0,
}

const searchWrapperStyle: React.CSSProperties = { marginBottom: 'var(--mmerp-space-xxs)' }

// A button that opens a searchable, single-select list — the look FilterChip (and anything else
// that picks one value from a button rather than a text field) builds its popup on.
export function ComboboxButton({
  trigger,
  data,
  value,
  onChange,
  nothingFound,
  searchable,
  searchLabel,
  renderOption,
  filter = defaultFilter,
  popupWidth = tokens.filterWidth.menu,
}: ComboboxButtonProps) {
  const [query, setQuery] = useState('')
  const triggerId = useId()
  return (
    <BaseCombobox.Root<string, false, SelectItem>
      items={data}
      value={value}
      onValueChange={(next) => {
        if (next !== null) onChange(next)
      }}
      inputValue={query}
      onInputValueChange={setQuery}
      filter={filter}
      itemToStringLabel={(itemValue) => data.find((item) => item.value === itemValue)?.label ?? ''}
      onOpenChange={(open) => {
        if (!open) setQuery('')
      }}
    >
      <BaseCombobox.Trigger id={triggerId} render={trigger} />
      <BaseCombobox.Portal>
        <BaseCombobox.Positioner sideOffset={4}>
          {/* The popup takes the role of a dialog around the input (see `inputInsidePopup` in the
              Combobox primitive); a dialog needs its own accessible name, borrowed from the trigger. */}
          <BaseCombobox.Popup className={optionStyles.popup} style={{ width: popupWidth }} aria-labelledby={triggerId}>
            {searchable ? (
              <BaseCombobox.InputGroup className={textInputStyles.wrapper} style={searchWrapperStyle} data-mmerp-control="">
                <BaseCombobox.Input placeholder={searchLabel} aria-label={searchLabel} className={textInputStyles.input} />
              </BaseCombobox.InputGroup>
            ) : (
              <BaseCombobox.Input style={hiddenInputStyle} />
            )}
            <BaseCombobox.Empty className={optionStyles.empty}>{nothingFound}</BaseCombobox.Empty>
            <BaseCombobox.List>
              {(item: SelectItem) => (
                <BaseCombobox.Item key={item.value} value={item.value} disabled={item.disabled} className={optionStyles.option}>
                  {renderOption ? renderOption(item) : item.label}
                </BaseCombobox.Item>
              )}
            </BaseCombobox.List>
          </BaseCombobox.Popup>
        </BaseCombobox.Positioner>
      </BaseCombobox.Portal>
    </BaseCombobox.Root>
  )
}
