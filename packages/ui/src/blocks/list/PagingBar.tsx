import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import { Select } from '../../primitives/input/Select.tsx'
import { Pagination, type PaginationLabels } from '../../primitives/navigation/Pagination.tsx'
import { Text } from '../../primitives/typography/Text.tsx'
import styles from './List.module.css'

export interface PagingBarProps extends Omit<React.ComponentPropsWithRef<'div'>, 'children'> {
  total: number
  page: number
  pageSize: number
  pageSizes: number[]
  onPage(page: number): void
  onPageSize(size: number): void
  totalText: string
  pageSizeText(n: number): string
  pageSizeLabel: string
  labels: PaginationLabels
}

// PagingBar sits under a list's table: the total on the left, the pages centred, the page size on
// the right — all three stay put whatever the total's or the page size's own width.
export function PagingBar({
  total,
  page,
  pageSize,
  pageSizes,
  onPage,
  onPageSize,
  totalText,
  pageSizeText,
  pageSizeLabel,
  labels,
  className,
  ...props
}: PagingBarProps) {
  if (total === 0) return null
  return (
    <div className={cx(styles.paging, className)} {...props}>
      <Text size="sm" tone="dimmed">
        {totalText}
      </Text>
      <Pagination total={Math.ceil(total / pageSize)} value={page} onChange={onPage} labels={labels} />
      <Select
        aria-label={pageSizeLabel}
        data={pageSizes.map((n) => ({ value: String(n), label: pageSizeText(n) }))}
        value={String(pageSize)}
        onChange={(v) => v && onPageSize(Number(v))}
        className={styles.pageSize}
      />
    </div>
  )
}
