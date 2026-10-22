import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import { ChevronDownIcon, CloseIcon } from '../../internal/icons.tsx'
import { Button } from '../../primitives/input/Button.tsx'
import { ComboboxButton } from '../../primitives/input/ComboboxButton.tsx'
import type { SelectItem } from '../../primitives/input/Select.tsx'
import { Tooltip } from '../../primitives/overlay/Tooltip.tsx'
import styles from './List.module.css'

export interface FilterChipProps extends Omit<React.ComponentPropsWithRef<'div'>, 'children' | 'onChange'> {
  label: string
  icon: React.ReactNode
  data: SelectItem[]
  value: string | null
  onChange(value: string | null): void
  // How the chosen value reads once set, e.g. "Legal entity: Acme Co".
  valueText(label: string, value: string): string
  clearLabel: string
  nothingFound: React.ReactNode
  // A filter that always holds a value (a legal entity, a year) is never cleared nor tinted.
  clearable?: boolean
  searchable?: boolean
  searchLabel?: string
  renderOption?: (item: SelectItem) => React.ReactNode
}

// FilterChip is one filter of a FilterBar: "Label" and a chevron while unset, "Label: value" tinted
// with its own clear button once set, so what the list is filtered by stays visible.
export function FilterChip({
  label,
  icon,
  data,
  value,
  onChange,
  valueText,
  clearLabel,
  nothingFound,
  clearable = true,
  searchable,
  searchLabel,
  renderOption,
  className,
  ...props
}: FilterChipProps) {
  const chosen = data.find((item) => item.value === value)?.label ?? null
  const active = chosen !== null && clearable
  const variant = active ? 'light' : 'default'
  const text = chosen !== null ? valueText(label, chosen) : label
  return (
    <Button.Group className={cx(styles.chip, className)} {...props}>
      <ComboboxButton
        trigger={
          // role="combobox" takes no accessible name from its content, so the visible text doubles as the label.
          <Button
            aria-label={text}
            variant={variant}
            leftSection={icon}
            rightSection={!active && <ChevronDownIcon />}
            title={chosen !== null ? text : undefined}
            className={styles.main}
          >
            <span className={styles.text}>{text}</span>
          </Button>
        }
        data={data}
        value={value}
        onChange={onChange}
        nothingFound={nothingFound}
        searchable={searchable}
        searchLabel={searchLabel}
        renderOption={renderOption}
      />
      {active && (
        <Tooltip label={clearLabel}>
          <Button variant={variant} aria-label={clearLabel} onClick={() => onChange(null)} className={styles.clear}>
            <CloseIcon />
          </Button>
        </Tooltip>
      )}
    </Button.Group>
  )
}
