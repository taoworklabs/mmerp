import { Button, Combobox, Tooltip, useCombobox } from '@mantine/core'
import { IconChevronDown, IconX, type Icon as TablerIcon } from '@tabler/icons-react'
import { useState, type ReactElement, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { filterWidth, icon } from '../theme'
import classes from './FilterChip.module.css'
import type { Option } from './fields'

type ChipProps = {
  label: string
  // The chip sizes it like any page-level button's icon.
  icon: TablerIcon
  // The chosen value's label; null while the filter is unset.
  chosen: string | null
  // A filter that always holds a value (a legal entity, a year) is never cleared nor tinted.
  clearable: boolean
  onClear: () => void
  onToggle: () => void
  // Wraps the main button in its dropdown's target.
  target: (button: ReactElement) => ReactNode
}

// ChipShell is the look every filter chip shares: "Name" and a chevron while unset; "Name: value",
// tinted, with its own clear button once set, so what the list is filtered by stays visible.
export function ChipShell({ label, icon: Icon, chosen, clearable, onClear, onToggle, target }: ChipProps) {
  const { t } = useTranslation()
  const active = chosen !== null && clearable
  const variant = active ? 'light' : 'default'
  const text = chosen !== null ? t('shared.list.filter_value', { name: label, value: chosen }) : label
  const clear = t('shared.list.clear_filter', { name: label })
  return (
    <Button.Group className={classes.chip}>
      {target(
        <Button
          className={classes.main}
          variant={variant}
          leftSection={<Icon {...icon.button} />}
          rightSection={!active && <IconChevronDown {...icon.text} />}
          maw={filterWidth.chip}
          classNames={{ label: classes.label }}
          title={chosen !== null ? text : undefined}
          onClick={onToggle}
        >
          <span className={classes.text}>{text}</span>
        </Button>,
      )}
      {active && (
        <Tooltip label={clear}>
          <Button variant={variant} px="xs" className={classes.clear} aria-label={clear} onClick={onClear}>
            <IconX {...icon.button} />
          </Button>
        </Tooltip>
      )}
    </Button.Group>
  )
}

type Props = {
  label: string
  icon: TablerIcon
  data: Option[]
  value: string | null
  onChange: (value: string | null) => void
  clearable?: boolean
  // A search box above the options, for long lists.
  searchable?: boolean
  renderOption?: (option: Option) => ReactNode
}

// FilterChip is one filter of a list's filter bar, picked from a list of options.
export function FilterChip({ label, icon, data, value, onChange, clearable = true, searchable, renderOption }: Props) {
  const { t } = useTranslation()
  const [search, setSearch] = useState('')
  const combobox = useCombobox({
    onDropdownClose: () => {
      combobox.resetSelectedOption()
      setSearch('')
    },
    onDropdownOpen: () => searchable && combobox.focusSearchInput(),
  })
  const needle = search.trim().toLowerCase()
  const shown = needle ? data.filter((o) => o.label.toLowerCase().includes(needle)) : data
  return (
    <Combobox
      store={combobox}
      position="bottom-start"
      width={filterWidth.menu}
      onOptionSubmit={(v) => {
        onChange(v)
        combobox.closeDropdown()
      }}
    >
      <ChipShell
        label={label}
        icon={icon}
        chosen={data.find((o) => o.value === value)?.label ?? null}
        clearable={clearable}
        onClear={() => onChange(null)}
        onToggle={() => combobox.toggleDropdown()}
        target={(button) => <Combobox.Target targetType="button">{button}</Combobox.Target>}
      />
      <Combobox.Dropdown>
        {searchable && <Combobox.Search value={search} onChange={(e) => setSearch(e.currentTarget.value)} placeholder={label} aria-label={label} />}
        <Combobox.Options className={classes.options}>
          {shown.length === 0 ? (
            <Combobox.Empty>{t('shared.list.no_options')}</Combobox.Empty>
          ) : (
            shown.map((o) => (
              <Combobox.Option key={o.value} value={o.value} data-checked={o.value === value || undefined} aria-selected={o.value === value}>
                {renderOption ? renderOption(o) : o.label}
              </Combobox.Option>
            ))
          )}
        </Combobox.Options>
      </Combobox.Dropdown>
    </Combobox>
  )
}
