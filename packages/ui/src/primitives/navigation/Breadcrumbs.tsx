import { Children } from 'react'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import styles from './Breadcrumbs.module.css'

export interface BreadcrumbsProps extends Omit<React.ComponentPropsWithRef<'nav'>, 'children'> {
  'aria-label': string
  separator?: React.ReactNode
  children: React.ReactNode
  // False for a trail that stops short of the page itself (e.g. a new-record form showing only
  // its parent list): the last item is then a link like any other, not the current page.
  currentLast?: boolean
}

export function Breadcrumbs({ separator = '/', className, children, currentLast = true, ...props }: BreadcrumbsProps) {
  const items = Children.toArray(children)
  return (
    <nav className={cx(styles.root, className)} {...props}>
      <ol className={styles.list}>
        {items.map((item, index) => {
          const isLast = index === items.length - 1
          return (
            <li key={index} className={styles.item} aria-current={isLast && currentLast ? 'page' : undefined}>
              {item}
              {!isLast && (
                <span className={styles.separator} aria-hidden="true">
                  {separator}
                </span>
              )}
            </li>
          )
        })}
      </ol>
    </nav>
  )
}
