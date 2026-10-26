import type * as React from 'react'
import { useMediaQuery } from '../../hooks/useMediaQuery.ts'
import { cx } from '../../internal/cx.ts'
import { UnstyledButton } from '../input/UnstyledButton.tsx'
import styles from './Pagination.module.css'

export interface PaginationLabels {
  previous: string
  next: string
  page(page: number): string
}

export interface PaginationProps extends Omit<React.ComponentPropsWithRef<'nav'>, 'onChange'> {
  total: number
  value: number
  onChange(page: number): void
  labels: PaginationLabels
}

type PaginationRangeItem = number | 'dots'

function range(start: number, end: number): number[] {
  return Array.from({ length: end - start + 1 }, (_, index) => start + index)
}

// Classic truncated-range algorithm: 1 sibling either side of the active page, 1 boundary page at each end.
function paginationRange(total: number, value: number, siblings = 1, boundaries = 1): PaginationRangeItem[] {
  const totalVisible = siblings * 2 + boundaries * 2 + 3
  if (totalVisible >= total) return range(1, total)

  const leftSibling = Math.max(value - siblings, boundaries + 1)
  const rightSibling = Math.min(value + siblings, total - boundaries)

  const showLeftDots = leftSibling > boundaries + 2
  const showRightDots = rightSibling < total - boundaries - 1

  if (!showLeftDots && showRightDots) {
    const leftCount = siblings * 2 + boundaries + 2
    return [...range(1, leftCount), 'dots', ...range(total - boundaries + 1, total)]
  }

  if (showLeftDots && !showRightDots) {
    const rightCount = siblings * 2 + boundaries + 2
    return [...range(1, boundaries), 'dots', ...range(total - rightCount + 1, total)]
  }

  return [...range(1, boundaries), 'dots', ...range(leftSibling, rightSibling), 'dots', ...range(total - boundaries + 1, total)]
}

function PrevIcon() {
  return (
    <svg width="10" height="10" viewBox="0 0 10 10" aria-hidden="true">
      <path d="M6 1 3 5l3 4" stroke="currentColor" strokeWidth="1.4" fill="none" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

function NextIcon() {
  return (
    <svg width="10" height="10" viewBox="0 0 10 10" aria-hidden="true">
      <path d="M4 1l3 4-3 4" stroke="currentColor" strokeWidth="1.4" fill="none" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

// Where controls grow to the 44px touch target; with siblings shown the row would outgrow a phone.
const touchQuery = '(max-width: 63.99em)'

export function Pagination({ total, value, onChange, labels, className, ...props }: PaginationProps) {
  const touch = useMediaQuery(touchQuery)
  const items = total > 0 ? paginationRange(total, value, touch ? 0 : 1) : []
  return (
    <nav className={cx(styles.root, className)} {...props}>
      <ul className={styles.list}>
        <li>
          <UnstyledButton
            data-mmerp-control=""
            className={styles.control}
            disabled={value <= 1}
            aria-label={labels.previous}
            onClick={() => onChange(value - 1)}
          >
            <PrevIcon />
          </UnstyledButton>
        </li>
        {items.map((item, index) =>
          item === 'dots' ? (
            <li key={`dots-${index}`} className={styles.dots} aria-hidden="true">
              …
            </li>
          ) : (
            <li key={item}>
              <UnstyledButton
                data-mmerp-control=""
                className={styles.control}
                data-active={item === value ? '' : undefined}
                aria-label={labels.page(item)}
                aria-current={item === value ? 'page' : undefined}
                onClick={() => onChange(item)}
              >
                {item}
              </UnstyledButton>
            </li>
          ),
        )}
        <li>
          <UnstyledButton
            data-mmerp-control=""
            className={styles.control}
            disabled={value >= total}
            aria-label={labels.next}
            onClick={() => onChange(value + 1)}
          >
            <NextIcon />
          </UnstyledButton>
        </li>
      </ul>
    </nav>
  )
}
