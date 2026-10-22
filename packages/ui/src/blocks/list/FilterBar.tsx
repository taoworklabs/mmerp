import type * as React from 'react'
import { useMediaQuery } from '../../hooks/useMediaQuery.ts'
import { cx } from '../../internal/cx.ts'
import { SearchIcon } from '../../internal/icons.tsx'
import { breakpoint } from '../../tokens/tokens.ts'
import { TextInput, type TextInputProps } from '../../primitives/input/TextInput.tsx'
import styles from './List.module.css'

export interface FilterBarProps {
  filters: React.ReactNode
  // Saved views, columns: stays at the bar's right end, only from 64em.
  tools?: React.ReactNode
  // While rows are selected, replaces the whole bar instead of sitting beside the filters — but
  // only once there's room for it; below 64em the filters stay in charge.
  bulk?: React.ReactNode
  label: string
}

const wideQuery = `(min-width: ${breakpoint.md})`

function FilterBarRoot({ filters, tools, bulk, label }: FilterBarProps) {
  const wide = useMediaQuery(wideQuery, true)
  if (wide && bulk) return <>{bulk}</>
  return (
    <div role="search" aria-label={label} className={styles.bar}>
      <div className={styles.filters}>{filters}</div>
      {wide && tools && <div className={styles.tools}>{tools}</div>}
    </div>
  )
}

export type FilterBarSearchProps = Omit<TextInputProps, 'type'>

// The bar's own search box: full width on its own line below 64em, sharing the row with the
// chips above it. Applying the typed value (on Enter/blur, the "/" shortcut) is the caller's.
function Search({ className, ...props }: FilterBarSearchProps) {
  return <TextInput type="search" leftSection={<SearchIcon />} className={cx(styles.search, className)} {...props} />
}

export const FilterBar = Object.assign(FilterBarRoot, { Search })
