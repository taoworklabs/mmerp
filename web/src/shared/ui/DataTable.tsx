import { DataTable as KitDataTable, type DataTableProps as KitDataTableProps } from '@mmerp/ui'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'

export type { Column, ColumnKind, ColumnRole, Selection } from '@mmerp/ui'

export type DataTableProps<T> = Omit<KitDataTableProps<T>, 'renderLink' | 'labels' | 'appearance'> & {
  // The first column links here and the whole row is clickable; without it rows are plain.
  href?: (row: T) => string
}

// DataTable is the only table areas use: the kit's table with router links and the app's labels.
export function DataTable<T>({ href, ...props }: DataTableProps<T>) {
  const { t } = useTranslation()
  return (
    <KitDataTable
      {...props}
      renderLink={href && ((row, children) => <Link to={href(row)}>{children}</Link>)}
      labels={{
        actions: t('shared.table.actions'),
        selectAll: t('shared.table.select_all'),
        selectRow: t('shared.table.select_row'),
        resize: (column) => t('shared.table.resize', { column }),
      }}
    />
  )
}
