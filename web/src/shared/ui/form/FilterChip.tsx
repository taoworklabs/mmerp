import { FilterChip as KitFilterChip } from '@mmerp/ui'
import type { Icon as TablerIcon } from '@tabler/icons-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { icon as iconSize } from '../theme'
import type { Option } from './fields'

type Props = {
  label: string
  // The chip sizes it like any page-level button's icon.
  icon: TablerIcon
  data: Option[]
  value: string | null
  onChange: (value: string | null) => void
  // A filter that always holds a value (a legal entity, a year) is never cleared nor tinted.
  clearable?: boolean
  // A search box above the options, for long lists.
  searchable?: boolean
  renderOption?: (option: Option) => ReactNode
}

// FilterChip is one filter of a list's filter bar, picked from a list of options.
export function FilterChip({ label, icon: Icon, ...props }: Props) {
  const { t } = useTranslation()
  return (
    <KitFilterChip
      {...props}
      label={label}
      icon={<Icon {...iconSize.button} />}
      valueText={(name, value) => t('shared.list.filter_value', { name, value })}
      clearLabel={t('shared.list.clear_filter', { name: label })}
      nothingFound={t('shared.list.no_options')}
      searchLabel={label}
    />
  )
}
